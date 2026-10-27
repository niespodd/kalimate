package ui

import (
	"math"
	"sort"
)

// pitchOrder is the tines sorted by ascending pitch (for "next key" selection).
var pitchOrder = func() []Tine {
	out := append([]Tine(nil), Layout...)
	sort.Slice(out, func(i, j int) bool { return out[i].Midi < out[j].Midi })
	return out
}()

// nextPending returns the next not-done tine by ascending pitch after target
// (wrapping), and whether every tine is already done.
func nextPending(target string, isDone func(string) bool) (string, bool) {
	allDone := true
	for _, t := range pitchOrder {
		if !isDone(t.Name) {
			allDone = false
			break
		}
	}
	if allDone {
		return "", true
	}
	start := 0
	for i, t := range pitchOrder {
		if t.Name == target {
			start = i
			break
		}
	}
	for off := 1; off <= len(pitchOrder); off++ {
		t := pitchOrder[(start+off)%len(pitchOrder)]
		if !isDone(t.Name) {
			return t.Name, false
		}
	}
	return "", true
}

// phase classifies the overall tuning situation for the widgets.
type phase int

const (
	phaseNoTarget phase = iota // nothing selected
	phaseNoSignal              // target set, silence
	phaseWrongKey              // hearing a different key than the target
	phaseMatch                 // hearing the target: meter is live
)

func phaseOf(s TunerState) phase {
	switch {
	case s.Target == "":
		return phaseNoTarget
	case !s.HasSignal:
		return phaseNoSignal
	case s.Heard != "" && s.Heard != s.Target:
		return phaseWrongKey
	default:
		return phaseMatch
	}
}

// centsBetween returns the interval from ref to freq in cents (1200·log2).
// Positive = freq is sharp of ref.
func centsBetween(freq, ref float64) float64 {
	if freq <= 0 || ref <= 0 {
		return 0
	}
	return 1200 * math.Log2(freq/ref)
}

// nearestTine returns the kalimba tine closest to freq, searching ONLY the
// instrument's tines (not the chromatic scale). The layout has a few semitone
// pairs (E4/F4, B4/C5, E5/F5, B5/C6), so a tine detuned far toward its
// neighbour would resolve to the wrong key on frequency alone; the controller
// biases toward the selected target to compensate (see ingest).
func nearestTine(freq float64, tines []Tine) Tine {
	best := tines[0]
	bestAbs := math.Abs(centsBetween(freq, best.FreqHz))
	for _, t := range tines[1:] {
		if a := math.Abs(centsBetween(freq, t.FreqHz)); a < bestAbs {
			best, bestAbs = t, a
		}
	}
	return best
}

// levelFromAmp maps RMS amplitude to a perceptual 0..1 level (log scale), so a
// normal pluck lights most of the input-level bars.
func levelFromAmp(amp float64) float64 {
	const lo, hi = 0.0015, 0.2
	if amp <= lo {
		return 0
	}
	return math.Max(0, math.Min(1, math.Log(amp/lo)/math.Log(hi/lo)))
}
