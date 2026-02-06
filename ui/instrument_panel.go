package ui

import (
	"github.com/tetrackt/tetrackt/audio"
)

// Instrument represents a complete instrument configuration
type Instrument struct {
	Name        string
	Oscillator1 audio.Oscillator
	Envelope1   audio.Envelope
	Oscillator2 audio.Oscillator
	Envelope2   audio.Envelope
	Mixer       audio.Mixer
}

// InstrumentPanel represents the UI component for managing instrument presets
type InstrumentPanel struct {
	Presets         []Instrument
	SelectedPreset  int
	CurrentTrackNum int
}

// NewInstrumentPanel initializes a new instrument panel with chiptune presets
func NewInstrumentPanel() *InstrumentPanel {
	presets := []Instrument{
		{
			Name:        "8-Bit Square Lead",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.1, Sustain: 0.7, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Silent, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 0, Release: 0},
			Mixer:       audio.Mixer{Balance: 0.0},
		},
		{
			Name:        "Classic Triangle",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.02, Decay: 0.15, Sustain: 0.6, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Silent, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 0, Release: 0},
			Mixer:       audio.Mixer{Balance: 0.0},
		},
		{
			Name:        "Retro Sawtooth",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.005, Decay: 0.2, Sustain: 0.5, Release: 0.25},
			Oscillator2: audio.Oscillator{Type: audio.Silent, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 0, Release: 0},
			Mixer:       audio.Mixer{Balance: 0.0},
		},
		{
			Name:        "Noise Percussion",
			Oscillator1: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0, Release: 0.05},
			Oscillator2: audio.Oscillator{Type: audio.Silent, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 0, Release: 0},
			Mixer:       audio.Mixer{Balance: 0.0},
		},
		{
			Name:        "Chiptune Bass",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.1, Sustain: 0.9, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.1, Sustain: 0.6, Release: 0.1},
			Mixer:       audio.Mixer{Balance: 0.3},
		},
		{
			Name:        "Synth Lead",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.7, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.25},
			Envelope2:   audio.Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.5, Release: 0.3},
			Mixer:       audio.Mixer{Balance: 0.4},
		},
		{
			Name:        "Vibrato Pad",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.3, Decay: 0.2, Sustain: 0.8, Release: 0.5},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.1},
			Envelope2:   audio.Envelope{Attack: 0.3, Decay: 0.2, Sustain: 0.8, Release: 0.5},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Arpeggiated Chords",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.4, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.33},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.4, Release: 0.1},
			Mixer:       audio.Mixer{Balance: 0.6},
		},
		{
			Name:        "Bit Crusher",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.SawtoothReverse, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "PWM Lead",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.15, Sustain: 0.7, Release: 0.25},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.15, Sustain: 0.7, Release: 0.25},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Vocal Synth",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.8, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Sawtooth, Phase: 0.25},
			Envelope2:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Mixer:       audio.Mixer{Balance: 0.35},
		},
		{
			Name:        "Chiptune Strings",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.2, Decay: 0.3, Sustain: 0.7, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.2, Decay: 0.3, Sustain: 0.5, Release: 0.4},
			Mixer:       audio.Mixer{Balance: 0.45},
		},
		{
			Name:        "Glitchy FX",
			Oscillator1: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.3, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.75},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.2, Release: 0.1},
			Mixer:       audio.Mixer{Balance: 0.6},
		},
		{
			Name:        "Retro Organ",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.05, Sustain: 0.9, Release: 0.15},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.05, Sustain: 0.8, Release: 0.15},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Digital Flute",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.08, Decay: 0.1, Sustain: 0.6, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.3},
			Envelope2:   audio.Envelope{Attack: 0.08, Decay: 0.1, Sustain: 0.4, Release: 0.3},
			Mixer:       audio.Mixer{Balance: 0.4},
		},
		{
			Name:        "Synth Brass",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.8, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Chiptune Bells",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.3, Sustain: 0.2, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.25},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.3, Sustain: 0.1, Release: 0.4},
			Mixer:       audio.Mixer{Balance: 0.6},
		},
		{
			Name:        "Funky Guitar",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0.4, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Sawtooth, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0.3, Release: 0.2},
			Mixer:       audio.Mixer{Balance: 0.45},
		},
		{
			Name:        "Ambient Pads",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.5, Decay: 0.3, Sustain: 0.9, Release: 0.8},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.6},
			Envelope2:   audio.Envelope{Attack: 0.5, Decay: 0.3, Sustain: 0.7, Release: 0.8},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Epic Lead",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.03, Decay: 0.2, Sustain: 0.8, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.3},
			Envelope2:   audio.Envelope{Attack: 0.03, Decay: 0.2, Sustain: 0.6, Release: 0.4},
			Mixer:       audio.Mixer{Balance: 0.45},
		},
		{
			Name:        "Game Over Sound",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.5, Sustain: 0, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.3, Sustain: 0, Release: 0.2},
			Mixer:       audio.Mixer{Balance: 0.7},
		},
		{
			Name:        "Retro Kick Drum",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0, Release: 0.05},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.08, Sustain: 0, Release: 0.03},
			Mixer:       audio.Mixer{Balance: 0.3},
		},
		{
			Name:        "Synthesized Choir",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.2, Decay: 0.2, Sustain: 0.9, Release: 0.5},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.2, Decay: 0.2, Sustain: 0.8, Release: 0.5},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Chiptune Harp",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.2, Sustain: 0.3, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.2},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.2, Sustain: 0.2, Release: 0.4},
			Mixer:       audio.Mixer{Balance: 0.6},
		},
		{
			Name:        "Vocoder Effect",
			Oscillator1: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.02, Decay: 0.1, Sustain: 0.7, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.02, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Retro Synth Bass",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.9, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.7, Release: 0.1},
			Mixer:       audio.Mixer{Balance: 0.35},
		},
		{
			Name:        "Chiptune Flanger",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.15, Sustain: 0.7, Release: 0.25},
			Oscillator2: audio.Oscillator{Type: audio.SawtoothReverse, Phase: 0.1},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.15, Sustain: 0.7, Release: 0.25},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Synthesized Trumpet",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.8, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.6, Release: 0.3},
			Mixer:       audio.Mixer{Balance: 0.45},
		},
		{
			Name:        "Chiptune Organ",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.05, Sustain: 0.9, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.05, Sustain: 0.8, Release: 0.2},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
		{
			Name:        "Retro Soundtrack",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.1, Decay: 0.2, Sustain: 0.8, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Sawtooth, Phase: 0.3},
			Envelope2:   audio.Envelope{Attack: 0.1, Decay: 0.2, Sustain: 0.6, Release: 0.4},
			Mixer:       audio.Mixer{Balance: 0.5},
		},
	}

	return &InstrumentPanel{
		Presets:         presets,
		SelectedPreset:  0,
		CurrentTrackNum: 0,
	}
}

// GetPreset returns the instrument at the specified index
func (ip *InstrumentPanel) GetPreset(index int) *Instrument {
	if index >= 0 && index < len(ip.Presets) {
		return &ip.Presets[index]
	}
	return nil
}

// ApplyPresetToTrack applies the selected preset to a track
func (ip *InstrumentPanel) ApplyPresetToTrack(preset *Instrument) (audio.Oscillator, audio.Envelope, audio.Oscillator, audio.Envelope, audio.Mixer) {
	return preset.Oscillator1, preset.Envelope1, preset.Oscillator2, preset.Envelope2, preset.Mixer
}
