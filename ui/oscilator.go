package ui

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
)

type editField int

const (
	oscillatorType editField = iota
	oscillatorPhase
)

type OscillatorModel struct {
	Oscillator          audio.Oscillator
	oscillatorList      []audio.OscillatorType
	selectedStyle       lipgloss.Style
	oscillatorTypeStyle lipgloss.Style
	editField           editField // 0 = type, 1 = phase
}

type OscillatorUpdated struct {
	Oscillator audio.Oscillator
}

func NewOscillatorModel(selectedStyle lipgloss.Style, oscillator audio.Oscillator) *OscillatorModel {
	oscillatorList := []audio.OscillatorType{audio.Sine, audio.Square, audio.Triangle, audio.Sawtooth, audio.SawtoothReverse, audio.Noise, audio.Silent}

	oscillatorTypeStyle := lipgloss.NewStyle().Width(calcOscWidth(oscillatorList))

	return &OscillatorModel{
		Oscillator:          oscillator,
		oscillatorList:      oscillatorList,
		selectedStyle:       selectedStyle,
		oscillatorTypeStyle: oscillatorTypeStyle,
	}
}

func (m *OscillatorModel) Init() tea.Cmd {
	return nil
}

func (m *OscillatorModel) View() tea.View {
	var oscillatorView strings.Builder
	oscillatorView.WriteString("Oscillator ")

	oscillatorView.WriteString(RenderOnOff(m.Oscillator.Type != audio.Silent))

	oscillatorView.WriteString("\n")
	oscType := renderFieldSelected(string(m.Oscillator.Type), m.editField == oscillatorType, m.selectedStyle)
	oscillatorView.WriteString(m.oscillatorTypeStyle.Render(oscType))

	oscillatorView.WriteString("\n")
	oscillatorView.WriteString(renderFieldSelected(RenderKnob("Phase", m.Oscillator.Phase), m.editField == oscillatorPhase, m.selectedStyle))

	return tea.NewView(oscillatorView.String())
}

func (m *OscillatorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			// Move to previous oscillator field
			m.editField = (m.editField - 1 + 2) % 2
		case "down":
			// Move to next oscillator field
			m.editField = (m.editField + 1) % 2
		case "left":
			switch m.editField {
			case oscillatorType:
				m.Oscillator.Type = cycle(m.oscillatorList, m.Oscillator.Type, -1)
			case oscillatorPhase:
				// decrease phase
				m.Oscillator.Phase -= 0.05
				if m.Oscillator.Phase < 0.0 {
					m.Oscillator.Phase = 0.0
				}
			}
			cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
		case "right":
			switch m.editField {
			case oscillatorType:
				m.Oscillator.Type = cycle(m.oscillatorList, m.Oscillator.Type, 1)
			case oscillatorPhase:
				// decrease phase
				m.Oscillator.Phase += 0.05
				if m.Oscillator.Phase > 1.0 {
					m.Oscillator.Phase = 1.0
				}
			}
			cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
		}
	}

	return m, cmd
}

func renderFieldSelected(content string, selected bool, style lipgloss.Style) string {
	if selected {
		return style.Render(content)
	}

	return content
}

func cycle(oscillatorList []audio.OscillatorType, current audio.OscillatorType, step int) audio.OscillatorType {
	nextIdx := slices.Index(oscillatorList, current) + step

	if nextIdx < 0 {
		return oscillatorList[len(oscillatorList)-1]
	}

	return oscillatorList[nextIdx%len(oscillatorList)]
}

func calcOscWidth(oscillatorList []audio.OscillatorType) int {
	oscWidth := 0
	for _, osc := range oscillatorList {
		oscWidth = max(oscWidth, len(osc))
	}
	return oscWidth
}
