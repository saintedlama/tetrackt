package audio

import (
	"math"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/generators"
)

// Volume controls the output gain of a signal chain.
// Level is a linear gain in [0, 1]; 1.0 is unity gain, 0 is silent.
type Volume struct {
	Level float64
}

// NewVolume returns a Volume with the given linear gain level.
func NewVolume(level float64) Volume {
	return Volume{Level: level}
}

// Streamer mixes the given streamers and applies the volume level.
// Passing zero streamers returns a silent, immediately-exhausted stream.
func (v Volume) Streamer(streamers ...beep.Streamer) beep.Streamer {
	var mixed beep.Streamer
	if len(streamers) == 0 {
		mixed = generators.Silence(-1)
	} else {
		mixed = beep.Mix(streamers...)
	}
	return &effects.Volume{
		Streamer: mixed,
		Base:     2,
		Volume:   volumeToDecibels(v.Level),
		Silent:   v.Level == 0,
	}
}

// volumeToDecibels converts a linear [0,1] gain to decibels (base-2 log scale).
func volumeToDecibels(level float64) float64 {
	if level <= 0 {
		return -999
	}
	return math.Log2(level) * 6
}
