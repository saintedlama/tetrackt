package audio

import (
	"math"
	"math/rand/v2"

	"github.com/gopxl/beep/v2"
)

type Oscillator struct {
	Type       OscillatorType
	Phase      float64 // normalized initial phase [0..1)
	PulseWidth float64 // duty cycle [0.01..0.99]; only used by Square; 0 defaults to 0.5
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
	Silent          OscillatorType = "silent"
)

// NewOscillator creates an oscillatorGenerator for the specified waveform.
// pulseWidth is used only by Square; a zero value defaults to 0.5 (50% duty).
func NewOscillator(oscillatorType OscillatorType, frequency float64, sampleRate beep.SampleRate, initialPhase float64, pulseWidth float64) *oscillatorGenerator {
	pw := pulseWidth
	if pw == 0 {
		pw = 0.5
	}
	return &oscillatorGenerator{
		oscillatorType: oscillatorType,
		frequency:      frequency,
		sampleRate:     sampleRate,
		phase:          math.Mod(initialPhase, 1.0),
		pulseWidth:     pw,
	}
}

// oscillatorGenerator implements beep.Streamer for oscillator waveform generation
type oscillatorGenerator struct {
	oscillatorType OscillatorType
	frequency      float64
	sampleRate     beep.SampleRate
	phase          float64
	pulseWidth     float64 // resolved duty cycle [0.01..0.99]
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
			if g.phase < g.pulseWidth {
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

		case Silent:
			sample = 0
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
