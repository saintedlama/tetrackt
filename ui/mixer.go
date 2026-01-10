package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Mixer struct {
	MixBalance PercentageKnob
}

func NewMixer() Mixer {
	return Mixer{
		MixBalance: NewPercentageKnob("Balance", 0.5, true, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))),
	}
}

func (m *Mixer) View() string {
	envView := strings.Builder{}
	envView.WriteString("Mixer:\n")

	envView.WriteString(m.MixBalance.View() + "\n")

	return envView.String()
}

func (m *Mixer) Update(msg tea.KeyMsg) *Mixer {
	switch msg.String() {
	case "left":
		if m.MixBalance.Value > 0.0 {
			m.MixBalance.Value -= 0.01
		}
	case "shift+left":
		if m.MixBalance.Value > 0.1 {
			m.MixBalance.Value -= 0.1
		}
	case "right":
		if m.MixBalance.Value < 1.0 {
			m.MixBalance.Value += 0.01
		}
	case "shift+right":
		if m.MixBalance.Value < 0.9 {
			m.MixBalance.Value += 0.1
		}
	}
	return m
}
