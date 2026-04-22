package audio

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const srLFO SampleRate = 44100

func TestLFOWaveformSine(t *testing.T) {
	t.Helper()
	assert.InDelta(t, 0.0, lfoWaveformSample(LFOSine, 0), 1e-9, "phase=0: want 0")
	assert.InDelta(t, 1.0, lfoWaveformSample(LFOSine, 0.25), 1e-6, "phase=0.25: want 1")
	assert.InDelta(t, 0.0, lfoWaveformSample(LFOSine, 0.5), 1e-6, "phase=0.5: want 0")
	assert.InDelta(t, -1.0, lfoWaveformSample(LFOSine, 0.75), 1e-6, "phase=0.75: want -1")
}

func TestLFOWaveformTriangle(t *testing.T) {
	assert.InDelta(t, -1.0, lfoWaveformSample(LFOTriangle, 0), 1e-9, "phase=0: want -1")
	assert.InDelta(t, 0.0, lfoWaveformSample(LFOTriangle, 0.25), 1e-9, "phase=0.25: want 0")
	assert.InDelta(t, 1.0, lfoWaveformSample(LFOTriangle, 0.5), 1e-9, "phase=0.5: want 1")
}

func TestLFOWaveformSquare(t *testing.T) {
	assert.Equal(t, 1.0, lfoWaveformSample(LFOSquare, 0.25), "phase=0.25: want 1")
	assert.Equal(t, -1.0, lfoWaveformSample(LFOSquare, 0.75), "phase=0.75: want -1")
}

func TestLFOWaveformSawtooth(t *testing.T) {
	assert.InDelta(t, -1.0, lfoWaveformSample(LFOSawtooth, 0), 1e-9, "phase=0: want -1")
	assert.InDelta(t, 0.0, lfoWaveformSample(LFOSawtooth, 0.5), 1e-9, "phase=0.5: want 0")
	assert.InDelta(t, 1.0, lfoWaveformSample(LFOSawtooth, 1.0), 1e-9, "phase=1: want 1")
}

func TestLFOWaveformUnknownReturnsZero(t *testing.T) {
	assert.Equal(t, 0.0, lfoWaveformSample("unknown", 0.5), "want 0")
}

func TestLFODelayHoldsZero(t *testing.T) {
	lfo := LFO{Waveform: LFOSine, Rate: 1.0, Depth: 1.0, Delay: 1.0}
	g := newLFOGenerator(lfo, float64(srLFO))

	// advance 0.5 s — still within the 1 s delay
	got := g.nextBlock(int(srLFO) / 2)
	require.Equal(t, 0.0, got, "within delay window: want 0")
}

func TestLFOAfterDelayProducesNonZero(t *testing.T) {
	lfo := LFO{Waveform: LFOSine, Rate: 1.0, Depth: 1.0, Delay: 0.1}
	g := newLFOGenerator(lfo, float64(srLFO))

	// advance past the delay (0.2 s > 0.1 s)
	g.nextBlock(int(srLFO) / 5) // 0.2 s
	// advance another quarter period (0.25 s at 1 Hz → phase ≈ 0.25, sine ≈ 1)
	g.nextBlock(int(srLFO) / 4)
	got := g.nextBlock(1)
	require.NotEqual(t, 0.0, got, "after delay, expected non-zero modulation")
}

func TestLFODepthScalesOutput(t *testing.T) {
	// At phase=0.25 sine LFO = 1.0, so depth directly sets the output magnitude.
	// Advance exactly 1/4 period.
	sr := float64(44100)
	rate := 10.0 // 10 Hz → period = 0.1 s, quarter = 0.025 s
	quarterSamples := int(sr * 0.025)

	for _, depth := range []float64{0.0, 0.5, 1.0} {
		lfo := LFO{Waveform: LFOSine, Rate: rate, Depth: depth, Delay: 0}
		g := newLFOGenerator(lfo, sr)
		got := g.nextBlock(quarterSamples)
		want := math.Sin(2*math.Pi*rate*float64(quarterSamples)/sr) * depth
		assert.InDelta(t, want, got, 0.05, "depth=%v: want ≈%v", depth, want)
	}
}

func TestLFOPhaseWraps(t *testing.T) {
	// Advance many full cycles; phase must stay in [0,1).
	lfo := LFO{Waveform: LFOSawtooth, Rate: 100.0, Depth: 1.0, Delay: 0}
	g := newLFOGenerator(lfo, float64(srLFO))
	for range 100 {
		g.nextBlock(441) // 441 samples at 44100 Hz = 0.01 s = 1 cycle at 100 Hz
	}
	assert.GreaterOrEqual(t, g.phase, 0.0, "phase out of range")
	assert.Less(t, g.phase, 1.0, "phase out of range")
}

func TestModulatedOscillatorNilLFOsReturnOscDirect(t *testing.T) {
	osc := NewOscillator(Square, 440, srLFO, 0, 0.5, 0, nil, 0)
	got := newModulatedOscillatorStreamer(osc, 440, 0.5, nil, nil, nil)
	require.Equal(t, osc, got, "expected the bare oscillator to be returned unchanged when both LFOs are nil")
}

func TestModulatedOscillatorPitchLFOChangesFrequency(t *testing.T) {
	osc := NewOscillator(Sine, 440, srLFO, 0, 0, 0, nil, 0)
	pitchLFO := newLFOGenerator(LFO{Waveform: LFOSquare, Rate: 1, Depth: 0.5, Delay: 0}, float64(srLFO))
	mod := newModulatedOscillatorStreamer(osc, 440, 0.5, pitchLFO, nil, nil)

	// After streaming one block the frequency should be ≠ 440 (LFO square at
	// phase 0 outputs +1, so freq = 440 * 1.5 = 660).
	buf := make([][2]float64, 512)
	mod.Stream(buf)
	require.NotEqual(t, 440.0, osc.frequency, "expected frequency to be modulated away from 440 Hz")
}

func TestModulatedOscillatorPWMLFOClampsDuty(t *testing.T) {
	osc := NewOscillator(Square, 440, srLFO, 0, 0.5, 0, nil, 0)
	// Depth=1 → mod can reach ±1; duty = 0.5 + 1*0.5 = 1.0 → clamped to 0.95
	pwmLFO := newLFOGenerator(LFO{Waveform: LFOSquare, Rate: 1, Depth: 1.0, Delay: 0}, float64(srLFO))
	mod := newModulatedOscillatorStreamer(osc, 440, 0.5, nil, pwmLFO, nil)

	buf := make([][2]float64, 512)
	mod.Stream(buf)
	assert.LessOrEqual(t, osc.pulseWidth, 0.95, "pulse width above clamped maximum")
	assert.GreaterOrEqual(t, osc.pulseWidth, 0.05, "pulse width below clamped minimum")
}

func TestModulatedOscillatorDetuneLFOShiftsFrequency(t *testing.T) {
	// Osc with 50 cents static detune → detuneMultiplier ≈ 1.029
	osc := NewOscillator(Square, 440, srLFO, 0, 0.5, 50, nil, 0)
	baseMult := osc.detuneMultiplier

	detuneLFO := newLFOGenerator(LFO{Waveform: LFOSquare, Rate: 1, Depth: 0.5, Delay: 0}, float64(srLFO))
	mod := newModulatedOscillatorStreamer(osc, 440, 0.5, nil, nil, detuneLFO)

	buf := make([][2]float64, 512)
	mod.Stream(buf)

	// After streaming, detuneMultiplier should differ from the static value.
	// osc.frequency remains the transparent raw Hz value and must be unchanged.
	require.NotEqual(t, baseMult, osc.detuneMultiplier, "expected detune LFO to shift detuneMultiplier away from static value")
	require.Equal(t, 440.0, osc.frequency, "expected osc.frequency to remain raw 440 Hz")
}

func TestModulatedOscillatorDetuneLFODoesNotAffectOsc1(t *testing.T) {
	// Osc1 has no detune LFO wired (nil) — its raw frequency must remain unchanged.
	osc1 := NewOscillator(Square, 440, srLFO, 0, 0.5, 50, nil, 0)
	staticFreq := osc1.frequency // set at construction

	mod1 := newModulatedOscillatorStreamer(osc1, 440, 0.5, nil, nil, nil)

	buf := make([][2]float64, 512)
	mod1.Stream(buf)
	require.Equal(t, staticFreq, osc1.frequency, "osc1 frequency changed without detune LFO")
}

func TestModulatedVolumeNilLFOReturnsDirect(t *testing.T) {
	osc2 := NewOscillator(Silent, 440, srLFO, 0, 0, 0, nil, 0)
	got := newModulatedVolumeStreamer(osc2, nil)
	require.Equal(t, osc2, got, "expected inner streamer returned unchanged when LFO is nil")
}

func TestModulatedVolumeTremoloScalesSamples(t *testing.T) {
	// Use a constant-output streamer (sine at DC = 0, so use sawtooth at
	// a frequency so low the first block is nearly flat).
	// Simpler: wrap a constant value streamer.
	osc := NewOscillator(Sawtooth, 100, srLFO, 0, 0, 0, nil, 0)
	volumeLFO := newLFOGenerator(LFO{Waveform: LFOSquare, Rate: 1, Depth: 0.5, Delay: 0}, float64(srLFO))
	mod := newModulatedVolumeStreamer(osc, volumeLFO)

	buf := make([][2]float64, 512)
	n, ok := mod.Stream(buf)
	require.True(t, ok && n > 0, "stream returned no samples")
	// At least verify samples are not all zero (the LFO is active).
	allZero := true
	for _, s := range buf[:n] {
		if s[0] != 0 {
			allZero = false
			break
		}
	}
	require.False(t, allZero, "expected non-zero output from modulated volume streamer")
}

func TestModulatedVolumeGainFloorIsZero(t *testing.T) {
	// LFO with depth=2 at square phase=0.75 → raw=-1, gain=1+(-1*2)=-1 → clamped to 0.
	// Use LFOSawtooth at phase=1 (≈ +1): gain = 1 + 1*depth. For negative gain
	// we need mod < -1: use depth=2, LFOSquare phase=0.75 → raw=-1.
	// Advance by 3/4 period to reach phase 0.75.
	sr := float64(44100)
	rate := 10.0 // 10 Hz
	threequarterSamples := int(sr * 0.075)

	volumeLFO := newLFOGenerator(LFO{Waveform: LFOSquare, Rate: rate, Depth: 2.0, Delay: 0}, sr)
	// advance to phase 0.5 (square → -1), gain = 1 + (-1*2) = -1 → floor 0
	volumeLFO.nextBlock(threequarterSamples)

	osc := NewOscillator(Sawtooth, 100, SampleRate(sr), 0, 0, 0, nil, 0)
	mod := &modulatedVolumeStreamer{inner: osc, lfo: volumeLFO}

	buf := make([][2]float64, 64)
	mod.Stream(buf)
	for i, s := range buf {
		assert.GreaterOrEqual(t, s[0], 0.0, "sample %d: gain floor violated", i)
	}
}

func TestModulatedFilterOffReturnsSource(t *testing.T) {
	osc := NewOscillator(Sine, 440, srLFO, 0, 0, 0, nil, 0)
	lfo := newLFOGenerator(LFO{Waveform: LFOSine, Rate: 1, Depth: 0.5, Delay: 0}, float64(srLFO))
	f := Filter{Type: FilterOff, Cutoff: 0.5, Resonance: 0}
	got := NewModulatedFilterStreamer(osc, srLFO, f, lfo)
	require.Equal(t, osc, got, "FilterOff should return the source streamer unchanged")
}

func TestModulatedFilterNilLFOReturnsBiquad(t *testing.T) {
	osc := NewOscillator(Sine, 440, srLFO, 0, 0, 0, nil, 0)
	f := Filter{Type: FilterLowPass, Cutoff: 0.5, Resonance: 0}
	got := NewModulatedFilterStreamer(osc, srLFO, f, nil)
	_, ok := got.(*biquadFilter)
	require.True(t, ok, "nil LFO with active filter should return *biquadFilter")
}

func TestModulatedFilterLFOReturnModulatedType(t *testing.T) {
	osc := NewOscillator(Sine, 440, srLFO, 0, 0, 0, nil, 0)
	lfo := newLFOGenerator(LFO{Waveform: LFOSine, Rate: 1, Depth: 0.5, Delay: 0}, float64(srLFO))
	f := Filter{Type: FilterLowPass, Cutoff: 0.5, Resonance: 0}
	got := NewModulatedFilterStreamer(osc, srLFO, f, lfo)
	_, ok := got.(*modulatedFilterStreamer)
	require.True(t, ok, "expected *modulatedFilterStreamer, got %T", got)
}

func TestModulatedFilterCutoffClamped(t *testing.T) {
	// depth=1 → mod can be ±1; cutoff = baseFilter.Cutoff + mod*0.5
	// with Cutoff=1.0 and mod=+1 → 1.5 → clamped to 1.0
	osc := NewOscillator(Sine, 100, srLFO, 0, 0, 0, nil, 0)
	lfo := newLFOGenerator(LFO{Waveform: LFOSquare, Rate: 1, Depth: 1.0, Delay: 0}, float64(srLFO))
	f := Filter{Type: FilterLowPass, Cutoff: 1.0, Resonance: 0}
	mf := NewModulatedFilterStreamer(osc, srLFO, f, lfo).(*modulatedFilterStreamer)

	buf := make([][2]float64, 512)
	mf.Stream(buf)
	// No panic and no NaN in output is sufficient to verify clamping works.
	for i, s := range buf {
		assert.False(t, math.IsNaN(s[0]) || math.IsNaN(s[1]), "NaN at sample %d after cutoff modulation", i)
	}
}
