package render

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tetrackt/tetrackt/audio"
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

func ExportWAV(pattern *Pattern, wavPath string, opts WavExportOptions) error {
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

	engine := NewRenderEngine(pattern, RenderConfig{
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
	return writeFileAtomically(s.outputPath, func(w io.Writer) error {
		return EncodeWAV(w, s.sampleRate, s.frames)
	})
}

// writeFileAtomically writes to outputPath via a temp file + rename.
func writeFileAtomically(outputPath string, write func(io.Writer) error) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), "tetrackt-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() { tmp.Close(); os.Remove(tmpPath) }
	if err := write(tmp); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	os.Remove(outputPath)
	if err := os.Rename(tmpPath, outputPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
