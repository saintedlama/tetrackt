package audio

import (
	"math"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
)

type Mixer struct {
	Volume1 float64 // 0.0 to 1.0, independent volume for oscillator 1
	Volume2 float64 // 0.0 to 1.0, independent volume for oscillator 2
}

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

// Streamer builds a playback pipeline for the given frequencies over duration d.
//
// frequencies is cycled across tickCount equal-length ticks. Passing a single
// frequency with tickCount=1 is equivalent to the previous single-note API.
//
// When continuous=true the ADSR envelope and LFOs run uninterrupted across all
// ticks (authentic chiptune behaviour); oscillators are simply retuned at each
// tick boundary. When continuous=false each tick gets a fresh synthesis chain
// (envelope + LFOs reset), producing a gate/stutter effect.
func (s *Synth) Streamer(sampleRate beep.SampleRate, frequencies []float64, tickCount int, continuous bool, d time.Duration) beep.Streamer {
	if len(frequencies) == 0 {
		frequencies = []float64{0}
	}
	if tickCount <= 0 {
		tickCount = 1
	}
	totalSamples := sampleRate.N(d)

	if continuous && tickCount > 1 {
		// One synthesis chain; oscillators retune at tick boundaries.
		chain, osc1, osc2 := s.buildChain(sampleRate, frequencies[0], totalSamples)
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
		chain, _, _ := s.buildChain(sampleRate, frequencies[0], totalSamples)
		return beep.Take(totalSamples, chain)
	}

	// continuous=false with multiple ticks: fresh ADSR/LFO per tick.
	tickSamples := totalSamples / tickCount
	seqStreamers := make([]beep.Streamer, tickCount)
	for i := 0; i < tickCount; i++ {
		freq := frequencies[i%len(frequencies)]
		chain, _, _ := s.buildChain(sampleRate, freq, tickSamples)
		seqStreamers[i] = beep.Take(tickSamples, chain)
	}
	return beep.Seq(seqStreamers...)
}

// buildChain constructs a full synthesis pipeline for a single frequency and
// sample duration. Returns the terminal streamer and both oscillator handles.
func (s *Synth) buildChain(sampleRate beep.SampleRate, frequency float64, sampleDuration int) (beep.Streamer, *oscillatorGenerator, *oscillatorGenerator) {
	sr := float64(sampleRate)

	osc1 := NewOscillator(s.Oscillator1.Type, frequency, sampleRate, s.Oscillator1.Phase, s.Oscillator1.PulseWidth)
	osc2 := NewOscillator(s.Oscillator2.Type, frequency, sampleRate, s.Oscillator2.Phase, s.Oscillator2.PulseWidth)

	makeLFO := func(dest ModDest) *lfoGenerator {
		if s.LFO1.Depth > 0 && s.LFO1.Dest == dest {
			return newLFOGenerator(s.LFO1, sr)
		}
		if s.LFO2.Depth > 0 && s.LFO2.Dest == dest {
			return newLFOGenerator(s.LFO2, sr)
		}
		return nil
	}

	src1 := newModulatedOscillatorStreamer(osc1, frequency, osc1.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth))
	src2 := newModulatedOscillatorStreamer(osc2, frequency, osc2.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth))

	streamer1 := NewEnvelope(src1, sampleDuration, s.Envelope1)
	streamer2 := NewEnvelope(src2, sampleDuration, s.Envelope2)

	mod1 := newModulatedVolumeStreamer(streamer1, makeLFO(ModVolume))
	mod2 := newModulatedVolumeStreamer(streamer2, makeLFO(ModVolume))

	mix1 := &effects.Volume{Streamer: mod1, Base: 2, Volume: math.Log2(s.Mixer.Volume1), Silent: s.Mixer.Volume1 == 0}
	mix2 := &effects.Volume{Streamer: mod2, Base: 2, Volume: math.Log2(s.Mixer.Volume2), Silent: s.Mixer.Volume2 == 0}

	mixed := beep.Mix(mix1, mix2)
	filtered := NewModulatedFilterStreamer(mixed, sampleRate, s.Filter, makeLFO(ModCutoff))

	return filtered, osc1, osc2
}
