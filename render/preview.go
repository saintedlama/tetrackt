package render

import (
	"time"

	"github.com/tetrackt/tetrackt/audio"
)

type PreviewPlayer struct {
	sink *SpeakerSink
}

func NewPreviewPlayer(sink *SpeakerSink) PreviewPlayer {
	return PreviewPlayer{sink: sink}
}

func (p *PreviewPlayer) Reset() {}

func (p *PreviewPlayer) Start(row Row, synth *audio.Synth, rowDuration time.Duration, sampleRate audio.SampleRate, globalVolume float64) bool {
	if row.Frequency == 0 || synth == nil {
		return false
	}

	song := &Pattern{
		Tracks:       []Track{{Synth: synth, Rows: []Row{row}}},
		NumRows:      1,
		NumTracks:    1,
		RowDuration:  rowDuration,
		DefaultTicks: 6,
	}

	collector := &bufferSink{}
	engine := NewRenderEngine(song, RenderConfig{
		SampleRate:   sampleRate,
		GlobalVolume: globalVolume,
		LoopCount:    1,
	})
	if err := engine.Run(collector); err != nil {
		return false
	}
	if len(collector.frames) == 0 {
		return false
	}
	p.sink.Play(&sampleStreamer{samples: collector.frames}, 1.0)
	return false
}

func (p *PreviewPlayer) Tick() bool {
	return false
}

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
