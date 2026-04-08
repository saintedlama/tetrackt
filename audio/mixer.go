package audio

import (
	"math"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
)

// Mixer holds per-oscillator volume levels and knows how to combine two
// streamers into a single mixed output stream.
type Mixer struct {
	Volume1 float64 // 0.0 to 1.0, independent volume for oscillator 1
	Volume2 float64 // 0.0 to 1.0, independent volume for oscillator 2
}

// Mix applies Volume1 to s1 and Volume2 to s2, then returns a single
// beep.Streamer that streams both channels summed together.
func (m Mixer) Mix(s1, s2 beep.Streamer) beep.Streamer {
	v1 := &effects.Volume{Streamer: s1, Base: 2, Volume: math.Log2(m.Volume1), Silent: m.Volume1 == 0}
	v2 := &effects.Volume{Streamer: s2, Base: 2, Volume: math.Log2(m.Volume2), Silent: m.Volume2 == 0}
	return beep.Mix(v1, v2)
}
