package render

import "github.com/tetrackt/tetrackt/audio"

type bufferSink struct {
	frames [][2]float64
}

func (s *bufferSink) Begin(sampleRate audio.SampleRate) error {
	s.frames = nil
	return nil
}

func (s *bufferSink) Write(samples [][2]float64) error {
	s.frames = append(s.frames, samples...)
	return nil
}

func (s *bufferSink) End() error {
	return nil
}
