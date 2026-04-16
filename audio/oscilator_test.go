package audio

import (
	"math"
	"testing"

	"github.com/gopxl/beep/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOscillatorSilent(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Silent, 440.0, sr, 0, 0, 0, nil, 0)
	samples := StreamN(osc, 100)
	for i, s := range samples {
		assert.Equal(t, 0.0, s[0], "sample %d L", i)
		assert.Equal(t, 0.0, s[1], "sample %d R", i)
	}
}

func TestOscillatorSquare(t *testing.T) {
	sr := beep.SampleRate(44100)
	// Half period: 44100 / (2*100) = 220.5 → flip at sample 221
	osc := NewOscillator(Square, 100.0, sr, 0, 0, 0, nil, 0)
	samples := StreamN(osc, 250)

	assert.Equal(t, 1.0, samples[0][0], "sample 0: want +1")
	assert.Equal(t, -1.0, samples[221][0], "sample 221: want -1 (past half period)")
	// All samples must be exactly ±1
	for i, s := range samples {
		assert.True(t, s[0] == 1.0 || s[0] == -1.0, "sample %d: want ±1, got %v", i, s[0])
	}
}

func TestOscillatorSawtooth(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Sawtooth, 100.0, sr, 0, 0, 0, nil, 0)
	samples := StreamN(osc, 10)

	// phase=0 → 2*0-1 = -1
	assert.InDelta(t, -1.0, samples[0][0], 1e-6, "sample 0: want -1")
	// Sawtooth increases linearly within a period
	for i := 1; i < len(samples); i++ {
		assert.Greater(t, samples[i][0], samples[i-1][0], "sample %d: expected increase", i)
	}
}

func TestOscillatorSawtoothReverse(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(SawtoothReverse, 100.0, sr, 0, 0, 0, nil, 0)
	samples := StreamN(osc, 10)

	// phase=0 → 1-2*0 = +1
	assert.InDelta(t, 1.0, samples[0][0], 1e-6, "sample 0: want +1")
	// Decreases linearly within a period
	for i := 1; i < len(samples); i++ {
		assert.Less(t, samples[i][0], samples[i-1][0], "sample %d: expected decrease", i)
	}
}

func TestOscillatorTriangle(t *testing.T) {
	sr := beep.SampleRate(44100)

	// phase=0 → 4*0-1 = -1
	s0 := StreamN(NewOscillator(Triangle, 100.0, sr, 0.0, 0, 0, nil, 0), 1)
	assert.InDelta(t, -1.0, s0[0][0], 1e-6, "phase 0.0: want -1")

	// phase=0.25 → 4*0.25-1 = 0
	s25 := StreamN(NewOscillator(Triangle, 100.0, sr, 0.25, 0, 0, nil, 0), 1)
	assert.InDelta(t, 0.0, s25[0][0], 1e-6, "phase 0.25: want 0")

	// phase=0.5 → -4*0.5+3 = 1
	s50 := StreamN(NewOscillator(Triangle, 100.0, sr, 0.5, 0, 0, nil, 0), 1)
	assert.InDelta(t, 1.0, s50[0][0], 1e-6, "phase 0.5: want 1")
}

func TestOscillatorSine(t *testing.T) {
	sr := beep.SampleRate(44100)
	tests := []struct {
		phase float64
		want  float64
	}{
		{0.0, 0.0},   // sin(0) = 0
		{0.25, 1.0},  // sin(π/2) = 1
		{0.5, 0.0},   // sin(π) ≈ 0
		{0.75, -1.0}, // sin(3π/2) = -1
	}
	for _, tt := range tests {
		s := StreamN(NewOscillator(Sine, 100.0, sr, tt.phase, 0, 0, nil, 0), 1)
		assert.InDelta(t, tt.want, s[0][0], 1e-6, "phase %v: want %v", tt.phase, tt.want)
	}
}

func TestOscillatorNoise(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Noise, 440.0, sr, 0, 0, 0, nil, 0)
	samples := StreamN(osc, 1000)

	first := samples[0][0]
	allSame := true
	for _, s := range samples {
		assert.LessOrEqual(t, math.Abs(s[0]), 1.0, "sample out of range [-1,1]")
		if s[0] != first {
			allSame = false
		}
	}
	require.False(t, allSame, "noise: all 1000 samples are identical")
}

func TestOscillatorStereo(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Sine, 440.0, sr, 0, 0, 0, nil, 0)
	samples := StreamN(osc, 100)
	for i, s := range samples {
		assert.Equal(t, s[0], s[1], "sample %d: L != R", i)
	}
}

func TestOscillatorInitialPhase(t *testing.T) {
	sr := beep.SampleRate(44100)
	// Square with phase=0.6, default pulseWidth=0.5 → 0.6 >= 0.5 → -1
	osc := NewOscillator(Square, 100.0, sr, 0.6, 0, 0, nil, 0)
	s := StreamN(osc, 1)
	assert.Equal(t, -1.0, s[0][0], "square at phase 0.6: want -1")
}

func TestOscillatorWavetableInterpolates(t *testing.T) {
	sr := beep.SampleRate(44100)
	// 4-sample table: [0, 1, 0, -1] — one cycle of a sine-like shape
	table := []float64{0, 1, 0, -1}
	osc := NewOscillator(Wavetable, 100.0, sr, 0, 0, 0, table, 0)
	s := StreamN(osc, 1)

	// phase=0 → pos=0.0 → i0=0, i1=1, frac=0 → table[0]*(1)+table[1]*0 = 0
	assert.InDelta(t, 0.0, s[0][0], 1e-9, "wavetable phase=0: want 0")
}

func TestOscillatorWavetableEmptyIsSilent(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Wavetable, 440.0, sr, 0, 0, 0, nil, 0)
	s := StreamN(osc, 100)
	for i, sample := range s {
		assert.Equal(t, 0.0, sample[0], "sample %d L: empty wavetable should be silent", i)
		assert.Equal(t, 0.0, sample[1], "sample %d R: empty wavetable should be silent", i)
	}
}

func TestOscillatorWavetableStereo(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Wavetable, 440.0, sr, 0, 0, 0, WavetableSoftSaw, 0)
	s := StreamN(osc, 100)
	for i, sample := range s {
		assert.Equal(t, sample[0], sample[1], "sample %d: L != R", i)
	}
}

func TestBuiltinWavetablesNormalized(t *testing.T) {
	tables := map[string][]float64{
		"SoftSaw":    WavetableSoftSaw,
		"SoftSquare": WavetableSoftSquare,
		"Organ":      WavetableOrgan,
		"Glass":      WavetableGlass,
	}
	for name, table := range tables {
		assert.Equal(t, wavetableSize, len(table), "%s: want len=%d", name, wavetableSize)
		peak := 0.0
		for _, v := range table {
			if a := math.Abs(v); a > peak {
				peak = a
			}
		}
		assert.InDelta(t, 1.0, peak, 1e-9, "%s: peak amplitude should be 1.0", name)
	}
}
