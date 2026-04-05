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

	oscillator1 := NewOscillator(s.oscillator1.Type, frequency, s.sampleRate, s.oscillator1.Phase)
	oscillator2 := NewOscillator(s.oscillator2.Type, frequency, s.sampleRate, s.oscillator2.Phase)

	sampleDuration := s.sampleRate.N(d)

	streamer1 := NewEnvelope(
		oscillator1,
		sampleDuration,
		Envelope{
			Attack:  s.envelope1.Attack,
			Decay:   s.envelope1.Decay,
			Sustain: s.envelope1.Sustain,
			Release: s.envelope1.Release,
		},
	)

	streamer2 := NewEnvelope(
		oscillator2,
		sampleDuration,
		Envelope{
			Attack:  s.envelope2.Attack,
			Decay:   s.envelope2.Decay,
			Sustain: s.envelope2.Sustain,
			Release: s.envelope2.Release,
		},
	)

	mix1 := &effects.Volume{Streamer: streamer1, Base: 2, Volume: math.Log2(s.mixer.Volume1), Silent: s.mixer.Volume1 == 0}
	mix2 := &effects.Volume{Streamer: streamer2, Base: 2, Volume: math.Log2(s.mixer.Volume2), Silent: s.mixer.Volume2 == 0}

	mixed := beep.Mix(mix1, mix2)
	filtered := NewFilterStreamer(mixed, s.sampleRate, s.filter)

	return beep.Take(sampleDuration, filtered)
}
