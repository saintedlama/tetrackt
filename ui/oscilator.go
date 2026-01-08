package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tetrackt/tetrackt/audio"
)

type OscillatorModel struct {
	Oscillator     audio.OscillatorType
	oscillatorIdx  int
	oscillatorList []audio.OscillatorType
	selectedStyle  lipgloss.Style
}

func NewOscillatorModel(selectedStyle lipgloss.Style, oscillator audio.OscillatorType) OscillatorModel {
	oscillatorList := []audio.OscillatorType{audio.Sine, audio.Square, audio.Triangle, audio.Sawtooth, audio.SawtoothReverse, audio.Noise}
	oscillatorIdx := 0

	for i, osc := range oscillatorList {
		if osc == oscillator {
			oscillatorIdx = i
			break
		}
	}

	return OscillatorModel{
		Oscillator:     oscillator,
		oscillatorIdx:  oscillatorIdx,
		oscillatorList: oscillatorList,
		selectedStyle:  selectedStyle,
	}
}

func (m OscillatorModel) View() string {
	var oscillatorView strings.Builder
	oscillatorView.WriteString("Oscillator:\n")

	for i, osc := range m.oscillatorList {
		if i == m.oscillatorIdx {
			oscillatorView.WriteString(m.selectedStyle.Render(fmt.Sprintf("%s", osc)))
			oscillatorView.WriteString("\n")
		} else {
			fmt.Fprintf(&oscillatorView, "%s\n", osc)
		}
	}

	return oscillatorView.String()
}

func (m OscillatorModel) Update(msg tea.KeyMsg) OscillatorModel {
	switch msg.String() {
	case "up", "left":
		// Move to previous oscillator
		m.oscillatorIdx = (m.oscillatorIdx - 1 + len(m.oscillatorList)) % len(m.oscillatorList)
		m.Oscillator = m.oscillatorList[m.oscillatorIdx]
	case "down", "right":
		// Move to next oscillator
		m.oscillatorIdx = (m.oscillatorIdx + 1) % len(m.oscillatorList)
		m.Oscillator = m.oscillatorList[m.oscillatorIdx]
	}

	return m
}
