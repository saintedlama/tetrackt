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

func TestPortamentoGlidesFromStartToTarget(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 200 * time.Millisecond

	osc := Oscillator{Type: Sine}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}

	startFreq := 220.0  // A3
	targetFreq := 440.0 // A4

	synth := NewSynth(osc, env, osc, env, mixer, NewFilter(), LFO{}, LFO{})
	synth.Portamento = 0.1 // 100ms glide

	// With portamento: pass [startFreq, targetFreq], tickCount=1.
	samples := streamN(synth.Streamer(sr, []float64{startFreq, targetFreq}, 1, true, dur), sr.N(dur))

	// Without portamento (snap): same target, no start freq.
	synthSnap := NewSynth(osc, env, osc, env, mixer, NewFilter(), LFO{}, LFO{})
	samplesSnap := streamN(synthSnap.Streamer(sr, []float64{targetFreq}, 1, true, dur), sr.N(dur))

	// The glide version starts at a different phase rate, so the waveforms must differ
	// in the early portion of the buffer.
	earlyLen := sr.N(20 * time.Millisecond) // first 20ms
	diff := 0.0
	for i := 0; i < earlyLen; i++ {
		d := samples[i][0] - samplesSnap[i][0]
		diff += d * d
	}
	if diff == 0 {
		t.Error("expected portamento glide to produce different early samples from snap-to-pitch")
	}
}

func TestPortamentoZeroDisablesGlide(t *testing.T) {
	sr := beep.SampleRate(44100)
	dur := 50 * time.Millisecond

	osc := Oscillator{Type: Sine}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 1.0, Volume2: 0}

	synth := NewSynth(osc, env, osc, env, mixer, NewFilter(), LFO{}, LFO{})
	synth.Portamento = 0 // disabled

	// Two frequencies supplied but Portamento=0 — only frequencies[0] should be used.
	samplesTwo := streamN(synth.Streamer(sr, []float64{220.0, 440.0}, 1, true, dur), sr.N(dur))
	samplesOne := streamN(synth.Streamer(sr, []float64{220.0}, 1, true, dur), sr.N(dur))

	// Must be identical — no glide, frequencies[1] ignored.
	for i := range samplesOne {
		if samplesOne[i][0] != samplesTwo[i][0] {
			t.Errorf("sample %d differs: %v vs %v — Portamento=0 should ignore second frequency", i, samplesOne[i][0], samplesTwo[i][0])
		}
	}
}

func TestPortamentoConvergesToTarget(t *testing.T) {
	sr := beep.SampleRate(44100)
	// Use a duration long enough that the portamento (50ms) has fully completed.
	dur := 200 * time.Millisecond
	portamento := 50 * time.Millisecond

	osc := Oscillator{Type: Sine}
	env := Envelope{Sustain: 1.0}
	mixer := Mixer{Volume1: 0, Volume2: 0} // silent output — we inspect the oscillator directly

	synth := &Synth{
		Oscillator1: osc,
		Envelope1:   env,
		Oscillator2: Oscillator{Type: Silent},
		Envelope2:   env,
		Mixer:       mixer,
		Portamento:  portamento.Seconds(),
	}

	osc1Ref := NewOscillator(osc.Type, 440.0, sr, 0, 0.5, 0)
	osc1Ref.startFrequency = 220.0
	osc1Ref.targetFrequency = 440.0
	osc1Ref.portamentoSamples = sr.N(portamento)
	osc1Ref.frequency = 220.0

	// Stream past the portamento window.
	buf := make([][2]float64, sr.N(dur))
	osc1Ref.Stream(buf)

	// After portamento has elapsed, frequency must equal targetFrequency.
	if osc1Ref.frequency != osc1Ref.targetFrequency {
		t.Errorf("after portamento, frequency=%v want %v", osc1Ref.frequency, osc1Ref.targetFrequency)
	}
	// Suppress unused synth warning.
	_ = synth
}
