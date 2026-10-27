// Package pitch calculates: it turns raw audio frames into a stable musical
// reading. It knows nothing about devices or display.
package pitch

import "math"

// Detection tuning constants.
const (
	// WindowSize is the YIN analysis window. ~46 ms at 44.1 kHz — enough
	// resolution for the lowest kalimba notes while staying cheap enough to
	// run on every hop.
	WindowSize = 2048

	yinThreshold = 0.15
	// rmsGate is the hard silence floor. Kept low so quiet, fast-decaying high
	// tines still register, while staying above typical room noise.
	rmsGate = 0.005
	minHz   = 40
	maxHz   = 3000
)

// Reading is a single-frame pitch estimate plus the frame's loudness.
type Reading struct {
	Freq       float64
	Confidence float64
	Amp        float64 // RMS amplitude of the frame (always set)
	HasPitch   bool
}

// Detect estimates the fundamental frequency of one window. Amp is always
// filled so callers can track the envelope even on unvoiced/quiet frames.
func Detect(frame []float64, sampleRate float64) Reading {
	r := Reading{Amp: rms(frame)}
	if r.Amp < rmsGate {
		return r
	}
	freq, conf := yin(frame, sampleRate, yinThreshold)
	// yinThreshold already bounds cmnd (and so confidence = 1 − cmnd); an
	// in-range fundamental is accepted. conf is still reported for callers.
	if freq > minHz && freq < maxHz {
		r.Freq, r.Confidence, r.HasPitch = freq, conf, true
	}
	return r
}

// Analyzer maintains a sliding window and produces one Reading per hop, so
// short audio hops give fine temporal resolution while YIN still sees a full
// window. Overlapping windows (hop < WindowSize) are exactly what lets us
// catch a pluck's attack and track its decay.
type Analyzer struct {
	sampleRate float64
	buf        []float64
}

func NewAnalyzer(sampleRate float64) *Analyzer {
	return &Analyzer{sampleRate: sampleRate, buf: make([]float64, 0, WindowSize*2)}
}

// Push appends a hop of samples. It returns (Reading, true) once enough
// samples have accumulated to fill the window; otherwise (zero, false).
func (a *Analyzer) Push(hop []float64) (Reading, bool) {
	a.buf = append(a.buf, hop...)
	if len(a.buf) < WindowSize {
		return Reading{}, false
	}
	if len(a.buf) > WindowSize {
		a.buf = a.buf[len(a.buf)-WindowSize:]
	}
	return Detect(a.buf, a.sampleRate), true
}

// yin estimates the fundamental frequency using the YIN algorithm
// (de Cheveigné & Kawahara, 2002). Returns (freq, confidence); freq is 0 if
// no pitch is found.
func yin(buf []float64, sampleRate, threshold float64) (float64, float64) {
	n := len(buf)
	tauMax := n / 2
	if tauMax < 2 {
		return 0, 0
	}

	// Difference function.
	diff := make([]float64, tauMax)
	for tau := 1; tau < tauMax; tau++ {
		var sum float64
		for i := 0; i < tauMax; i++ {
			d := buf[i] - buf[i+tau]
			sum += d * d
		}
		diff[tau] = sum
	}

	// Cumulative mean normalized difference.
	cmnd := make([]float64, tauMax)
	cmnd[0] = 1
	var running float64
	for tau := 1; tau < tauMax; tau++ {
		running += diff[tau]
		if running == 0 {
			cmnd[tau] = 1
		} else {
			cmnd[tau] = diff[tau] * float64(tau) / running
		}
	}

	// Absolute threshold: first local minimum below threshold.
	tau := -1
	for t := 2; t < tauMax; t++ {
		if cmnd[t] < threshold {
			for t+1 < tauMax && cmnd[t+1] < cmnd[t] {
				t++
			}
			tau = t
			break
		}
	}
	if tau == -1 {
		return 0, 0
	}

	// Parabolic interpolation for sub-sample accuracy.
	betterTau := float64(tau)
	if tau > 0 && tau < tauMax-1 {
		s0, s1, s2 := cmnd[tau-1], cmnd[tau], cmnd[tau+1]
		denom := 2 * (2*s1 - s2 - s0)
		if denom != 0 {
			betterTau = float64(tau) + (s2-s0)/denom
		}
	}

	return sampleRate / betterTau, 1 - cmnd[tau]
}

// rms returns the root-mean-square amplitude of buf.
func rms(buf []float64) float64 {
	if len(buf) == 0 {
		return 0
	}
	var sum float64
	for _, v := range buf {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(buf)))
}
