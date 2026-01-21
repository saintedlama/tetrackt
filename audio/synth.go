package audio

import (
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
)

type Mixer struct {
	Balance float64 // 0.0 = full left, 1.0 = full right
}

// Synth represents the audio synthesis engine
type Synth struct {
	sampleRate  beep.SampleRate
	oscillator1 Oscillator
	envelope1   Envelope
	oscillator2 Oscillator
	envelope2   Envelope
	mixer       Mixer
}

// NewSynth creates a new synthesis engine
func NewSynth(sampleRate beep.SampleRate, oscillator1 Oscillator, envelope1 Envelope, oscillator2 Oscillator, envelope2 Envelope, mixer Mixer) *Synth {
	return &Synth{
		sampleRate:  sampleRate,
		oscillator1: oscillator1,
		envelope1:   envelope1,
		oscillator2: oscillator2,
		envelope2:   envelope2,
		mixer:       mixer,
	}
}

func (s *Synth) Streamer(note Note, d time.Duration) beep.Streamer {
	frequency := note.Frequency()

	oscillator1 := NewOscillator(s.oscillator1.Type, frequency, s.sampleRate)
	oscillator2 := NewOscillator(s.oscillator2.Type, frequency, s.sampleRate)

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

	v := (s.mixer.Balance - 0.5) * 2 // Scale to -1 to 1
	mix1 := &effects.Volume{Streamer: streamer1, Base: 2, Volume: -v, Silent: v >= 1}
	mix2 := &effects.Volume{Streamer: streamer2, Base: 2, Volume: v, Silent: v <= -1}

	mixed := beep.Mix(mix1, mix2)

	return beep.Take(sampleDuration, mixed)
}
