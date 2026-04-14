package render

import (
	"fmt"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"
)

type WavExportOptions struct {
	SampleRate   audio.SampleRate
	GlobalVolume float64
	LoopCount    int
}

type WAVSink struct {
	sampleRate audio.SampleRate
	frames     [][2]float64
	outputPath string
}

func ExportWAVFromTracks(model *tracker.TrackerModel, wavPath string, opts WavExportOptions) error {
	if wavPath == "" {
		return fmt.Errorf("render: output path is empty")
	}
	if opts.SampleRate <= 0 {
		opts.SampleRate = 44100
	}
	if opts.GlobalVolume < 0 {
		opts.GlobalVolume = 1.0
	}
	if opts.LoopCount <= 0 {
		opts.LoopCount = 1
	}

	engine := NewRenderEngine(model, RenderConfig{
		SampleRate:   opts.SampleRate,
		GlobalVolume: opts.GlobalVolume,
		LoopCount:    opts.LoopCount,
	})
	return engine.Run(&WAVSink{outputPath: wavPath})
}

func (s *WAVSink) Begin(sampleRate audio.SampleRate) error {
	s.sampleRate = sampleRate
	s.frames = nil
	return nil
}

func (s *WAVSink) Write(samples [][2]float64) error {
	if len(samples) == 0 {
		return nil
	}
	s.frames = append(s.frames, samples...)
	return nil
}

func (s *WAVSink) End() error {
	if s.outputPath == "" {
		return fmt.Errorf("render: output path is empty")
	}
	return audio.WriteWAV(s.outputPath, s.sampleRate, s.frames)
}
