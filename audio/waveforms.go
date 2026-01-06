package audio

import (
	"math"
	"math/rand/v2"

	"github.com/gopxl/beep/v2"
)

// WaveformType represents the type of waveform to generate
type WaveformType string

const (
	Sine            WaveformType = "sine"
	Square          WaveformType = "square"
	Triangle        WaveformType = "triangle"
	Sawtooth        WaveformType = "sawtooth"
	SawtoothReverse WaveformType = "sawtooth_reverse"
	Noise           WaveformType = "noise"
)

// NewWaveform creates a beep.Streamer that generates the specified waveform
func (s *Synth) NewWaveform(waveType WaveformType, frequency float64) beep.Streamer {
	return &waveformGenerator{
		waveType:   waveType,
		frequency:  frequency,
		sampleRate: s.SampleRate,
		phase:      0,
	}
}

// waveformGenerator implements beep.Streamer for waveform generation
type waveformGenerator struct {
	waveType   WaveformType
	frequency  float64
	sampleRate beep.SampleRate
	phase      float64
}

// Stream fills the samples buffer with waveform data
func (g *waveformGenerator) Stream(samples [][2]float64) (n int, ok bool) {
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
func (g *waveformGenerator) Err() error {
	return nil
}
