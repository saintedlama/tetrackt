package audio

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
func (v Volume) Streamer(streamers ...Streamer) Streamer {
	var mixed Streamer
	if len(streamers) == 0 {
		mixed = &silenceStreamer{}
	} else {
		mixed = mixAll(streamers...)
	}
	return newScaledStreamer(mixed, v.Level)
}
