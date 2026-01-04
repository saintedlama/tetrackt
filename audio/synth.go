package audio

import (
	"math"

	"github.com/faiface/beep"
)

// WaveformType represents the type of waveform to generate
type WaveformType string

const (
	Sine            WaveformType = "sine"
	Square          WaveformType = "square"
	Triangle        WaveformType = "triangle"
	Sawtooth        WaveformType = "sawtooth"
	SawtoothReverse WaveformType = "sawtooth_reverse"
)

// Synth represents the audio synthesis engine
type Synth struct {
	sampleRate beep.SampleRate
}

// NewSynth creates a new synthesis engine
func NewSynth(sampleRate beep.SampleRate) *Synth {
	return &Synth{
		sampleRate: sampleRate,
	}
}

// Start initializes the audio engine
func (s *Synth) Start() error {
	// TODO: Initialize beep speaker
	return nil
}

// Stop shuts down the audio engine
func (s *Synth) Stop() error {
	// TODO: Stop beep speaker
	return nil
}

// NewGenerator creates a beep.Streamer that generates the specified waveform
func (s *Synth) NewGenerator(waveType WaveformType, frequency float64) beep.Streamer {
	return &generator{
		waveType:   waveType,
		frequency:  frequency,
		sampleRate: s.sampleRate,
		phase:      0,
	}
}

// generator implements beep.Streamer for waveform generation
type generator struct {
	waveType   WaveformType
	frequency  float64
	sampleRate beep.SampleRate
	phase      float64
}

// Stream fills the samples buffer with waveform data
func (g *generator) Stream(samples [][2]float64) (n int, ok bool) {
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
func (g *generator) Err() error {
	return nil
}
