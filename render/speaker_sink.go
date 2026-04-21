package render

import (
	"io"
	"math"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/tetrackt/tetrackt/audio"
)

type SpeakerSink struct {
	mu         sync.Mutex
	ctx        *oto.Context
	sampleRate audio.SampleRate
	live       *queuedReader
	livePlayer *oto.Player
	oneShots   []*oto.Player
}

func NewSpeakerSink(sampleRate audio.SampleRate) *SpeakerSink {
	sink := &SpeakerSink{}
	_ = sink.Begin(sampleRate)
	return sink
}

func (s *SpeakerSink) Begin(sampleRate audio.SampleRate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx == nil {
		ctx, readyCh, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   int(sampleRate),
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   100 * time.Millisecond,
		})
		if err != nil {
			return err
		}
		<-readyCh
		s.ctx = ctx
		s.sampleRate = sampleRate
	}
	qr := &queuedReader{}
	player := s.ctx.NewPlayer(qr)
	player.Play()
	s.live = qr
	s.livePlayer = player
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
	player := s.livePlayer
	s.live = nil
	s.livePlayer = nil
	s.mu.Unlock()
	if player != nil {
		player.Close()
	}
	return nil
}

func (s *SpeakerSink) Clear() {
	s.mu.Lock()
	player := s.livePlayer
	oneShots := s.oneShots
	s.live = nil
	s.livePlayer = nil
	s.oneShots = nil
	s.mu.Unlock()
	if player != nil {
		player.Close()
	}
	for _, p := range oneShots {
		p.Close()
	}
}

func (s *SpeakerSink) Play(streamer audio.Streamer, globalVolume float64) {
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		return
	}
	r := &pcmReader{streamer: audio.NewVolume(globalVolume).Streamer(streamer)}
	player := ctx.NewPlayer(r)
	s.mu.Lock()
	s.oneShots = append(s.oneShots, player)
	s.mu.Unlock()
	player.Play()
	go func() {
		for player.IsPlaying() {
			time.Sleep(time.Millisecond)
		}
		player.Close()
		s.mu.Lock()
		for i, p := range s.oneShots {
			if p == player {
				s.oneShots = append(s.oneShots[:i], s.oneShots[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()
}

// sampleStreamer plays back a pre-rendered slice of samples, used by RenderToStream.
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

func (s *sampleStreamer) Err() error { return nil }

// queuedReader is an io.Reader fed by Write() calls during live pattern playback.
// When there is no queued audio it returns silence so the oto player stays active.
type queuedReader struct {
	mu     sync.Mutex
	queue  []byte
	closed bool
}

func (r *queuedReader) Append(samples [][2]float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	for _, s := range samples {
		l := floatToInt16(s[0])
		ch := floatToInt16(s[1])
		r.queue = append(r.queue,
			byte(uint16(l)),
			byte(uint16(l)>>8),
			byte(uint16(ch)),
			byte(uint16(ch)>>8),
		)
	}
}

func (r *queuedReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed && len(r.queue) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.queue)
	r.queue = r.queue[n:]
	for i := n; i < len(p); i++ {
		p[i] = 0
	}
	return len(p), nil
}

// pcmReader wraps an audio.Streamer and exposes it as an io.Reader of 16-bit
// little-endian stereo PCM, used for one-shot note previews in Play().
type pcmReader struct {
	streamer audio.Streamer
	buf      [][2]float64
	pcm      []byte
	offset   int
	done     bool
}

func (r *pcmReader) Read(p []byte) (int, error) {
	written := 0
	for written < len(p) {
		if r.offset < len(r.pcm) {
			n := copy(p[written:], r.pcm[r.offset:])
			r.offset += n
			written += n
			continue
		}
		if r.done {
			for i := written; i < len(p); i++ {
				p[i] = 0
			}
			return len(p), io.EOF
		}
		need := (len(p) - written) / 4
		if need == 0 {
			break
		}
		if need > 512 {
			need = 512
		}
		if len(r.buf) < need {
			r.buf = make([][2]float64, need)
		}
		n, ok := r.streamer.Stream(r.buf[:need])
		if !ok {
			r.done = true
		}
		if n == 0 {
			continue
		}
		r.pcm = r.pcm[:0]
		for _, s := range r.buf[:n] {
			l := floatToInt16(s[0])
			ch := floatToInt16(s[1])
			r.pcm = append(r.pcm,
				byte(uint16(l)),
				byte(uint16(l)>>8),
				byte(uint16(ch)),
				byte(uint16(ch)>>8),
			)
		}
		r.offset = 0
	}
	return written, nil
}

func floatToInt16(v float64) int16 {
	if v > 1 {
		v = 1
	} else if v < -1 {
		v = -1
	}
	return int16(v * math.MaxInt16)
}
