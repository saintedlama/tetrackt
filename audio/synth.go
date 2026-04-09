package audio

import (
	"time"

	"github.com/gopxl/beep/v2"
)

// SampleRate is the number of audio samples per second.
// It is an alias for beep.SampleRate, exposed so callers outside of audio
// don't need to import beep directly.
type SampleRate = beep.SampleRate

// Streamer is the interface for pulling audio samples.
// It is an alias for beep.Streamer, exposed so callers outside of audio
// don't need to import beep directly.
type Streamer = beep.Streamer

// Synth represents the audio synthesis engine
type Synth struct {
	Oscillator1 Oscillator
	Envelope1   Envelope
	Oscillator2 Oscillator
	Envelope2   Envelope
	Mixer       Mixer
	Filter      Filter
	LFO1        LFO
	LFO2        LFO
	Portamento  float64 // glide duration in seconds; 0 = snap to pitch immediately
}

// NewSynth creates a new synthesis engine
func NewSynth(oscillator1 Oscillator, envelope1 Envelope, oscillator2 Oscillator, envelope2 Envelope, mixer Mixer, filter Filter, lfo1, lfo2 LFO) *Synth {
	return &Synth{
		Oscillator1: oscillator1,
		Envelope1:   envelope1,
		Oscillator2: oscillator2,
		Envelope2:   envelope2,
		Mixer:       mixer,
		Filter:      filter,
		LFO1:        lfo1,
		LFO2:        lfo2,
	}
}

// PlayParams holds per-trigger playback context passed to Synth.Streamer.
// It is separate from Synth (the instrument patch) so callers supply per-note
// information without mutating the patch.
type PlayParams struct {
	// Frequencies lists pitches to cycle across ticks.
	// A single-element slice plays one note; multiple elements enable arpeggio.
	Frequencies []float64
	// StartFreq enables a portamento glide from StartFreq to Frequencies[0].
	// Only active when Synth.Portamento > 0 and StartFreq > 0.
	StartFreq float64
	// Duration is the total note length.
	Duration time.Duration
	// Continuous keeps the ADSR envelope and LFOs running uninterrupted across
	// arpeggio ticks. When false, each tick gets a fresh synthesis chain.
	Continuous bool
}

// tickingStreamer wraps a synthesis pipeline and retunes oscillators at tick
// boundaries, enabling continuous arpeggio cycling within a single ADSR envelope.
type tickingStreamer struct {
	inner          beep.Streamer
	osc1           *oscillatorGenerator
	osc2           *oscillatorGenerator
	frequencies    []float64
	samplesPerTick int
	sampleCount    int
	tickIdx        int
}

func (t *tickingStreamer) Stream(samples [][2]float64) (int, bool) {
	total := 0
	for len(samples) > 0 {
		toNext := t.samplesPerTick - t.sampleCount
		if toNext <= 0 {
			toNext = 1
		}
		chunk := samples
		if len(chunk) > toNext {
			chunk = chunk[:toNext]
		}
		n, ok := t.inner.Stream(chunk)
		total += n
		t.sampleCount += n
		samples = samples[n:]
		if t.sampleCount >= t.samplesPerTick {
			t.sampleCount = 0
			t.tickIdx++
			freq := t.frequencies[t.tickIdx%len(t.frequencies)]
			t.osc1.SetFrequency(freq)
			t.osc2.SetFrequency(freq)
		}
		if !ok || n == 0 {
			break
		}
	}
	return total, total > 0
}

func (t *tickingStreamer) Err() error { return t.inner.Err() }

// Streamer builds a playback pipeline from the given PlayParams.
//
// p.Frequencies is cycled across equal-length ticks; a single-element slice
// plays one note, multiple elements enable arpeggio.
//
// When p.Continuous is true the ADSR envelope and LFOs run uninterrupted across
// all arpeggio ticks (authentic chiptune behaviour). When false, each tick gets
// a fresh synthesis chain (envelope + LFOs reset), producing a gate/stutter effect.
//
// When Synth.Portamento > 0 and p.StartFreq > 0, a pitch glide is applied from
// p.StartFreq to p.Frequencies[0].
func (s *Synth) Streamer(sampleRate beep.SampleRate, p PlayParams) beep.Streamer {
	frequencies := p.Frequencies
	if len(frequencies) == 0 {
		frequencies = []float64{0}
	}
	tickCount := len(frequencies)
	totalSamples := sampleRate.N(p.Duration)

	if p.Continuous && tickCount > 1 {
		// One synthesis chain; oscillators retune at tick boundaries.
		chain, osc1, osc2 := s.buildChain(sampleRate, 0, frequencies[0], totalSamples)
		samplesPerTick := totalSamples / tickCount
		ticking := &tickingStreamer{
			inner:          chain,
			osc1:           osc1,
			osc2:           osc2,
			frequencies:    frequencies,
			samplesPerTick: samplesPerTick,
		}
		return beep.Take(totalSamples, ticking)
	}

	if tickCount == 1 {
		startFreq := 0.0
		if s.Portamento > 0 {
			startFreq = p.StartFreq
		}
		chain, _, _ := s.buildChain(sampleRate, startFreq, frequencies[0], totalSamples)
		return beep.Take(totalSamples, chain)
	}

	// p.Continuous=false with multiple ticks: fresh ADSR/LFO per tick.
	tickSamples := totalSamples / tickCount
	seqStreamers := make([]beep.Streamer, tickCount)
	for i := 0; i < tickCount; i++ {
		freq := frequencies[i%len(frequencies)]
		chain, _, _ := s.buildChain(sampleRate, 0, freq, tickSamples)
		seqStreamers[i] = beep.Take(tickSamples, chain)
	}
	return beep.Seq(seqStreamers...)
}

// buildChain constructs a full synthesis pipeline for a single frequency and
// sample duration. Returns the terminal streamer and both oscillator handles.
// startFreq > 0 and s.Portamento > 0 enables a pitch glide from startFreq to frequency.
func (s *Synth) buildChain(sampleRate beep.SampleRate, startFreq, frequency float64, sampleDuration int) (beep.Streamer, *oscillatorGenerator, *oscillatorGenerator) {
	sr := float64(sampleRate)

	osc1 := NewOscillator(s.Oscillator1.Type, frequency, sampleRate, s.Oscillator1.Phase, s.Oscillator1.PulseWidth, s.Oscillator1.Detune)
	osc2 := NewOscillator(s.Oscillator2.Type, frequency, sampleRate, s.Oscillator2.Phase, s.Oscillator2.PulseWidth, s.Oscillator2.Detune)

	if s.Portamento > 0 && startFreq > 0 {
		portamentoSamples := int(s.Portamento * sr)
		osc1.startFrequency = startFreq * osc1.detuneMultiplier
		osc1.targetFrequency = osc1.frequency
		osc1.portamentoSamples = portamentoSamples
		osc1.frequency = osc1.startFrequency
		osc2.startFrequency = startFreq * osc2.detuneMultiplier
		osc2.targetFrequency = osc2.frequency
		osc2.portamentoSamples = portamentoSamples
		osc2.frequency = osc2.startFrequency
	}

	makeLFO := func(dest ModDest) *lfoGenerator {
		if s.LFO1.Depth > 0 && s.LFO1.Dest == dest {
			return newLFOGenerator(s.LFO1, sr)
		}
		if s.LFO2.Depth > 0 && s.LFO2.Dest == dest {
			return newLFOGenerator(s.LFO2, sr)
		}
		return nil
	}

	src1 := newModulatedOscillatorStreamer(osc1, osc1.frequency, osc1.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth), makeLFO(ModDetune))
	src2 := newModulatedOscillatorStreamer(osc2, osc2.frequency, osc2.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth), makeLFO(ModDetune))

	streamer1 := NewEnvelope(src1, sampleDuration, s.Envelope1)
	streamer2 := NewEnvelope(src2, sampleDuration, s.Envelope2)

	mod1 := newModulatedVolumeStreamer(streamer1, makeLFO(ModVolume))
	mod2 := newModulatedVolumeStreamer(streamer2, makeLFO(ModVolume))

	mixed := s.Mixer.Mix(mod1, mod2)
	filtered := NewModulatedFilterStreamer(mixed, sampleRate, s.Filter, makeLFO(ModCutoff))

	return filtered, osc1, osc2
}
