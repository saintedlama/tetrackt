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
	sampleRate  beep.SampleRate
	oscillator1 Oscillator
	envelope1   Envelope
	oscillator2 Oscillator
	envelope2   Envelope
	mixer       Mixer
	filter      Filter
	LFOs        map[ModDest]*LFO // optional modulation sources; nil = no modulation
}

// NewSynth creates a new synthesis engine
func NewSynth(sampleRate beep.SampleRate, oscillator1 Oscillator, envelope1 Envelope, oscillator2 Oscillator, envelope2 Envelope, mixer Mixer, filter Filter) *Synth {
	return &Synth{
		sampleRate:  sampleRate,
		oscillator1: oscillator1,
		envelope1:   envelope1,
		oscillator2: oscillator2,
		envelope2:   envelope2,
		mixer:       mixer,
		filter:      filter,
	}
}

func (s *Synth) Streamer(note Note, d time.Duration) beep.Streamer {
	frequency := note.Frequency()

	osc1 := NewOscillator(s.oscillator1.Type, frequency, s.sampleRate, s.oscillator1.Phase, s.oscillator1.PulseWidth)
	osc2 := NewOscillator(s.oscillator2.Type, frequency, s.sampleRate, s.oscillator2.Phase, s.oscillator2.PulseWidth)

	sampleDuration := s.sampleRate.N(d)
	sr := float64(s.sampleRate)

	// Helper: create a fresh lfoGenerator for the given destination, or nil.
	makeLFO := func(dest ModDest) *lfoGenerator {
		if s.LFOs == nil {
			return nil
		}
		if lfo := s.LFOs[dest]; lfo != nil {
			return newLFOGenerator(*lfo, sr)
		}
		return nil
	}

	src1 := newModulatedOscillatorStreamer(osc1, frequency, osc1.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth))
	src2 := newModulatedOscillatorStreamer(osc2, frequency, osc2.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth))

	streamer1 := NewEnvelope(src1, sampleDuration, s.envelope1)
	streamer2 := NewEnvelope(src2, sampleDuration, s.envelope2)

	mod1 := newModulatedVolumeStreamer(streamer1, makeLFO(ModVolume))
	mod2 := newModulatedVolumeStreamer(streamer2, makeLFO(ModVolume))

	mix1 := &effects.Volume{Streamer: mod1, Base: 2, Volume: math.Log2(s.mixer.Volume1), Silent: s.mixer.Volume1 == 0}
	mix2 := &effects.Volume{Streamer: mod2, Base: 2, Volume: math.Log2(s.mixer.Volume2), Silent: s.mixer.Volume2 == 0}

	mixed := beep.Mix(mix1, mix2)
	filtered := NewModulatedFilterStreamer(mixed, s.sampleRate, s.filter, makeLFO(ModCutoff))

	return beep.Take(sampleDuration, filtered)
}
