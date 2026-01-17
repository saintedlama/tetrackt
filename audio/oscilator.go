package audio

import (
	"math"
	"math/rand/v2"

	"github.com/gopxl/beep/v2"
)

type Oscillator struct {
	Type OscillatorType
}

// OscillatorType represents the type of oscillator waveform to generate
type OscillatorType string

const (
	Sine            OscillatorType = "sine"
	Square          OscillatorType = "square"
	Triangle        OscillatorType = "triangle"
	Sawtooth        OscillatorType = "sawtooth"
	SawtoothReverse OscillatorType = "sawtooth_reverse"
	Noise           OscillatorType = "noise"
)

// NewOscillator creates a beep.Streamer that generates the specified oscillator waveform
func (s *Synth) NewOscillator(oscillatorType OscillatorType, frequency float64) beep.Streamer {
	return &oscillatorGenerator{
		oscillatorType: oscillatorType,
		frequency:      frequency,
		sampleRate:     s.SampleRate,
		phase:          0,
	}
}

// oscillatorGenerator implements beep.Streamer for oscillator waveform generation
type oscillatorGenerator struct {
	oscillatorType OscillatorType
	frequency      float64
	sampleRate     beep.SampleRate
	phase          float64
}

// Stream fills the samples buffer with oscillator waveform data
func (g *oscillatorGenerator) Stream(samples [][2]float64) (n int, ok bool) {
	phaseIncrement := g.frequency / float64(g.sampleRate)

	for i := range samples {
		var sample float64

		switch g.oscillatorType {
		case Sine:
			sample = math.Sin(2 * math.Pi * g.phase)

		case Square:
			if g.phase < 0.5 {
				sample = 1.0
			} else {
				sample = -1.0
			}

		case Triangle:
			if g.phase < 0.5 {
				sample = 4*g.phase - 1
			} else {
				sample = -4*g.phase + 3
			}

		case Sawtooth:
			sample = 2*g.phase - 1

		case SawtoothReverse:
			sample = 1 - 2*g.phase

		case Noise:
			sample = rand.Float64()*2 - 1
		}

		samples[i][0] = sample
		samples[i][1] = sample

		g.phase += phaseIncrement
		if g.phase >= 1.0 {
			g.phase -= 1.0
		}
	}

	return len(samples), true
}

// Err returns any error that occurred during streaming
func (g *oscillatorGenerator) Err() error {
	return nil
}
