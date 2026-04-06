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
	synth := NewSynth(sr, osc, env, osc, env, Mixer{Volume1: 1.0, Volume2: 1.0}, NewFilter())
	streamer := synth.Streamer(NewNote(BaseA, Octave4), dur)

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
	synth := NewSynth(sr, osc, env, osc, env, Mixer{Volume1: 1.0, Volume2: 1.0}, NewFilter())
	samples := streamN(synth.Streamer(NewNote(BaseA, Octave4), dur), sr.N(dur))

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
	synth := NewSynth(sr, osc, env, osc, env, Mixer{Volume1: 0, Volume2: 0}, NewFilter())
	samples := streamN(synth.Streamer(NewNote(BaseA, Octave4), dur), sr.N(dur))

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
	synth := NewSynth(sr, osc1, env, osc2, env, Mixer{Volume1: 1.0, Volume2: 0}, NewFilter())
	samples := streamN(synth.Streamer(NewNote(BaseA, Octave4), dur), sr.N(dur))

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
	synthOff := NewSynth(sr, osc, env, osc, env, mixer, Filter{Type: FilterOff, Cutoff: 0.5})
	samplesOff := streamN(synthOff.Streamer(note, dur), n)

	// Aggressive LP filter (~21 Hz cutoff) should heavily attenuate 440 Hz content
	synthLP := NewSynth(sr, osc, env, osc, env, mixer, Filter{Type: FilterLowPass, Cutoff: 0.01})
	samplesLP := streamN(synthLP.Streamer(note, dur), n)

	rmsOff := rms(samplesOff)
	rmsLP := rms(samplesLP)

	if rmsOff <= rmsLP {
		t.Errorf("FilterOff RMS (%v) should exceed LP-filtered RMS (%v)", rmsOff, rmsLP)
	}
}
