package ui

import "github.com/tetrackt/tetrackt/audio"

type TrackChanged struct {
	Synth *audio.Synth
}

type SynthUpdated struct {
	Synth *audio.Synth
}
