package render

import (
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/audio/effects"
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

	subticks := row.Ticks
	durationMs := float64(rowDuration) / float64(time.Millisecond)
	fx := rowToEffectDefs(row, subticks)
	ep := effects.NewEffectsPatch(synth, fx, durationMs, subticks)
	streamer := ep.Streamer(sampleRate, row.Frequency, p.prevFrequency)
	p.prevFrequency = row.Frequency

	vol := globalVolume
	if row.Volume > 0 {
		vol *= row.Volume
	}

	p.sink.Play(streamer, vol)
	return false
}

func (p *PreviewPlayer) Tick() bool {
	return false
}
