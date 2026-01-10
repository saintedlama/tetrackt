package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tetrackt/tetrackt/audio"
)

type envelopePreset struct {
	Name    string
	Type    string
	Attack  float64
	Decay   float64
	Sustain float64
	Release float64
}

type PresetModel struct {
	presetIndex   int
	envelope      audio.Envelope
	selectedStyle lipgloss.Style
}

var envelopePresets = []envelopePreset{
	{Name: "Off", Type: "Utility", Attack: 0.00, Decay: 0.00, Sustain: 1.00, Release: 0.00},
	{Name: "Pluck Clean", Type: "Pluck", Attack: 0.01, Decay: 0.12, Sustain: 0.00, Release: 0.10},
	{Name: "Bright Lead", Type: "Lead", Attack: 0.02, Decay: 0.10, Sustain: 0.60, Release: 0.12},
	{Name: "Organ Hold", Type: "Organ", Attack: 0.00, Decay: 0.00, Sustain: 1.00, Release: 0.08},
	{Name: "Perc Hit", Type: "Percussion", Attack: 0.00, Decay: 0.18, Sustain: 0.00, Release: 0.20},
	{Name: "Bass Pluck", Type: "Bass", Attack: 0.01, Decay: 0.15, Sustain: 0.10, Release: 0.08},
	{Name: "Piano", Type: "Keys", Attack: 0.02, Decay: 0.40, Sustain: 0.20, Release: 0.15},
	{Name: "Brass", Type: "Brass", Attack: 0.12, Decay: 0.25, Sustain: 0.70, Release: 0.20},
	{Name: "Warm Strings", Type: "Strings", Attack: 0.50, Decay: 0.20, Sustain: 0.80, Release: 0.25},
	{Name: "Soft Pad", Type: "Pad", Attack: 0.30, Decay: 0.30, Sustain: 0.80, Release: 0.35},
	{Name: "Slow Swell", Type: "Pad", Attack: 0.65, Decay: 0.10, Sustain: 0.90, Release: 0.20},
	{Name: "Blip Lead", Type: "Chiptune", Attack: 0.00, Decay: 0.10, Sustain: 0.20, Release: 0.05},
	{Name: "Square Stab", Type: "Chiptune", Attack: 0.00, Decay: 0.08, Sustain: 0.00, Release: 0.04},
	{Name: "Arp Pluck", Type: "Chiptune", Attack: 0.00, Decay: 0.12, Sustain: 0.10, Release: 0.06},
	{Name: "Pulse Bass", Type: "Chiptune", Attack: 0.00, Decay: 0.18, Sustain: 0.50, Release: 0.12},
	{Name: "Duty Sweep", Type: "Chiptune", Attack: 0.00, Decay: 0.25, Sustain: 0.30, Release: 0.18},
	{Name: "Noise Hat", Type: "Chiptune", Attack: 0.00, Decay: 0.05, Sustain: 0.00, Release: 0.03},
	{Name: "Click Kick", Type: "Chiptune", Attack: 0.00, Decay: 0.20, Sustain: 0.00, Release: 0.08},
	{Name: "Glide Pad 8-bit", Type: "Chiptune", Attack: 0.05, Decay: 0.30, Sustain: 0.35, Release: 0.25},
	{Name: "Game Intro Bell", Type: "Chiptune", Attack: 0.01, Decay: 0.35, Sustain: 0.15, Release: 0.30},
	{Name: "Laser Zap", Type: "Chiptune", Attack: 0.00, Decay: 0.10, Sustain: 0.00, Release: 0.12},
}

func NewPresetModel(selectedStyle lipgloss.Style) PresetModel {
	return PresetModel{
		presetIndex:   0,
		envelope:      audio.Envelope{},
		selectedStyle: selectedStyle,
	}
}

func (m PresetModel) Update(msg tea.KeyMsg) PresetModel {
	switch msg.String() {
	case "up":
		m.presetIndex = (m.presetIndex - 1 + len(envelopePresets)) % len(envelopePresets)
	case "down":
		m.presetIndex = (m.presetIndex + 1) % len(envelopePresets)
	}

	// TODO: Do I need this or expose a function to get the envelope?
	m.envelope.Attack = envelopePresets[m.presetIndex].Attack
	m.envelope.Decay = envelopePresets[m.presetIndex].Decay
	m.envelope.Sustain = envelopePresets[m.presetIndex].Sustain
	m.envelope.Release = envelopePresets[m.presetIndex].Release

	return m
}

func (m PresetModel) View() string {
	var view strings.Builder
	view.WriteString("Envelope Presets (Enter to apply, Esc to cancel)\n")

	for idx, p := range envelopePresets {
		line := fmt.Sprintf("%s (%s)  A:%3d%% D:%3d%% S:%3d%% R:%3d%%",
			p.Name,
			p.Type,
			int(p.Attack*100),
			int(p.Decay*100),
			int(p.Sustain*100),
			int(p.Release*100),
		)

		if idx == m.presetIndex {
			line = m.selectedStyle.Render(line)
		}

		view.WriteString(line + "\n")
	}

	return view.String()
}
