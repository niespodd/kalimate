package ui

import (
	"math"
	"testing"
)

func TestCentsBetween(t *testing.T) {
	// An octave up is +1200¢, a semitone is ~+100¢, unison is 0.
	if c := centsBetween(880, 440); math.Abs(c-1200) > 0.01 {
		t.Errorf("octave: got %.2f, want 1200", c)
	}
	if c := centsBetween(440, 440); c != 0 {
		t.Errorf("unison: got %.2f, want 0", c)
	}
	semitone := 440 * math.Pow(2, 1.0/12)
	if c := centsBetween(semitone, 440); math.Abs(c-100) > 0.01 {
		t.Errorf("semitone: got %.2f, want 100", c)
	}
}

func TestNearestTine(t *testing.T) {
	a4, _ := TineByName("A4") // 440 Hz
	c4, _ := TineByName("C4")

	// Exactly on a tine.
	if got := nearestTine(a4.FreqHz, Layout); got.Name != "A4" {
		t.Errorf("exact A4: got %s", got.Name)
	}
	// 40¢ sharp of C4 still resolves to C4 (tines are far apart).
	sharpC4 := c4.FreqHz * math.Pow(2, 40.0/1200)
	if got := nearestTine(sharpC4, Layout); got.Name != "C4" {
		t.Errorf("detuned C4: got %s, want C4", got.Name)
	}
	// Halfway between C4 and D4 is ambiguous but must still be one of them.
	d4, _ := TineByName("D4")
	mid := math.Sqrt(c4.FreqHz * d4.FreqHz)
	if got := nearestTine(mid, Layout); got.Name != "C4" && got.Name != "D4" {
		t.Errorf("between C4/D4: got %s", got.Name)
	}
}
