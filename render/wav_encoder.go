package render

import (
	"io"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/youpy/go-wav"
)

// EncodeWAV writes a standard RIFF/WAV file with 16-bit signed PCM stereo data to w.
func EncodeWAV(w io.Writer, sampleRate audio.SampleRate, frames [][2]float64) error {
	const numChannels = 2
	const bitsPerSample = 16

	numSamples := uint32(len(frames))
	
	// Create WAV writer
	writer := wav.NewWriter(w, numSamples, numChannels, uint32(sampleRate), bitsPerSample)

	// Convert float64 frames to wav.Sample
	samples := make([]wav.Sample, numSamples)
	for i, frame := range frames {
		samples[i] = wav.Sample{
			Values: [2]int{
				int(floatToInt16(frame[0])), // left channel
				int(floatToInt16(frame[1])), // right channel
			},
		}
	}

	// Write samples
	return writer.WriteSamples(samples)
}
