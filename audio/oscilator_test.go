package audio

import (
	"math"
	"testing"

	"github.com/gopxl/beep/v2"
)

// streamN streams exactly n samples from s into a fresh buffer and returns it.
func streamN(s beep.Streamer, n int) [][2]float64 {
	buf := make([][2]float64, n)
	s.Stream(buf)
	return buf
}

func TestOscillatorSilent(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Silent, 440.0, sr, 0, 0, 0, nil)
	samples := streamN(osc, 100)
	for i, s := range samples {
		if s[0] != 0 || s[1] != 0 {
			t.Errorf("sample %d: expected 0,0 got %v,%v", i, s[0], s[1])
		}
	}
}

func TestOscillatorSquare(t *testing.T) {
	sr := beep.SampleRate(44100)
	// Half period: 44100 / (2*100) = 220.5 → flip at sample 221
	osc := NewOscillator(Square, 100.0, sr, 0, 0, 0, nil)
	samples := streamN(osc, 250)

	if samples[0][0] != 1.0 {
		t.Errorf("sample 0: want +1, got %v", samples[0][0])
	}
	if samples[221][0] != -1.0 {
		t.Errorf("sample 221: want -1 (past half period), got %v", samples[221][0])
	}
	// All samples must be exactly ±1
	for i, s := range samples {
		if s[0] != 1.0 && s[0] != -1.0 {
			t.Errorf("sample %d: want ±1, got %v", i, s[0])
			break
		}
	}
}

func TestOscillatorSawtooth(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Sawtooth, 100.0, sr, 0, 0, 0, nil)
	samples := streamN(osc, 10)

	// phase=0 → 2*0-1 = -1
	if math.Abs(samples[0][0]-(-1.0)) > 1e-6 {
		t.Errorf("sample 0: want -1, got %v", samples[0][0])
	}
	// Sawtooth increases linearly within a period
	for i := 1; i < len(samples); i++ {
		if samples[i][0] <= samples[i-1][0] {
			t.Errorf("sample %d: expected increase, %v <= %v", i, samples[i][0], samples[i-1][0])
		}
	}
}

func TestOscillatorSawtoothReverse(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(SawtoothReverse, 100.0, sr, 0, 0, 0, nil)
	samples := streamN(osc, 10)

	// phase=0 → 1-2*0 = +1
	if math.Abs(samples[0][0]-1.0) > 1e-6 {
		t.Errorf("sample 0: want +1, got %v", samples[0][0])
	}
	// Decreases linearly within a period
	for i := 1; i < len(samples); i++ {
		if samples[i][0] >= samples[i-1][0] {
			t.Errorf("sample %d: expected decrease, %v >= %v", i, samples[i][0], samples[i-1][0])
		}
	}
}

func TestOscillatorTriangle(t *testing.T) {
	sr := beep.SampleRate(44100)

	// phase=0 → 4*0-1 = -1
	s0 := streamN(NewOscillator(Triangle, 100.0, sr, 0.0, 0, 0, nil), 1)
	if math.Abs(s0[0][0]-(-1.0)) > 1e-6 {
		t.Errorf("phase 0.0: want -1, got %v", s0[0][0])
	}

	// phase=0.25 → 4*0.25-1 = 0
	s25 := streamN(NewOscillator(Triangle, 100.0, sr, 0.25, 0, 0, nil), 1)
	if math.Abs(s25[0][0]) > 1e-6 {
		t.Errorf("phase 0.25: want 0, got %v", s25[0][0])
	}

	// phase=0.5 → -4*0.5+3 = 1
	s50 := streamN(NewOscillator(Triangle, 100.0, sr, 0.5, 0, 0, nil), 1)
	if math.Abs(s50[0][0]-1.0) > 1e-6 {
		t.Errorf("phase 0.5: want 1, got %v", s50[0][0])
	}
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
		s := streamN(NewOscillator(Sine, 100.0, sr, tt.phase, 0, 0, nil), 1)
		if math.Abs(s[0][0]-tt.want) > 1e-6 {
			t.Errorf("phase %v: want %v, got %v", tt.phase, tt.want, s[0][0])
		}
	}
}

func TestOscillatorNoise(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Noise, 440.0, sr, 0, 0, 0, nil)
	samples := streamN(osc, 1000)

	first := samples[0][0]
	allSame := true
	for _, s := range samples {
		if math.Abs(s[0]) > 1.0 {
			t.Errorf("sample out of range [-1,1]: %v", s[0])
		}
		if s[0] != first {
			allSame = false
		}
	}
	if allSame {
		t.Error("noise: all 1000 samples are identical")
	}
}

func TestOscillatorStereo(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Sine, 440.0, sr, 0, 0, 0, nil)
	samples := streamN(osc, 100)
	for i, s := range samples {
		if s[0] != s[1] {
			t.Errorf("sample %d: L(%v) != R(%v)", i, s[0], s[1])
		}
	}
}

func TestOscillatorInitialPhase(t *testing.T) {
	sr := beep.SampleRate(44100)
	// Square with phase=0.6, default pulseWidth=0.5 → 0.6 >= 0.5 → -1
	osc := NewOscillator(Square, 100.0, sr, 0.6, 0, 0, nil)
	s := streamN(osc, 1)
	if s[0][0] != -1.0 {
		t.Errorf("square at phase 0.6: want -1, got %v", s[0][0])
	}
}

func TestOscillatorWavetableInterpolates(t *testing.T) {
	sr := beep.SampleRate(44100)
	// 4-sample table: [0, 1, 0, -1] — one cycle of a sine-like shape
	table := []float64{0, 1, 0, -1}
	osc := NewOscillator(Wavetable, 100.0, sr, 0, 0, 0, table)
	s := streamN(osc, 1)

	// phase=0 → pos=0.0 → i0=0, i1=1, frac=0 → table[0]*(1)+table[1]*0 = 0
	if math.Abs(s[0][0]) > 1e-9 {
		t.Errorf("wavetable phase=0: want 0, got %v", s[0][0])
	}
}

func TestOscillatorWavetableEmptyIsSilent(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Wavetable, 440.0, sr, 0, 0, 0, nil)
	s := streamN(osc, 100)
	for i, sample := range s {
		if sample[0] != 0 || sample[1] != 0 {
			t.Errorf("sample %d: empty wavetable should be silent, got %v", i, sample)
		}
	}
}

func TestOscillatorWavetableStereo(t *testing.T) {
	sr := beep.SampleRate(44100)
	osc := NewOscillator(Wavetable, 440.0, sr, 0, 0, 0, WavetableSoftSaw)
	s := streamN(osc, 100)
	for i, sample := range s {
		if sample[0] != sample[1] {
			t.Errorf("sample %d: L(%v) != R(%v)", i, sample[0], sample[1])
		}
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
		if len(table) != wavetableSize {
			t.Errorf("%s: want len=%d, got %d", name, wavetableSize, len(table))
		}
		peak := 0.0
		for _, v := range table {
			if a := math.Abs(v); a > peak {
				peak = a
			}
		}
		if math.Abs(peak-1.0) > 1e-9 {
			t.Errorf("%s: peak amplitude should be 1.0, got %v", name, peak)
		}
	}
}
