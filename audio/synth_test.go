package audio

import (
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
)

func TestSynthPatchLength(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 100 * time.Millisecond
	n := sr.N(dur)

	osc := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Mixer{Volume1: 1.0, Volume2: 1.0}, NewFilter(), LFO{}, LFO{}, LFO{})
	patch := synth.NewPatch(sr, NewNote(BaseA, Octave4).Frequency(), n)

	buf := make([][2]float64, 512)
	total := 0
	limited := beep.Take(n, patch)
	for {
		count, ok := limited.Stream(buf)
		total += count
		if !ok {
			break
		}
	}

	if total != n {
		t.Errorf("want %d samples, got %d", n, total)
	}
}

func TestSynthSilentOscillators(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 10 * time.Millisecond
	n := sr.N(dur)

	osc := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Mixer{Volume1: 1.0, Volume2: 1.0}, NewFilter(), LFO{}, LFO{}, LFO{})
	samples := StreamN(synth.NewPatch(sr, NewNote(BaseA, Octave4).Frequency(), n), n)

	for i, s := range samples {
		if s[0] != 0 || s[1] != 0 {
			t.Errorf("sample %d: want 0,0 got %v,%v", i, s[0], s[1])
		}
	}
}

func TestSynthMixerZeroVolume(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 10 * time.Millisecond
	n := sr.N(dur)

	osc := Oscillator{Type: Square}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Mixer{Volume1: 0, Volume2: 0}, NewFilter(), LFO{}, LFO{}, LFO{})
	samples := StreamN(synth.NewPatch(sr, NewNote(BaseA, Octave4).Frequency(), n), n)

	for i, s := range samples {
		if s[0] != 0 || s[1] != 0 {
			t.Errorf("sample %d: want silent, got %v,%v", i, s[0], s[1])
		}
	}
}

func TestSynthMixerBalance(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 10 * time.Millisecond
	n := sr.N(dur)

	osc1 := Oscillator{Type: Square}
	osc2 := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc1, env, osc2, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Mixer{Volume1: 1.0, Volume2: 0}, NewFilter(), LFO{}, LFO{}, LFO{})
	samples := StreamN(synth.NewPatch(sr, NewNote(BaseA, Octave4).Frequency(), n), n)

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
	freq := NewNote(BaseA, Octave4).Frequency()

	osc := Oscillator{Type: Square}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}

	synthOff := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, Filter{Type: FilterOff, Cutoff: 0.5}, LFO{}, LFO{}, LFO{})
	samplesOff := StreamN(synthOff.NewPatch(sr, freq, n), n)

	synthLP := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, Filter{Type: FilterLowPass, Cutoff: 0.01}, LFO{}, LFO{}, LFO{})
	samplesLP := StreamN(synthLP.NewPatch(sr, freq, n), n)

	rmsOff := rms(samplesOff)
	rmsLP := rms(samplesLP)

	if rmsOff <= rmsLP {
		t.Errorf("FilterOff RMS (%v) should exceed LP-filtered RMS (%v)", rmsOff, rmsLP)
	}
}

func TestSynthDetuneShiftsFrequency(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 100 * time.Millisecond
	n := sr.N(dur)
	baseFreq := 440.0
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}

	oscNone := Oscillator{Type: Sine}
	synthNone := NewSynth(oscNone, env, oscNone, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, NewFilter(), LFO{}, LFO{}, LFO{})

	oscUp := Oscillator{Type: Sine, Detune: 1200}
	synthUp := NewSynth(oscUp, env, oscUp, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, NewFilter(), LFO{}, LFO{}, LFO{})

	samplesNone := StreamN(synthNone.NewPatch(sr, baseFreq, n), n)
	samplesUp := StreamN(synthUp.NewPatch(sr, baseFreq, n), n)

	countCrossings := func(s [][2]float64) int {
		count := 0
		for i := 1; i < len(s); i++ {
			if s[i-1][0] <= 0 && s[i][0] > 0 {
				count++
			}
		}
		return count
	}
	cNone := countCrossings(samplesNone)
	cUp := countCrossings(samplesUp)
	ratio := float64(cUp) / float64(cNone)
	if ratio < 1.8 || ratio > 2.2 {
		t.Errorf("expected ~2x crossings for +1200 cent detune, got ratio %v (%d vs %d)", ratio, cUp, cNone)
	}
}

func TestSynthDetuneZeroNoEffect(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 10 * time.Millisecond
	n := sr.N(dur)

	osc := Oscillator{Type: Sine, Detune: 0}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, NewFilter(), LFO{}, LFO{}, LFO{})
	s := StreamN(synth.NewPatch(sr, 440.0, n), n)
	if len(s) == 0 {
		t.Fatal("no samples")
	}
}

func TestPatchSetFrequencyRetunes(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 100 * time.Millisecond
	n := sr.N(dur)
	half := n / 2

	osc := Oscillator{Type: Sine}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, NewFilter(), LFO{}, LFO{}, LFO{})

	patch := synth.NewPatch(sr, 440.0, n)
	first := StreamN(patch, half)

	patch.SetFrequency(880.0)
	second := StreamN(patch, half)

	countCrossings := func(s [][2]float64) int {
		count := 0
		for i := 1; i < len(s); i++ {
			if s[i-1][0] <= 0 && s[i][0] > 0 {
				count++
			}
		}
		return count
	}
	c1 := countCrossings(first)
	c2 := countCrossings(second)
	ratio := float64(c2) / float64(c1)
	if ratio < 1.8 || ratio > 2.2 {
		t.Errorf("expected ~2x crossings after SetFrequency(880), got ratio %v (%d vs %d)", ratio, c2, c1)
	}
}

func TestPatchSetFrequencyPreservesDetune(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 100 * time.Millisecond
	n := sr.N(dur)

	// +1200 cents detune on osc1 → frequency doubles
	oscDetuned := Oscillator{Type: Sine, Detune: 1200}
	oscSilent := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}
	synth := NewSynth(oscDetuned, env, oscSilent, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, NewFilter(), LFO{}, LFO{}, LFO{})

	// base=220Hz + detune → 440Hz; after SetFrequency base=440Hz + detune → 880Hz
	patch := synth.NewPatch(sr, 220.0, n)
	first := StreamN(patch, n/2)
	patch.SetFrequency(440.0)
	second := StreamN(patch, n/2)

	countCrossings := func(s [][2]float64) int {
		count := 0
		for i := 1; i < len(s); i++ {
			if s[i-1][0] <= 0 && s[i][0] > 0 {
				count++
			}
		}
		return count
	}
	ratio := float64(countCrossings(second)) / float64(countCrossings(first))
	if ratio < 1.8 || ratio > 2.2 {
		t.Errorf("expected ~2x crossings after doubling frequency (detune preserved), got ratio %v", ratio)
	}
}

func TestPatchReset(t *testing.T) {
	// 1000 Hz so 1 sample = 1ms for easy math
	const sr = beep.SampleRate(1000)
	const n = 1000

	osc := Oscillator{Type: Sine}
	// 200ms attack at 1000Hz → 200 sample attack, then flat sustain
	env := Envelope{Attack: 200 * time.Millisecond, Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}
	synth := NewSynth(osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, NewFilter(), LFO{}, LFO{}, LFO{})

	patch := synth.NewPatch(sr, 50.0, n)

	// Stream 300 samples: 200 attack + 100 into sustain
	StreamN(patch, 300)
	sustainSamples := StreamN(patch, 100)
	sustainRMS := rms(sustainSamples)

	// Reset and stream 10 samples from the very start of attack
	patch.Reset()
	attackStartSamples := StreamN(patch, 10)
	attackStartRMS := rms(attackStartSamples)

	if sustainRMS < 0.5 {
		t.Errorf("expected sustain RMS near 1, got %v", sustainRMS)
	}
	if attackStartRMS > sustainRMS*0.3 {
		t.Errorf("expected attack-start RMS much lower than sustain after Reset, got %v vs %v", attackStartRMS, sustainRMS)
	}
}
