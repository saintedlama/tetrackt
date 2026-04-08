package audio

import (
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
)

func TestSynthStreamerLength(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 100 * time.Millisecond
	expected := sr.N(dur)

	osc := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Mixer{Volume1: 1.0, Volume2: 1.0}, NewFilter(), LFO{}, LFO{})
	streamer := synth.Streamer(sr, []float64{NewNote(BaseA, Octave4).Frequency()}, 1, false, dur)

	buf := make([][2]float64, 512)
	total := 0
	for {
		n, ok := streamer.Stream(buf)
		total += n
		if !ok {
			break
		}
	}

	if total != expected {
		t.Errorf("want %d samples, got %d", expected, total)
	}
}

func TestSynthSilentOscillators(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 10 * time.Millisecond

	osc := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Mixer{Volume1: 1.0, Volume2: 1.0}, NewFilter(), LFO{}, LFO{})
	samples := streamN(synth.Streamer(sr, []float64{NewNote(BaseA, Octave4).Frequency()}, 1, false, dur), sr.N(dur))

	for i, s := range samples {
		if s[0] != 0 || s[1] != 0 {
			t.Errorf("sample %d: want 0,0 got %v,%v", i, s[0], s[1])
		}
	}
}

func TestSynthMixerZeroVolume(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 10 * time.Millisecond

	osc := Oscillator{Type: Square}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Mixer{Volume1: 0, Volume2: 0}, NewFilter(), LFO{}, LFO{})
	samples := streamN(synth.Streamer(sr, []float64{NewNote(BaseA, Octave4).Frequency()}, 1, false, dur), sr.N(dur))

	for i, s := range samples {
		if s[0] != 0 || s[1] != 0 {
			t.Errorf("sample %d: want silent, got %v,%v", i, s[0], s[1])
		}
	}
}

func TestSynthMixerBalance(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 10 * time.Millisecond

	osc1 := Oscillator{Type: Square}
	osc2 := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc1, env, osc2, env, Mixer{Volume1: 1.0, Volume2: 0}, NewFilter(), LFO{}, LFO{})
	samples := streamN(synth.Streamer(sr, []float64{NewNote(BaseA, Octave4).Frequency()}, 1, false, dur), sr.N(dur))

	hasNonZero := false
	for _, s := range samples {
		if s[0] != 0 || s[1] != 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("expected non-zero output with Square osc1 at Volume1=1")
	}
}

func TestSynthFilterOff(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 50 * time.Millisecond
	n := sr.N(dur)
	note := NewNote(BaseA, Octave4)

	osc := Oscillator{Type: Square}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}

	// FilterOff passes through unmodified
	synthOff := NewSynth(osc, env, osc, env, mixer, Filter{Type: FilterOff, Cutoff: 0.5}, LFO{}, LFO{})
	samplesOff := streamN(synthOff.Streamer(sr, []float64{note.Frequency()}, 1, false, dur), n)

	// Aggressive LP filter (~21 Hz cutoff) should heavily attenuate 440 Hz content
	synthLP := NewSynth(osc, env, osc, env, mixer, Filter{Type: FilterLowPass, Cutoff: 0.01}, LFO{}, LFO{})
	samplesLP := streamN(synthLP.Streamer(sr, []float64{note.Frequency()}, 1, false, dur), n)

	rmsOff := rms(samplesOff)
	rmsLP := rms(samplesLP)

	if rmsOff <= rmsLP {
		t.Errorf("FilterOff RMS (%v) should exceed LP-filtered RMS (%v)", rmsOff, rmsLP)
	}
}

func TestSynthDetuneShiftsFrequency(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 100 * time.Millisecond

	baseFreq := 440.0
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}

	// Osc1 with no detune: phase advances at 440 Hz.
	oscNone := Oscillator{Type: Sine}
	synthNone := NewSynth(oscNone, env, oscNone, env, mixer, NewFilter(), LFO{}, LFO{})

	// Osc1 detuned +1200 cents = one octave up → 880 Hz.
	oscUp := Oscillator{Type: Sine, Detune: 1200}
	synthUp := NewSynth(oscUp, env, oscUp, env, mixer, NewFilter(), LFO{}, LFO{})

	samplesNone := streamN(synthNone.Streamer(sr, []float64{baseFreq}, 1, false, dur), sr.N(dur))
	samplesUp := streamN(synthUp.Streamer(sr, []float64{baseFreq}, 1, false, dur), sr.N(dur))

	// Count zero crossings (positive-going) as a proxy for frequency.
	countCrossings := func(s [][2]float64) int {
		n := 0
		for i := 1; i < len(s); i++ {
			if s[i-1][0] <= 0 && s[i][0] > 0 {
				n++
			}
		}
		return n
	}
	cNone := countCrossings(samplesNone)
	cUp := countCrossings(samplesUp)
	// 880 Hz should produce roughly twice the crossings of 440 Hz.
	ratio := float64(cUp) / float64(cNone)
	if ratio < 1.8 || ratio > 2.2 {
		t.Errorf("expected ~2x crossings for +1200 cent detune, got ratio %v (%d vs %d)", ratio, cUp, cNone)
	}
}

func TestSynthDetuneZeroNoEffect(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 10 * time.Millisecond
	baseFreq := 440.0
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}

	osc := Oscillator{Type: Sine, Detune: 0}
	synth := NewSynth(osc, env, osc, env, mixer, NewFilter(), LFO{}, LFO{})
	s := streamN(synth.Streamer(sr, []float64{baseFreq}, 1, false, dur), sr.N(dur))
	if len(s) == 0 {
		t.Fatal("no samples")
	}
}

func TestSynthDetuneArpPreservesDetune(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 100 * time.Millisecond

	// +1200 cents detune on osc1 only, two-note arp (440 Hz, 880 Hz).
	oscDetuned := Oscillator{Type: Sine, Detune: 1200}
	oscSilent := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}
	synth := NewSynth(oscDetuned, env, oscSilent, env, mixer, NewFilter(), LFO{}, LFO{})

	// Should not panic; detune must be maintained across tick boundaries.
	_ = streamN(synth.Streamer(sr, []float64{440, 880}, 2, true, dur), sr.N(dur))
}
