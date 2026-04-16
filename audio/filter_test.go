package audio

import (
	"math"
	"testing"

	"github.com/gopxl/beep/v2"
)

// rms computes the root-mean-square amplitude across both channels.
func rms(samples [][2]float64) float64 {
	var sum float64
	for _, s := range samples {
		sum += s[0]*s[0] + s[1]*s[1]
	}
	return math.Sqrt(sum / float64(2*len(samples)))
}

func TestNewFilterDefaults(t *testing.T) {
	f := NewFilter()
	if f.Type != FilterOff {
		t.Errorf("want FilterOff, got %v", f.Type)
	}
	if math.Abs(f.Cutoff-0.5) > 1e-6 {
		t.Errorf("want Cutoff=0.5, got %v", f.Cutoff)
	}
	if math.Abs(f.Resonance-0.0) > 1e-6 {
		t.Errorf("want Resonance=0, got %v", f.Resonance)
	}
}

func TestFilterOffPassthrough(t *testing.T) {
	sr := beep.SampleRate(44100)
	src := ConstantStreamer(1.0)
	result := NewFilterStreamer(src, sr, Filter{Type: FilterOff, Cutoff: 0.5})
	if result != src {
		t.Error("FilterOff should return the original streamer (pointer equality)")
	}
}

func TestFilterCutoffHz(t *testing.T) {
	tests := []struct {
		cutoff float64
		wantHz float64
	}{
		{0.0, 20.0},    // 20 * 900^0 = 20 Hz
		{1.0, 18000.0}, // 20 * 900^1 = 18000 Hz
		{0.5, 600.0},   // geometric mean: 20 * sqrt(900) = 20*30 = 600 Hz
	}
	for _, tt := range tests {
		f := Filter{Cutoff: tt.cutoff}
		got := f.cutoffHz()
		if math.Abs(got-tt.wantHz) > 1e-6 {
			t.Errorf("Cutoff=%v: want %v Hz, got %v Hz", tt.cutoff, tt.wantHz, got)
		}
	}
}

func TestFilterQResonance(t *testing.T) {
	tests := []struct {
		resonance float64
		wantQ     float64
	}{
		{0.0, 0.5},
		{1.0, 20.0},
	}
	for _, tt := range tests {
		f := Filter{Resonance: tt.resonance}
		got := f.q()
		if math.Abs(got-tt.wantQ) > 1e-6 {
			t.Errorf("Resonance=%v: want Q=%v, got Q=%v", tt.resonance, tt.wantQ, got)
		}
	}
}

func TestLowPassAttenuatesHighFreq(t *testing.T) {
	sr := beep.SampleRate(44100)
	const numSamples = 44100

	// Reference: 1000 Hz sine, no filter
	inputSamples := streamN(NewOscillator(Sine, 1000.0, sr, 0, 0, 0, nil, 0), numSamples)
	inputRMS := rms(inputSamples)

	// LP filter with Cutoff=0.1 → cutoff ≈ 39 Hz, well below 1000 Hz
	f := Filter{Type: FilterLowPass, Cutoff: 0.1}
	filtered := NewFilterStreamer(NewOscillator(Sine, 1000.0, sr, 0, 0, 0, nil, 0), sr, f)
	filteredRMS := rms(streamN(filtered, numSamples))

	if filteredRMS >= inputRMS*0.1 {
		t.Errorf("LP filter should heavily attenuate 1000 Hz; inputRMS=%v filteredRMS=%v",
			inputRMS, filteredRMS)
	}
}

func TestHighPassAttenuatesLowFreq(t *testing.T) {
	sr := beep.SampleRate(44100)
	const numSamples = 44100

	// Reference: 100 Hz sine, no filter
	inputSamples := streamN(NewOscillator(Sine, 100.0, sr, 0, 0, 0, nil, 0), numSamples)
	inputRMS := rms(inputSamples)

	// HP filter with Cutoff=0.9 → cutoff ≈ 9128 Hz, well above 100 Hz
	f := Filter{Type: FilterHighPass, Cutoff: 0.9}
	filtered := NewFilterStreamer(NewOscillator(Sine, 100.0, sr, 0, 0, 0, nil, 0), sr, f)
	filteredRMS := rms(streamN(filtered, numSamples))

	if filteredRMS >= inputRMS*0.1 {
		t.Errorf("HP filter should heavily attenuate 100 Hz; inputRMS=%v filteredRMS=%v",
			inputRMS, filteredRMS)
	}
}
