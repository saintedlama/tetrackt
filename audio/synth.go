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

// ActiveVoice is the handle returned by Synth.Streamer. It wraps the composed
// beep.Streamer and exposes SetFrequency for live retuning (e.g. arpeggio).
type ActiveVoice struct {
	beep.Streamer
	osc1 *oscillatorGenerator
	osc2 *oscillatorGenerator
}

// SetFrequency retunes both oscillators to hz. Call between speaker.Lock/Unlock.
func (av *ActiveVoice) SetFrequency(hz float64) {
	av.osc1.SetFrequency(hz)
	av.osc2.SetFrequency(hz)
}

func (s *Synth) Streamer(sampleRate beep.SampleRate, frequency float64, d time.Duration) *ActiveVoice {
	osc1 := NewOscillator(s.Oscillator1.Type, frequency, sampleRate, s.Oscillator1.Phase, s.Oscillator1.PulseWidth)
	osc2 := NewOscillator(s.Oscillator2.Type, frequency, sampleRate, s.Oscillator2.Phase, s.Oscillator2.PulseWidth)

	sampleDuration := sampleRate.N(d)
	sr := float64(sampleRate)

	// Helper: create a fresh lfoGenerator for the given destination, or nil.
	// LFO1 is checked before LFO2; if both target the same destination, LFO1 wins.
	// An LFO with Depth == 0 is considered disabled.
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

	return &ActiveVoice{
		Streamer: beep.Take(sampleDuration, filtered),
		osc1:     osc1,
		osc2:     osc2,
	}
}
