package render

import (
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/tetrackt/tetrackt/audio"
)

type SpeakerSink struct {
	mu          sync.Mutex
	initialized bool
	sampleRate  audio.SampleRate
}

func NewSpeakerSink(sampleRate audio.SampleRate) *SpeakerSink {
	sink := &SpeakerSink{}
	_ = sink.Begin(sampleRate)
	return sink
}

func (s *SpeakerSink) Begin(sampleRate audio.SampleRate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initialized {
		return nil
	}
	bufferSize := sampleRate.N(100 * time.Millisecond)
	speaker.Init(beep.SampleRate(sampleRate), bufferSize)
	s.sampleRate = sampleRate
	s.initialized = true
	return nil
}

func (s *SpeakerSink) Write(samples [][2]float64) error {
	if len(samples) == 0 {
		return nil
	}
	speaker.Play(audio.NewVolume(1.0).Streamer(&sampleStreamer{samples: append([][2]float64(nil), samples...)}))
	return nil
}

func (s *SpeakerSink) End() error {
	return nil
}

func (s *SpeakerSink) Clear() {
	speaker.Clear()
}

func (s *SpeakerSink) Play(streamer audio.Streamer, globalVolume float64) {
	speaker.Play(audio.NewVolume(globalVolume).Streamer(streamer))
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
