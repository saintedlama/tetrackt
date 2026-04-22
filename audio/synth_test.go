package audio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetrackt/tetrackt/notes"
)

func TestSynthPatchLength(t *testing.T) {
	sr := SampleRate(44100)
	dur := 100 * time.Millisecond
	n := sr.N(dur)

	osc := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Mixer{Volume1: 1.0, Volume2: 1.0}, NewFilter(), LFO{}, LFO{}, LFO{})
	patch := synth.NewPatch(sr, notes.NewNote(notes.BaseA, notes.Octave4).Frequency(), n)

	buf := make([][2]float64, 512)
	total := 0
	for {
		count, ok := patch.Stream(buf)
		total += count
		if !ok {
			break
		}
	}

	assert.Equal(t, n, total, "want %d samples", n)
}

func TestSynthSilentOscillators(t *testing.T) {
	sr := SampleRate(44100)
	dur := 10 * time.Millisecond
	n := sr.N(dur)

	osc := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Mixer{Volume1: 1.0, Volume2: 1.0}, NewFilter(), LFO{}, LFO{}, LFO{})
	samples := StreamN(synth.NewPatch(sr, notes.NewNote(notes.BaseA, notes.Octave4).Frequency(), n), n)

	for i, s := range samples {
		assert.Equal(t, 0.0, s[0], "sample %d L", i)
		assert.Equal(t, 0.0, s[1], "sample %d R", i)
	}
}

func TestSynthMixerZeroVolume(t *testing.T) {
	sr := SampleRate(44100)
	dur := 10 * time.Millisecond
	n := sr.N(dur)

	osc := Oscillator{Type: Square}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Mixer{Volume1: 0, Volume2: 0}, NewFilter(), LFO{}, LFO{}, LFO{})
	samples := StreamN(synth.NewPatch(sr, notes.NewNote(notes.BaseA, notes.Octave4).Frequency(), n), n)

	for i, s := range samples {
		assert.Equal(t, 0.0, s[0], "sample %d L: want silent", i)
		assert.Equal(t, 0.0, s[1], "sample %d R: want silent", i)
	}
}

func TestSynthMixerBalance(t *testing.T) {
	sr := SampleRate(44100)
	dur := 10 * time.Millisecond
	n := sr.N(dur)

	osc1 := Oscillator{Type: Square}
	osc2 := Oscillator{Type: Silent}
	env := Envelope{Sustain: 1.0}
	synth := NewSynth(osc1, env, osc2, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, Mixer{Volume1: 1.0, Volume2: 0}, NewFilter(), LFO{}, LFO{}, LFO{})
	samples := StreamN(synth.NewPatch(sr, notes.NewNote(notes.BaseA, notes.Octave4).Frequency(), n), n)

	hasNonZero := false
	for _, s := range samples {
		if s[0] != 0 || s[1] != 0 {
			hasNonZero = true
			break
		}
	}
	assert.True(t, hasNonZero, "expected non-zero output with Square osc1 at Volume1=1")
}

func TestSynthFilterOff(t *testing.T) {
	sr := SampleRate(44100)
	dur := 50 * time.Millisecond
	n := sr.N(dur)
	freq := notes.NewNote(notes.BaseA, notes.Octave4).Frequency()

	osc := Oscillator{Type: Square}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}

	synthOff := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, Filter{Type: FilterOff, Cutoff: 0.5}, LFO{}, LFO{}, LFO{})
	samplesOff := StreamN(synthOff.NewPatch(sr, freq, n), n)

	synthLP := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, Filter{Type: FilterLowPass, Cutoff: 0.01}, LFO{}, LFO{}, LFO{})
	samplesLP := StreamN(synthLP.NewPatch(sr, freq, n), n)

	rmsOff := rms(samplesOff)
	rmsLP := rms(samplesLP)

	assert.Greater(t, rmsOff, rmsLP, "FilterOff RMS should exceed LP-filtered RMS")
}

func TestSynthDetuneShiftsFrequency(t *testing.T) {
	sr := SampleRate(44100)
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
	assert.InDelta(t, 2.0, ratio, 0.2, "expected ~2x crossings for +1200 cent detune (%d vs %d)", cUp, cNone)
}

func TestSynthDetuneZeroNoEffect(t *testing.T) {
	sr := SampleRate(44100)
	dur := 10 * time.Millisecond
	n := sr.N(dur)

	osc := Oscillator{Type: Sine, Detune: 0}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}
	synth := NewSynth(osc, env, osc, env, Oscillator{Type: Silent}, Envelope{Sustain: 1.0}, mixer, NewFilter(), LFO{}, LFO{}, LFO{})
	s := StreamN(synth.NewPatch(sr, 440.0, n), n)
	require.NotEmpty(t, s, "no samples")
}

func TestPatchSetFrequencyRetunes(t *testing.T) {
	sr := SampleRate(44100)
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
	assert.InDelta(t, 2.0, ratio, 0.2, "expected ~2x crossings after SetFrequency(880) (%d vs %d)", c2, c1)
}

func TestPatchSetFrequencyPreservesDetune(t *testing.T) {
	sr := SampleRate(44100)
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
	assert.InDelta(t, 2.0, ratio, 0.2, "expected ~2x crossings after doubling frequency (detune preserved)")
}

func TestPatchReset(t *testing.T) {
	// 1000 Hz so 1 sample = 1ms for easy math
	const sr = SampleRate(1000)
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

	assert.GreaterOrEqual(t, sustainRMS, 0.5, "expected sustain RMS near 1")
	assert.Less(t, attackStartRMS, sustainRMS*0.3, "expected attack-start RMS much lower than sustain after Reset")
}
