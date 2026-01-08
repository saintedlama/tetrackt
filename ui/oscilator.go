package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tetrackt/tetrackt/audio"
)

type OscilatorModel struct {
	Oscilator     audio.OscilatorType
	oscilatorIdx  int
	oscilatorList []audio.OscilatorType
	selectedStyle lipgloss.Style
}

func NewOscilatorModel(selectedStyle lipgloss.Style) OscilatorModel {
	return OscilatorModel{
		Oscilator:     audio.Sine,
		oscilatorIdx:  0,
		oscilatorList: []audio.OscilatorType{audio.Sine, audio.Square, audio.Triangle, audio.Sawtooth, audio.SawtoothReverse, audio.Noise},
		selectedStyle: selectedStyle,
	}
}

func (m OscilatorModel) View() string {
	var oscilatorView strings.Builder
	oscilatorView.WriteString("Oscilator:\n")

	for i, osc := range m.oscilatorList {
		if i == m.oscilatorIdx {
			oscilatorView.WriteString(m.selectedStyle.Render(fmt.Sprintf("%s", osc)))
			oscilatorView.WriteString("\n")
		} else {
			fmt.Fprintf(&oscilatorView, "%s\n", osc)
		}
	}

	return oscilatorView.String()
}

func (m OscilatorModel) Update(msg tea.KeyMsg) (OscilatorModel, tea.Cmd) {
	switch msg.String() {
	case "up", "left":
		// Move to previous oscilator
		m.oscilatorIdx = (m.oscilatorIdx - 1 + len(m.oscilatorList)) % len(m.oscilatorList)
		m.Oscilator = m.oscilatorList[m.oscilatorIdx]
	case "down", "right":
		// Move to next oscilator
		m.oscilatorIdx = (m.oscilatorIdx + 1) % len(m.oscilatorList)
		m.Oscilator = m.oscilatorList[m.oscilatorIdx]
	}

	return m, nil
}
