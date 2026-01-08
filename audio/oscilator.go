package audio

import (
	"math"
	"math/rand/v2"

	"github.com/gopxl/beep/v2"
)

// OscilatorType represents the type of oscilator waveform to generate
type OscilatorType string

const (
	Sine            OscilatorType = "sine"
	Square          OscilatorType = "square"
	Triangle        OscilatorType = "triangle"
	Sawtooth        OscilatorType = "sawtooth"
	SawtoothReverse OscilatorType = "sawtooth_reverse"
	Noise           OscilatorType = "noise"
)

// NewOscilator creates a beep.Streamer that generates the specified oscilator waveform
func (s *Synth) NewOscilator(waveType OscilatorType, frequency float64) beep.Streamer {
	return &oscilatorGenerator{
		waveType:   waveType,
		frequency:  frequency,
		sampleRate: s.SampleRate,
		phase:      0,
	}
}

// oscilatorGenerator implements beep.Streamer for oscilator waveform generation
type oscilatorGenerator struct {
	waveType   OscilatorType
	frequency  float64
	sampleRate beep.SampleRate
	phase      float64
}

// Stream fills the samples buffer with oscilator waveform data
func (g *oscilatorGenerator) Stream(samples [][2]float64) (n int, ok bool) {
	phaseIncrement := g.frequency / float64(g.sampleRate)

	for i := range samples {
		var sample float64

		switch g.waveType {
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
func (g *oscilatorGenerator) Err() error {
	return nil
}
