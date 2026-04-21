package render

import (
	"time"

	"github.com/tetrackt/tetrackt/audio"
)

type PreviewPlayer struct {
	sink          *SpeakerSink
	prevFrequency float64
}

func NewPreviewPlayer(sink *SpeakerSink) PreviewPlayer {
	return PreviewPlayer{sink: sink}
}

func (p *PreviewPlayer) Reset() {
	p.prevFrequency = 0
}

func (p *PreviewPlayer) Start(row Row, synth *audio.Synth, rowDuration time.Duration, sampleRate audio.SampleRate, globalVolume float64) bool {
	if row.Frequency == 0 || synth == nil {
		return false
	}

	durationMs := float64(rowDuration) / float64(time.Millisecond)
	ep := audio.NewEffectsPatch(synth, row.FX, durationMs)
	streamer := ep.Streamer(sampleRate, row.Frequency, p.prevFrequency)
	p.prevFrequency = row.Frequency

	p.sink.Play(streamer, globalVolume)
	return false
}

func (p *PreviewPlayer) Tick() bool {
	return false
}
