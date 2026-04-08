package audio

import (
	"math"

	"github.com/gopxl/beep/v2"
)

// LFOWaveform selects the LFO waveform shape.
type LFOWaveform string

const (
	LFOSine     LFOWaveform = "sine"
	LFOTriangle LFOWaveform = "triangle"
	LFOSquare   LFOWaveform = "square"
	LFOSawtooth LFOWaveform = "sawtooth"
)

// ModDest selects the modulation destination.
type ModDest int

const (
	ModPitch      ModDest = iota // modulates oscillator frequency (vibrato)
	ModVolume                    // modulates envelope output gain (tremolo)
	ModCutoff                    // modulates filter cutoff (auto-wah)
	ModPulseWidth                // modulates square-wave duty cycle (PWM sweep)
	ModDetune                    // modulates detune offset (chorus/beating)
)

// LFO describes a low-frequency oscillator modulation source.
type LFO struct {
	Waveform LFOWaveform
	Rate     float64 // Hz (0.01–20)
	Depth    float64 // [0, 1] — scales the [-1,+1] raw output
	Delay    float64 // seconds before onset
	Dest     ModDest // modulation destination
}

// lfoGenerator produces per-block modulation values.
type lfoGenerator struct {
	lfo        LFO
	phase      float64
	elapsed    float64
	sampleRate float64
}

func newLFOGenerator(lfo LFO, sampleRate float64) *lfoGenerator {
	return &lfoGenerator{lfo: lfo, sampleRate: sampleRate}
}

// nextBlock advances the LFO by n samples and returns the modulation scalar.
func (g *lfoGenerator) nextBlock(n int) float64 {
	g.elapsed += float64(n) / g.sampleRate
	if g.elapsed < g.lfo.Delay {
		return 0
	}
	g.phase += g.lfo.Rate * float64(n) / g.sampleRate
	for g.phase >= 1.0 {
		g.phase -= 1.0
	}
	return lfoWaveformSample(g.lfo.Waveform, g.phase) * g.lfo.Depth
}

func lfoWaveformSample(w LFOWaveform, phase float64) float64 {
	switch w {
	case LFOSine:
		return math.Sin(2 * math.Pi * phase)
	case LFOTriangle:
		if phase < 0.5 {
			return 4*phase - 1
		}
		return -4*phase + 3
	case LFOSquare:
		if phase < 0.5 {
			return 1
		}
		return -1
	case LFOSawtooth:
		return 2*phase - 1
	default:
		return 0
	}
}

type modulatedOscillatorStreamer struct {
	osc       *oscillatorGenerator
	pitchLFO  *lfoGenerator
	pwmLFO    *lfoGenerator
	detuneLFO *lfoGenerator // modulates osc2 only; nil = no detune mod
	baseFreq  float64
	baseDuty  float64
}

// newModulatedOscillatorStreamer wraps osc with optional LFO-driven pitch,
// pulse-width, and/or detune modulation. Returns osc unchanged when all LFOs are nil.
func newModulatedOscillatorStreamer(osc *oscillatorGenerator, baseFreq, baseDuty float64, pitchLFO, pwmLFO, detuneLFO *lfoGenerator) beep.Streamer {
	if pitchLFO == nil && pwmLFO == nil && detuneLFO == nil {
		return osc
	}
	return &modulatedOscillatorStreamer{
		osc:       osc,
		pitchLFO:  pitchLFO,
		pwmLFO:    pwmLFO,
		detuneLFO: detuneLFO,
		baseFreq:  baseFreq,
		baseDuty:  baseDuty,
	}
}

func (m *modulatedOscillatorStreamer) Stream(samples [][2]float64) (int, bool) {
	n := len(samples)
	if m.pitchLFO != nil {
		mod := m.pitchLFO.nextBlock(n)
		m.osc.frequency = m.baseFreq * (1.0 + mod)
	}
	if m.pwmLFO != nil {
		mod := m.pwmLFO.nextBlock(n)
		duty := m.baseDuty + mod*0.5
		if duty < 0.05 {
			duty = 0.05
		} else if duty > 0.95 {
			duty = 0.95
		}
		m.osc.pulseWidth = duty
	}
	if m.detuneLFO != nil {
		mod := m.detuneLFO.nextBlock(n)
		// mod ∈ [-depth, +depth]; treat as a fractional cent offset relative to the
		// base detune. A full-depth modulation sweeps ±1 octave (1200 cents).
		effectiveMult := m.osc.detuneMultiplier * math.Pow(2, mod*1200.0/1200.0)
		m.osc.frequency = m.baseFreq * effectiveMult
	}
	return m.osc.Stream(samples)
}

func (m *modulatedOscillatorStreamer) Err() error { return m.osc.Err() }

type modulatedVolumeStreamer struct {
	inner beep.Streamer
	lfo   *lfoGenerator
}

// newModulatedVolumeStreamer wraps inner with LFO-driven gain modulation.
// Returns inner unchanged when lfo is nil.
func newModulatedVolumeStreamer(inner beep.Streamer, lfo *lfoGenerator) beep.Streamer {
	if lfo == nil {
		return inner
	}
	return &modulatedVolumeStreamer{inner: inner, lfo: lfo}
}

func (m *modulatedVolumeStreamer) Stream(samples [][2]float64) (int, bool) {
	n, ok := m.inner.Stream(samples)
	gain := 1.0 + m.lfo.nextBlock(n)
	if gain < 0 {
		gain = 0
	}
	for i := 0; i < n; i++ {
		samples[i][0] *= gain
		samples[i][1] *= gain
	}
	return n, ok
}

func (m *modulatedVolumeStreamer) Err() error { return m.inner.Err() }

type modulatedFilterStreamer struct {
	filter     *biquadFilter
	lfo        *lfoGenerator
	baseFilter Filter
	sampleRate beep.SampleRate
}

// NewModulatedFilterStreamer builds a biquad filter with optional LFO cutoff
// modulation. Behaves identically to NewFilterStreamer when lfo is nil.
func NewModulatedFilterStreamer(src beep.Streamer, sampleRate beep.SampleRate, f Filter, lfo *lfoGenerator) beep.Streamer {
	if f.Type == FilterOff {
		return src
	}
	bf := &biquadFilter{src: src, coeffs: calcCoeffs(f, float64(sampleRate))}
	if lfo == nil {
		return bf
	}
	return &modulatedFilterStreamer{
		filter:     bf,
		lfo:        lfo,
		baseFilter: f,
		sampleRate: sampleRate,
	}
}

func (m *modulatedFilterStreamer) Stream(samples [][2]float64) (int, bool) {
	n := len(samples)
	mod := m.lfo.nextBlock(n)
	newCutoff := m.baseFilter.Cutoff + mod*0.5
	if newCutoff < 0 {
		newCutoff = 0
	} else if newCutoff > 1 {
		newCutoff = 1
	}
	modFilter := m.baseFilter
	modFilter.Cutoff = newCutoff
	m.filter.coeffs = calcCoeffs(modFilter, float64(m.sampleRate))
	return m.filter.Stream(samples)
}

func (m *modulatedFilterStreamer) Err() error { return m.filter.Err() }
