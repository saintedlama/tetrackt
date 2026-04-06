package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/common"
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
	Envelope      audio.Envelope
}

type EnvelopeUpdated struct {
	Envelope audio.Envelope
}

func NewEnvelopeModel(envelope audio.Envelope) *EnvelopeModel {
	return &EnvelopeModel{
		envelopeField: EnvelopeAttack,
		Envelope:      envelope,
	}
}

func (m *EnvelopeModel) Init() tea.Cmd {
	return nil
}

func (m *EnvelopeModel) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			m.envelopeField = (m.envelopeField - 1 + 4) % 4
		case "down":
			m.envelopeField = (m.envelopeField + 1) % 4
		case "left":
			m.adjustEnvelopeValue(-0.01)
		case "shift+left":
			m.adjustEnvelopeValue(-0.10)
		case "right":
			m.adjustEnvelopeValue(0.01)
		case "shift+right":
			m.adjustEnvelopeValue(0.10)
		}
	}

	// TODO: Fired too often - only when value changes for clarity
	cmd = func() tea.Msg {
		return EnvelopeUpdated{
			Envelope: m.Envelope,
		}
	}

	return m, cmd
}

// adjustEnvelopeValue adjusts the current envelope field by a delta value
func (m *EnvelopeModel) adjustEnvelopeValue(delta float64) {
	var currentValue *float64

	switch m.envelopeField {
	case EnvelopeAttack:
		currentValue = &m.Envelope.Attack
	case EnvelopeDecay:
		currentValue = &m.Envelope.Decay
	case EnvelopeSustain:
		currentValue = &m.Envelope.Sustain
	case EnvelopeRelease:
		currentValue = &m.Envelope.Release
	}

	if currentValue != nil {
		newValue := *currentValue + delta

		// For A, D, R: prevent increases that would make A+D+R exceed 100
		if m.envelopeField != EnvelopeSustain && delta > 0 {
			otherSum := m.Envelope.Attack + m.Envelope.Decay + m.Envelope.Release - *currentValue
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

func (m *EnvelopeModel) View() string {
	envView := strings.Builder{}

	envView.WriteString(common.RenderKnobSelected("Attack", m.Envelope.Attack, m.envelopeField == EnvelopeAttack) + "\n")
	envView.WriteString(common.RenderKnobSelected("Decay", m.Envelope.Decay, m.envelopeField == EnvelopeDecay) + "\n")
	envView.WriteString(common.RenderKnobSelected("Sustain", m.Envelope.Sustain, m.envelopeField == EnvelopeSustain) + "\n")
	envView.WriteString(common.RenderKnobSelected("Release", m.Envelope.Release, m.envelopeField == EnvelopeRelease))

	return envView.String()
}
