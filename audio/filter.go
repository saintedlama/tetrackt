package audio

import (
	"math"

	"github.com/gopxl/beep/v2"
)

// FilterType selects the filter mode.
type FilterType string

const (
	FilterOff      FilterType = "off"
	FilterLowPass  FilterType = "lowpass"
	FilterHighPass FilterType = "highpass"
	FilterBandPass FilterType = "bandpass"
)

// Filter holds the parameters for the biquad filter.
// Cutoff and Resonance are normalised 0.0–1.0.
//   - Cutoff 0 → ~20 Hz, Cutoff 1 → ~18 000 Hz (log scale)
//   - Resonance 0 → Q ≈ 0.5 (no peak), Resonance 1 → Q ≈ 20 (sharp resonant peak)
type Filter struct {
	Type      FilterType
	Cutoff    float64 // 0.0–1.0
	Resonance float64 // 0.0–1.0
}

// NewFilter creates a filter with default (bypassed) settings.
func NewFilter() Filter {
	return Filter{Type: FilterOff, Cutoff: 0.5, Resonance: 0.0}
}

// cutoffHz maps the normalised Cutoff to a frequency in Hz using a log scale.
func (f Filter) cutoffHz() float64 {
	const minHz = 20.0
	const maxHz = 18000.0
	return minHz * math.Pow(maxHz/minHz, f.Cutoff)
}

// q maps normalised Resonance to a Q factor (0.5 – 20).
func (f Filter) q() float64 {
	const minQ = 0.5
	const maxQ = 20.0
	return minQ + (maxQ-minQ)*f.Resonance
}

// NewFilterStreamer wraps a beep.Streamer with the biquad filter described by f.
// If f.Type == FilterOff the original streamer is returned unchanged.
func NewFilterStreamer(src beep.Streamer, sampleRate beep.SampleRate, f Filter) beep.Streamer {
	if f.Type == FilterOff {
		return src
	}
	return &biquadFilter{
		src:    src,
		coeffs: calcCoeffs(f, float64(sampleRate)),
	}
}

// biquadFilter applies a two-pole IIR biquad filter to both stereo channels.
type biquadFilter struct {
	src    beep.Streamer
	coeffs biquadCoeffs
	// per-channel delay elements
	x1L, x2L, y1L, y2L float64
	x1R, x2R, y1R, y2R float64
}

// biquadCoeffs holds the five biquad coefficients.
type biquadCoeffs struct {
	b0, b1, b2 float64
	a1, a2     float64
}

// calcCoeffs computes RBJ Audio EQ Cookbook coefficients.
func calcCoeffs(f Filter, fs float64) biquadCoeffs {
	f0 := f.cutoffHz()
	Q := f.q()

	omega := 2 * math.Pi * f0 / fs
	sinW := math.Sin(omega)
	cosW := math.Cos(omega)
	alpha := sinW / (2 * Q)

	var b0, b1, b2, a0, a1, a2 float64

	switch f.Type {
	case FilterLowPass:
		b0 = (1 - cosW) / 2
		b1 = 1 - cosW
		b2 = (1 - cosW) / 2
		a0 = 1 + alpha
		a1 = -2 * cosW
		a2 = 1 - alpha

	case FilterHighPass:
		b0 = (1 + cosW) / 2
		b1 = -(1 + cosW)
		b2 = (1 + cosW) / 2
		a0 = 1 + alpha
		a1 = -2 * cosW
		a2 = 1 - alpha

	case FilterBandPass:
		// constant 0 dB peak gain BPF
		b0 = sinW / 2
		b1 = 0
		b2 = -sinW / 2
		a0 = 1 + alpha
		a1 = -2 * cosW
		a2 = 1 - alpha

	default:
		// pass-through (should not happen because FilterOff is handled above)
		return biquadCoeffs{b0: 1}
	}

	return biquadCoeffs{
		b0: b0 / a0,
		b1: b1 / a0,
		b2: b2 / a0,
		a1: a1 / a0,
		a2: a2 / a0,
	}
}

func (b *biquadFilter) Stream(samples [][2]float64) (int, bool) {
	n, ok := b.src.Stream(samples)
	c := b.coeffs
	for i := range n {
		inL := samples[i][0]
		outL := c.b0*inL + c.b1*b.x1L + c.b2*b.x2L - c.a1*b.y1L - c.a2*b.y2L
		b.x2L, b.x1L = b.x1L, inL
		b.y2L, b.y1L = b.y1L, outL
		samples[i][0] = outL

		inR := samples[i][1]
		outR := c.b0*inR + c.b1*b.x1R + c.b2*b.x2R - c.a1*b.y1R - c.a2*b.y2R
		b.x2R, b.x1R = b.x1R, inR
		b.y2R, b.y1R = b.y1R, outR
		samples[i][1] = outR
	}
	return n, ok
}

func (b *biquadFilter) Err() error { return b.src.Err() }
