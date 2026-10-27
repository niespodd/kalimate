package pitch

import (
	"fmt"
	"math"
)

var noteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// Note is the musical interpretation of a frequency.
type Note struct {
	Name   string  // e.g. "D4"
	Cents  float64 // deviation from the exact note, in cents [-50, +50]
	Target float64 // exact frequency of the nearest note, in Hz
}

// nearestNote maps a frequency to the closest equal-tempered note (A4 = 440 Hz).
func nearestNote(freq float64) Note {
	midi := 69.0 + 12.0*math.Log2(freq/440.0)
	nearest := math.Round(midi)
	target := 440.0 * math.Pow(2, (nearest-69.0)/12.0)

	idx := int(nearest) % 12
	if idx < 0 {
		idx += 12
	}
	octave := int(nearest)/12 - 1

	return Note{
		Name:   fmt.Sprintf("%s%d", noteNames[idx], octave),
		Cents:  (midi - nearest) * 100.0,
		Target: target,
	}
}
