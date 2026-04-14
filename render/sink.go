package render

import "github.com/tetrackt/tetrackt/audio"

// RenderSink receives mixed stereo sample frames from the offline render engine.
type RenderSink interface {
	Begin(sampleRate audio.SampleRate) error
	Write(samples [][2]float64) error
	End() error
}
