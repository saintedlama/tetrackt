package render

import (
	"math"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"
)

type PreviewPlayer struct {
	sink            *SpeakerSink
	previewPatch    *audio.Patch
	previewArp      audio.ArpeggioEffect
	previewBaseFreq float64
	previewSubTick  int
	previewSpeed    int
}

func NewPreviewPlayer(sink *SpeakerSink) PreviewPlayer {
	return PreviewPlayer{sink: sink}
}

func (p *PreviewPlayer) Reset() {
	p.previewPatch = nil
	p.previewArp = audio.ArpeggioEffect{}
	p.previewBaseFreq = 0
	p.previewSubTick = 0
	p.previewSpeed = 0
}

func (p *PreviewPlayer) Start(note audio.Note, arp audio.ArpeggioEffect, s *audio.Synth, duration time.Duration, sampleRate audio.SampleRate, globalVolume float64, speed int) bool {
	noteSamples := sampleRate.N(duration)
	patch := s.NewPatch(sampleRate, note.Frequency(), noteSamples)
	p.previewPatch = patch

	if arp.IsActive() {
		if speed <= 0 {
			speed = tracker.DefaultSpeed
		}
		p.previewArp = arp
		p.previewBaseFreq = note.Frequency()
		p.previewSubTick = 0
		p.previewSpeed = speed
		mult := math.Pow(2, float64(arp.Offsets[0])/12)
		patch.SetFrequency(p.previewBaseFreq * mult)
		p.sink.Play(patch, globalVolume)
		return true
	}

	p.previewArp = audio.ArpeggioEffect{}
	p.sink.Play(patch, globalVolume)
	return false
}

func (p *PreviewPlayer) Tick() bool {
	if p.previewPatch == nil || !p.previewArp.IsActive() {
		return false
	}
	p.previewSubTick++
	if p.previewSubTick >= p.previewSpeed {
		p.previewArp = audio.ArpeggioEffect{}
		return false
	}
	idx := p.previewSubTick % len(p.previewArp.Offsets)
	mult := math.Pow(2, float64(p.previewArp.Offsets[idx])/12)
	p.previewPatch.SetFrequency(p.previewBaseFreq * mult)
	return true
}
