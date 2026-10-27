package ui

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
)

const (
	idleTimeout = 600 * time.Millisecond  // no fresh pitch this long → no signal
	heardHold   = 1500 * time.Millisecond // keep the heard dot this long after signal drops
	doneHold    = 400 * time.Millisecond  // continuously in tune this long → Done
	centsAlpha  = 0.30                    // cents EMA factor per frame

	// attributeCents is how far a pluck may sit from the target and still be
	// read as the target (push/pull guidance) rather than resolving to the
	// nearest key. Semitone neighbours (E4/F4, B4/C5, E5/F5, B5/C6) are 100¢
	// apart, so a full semitone would let a slightly-flat neighbour masquerade
	// as the target; two-thirds of a semitone still catches realistic detuning
	// while handing an actual neighbour back to the key it belongs to (bug 10).
	attributeCents = 65.0
)

// Controller owns the TunerState and the target-first tuning rules. The audio
// goroutine calls Feed with raw (freq, level, ok); user actions arrive via the
// view's callbacks. It renders through view.Update (fyne.Do), coalesced to 60Hz.
type Controller struct {
	// RestartCapture reopens capture on the given device index; returns an error
	// if the device is unavailable. Wired by the app.
	RestartCapture func(index int) error
	// RefreshDevices re-enumerates input devices. Wired by the app.
	RefreshDevices func()

	view *View

	mu          sync.Mutex
	target      string
	started     bool
	done        map[string]bool
	emaCents    float64
	hasEMA      bool
	lastFresh   time.Time
	heard       string
	lastHeard   time.Time
	greenActive bool
	greenSince  time.Time
	level       float64

	devices  []string
	current  int
	micName  string
	micError bool

	lastRender int64 // unix-nanos, for ≤60 Hz Feed coalescing
	now        func() time.Time
}

// NewController builds the controller and wires the view's user-action callbacks.
func NewController(v *View) *Controller {
	c := &Controller{view: v, done: map[string]bool{}, now: time.Now}
	v.OnSelectTarget = c.SelectTarget
	v.OnNext = c.NextOrReset
	v.OnStepTarget = c.StepTarget
	v.OnDeviceSelected = c.SelectDevice
	v.OnRefreshDevices = func() {
		if c.RefreshDevices != nil {
			c.RefreshDevices()
		}
	}
	return c
}

// Feed ingests one detection frame from the audio goroutine.
func (c *Controller) Feed(freq, level float64, ok bool) {
	c.mu.Lock()
	c.ingest(freq, level, ok)
	st := c.buildState()
	c.mu.Unlock()

	now := c.now().UnixNano()
	if now-atomic.LoadInt64(&c.lastRender) < int64(time.Second/60) {
		return
	}
	atomic.StoreInt64(&c.lastRender, now)
	c.push(st)
}

// ingest applies the tuning rules. Caller holds the lock.
func (c *Controller) ingest(freq, level float64, ok bool) {
	now := c.now()
	c.level = level

	if ok {
		c.lastFresh = now
		heard := nearestTine(freq, Layout) // rule 1: nearest tine, not chromatic

		if c.target == "" { // rule 2: auto-select the first key heard
			c.setTargetLocked(heard.Name)
		}

		// With a target chosen, measure the pluck against IT: fold YIN octave
		// errors toward the target (so C5/C6 read as C4 — bug 8) and keep
		// attributing the pitch to the target while it is within attributeCents,
		// so a tine detuned toward its semitone neighbour (E4/F4, B4/C5, E5/F5,
		// B5/C6) still gets push/pull guidance instead of reading as the wrong
		// key (bug 7) — but a genuine neighbour, ~100¢ away, is no longer
		// swallowed (bug 10).
		if t, isTine := TineByName(c.target); isTine {
			folded := freq
			for folded < t.FreqHz/1.5 {
				folded *= 2
			}
			for folded > t.FreqHz*1.5 {
				folded /= 2
			}
			if math.Abs(centsBetween(folded, t.FreqHz)) < attributeCents {
				heard, freq = t, folded
			} else {
				heard = nearestTine(freq, Layout) // genuinely a different key
			}
		}
		c.heard = heard.Name
		c.lastHeard = now

		if c.heard == c.target { // rule 3: measure against the target
			cents := centsBetween(freq, heard.FreqHz)
			if c.hasEMA {
				c.emaCents += centsAlpha * (cents - c.emaCents)
			} else {
				c.emaCents, c.hasEMA = cents, true
			}
		} else { // rule 4: wrong key — don't feed the meter
			c.hasEMA = false
		}
	}

	hasSignal := !c.lastFresh.IsZero() && now.Sub(c.lastFresh) < idleTimeout
	if !hasSignal {
		c.hasEMA = false // rule 3: reset EMA when the signal drops
	}

	// rule 7: a target becomes Done after staying in tune continuously.
	inTune := hasSignal && c.target != "" && c.heard == c.target &&
		c.hasEMA && math.Abs(c.emaCents) <= InTuneCents
	if inTune {
		if !c.greenActive {
			c.greenActive, c.greenSince = true, now
		} else if now.Sub(c.greenSince) >= doneHold {
			c.done[c.target] = true
		}
	} else {
		c.greenActive = false
	}
}

func (c *Controller) buildState() TunerState {
	now := c.now()
	hasSignal := !c.lastFresh.IsZero() && now.Sub(c.lastFresh) < idleTimeout

	heardShown := ""
	if !c.lastHeard.IsZero() && now.Sub(c.lastHeard) < heardHold {
		heardShown = c.heard // rule 5: hold the dot briefly after signal drops
	}

	var status map[string]TineStatus
	if c.started {
		status = make(map[string]TineStatus, len(Layout))
		for _, t := range Layout {
			if c.done[t.Name] {
				status[t.Name] = Done
			} else {
				status[t.Name] = Pending
			}
		}
	}

	cents := 0.0
	if c.hasEMA {
		cents = c.emaCents
	}
	return TunerState{
		Target:    c.target,
		Heard:     heardShown,
		Cents:     cents,
		HasSignal: hasSignal,
		Level:     levelFromAmp(c.level),
		Status:    status,
		MicName:   c.micName,
		MicError:  c.micError,
	}
}

// setTargetLocked selects a key without touching its Done status (rule 9).
func (c *Controller) setTargetLocked(name string) {
	if _, ok := TineByName(name); !ok {
		return
	}
	c.target = name
	c.started = true
	c.hasEMA = false
	c.greenActive = false
}

// SelectTarget is invoked when the user taps a key.
func (c *Controller) SelectTarget(name string) {
	c.mu.Lock()
	c.setTargetLocked(name)
	st := c.buildState()
	c.mu.Unlock()
	c.push(st)
}

// StepTarget moves the selection one tine left (delta<0) or right (delta>0) in
// physical order, wrapping at the ends. With nothing selected yet, the first
// press lands on the center (lowest) key.
func (c *Controller) StepTarget(delta int) {
	c.mu.Lock()
	idx := len(Layout) / 2
	for i, t := range Layout {
		if t.Name == c.target {
			idx = (i + delta + len(Layout)) % len(Layout)
			break
		}
	}
	c.setTargetLocked(Layout[idx].Name)
	st := c.buildState()
	c.mu.Unlock()
	c.push(st)
}

// NextOrReset advances to the next pending key, or starts over when all are done.
func (c *Controller) NextOrReset() {
	c.mu.Lock()
	next, allDone := nextPending(c.target, func(n string) bool { return c.done[n] })
	if allDone {
		c.done = map[string]bool{} // "Start over"
	} else if next != "" {
		c.setTargetLocked(next)
	}
	st := c.buildState()
	c.mu.Unlock()
	c.push(st)
}

// SelectDevice switches the input device and restarts capture.
func (c *Controller) SelectDevice(idx int) {
	c.mu.Lock()
	c.current = idx
	if idx >= 0 && idx < len(c.devices) {
		c.micName = c.devices[idx]
	}
	c.hasEMA = false
	st := c.buildState()
	c.mu.Unlock()
	c.push(st) // reflect the chosen device immediately

	// Opening a device can block; keep it off the UI (menu-callback) thread.
	go func() {
		var err error
		if c.RestartCapture != nil {
			err = c.RestartCapture(idx)
		}
		c.mu.Lock()
		c.micError = err != nil
		moved := err == nil && idx >= 0 && idx < len(c.devices)
		devices := c.devices
		st := c.buildState()
		c.mu.Unlock()
		fyne.Do(func() {
			if moved {
				c.view.SetDeviceList(devices, idx) // move the checkmark
			}
			c.view.Update(st)
		})
	}()
}

// SetDevices sets the available input list and the current selection, and
// renders the initial state. Safe to call once the app has started.
func (c *Controller) SetDevices(names []string, current int) {
	c.mu.Lock()
	c.devices = names

	// Preserve the active selection across a refresh: only honour the provided
	// (default) index when nothing is selected yet, so refreshing the list does
	// not move the checkmark off the device capture is actually using (bug 16).
	sel := current
	if c.micName != "" {
		for i, n := range names {
			if n == c.micName {
				sel = i
				break
			}
		}
	}
	c.current = sel
	if sel >= 0 && sel < len(names) {
		c.micName = names[sel]
	} else if len(names) == 0 {
		c.micName = "System default" // default device opened, but none enumerated (bug 20)
	}
	st := c.buildState()
	c.mu.Unlock()

	fyne.Do(func() {
		c.view.SetDeviceList(names, sel)
		c.view.Update(st)
	})
}

// SetMicError flags capture as unavailable (e.g. permission denied at startup).
func (c *Controller) SetMicError(bad bool) {
	c.mu.Lock()
	c.micError = bad
	st := c.buildState()
	c.mu.Unlock()
	c.push(st)
}

func (c *Controller) push(st TunerState) {
	fyne.Do(func() { c.view.Update(st) })
}
