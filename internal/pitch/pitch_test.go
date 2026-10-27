package pitch

import (
	"math"
	"testing"
)

// sine builds a frameSize buffer at freq Hz for testing the detector.
func sine(freq, sampleRate float64, n int) []float64 {
	buf := make([]float64, n)
	for i := range buf {
		buf[i] = 0.5 * math.Sin(2*math.Pi*freq*float64(i)/sampleRate)
	}
	return buf
}

func TestDetectRecoversKnownPitches(t *testing.T) {
	const sr = 44100
	for _, want := range []float64{261.63, 440.0, 1318.51} { // C4, A4, E6
		r := Detect(sine(want, sr, 4096), sr)
		if !r.HasPitch {
			t.Fatalf("no pitch detected for %.2f Hz", want)
		}
		if math.Abs(r.Freq-want) > want*0.01 { // within 1%
			t.Errorf("freq = %.2f, want ~%.2f", r.Freq, want)
		}
	}
}

func TestSilenceHasNoPitch(t *testing.T) {
	if r := Detect(make([]float64, 4096), 44100); r.HasPitch {
		t.Errorf("silence reported a pitch: %+v", r)
	}
}

func TestNearestNote(t *testing.T) {
	n := nearestNote(440)
	if n.Name != "A4" {
		t.Errorf("440 Hz = %q, want A4", n.Name)
	}
	if math.Abs(n.Cents) > 0.1 {
		t.Errorf("440 Hz cents = %.2f, want ~0", n.Cents)
	}
	if nearestNote(261.63).Name != "C4" {
		t.Errorf("261.63 Hz = %q, want C4", nearestNote(261.63).Name)
	}
}

// pushPluck simulates one kalimba pluck: a bit of silence to re-arm, an onset
// with an exponentially decaying sustain at freq, then a silent tail.
func pushPluck(tr *PluckTracker, freq float64) Result {
	var last Result
	for i := 0; i < 3; i++ {
		last = tr.Push(Reading{Amp: 0.001})
	}
	amp := 0.25
	for i := 0; i < 25; i++ {
		last = tr.Push(Reading{Freq: freq, Confidence: 0.95, Amp: amp, HasPitch: true})
		amp *= 0.92 // decay
	}
	for i := 0; i < 3; i++ {
		last = tr.Push(Reading{Amp: 0.001})
	}
	return last
}

// TestTrackerLocksAfterThreePlucks verifies the "3 tries" behavior: the
// reading only locks once enough consistent plucks are captured.
func TestTrackerLocksAfterThreePlucks(t *testing.T) {
	tr := NewPluckTracker(1024.0 / 44100.0)

	r := pushPluck(tr, 440)
	if !r.HasNote || r.Name != "A4" {
		t.Fatalf("after 1 pluck: got %+v, want A4", r)
	}
	if r.Locked {
		t.Errorf("locked after only 1 pluck: %+v", r)
	}

	pushPluck(tr, 440)
	r = pushPluck(tr, 440)
	if !r.Locked || r.Captured < 3 {
		t.Errorf("after 3 plucks: got %+v, want locked with Captured>=3", r)
	}
	if math.Abs(r.Cents) > 3 {
		t.Errorf("cents = %.2f, want ~0 for exact 440 Hz", r.Cents)
	}
}

// TestTrackerCapturesQuietHighPluck guards the quiet/high-tine case: a high
// kalimba tine is much softer than a low one and decays fast, so its usable
// sustain is short. It must still be captured rather than discarded as noise
// (the symptom was high tones not registering at all).
func TestTrackerCapturesQuietHighPluck(t *testing.T) {
	tr := NewPluckTracker(1024.0 / 44100.0)

	var last Result
	for i := 0; i < 3; i++ {
		last = tr.Push(Reading{Amp: 0.001}) // re-arm on near silence
	}
	// A soft E6 that fades ~15% per hop — only a handful of hops clear the gate.
	amp := 0.03
	for i := 0; i < 14; i++ {
		last = tr.Push(Reading{
			Freq: 1318.51, Confidence: 0.9, Amp: amp,
			HasPitch: amp >= rmsGate, // mirror Detect's silence gate
		})
		amp *= 0.85
	}
	for i := 0; i < 3; i++ {
		last = tr.Push(Reading{Amp: 0.001})
	}

	if !last.HasNote || last.Name != "E6" {
		t.Fatalf("quiet high pluck not captured: got %+v, want E6", last)
	}
}

// TestTrackerResetsOnNewNote verifies that playing a different key starts a
// fresh capture rather than averaging across unrelated notes.
func TestTrackerResetsOnNewNote(t *testing.T) {
	tr := NewPluckTracker(1024.0 / 44100.0)
	pushPluck(tr, 440) // A4
	pushPluck(tr, 440)
	r := pushPluck(tr, 329.63) // E4 — different key
	if r.Name != "E4" {
		t.Fatalf("after switching key: got %q, want E4", r.Name)
	}
	if r.Captured != 1 {
		t.Errorf("Captured = %d, want 1 (history reset on new note)", r.Captured)
	}
}
