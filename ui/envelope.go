package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tetrackt/tetrackt/audio"
)

// EnvelopeEditField represents which envelope parameter is being edited
type EnvelopeEditField int

const (
	EnvelopeAttack EnvelopeEditField = iota
	EnvelopeDecay
	EnvelopeSustain
	EnvelopeRelease
)

type EnvelopeModel struct {
	envelopeField EnvelopeEditField

	Attack  PercentageKnob
	Decay   PercentageKnob
	Sustain PercentageKnob
	Release PercentageKnob
}

func NewEnvelopeModel(selectedStyle lipgloss.Style, envelope audio.Envelope) EnvelopeModel {
	return EnvelopeModel{
		envelopeField: EnvelopeAttack,
		Attack:        NewPercentageKnob("Attack", envelope.Attack, true, selectedStyle),
		Decay:         NewPercentageKnob("Decay", envelope.Decay, false, selectedStyle),
		Sustain:       NewPercentageKnob("Sustain", envelope.Sustain, false, selectedStyle),
		Release:       NewPercentageKnob("Release", envelope.Release, false, selectedStyle),
	}
}

func (m EnvelopeModel) Update(msg tea.KeyMsg) EnvelopeModel {
	switch msg.String() {
	case "up":
		// Move to previous envelope field
		m.envelopeField = (m.envelopeField - 1 + 4) % 4
	case "down":
		// Move to next envelope field
		m.envelopeField = (m.envelopeField + 1) % 4
	case "left":
		// Decrease value by 1%
		m.adjustEnvelopeValue(-0.01)
	case "shift+left":
		// Decrease value by 10%
		m.adjustEnvelopeValue(-0.10)
	case "right":
		// Increase value by 1%
		m.adjustEnvelopeValue(0.01)
	case "shift+right":
		// Increase value by 10%
		m.adjustEnvelopeValue(0.10)
	}

	// TODO: Use a selection index instead of "selected" flags in knobs
	// TODO: Refactor to avoid repetition
	m.Attack.Selected = false
	m.Decay.Selected = false
	m.Sustain.Selected = false
	m.Release.Selected = false

	if m.envelopeField == EnvelopeAttack {
		m.Attack.Selected = true
	}
	if m.envelopeField == EnvelopeDecay {
		m.Decay.Selected = true
	}
	if m.envelopeField == EnvelopeSustain {
		m.Sustain.Selected = true
	}
	if m.envelopeField == EnvelopeRelease {
		m.Release.Selected = true
	}

	return m
}

// adjustEnvelopeValue adjusts the current envelope field by a delta value
func (m *EnvelopeModel) adjustEnvelopeValue(delta float64) {
	var currentValue *float64

	switch m.envelopeField {
	case EnvelopeAttack:
		currentValue = &m.Attack.Value
	case EnvelopeDecay:
		currentValue = &m.Decay.Value
	case EnvelopeSustain:
		currentValue = &m.Sustain.Value
	case EnvelopeRelease:
		currentValue = &m.Release.Value
	}

	if currentValue != nil {
		newValue := *currentValue + delta

		// For A, D, R: prevent increases that would make A+D+R exceed 100
		if m.envelopeField != EnvelopeSustain && delta > 0 {
			otherSum := m.Attack.Value + m.Decay.Value + m.Release.Value - *currentValue
			if newValue+otherSum > 1.0 {
				return // block the increase
			}
		}

		// Clamp value between 0 and 1.0
		if newValue < 0 {
			newValue = 0
		} else if newValue > 1.0 {
			newValue = 1.0
		}

		*currentValue = newValue
	}
}

func (m EnvelopeModel) View() string {
	envView := strings.Builder{}
	envView.WriteString("ADSR Envelope:\n")

	envView.WriteString(m.Attack.View() + "\n")
	envView.WriteString(m.Decay.View() + "\n")
	envView.WriteString(m.Sustain.View() + "\n")
	envView.WriteString(m.Release.View() + "\n")

	return envView.String()
}
