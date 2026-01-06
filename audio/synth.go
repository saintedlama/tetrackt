package audio

import (
	"github.com/gopxl/beep/v2"
)

// Synth represents the audio synthesis engine
type Synth struct {
	SampleRate beep.SampleRate
}

// NewSynth creates a new synthesis engine
func NewSynth(sampleRate beep.SampleRate) *Synth {
	return &Synth{
		SampleRate: sampleRate,
	}
}
