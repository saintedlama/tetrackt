package synth

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
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

func (m *EnvelopeModel) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
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

// adjustEnvelopeValue adjusts the current envelope field by a delta value.
// For Attack, Decay, Release: delta is interpreted as seconds (0.01 = 10ms, 0.10 = 100ms).
// For Sustain: delta is a fraction of the 0–1 level range.
func (m *EnvelopeModel) adjustEnvelopeValue(delta float64) {
	switch m.envelopeField {
	case EnvelopeAttack:
		d := m.Envelope.Attack + time.Duration(delta*float64(time.Second))
		if d < 0 {
			d = 0
		}
		m.Envelope.Attack = d
	case EnvelopeDecay:
		d := m.Envelope.Decay + time.Duration(delta*float64(time.Second))
		if d < 0 {
			d = 0
		}
		m.Envelope.Decay = d
	case EnvelopeSustain:
		newValue := m.Envelope.Sustain + delta
		if newValue < 0 {
			newValue = 0
		} else if newValue > 1.0 {
			newValue = 1.0
		}
		m.Envelope.Sustain = newValue
	case EnvelopeRelease:
		d := m.Envelope.Release + time.Duration(delta*float64(time.Second))
		if d < 0 {
			d = 0
		}
		m.Envelope.Release = d
	}
}

func (m *EnvelopeModel) View() string {
	envView := strings.Builder{}

	envView.WriteString(common.RenderKnobDurationSelected("Attack", m.Envelope.Attack, m.envelopeField == EnvelopeAttack) + "\n")
	envView.WriteString(common.RenderKnobDurationSelected("Decay", m.Envelope.Decay, m.envelopeField == EnvelopeDecay) + "\n")
	envView.WriteString(common.RenderKnobSelected("Sustain", m.Envelope.Sustain, m.envelopeField == EnvelopeSustain) + "\n")
	envView.WriteString(common.RenderKnobDurationSelected("Release", m.Envelope.Release, m.envelopeField == EnvelopeRelease))

	return envView.String()
}
