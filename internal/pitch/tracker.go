package pitch

import (
	"math"
	"sort"
)

// PluckTracker turns a stream of per-hop Readings into a stable, confident
// measurement by modeling how a kalimba actually sounds.
//
// A plucked metal tine has a sharp, noisy attack, then a long exponentially
// decaying tone whose fundamental is cleanest in the first few hundred
// milliseconds. The tracker therefore:
//
//  1. detects each pluck as an onset event (adaptive noise floor + hysteresis),
//  2. discards the attack transient (where the tine's pitch is momentarily
//     sharp — the "ping"),
//  3. collects pitch estimates over the sustain, folding octave jumps back
//     toward the center,
//  4. reduces the whole pluck to one frequency via an outlier-robust median
//     (deliberately *not* amplitude-weighted, which would bias toward the
//     sharp attack), and
//  5. aggregates several plucks of the same note and reports their median —
//     so the reading only settles after a few consistent tries instead of
//     jumping around on every tap.
type PluckTracker struct {
	needed     int
	attackSkip int // hops to ignore after onset (attack transient)

	noiseFloor float64
	prevAmp    float64 // amp one hop before lastAmp; a re-pluck's rise spans two hops
	lastAmp    float64
	armed      bool // amplitude has dropped low enough to accept a new onset
	inPluck    bool
	sinceOnset int
	curFreqs   []float64

	name   string    // note name the accumulator is collecting
	plucks []float64 // one frequency per captured pluck of that note
}

// Tracker tuning.
const (
	neededPlucks    = 3
	minPluckSamples = 4    // a pluck with fewer good hops is ignored as noise
	maxPluckSamples = 50   // stop sampling the long quiet tail
	maxAccum        = 6    // sliding history of recent plucks
	attackMs        = 90.0 // skip the attack "ping", where the tine reads sharp

	minOnsetAmp   = 0.008   // absolute onset floor regardless of noise estimate
	minReleaseAmp = rmsGate // end a pluck no later than detection's silence floor
	onsetMult     = 5.0     // onset must exceed noiseFloor × this
	releaseMult   = 2.5
	noiseEMA      = 0.02
	repluckRise   = 1.8 // a mid-decay jump this much above the last hop = new pluck
)

// NewPluckTracker builds a tracker. hopDur is the real time between Readings,
// used to size the attack-rejection window.
func NewPluckTracker(hopDur float64) *PluckTracker {
	skip := int(math.Round(attackMs / 1000 / hopDur))
	if skip < 1 {
		skip = 1
	}
	return &PluckTracker{needed: neededPlucks, attackSkip: skip, armed: true}
}

// Result is the current measurement for display.
type Result struct {
	HasNote   bool
	Name      string
	Cents     float64
	Freq      float64
	Target    float64
	Captured  int  // plucks captured toward Needed
	Needed    int  // target number of consistent plucks
	Listening bool // a pluck is currently ringing
	Locked    bool // captured >= needed
}

// Push feeds one Reading and returns what should currently be displayed.
func (t *PluckTracker) Push(r Reading) Result {
	amp := r.Amp

	if !t.inPluck && amp < t.onsetLevel() {
		t.updateNoise(amp)
	}
	onset, release := t.onsetLevel(), t.releaseLevel()

	switch {
	case !t.inPluck:
		if t.armed && amp > onset {
			t.startPluck()
		} else if amp < release {
			t.armed = true
		}

	case t.sinceOnset > t.attackSkip && amp > onset && amp > t.prevAmp*repluckRise:
		// A fresh onset while the previous note is still ringing: close the old
		// pluck and begin a new one. The onset's rise is spread across two hops
		// (the YIN window overlaps), so it is measured against prevAmp — two hops
		// back — not the immediately preceding hop, which can stay under the rise.
		t.finalizePluck()
		t.startPluck()

	default:
		t.sinceOnset++
		// Collect clean sustain estimates. Equal votes: a later median is
		// unbiased, whereas amplitude-weighting would pull toward the sharp attack.
		if t.sinceOnset >= t.attackSkip && r.HasPitch && amp > release {
			t.curFreqs = append(t.curFreqs, r.Freq)
		}
		switch {
		case amp < release:
			t.finalizePluck()
			t.inPluck = false
			t.armed = true
		case len(t.curFreqs) >= maxPluckSamples:
			// Sample cap reached while the tine still rings: fold this window into
			// the note's history and begin a new one, so a re-pluck during the long
			// ring is still detected instead of waiting for full decay below release.
			t.finalizePluck()
			t.startPluck()
		}
	}

	t.prevAmp, t.lastAmp = t.lastAmp, amp
	return t.result()
}

func (t *PluckTracker) startPluck() {
	t.inPluck = true
	t.armed = false
	t.sinceOnset = 0
	t.curFreqs = t.curFreqs[:0]
}

// finalizePluck collapses the current pluck into one frequency and folds it
// into the accumulator for its note.
func (t *PluckTracker) finalizePluck() {
	freqs := t.curFreqs
	t.curFreqs, t.sinceOnset = nil, 0

	if len(freqs) < minPluckSamples {
		return // too short/noisy to trust
	}
	f := pluckFrequency(freqs)
	if f <= 0 {
		return
	}

	if name := nearestNote(f).Name; name != t.name {
		t.name = name // a different key — start its history fresh
		t.plucks = []float64{f}
	} else {
		t.plucks = append(t.plucks, f)
		if len(t.plucks) > maxAccum {
			t.plucks = t.plucks[len(t.plucks)-maxAccum:]
		}
	}
}

func (t *PluckTracker) result() Result {
	res := Result{Needed: t.needed, Listening: t.inPluck}

	// A pluck in progress drives the display from its own samples, so a freshly
	// adjusted tine is reflected immediately instead of lagging behind the median
	// of earlier plucks. Until it has enough clean samples we report no note (just
	// Listening), so the previous key doesn't flash during the new attack.
	if t.inPluck {
		if len(t.curFreqs) >= minPluckSamples {
			if pf := pluckFrequency(t.curFreqs); pf > 0 {
				t.fill(&res, pf, min(len(t.plucks), t.needed))
				res.Locked = res.Captured >= t.needed
			}
		}
		return res
	}

	// Between plucks the headline settles on the stable median across captured
	// plucks — replaying the same note refines that median.
	if len(t.plucks) > 0 {
		t.fill(&res, median(t.plucks), min(len(t.plucks), t.needed))
		res.Locked = res.Captured >= t.needed
	}
	return res
}

func (t *PluckTracker) fill(res *Result, freq float64, captured int) {
	n := nearestNote(freq)
	res.HasNote = true
	res.Name = n.Name
	res.Freq = freq
	res.Target = n.Target
	res.Cents = n.Cents
	res.Captured = captured
}

func (t *PluckTracker) onsetLevel() float64 { return math.Max(minOnsetAmp, t.noiseFloor*onsetMult) }
func (t *PluckTracker) releaseLevel() float64 {
	return math.Max(minReleaseAmp, t.noiseFloor*releaseMult)
}

func (t *PluckTracker) updateNoise(amp float64) {
	if t.noiseFloor == 0 {
		t.noiseFloor = amp
		return
	}
	t.noiseFloor += noiseEMA * (amp - t.noiseFloor)
}

// pluckFrequency reduces one pluck's estimates to a single frequency: fold
// each sample toward the median octave (undoing YIN octave errors), then take
// the median (robust to the stray outlier, unbiased across the sustain).
func pluckFrequency(freqs []float64) float64 {
	if len(freqs) == 0 {
		return 0
	}
	m := median(freqs)
	folded := make([]float64, len(freqs))
	for i, f := range freqs {
		for f < m/1.5 {
			f *= 2
		}
		for f > m*1.5 {
			f /= 2
		}
		folded[i] = f
	}
	return median(folded)
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}
