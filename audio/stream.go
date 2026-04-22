package audio

import "time"

// SampleRate is the number of audio samples per second.
type SampleRate int

// N returns the number of samples needed for the given duration at this sample rate.
func (sr SampleRate) N(d time.Duration) int {
	return int(float64(sr)*d.Seconds() + 0.5)
}

// Streamer pulls stereo audio frames one buffer at a time.
// Stream fills samples with stereo frames and returns (n, ok):
//
//	n   number of frames written (may be less than len(samples) on final call)
//	ok  false when the stream is permanently exhausted
type Streamer interface {
	Stream(samples [][2]float64) (n int, ok bool)
	Err() error
}
