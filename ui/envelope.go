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

	Attack  float64
	Decay   float64
	Sustain float64
	Release float64

	ShowModal     bool
	selectedStyle lipgloss.Style
	PresetModel   PresetModel
}

type EnvelopeUpdated struct {
	Envelope audio.Envelope
}

func NewEnvelopeModel(selectedStyle lipgloss.Style, envelope audio.Envelope) *EnvelopeModel {
	return &EnvelopeModel{
		envelopeField: EnvelopeAttack,
		Attack:        envelope.Attack,
		Decay:         envelope.Decay,
		Sustain:       envelope.Sustain,
		Release:       envelope.Release,

		selectedStyle: selectedStyle,
		PresetModel:   NewPresetModel(selectedStyle),
	}
}

// TODO: Need to get the values back to the track somehow (through main model?)
func (m *EnvelopeModel) Update(msg tea.Msg) (*EnvelopeModel, tea.Cmd) {
	var cmd tea.Cmd

	if m.ShowModal {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.Attack = m.PresetModel.envelope.Attack
				m.Decay = m.PresetModel.envelope.Decay
				m.Sustain = m.PresetModel.envelope.Sustain
				m.Release = m.PresetModel.envelope.Release
				m.ShowModal = false

				cmd = func() tea.Msg {
					return EnvelopeUpdated{
						Envelope: audio.Envelope{
							Attack:  m.Attack,
							Decay:   m.Decay,
							Sustain: m.Sustain,
							Release: m.Release,
						},
					}
				}

				return m, cmd
			case "esc", "p":
				m.ShowModal = false
				return m, nil
			}

			m.PresetModel = m.PresetModel.Update(msg)
			return m, nil
		}
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		case "p":
			m.ShowModal = !m.ShowModal
		}
	}

	// TODO: Fired too often - only when value changes for clarity
	cmd = func() tea.Msg {
		return EnvelopeUpdated{
			Envelope: audio.Envelope{
				Attack:  m.Attack,
				Decay:   m.Decay,
				Sustain: m.Sustain,
				Release: m.Release,
			},
		}
	}

	return m, cmd
}

// adjustEnvelopeValue adjusts the current envelope field by a delta value
func (m *EnvelopeModel) adjustEnvelopeValue(delta float64) {
	var currentValue *float64

	switch m.envelopeField {
	case EnvelopeAttack:
		currentValue = &m.Attack
	case EnvelopeDecay:
		currentValue = &m.Decay
	case EnvelopeSustain:
		currentValue = &m.Sustain
	case EnvelopeRelease:
		currentValue = &m.Release
	}

	if currentValue != nil {
		newValue := *currentValue + delta

		// For A, D, R: prevent increases that would make A+D+R exceed 100
		if m.envelopeField != EnvelopeSustain && delta > 0 {
			otherSum := m.Attack + m.Decay + m.Release - *currentValue
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
	if m.ShowModal {
		return m.PresetModel.View()
	}

	envView := strings.Builder{}
	envView.WriteString("ADSR Envelope:\n")

	envView.WriteString(RenderKnobSelected("Attack", m.Attack, m.envelopeField == EnvelopeAttack, m.selectedStyle) + "\n")
	envView.WriteString(RenderKnobSelected("Decay", m.Decay, m.envelopeField == EnvelopeDecay, m.selectedStyle) + "\n")
	envView.WriteString(RenderKnobSelected("Sustain", m.Sustain, m.envelopeField == EnvelopeSustain, m.selectedStyle) + "\n")
	envView.WriteString(RenderKnobSelected("Release", m.Release, m.envelopeField == EnvelopeRelease, m.selectedStyle))

	return envView.String()
}

func RenderKnobSelected(label string, value float64, selected bool, selectedStyle lipgloss.Style) string {
	knobChar := RenderKnob(label, value)

	if selected {
		return selectedStyle.Render(knobChar)
	} else {
		return knobChar
	}
}
