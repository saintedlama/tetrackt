package render

import (
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"
)

type PreviewPlayer struct {
	sink *SpeakerSink
}

func NewPreviewPlayer(sink *SpeakerSink) PreviewPlayer {
	return PreviewPlayer{sink: sink}
}

func (p *PreviewPlayer) Reset() {}

func (p *PreviewPlayer) Start(row tracker.TrackRow, synth *audio.Synth, bpm int, sampleRate audio.SampleRate, globalVolume float64) bool {
	if audio.IsOff(row.Note) || synth == nil {
		return false
	}

	// TODO: This is a bit hacky
	tm := tracker.NewTracker(1, 1, 0, 0)
	tm.BPM = tracker.NewBPM(bpm)
	tm.Tracks[0].Synth = synth
	tm.Tracks[0].Rows[0] = row

	collector := &bufferSink{}
	engine := NewRenderEngine(tm, RenderConfig{
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
