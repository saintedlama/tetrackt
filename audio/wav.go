package audio

import (
	"os"
	"path/filepath"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/wav"
)

func WriteWAV(outputPath string, sampleRate SampleRate, frames [][2]float64) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	dir := filepath.Dir(outputPath)
	tmpFile, err := os.CreateTemp(dir, "tetrackt-*.wav")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	cleanup := func() {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
	}

	streamer := &sampleStreamer{samples: frames}
	format := beep.Format{SampleRate: beep.SampleRate(sampleRate), NumChannels: 2, Precision: 2}
	if err := wav.Encode(tmpFile, streamer, format); err != nil {
		cleanup()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Remove(outputPath)
	if err := os.Rename(tmpPath, outputPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

type sampleStreamer struct {
	samples [][2]float64
	offset  int
}

func (s *sampleStreamer) Stream(buf [][2]float64) (int, bool) {
	if s.offset >= len(s.samples) {
		return 0, false
	}
	n := copy(buf, s.samples[s.offset:])
	s.offset += n
	return n, s.offset < len(s.samples)
}

func (s *sampleStreamer) Err() error {
	return nil
}
