// Command kalimate is a minimal macOS kalimba tuner.
//
// The code is split by concern:
//   - internal/audio  interfaces with capture devices (streams raw samples)
//   - internal/pitch  calculates (pitch detection)
//   - internal/ui     displays and owns the target-first tuning rules
//
// This file wires those together and owns the device lifecycle.
package main

import (
	"log"
	"sync"

	"fyne.io/fyne/v2/app"

	"kalimate/internal/audio"
	"kalimate/internal/pitch"
	"kalimate/internal/ui"
)

const (
	sampleRate = 44100
	hopSize    = 1024 // ~23 ms between readings; overlaps the analysis window
)

type controller struct {
	view *ui.View
	ctrl *ui.Controller

	mu      sync.Mutex
	devices []audio.Device
	cap     *audio.Capture
	stop    chan struct{}
}

func main() {
	a := app.NewWithID("io.github.niespodd.kalimate")
	a.SetIcon(appIcon)

	view := ui.New(a)
	c := &controller{view: view, ctrl: ui.NewController(view)}
	c.ctrl.RestartCapture = c.switchDevice
	c.ctrl.RefreshDevices = c.refreshDevices

	// Enumerate devices up front (no UI calls — safe before the loop starts).
	names, defaultIdx := c.enumerate()

	// Start capture and publish the initial UI only once the event loop runs:
	// the controller's renders use fyne.Do, invalid before the app has started.
	a.Lifecycle().SetOnStarted(func() {
		c.ctrl.SetDevices(names, defaultIdx)
		if err := c.switchDevice(defaultIdx); err != nil {
			log.Printf("could not open input: %v", err)
			c.ctrl.SetMicError(true)
		}
	})

	win := c.view.Window()
	win.SetIcon(appIcon)
	win.SetCloseIntercept(func() {
		c.shutdown()
		win.Close()
	})
	win.ShowAndRun()
}

// enumerate refreshes the device list and returns (names, defaultIndex).
func (c *controller) enumerate() ([]string, int) {
	devices, err := audio.ListDevices()
	if err != nil {
		log.Printf("could not list input devices: %v", err)
	}
	c.mu.Lock()
	c.devices = devices
	c.mu.Unlock()

	names := make([]string, len(devices))
	defaultIdx := 0
	for i, d := range devices {
		names[i] = d.Name
		if d.IsDefault {
			defaultIdx = i
		}
	}
	return names, defaultIdx
}

// refreshDevices re-enumerates inputs and updates the menu.
func (c *controller) refreshDevices() {
	names, defaultIdx := c.enumerate()
	c.ctrl.SetDevices(names, defaultIdx)
}

// switchDevice stops any running capture and starts a fresh one on the chosen
// device, with its processing loop.
func (c *controller) switchDevice(index int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var dev *audio.Device
	if index >= 0 && index < len(c.devices) {
		dev = &c.devices[index]
	}

	// Open the new device before stopping the old one, so a failed switch leaves
	// the existing capture running instead of nothing.
	capture, err := audio.Open(dev, sampleRate, hopSize)
	if err != nil {
		return err
	}
	c.stopLocked()
	c.cap = capture
	c.stop = make(chan struct{})
	go c.loop(capture, c.stop)
	return nil
}

// loop reads audio hops, detects pitch, and feeds raw (freq, level, ok) to the
// UI controller, which owns all state and rendering.
func (c *controller) loop(capture *audio.Capture, stop chan struct{}) {
	analyzer := pitch.NewAnalyzer(sampleRate)
	tracker := pitch.NewPluckTracker(float64(hopSize) / sampleRate)
	dropped := capture.Dropped()
	for {
		select {
		case <-stop:
			return
		case hop := <-capture.Frames:
			// A dropped hop breaks window continuity; restart the analyzer so YIN
			// never splices samples across the gap into one garbage window.
			if d := capture.Dropped(); d != dropped {
				dropped = d
				analyzer = pitch.NewAnalyzer(sampleRate)
			}
			r, ok := analyzer.Push(hop)
			if !ok {
				continue
			}
			res := tracker.Push(r)
			// Freq and the valid flag both come from the tracker, so they describe
			// the same measurement: the current pluck's estimate, or nothing while
			// no pluck is ringing. Level stays the raw frame amplitude.
			c.ctrl.Feed(res.Freq, r.Amp, res.HasNote && res.Listening)
		}
	}
}

func (c *controller) stopLocked() {
	if c.stop != nil {
		close(c.stop)
		c.stop = nil
	}
	if c.cap != nil {
		c.cap.Close()
		c.cap = nil
	}
}

func (c *controller) shutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopLocked()
}
