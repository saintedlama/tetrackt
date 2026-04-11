package audio

import (
	"math"
	"time"

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

// FilterEnvelope defines an ADSR envelope applied as an additive offset to
// Filter.Cutoff, scaled by Depth. Depth 0 disables the feature entirely.
type FilterEnvelope struct {
	Attack  time.Duration
	Decay   time.Duration
	Sustain float64 // [0, 1]
	Release time.Duration
	Depth   float64 // [0, 1]; maximum additive offset on Filter.Cutoff
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

// filterEnvelopeGenerator applies an ADSR envelope to the filter cutoff.
// The level drives an additive offset on Filter.Cutoff scaled by Depth.
// Coefficient updates happen once per Stream call (block granularity).
type filterEnvelopeGenerator struct {
	filter     *biquadFilter
	lfo        *lfoGenerator
	baseFilter Filter
	sampleRate beep.SampleRate
	depth      float64

	idx            int
	sustain        float64
	attackSamples  int
	decaySamples   int
	sustainSamples int
	releaseSamples int
}

// newFilterEnvelopeGenerator creates a filter pipeline driven by an ADSR
// envelope. Only call when fe.Depth > 0 and f.Type != FilterOff.
func newFilterEnvelopeGenerator(src beep.Streamer, sampleRate beep.SampleRate, noteSamples int, f Filter, fe FilterEnvelope, lfo *lfoGenerator) *filterEnvelopeGenerator {
	sr := float64(sampleRate)
	attackSamples := int(fe.Attack.Seconds() * sr)
	decaySamples := int(fe.Decay.Seconds() * sr)
	releaseSamples := int(fe.Release.Seconds() * sr)
	sustainSamples := max(0, noteSamples-(attackSamples+decaySamples+releaseSamples))
	return &filterEnvelopeGenerator{
		filter:         &biquadFilter{src: src, coeffs: calcCoeffs(f, float64(sampleRate))},
		lfo:            lfo,
		baseFilter:     f,
		sampleRate:     sampleRate,
		depth:          fe.Depth,
		sustain:        math.Max(minEnvelopeLevel, fe.Sustain),
		attackSamples:  attackSamples,
		decaySamples:   decaySamples,
		sustainSamples: sustainSamples,
		releaseSamples: releaseSamples,
	}
}

// level computes the ADSR envelope level at sample index idx analytically,
// using exact exponential curves rather than per-sample multiplication.
func (g *filterEnvelopeGenerator) level() float64 {
	idx := g.idx
	switch {
	case g.attackSamples > 0 && idx < g.attackSamples:
		return minEnvelopeLevel * math.Pow(1.0/minEnvelopeLevel, float64(idx)/float64(g.attackSamples))
	case g.decaySamples > 0 && idx < g.attackSamples+g.decaySamples:
		t := float64(idx-g.attackSamples) / float64(g.decaySamples)
		return math.Pow(g.sustain, t)
	case idx < g.attackSamples+g.decaySamples+g.sustainSamples:
		return g.sustain
	case g.releaseSamples > 0 && idx < g.attackSamples+g.decaySamples+g.sustainSamples+g.releaseSamples:
		t := float64(idx-g.attackSamples-g.decaySamples-g.sustainSamples) / float64(g.releaseSamples)
		return g.sustain * math.Pow(minEnvelopeLevel/g.sustain, t)
	default:
		return 0.0
	}
}

func (g *filterEnvelopeGenerator) Stream(samples [][2]float64) (int, bool) {
	effectiveCutoff := g.baseFilter.Cutoff + g.level()*g.depth
	if g.lfo != nil {
		effectiveCutoff += g.lfo.nextBlock(len(samples)) * 0.5
	}
	if effectiveCutoff < 0 {
		effectiveCutoff = 0
	} else if effectiveCutoff > 1 {
		effectiveCutoff = 1
	}
	modFilter := g.baseFilter
	modFilter.Cutoff = effectiveCutoff
	g.filter.coeffs = calcCoeffs(modFilter, float64(g.sampleRate))
	n, ok := g.filter.Stream(samples)
	g.idx += n
	return n, ok
}

func (g *filterEnvelopeGenerator) Err() error { return g.filter.Err() }

func (g *filterEnvelopeGenerator) reset() {
	g.idx = 0
}
