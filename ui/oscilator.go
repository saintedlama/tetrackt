package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tetrackt/tetrackt/audio"
)

type OscillatorModel struct {
	Oscillator     audio.Oscillator
	oscillatorList []audio.OscillatorType
	selectedStyle  lipgloss.Style
}

type OscillatorUpdated struct {
	Oscillator audio.Oscillator
}

func NewOscillatorModel(selectedStyle lipgloss.Style, oscillator audio.OscillatorType) *OscillatorModel {
	oscillatorList := []audio.OscillatorType{audio.Sine, audio.Square, audio.Triangle, audio.Sawtooth, audio.SawtoothReverse, audio.Noise}

	return &OscillatorModel{
		Oscillator:     audio.Oscillator{Type: oscillator},
		oscillatorList: oscillatorList,
		selectedStyle:  selectedStyle,
	}
}

func (m *OscillatorModel) Init() tea.Cmd {
	return nil
}

func (m *OscillatorModel) View() string {
	var oscillatorView strings.Builder
	oscillatorView.WriteString("Oscillator:\n")

	for i, osc := range m.oscillatorList {
		if osc == m.Oscillator.Type {
			oscillatorView.WriteString(m.selectedStyle.Render(fmt.Sprintf("%s", osc)))
		} else {
			fmt.Fprint(&oscillatorView, osc)
		}

		if i < len(m.oscillatorList)-1 {
			oscillatorView.WriteString("\n")
		}
	}

	return oscillatorView.String()
}

func (m *OscillatorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "left":
			// Move to previous oscillator
			idx := slices.Index(m.oscillatorList, m.Oscillator.Type)
			idx = (idx - 1 + len(m.oscillatorList)) % len(m.oscillatorList)
			m.Oscillator.Type = m.oscillatorList[idx]

			cmd = func() tea.Msg {
				return OscillatorUpdated{
					Oscillator: m.Oscillator,
				}
			}
		case "down", "right":
			// Move to next oscillator
			idx := slices.Index(m.oscillatorList, m.Oscillator.Type)
			idx = (idx + 1) % len(m.oscillatorList)
			m.Oscillator.Type = m.oscillatorList[idx]

			cmd = func() tea.Msg {
				return OscillatorUpdated{
					Oscillator: m.Oscillator,
				}
			}
		}
	}

	return m, cmd
}
