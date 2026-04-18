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
	live        *queuedStreamer
}

func NewSpeakerSink(sampleRate audio.SampleRate) *SpeakerSink {
	sink := &SpeakerSink{}
	_ = sink.Begin(sampleRate)
	return sink
}

func (s *SpeakerSink) Begin(sampleRate audio.SampleRate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.initialized {
		bufferSize := sampleRate.N(100 * time.Millisecond)
		speaker.Init(beep.SampleRate(sampleRate), bufferSize)
		s.sampleRate = sampleRate
		s.initialized = true
	}
	s.live = &queuedStreamer{}
	speaker.Play(s.live)
	return nil
}

func (s *SpeakerSink) Write(samples [][2]float64) error {
	if len(samples) == 0 {
		return nil
	}
	s.mu.Lock()
	live := s.live
	s.mu.Unlock()
	if live == nil {
		return nil
	}
	live.Append(samples)
	return nil
}

func (s *SpeakerSink) End() error {
	s.mu.Lock()
	if s.live != nil {
		s.live.Stop()
		s.live = nil
	}
	s.mu.Unlock()
	return nil
}

func (s *SpeakerSink) Clear() {
	s.mu.Lock()
	if s.live != nil {
		s.live.Stop()
		s.live = nil
	}
	s.mu.Unlock()
	speaker.Clear()
}

func (s *SpeakerSink) Play(streamer audio.Streamer, globalVolume float64) {
	speaker.Play(audio.NewVolume(globalVolume).Streamer(streamer))
}

type sampleStreamer struct {
	samples [][2]float64
	offset  int
	loop    bool
}

func (s *sampleStreamer) Stream(buf [][2]float64) (int, bool) {
	if len(s.samples) == 0 {
		return 0, false
	}
	total := 0
	for total < len(buf) {
		n := copy(buf[total:], s.samples[s.offset:])
		total += n
		s.offset += n
		if s.offset >= len(s.samples) {
			if !s.loop {
				return total, false
			}
			s.offset = 0
		}
	}
	return total, true
}

func (s *sampleStreamer) Err() error {
	return nil
}

type queuedStreamer struct {
	mu     sync.Mutex
	queue  [][2]float64
	closed bool
	offset int
}

func (s *queuedStreamer) Append(samples [][2]float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || len(samples) == 0 {
		return
	}
	s.queue = append(s.queue, samples...)
}

func (s *queuedStreamer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.queue = nil
	s.offset = 0
}

func (s *queuedStreamer) Stream(buf [][2]float64) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, false
	}
	if len(s.queue) == 0 {
		for i := range buf {
			buf[i] = [2]float64{}
		}
		return len(buf), true
	}

	n := copy(buf, s.queue)
	for i := n; i < len(buf); i++ {
		buf[i] = [2]float64{}
	}
	if n == len(s.queue) {
		s.queue = s.queue[:0]
	} else {
		s.queue = append(s.queue[:0], s.queue[n:]...)
	}
	return len(buf), true
}

func (s *queuedStreamer) Err() error {
	return nil
}
