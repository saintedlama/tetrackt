package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

type PercentageKnob struct {
	Label         string
	Value         float64
	Selected      bool
	Style         lipgloss.Style
	SelectedStyle lipgloss.Style
}

func NewPercentageKnob(label string, value float64, selected bool, style lipgloss.Style, selectedStyle lipgloss.Style) PercentageKnob {
	return PercentageKnob{
		Label:         label,
		Value:         value,
		Selected:      selected,
		Style:         style,
		SelectedStyle: selectedStyle,
	}
}

func (k PercentageKnob) View() string {
	knobChar := percentageToKnob(float64(k.Value))

	knobDisplay := fmt.Sprintf("%s: %s %3d%%", k.Label, knobChar, int(k.Value*100))

	if k.Selected {
		return k.SelectedStyle.Render(knobDisplay)
	}
	return k.Style.Render(knobDisplay)
}

// percentageToKnob converts a percentage value to a knob character
func percentageToKnob(percentage float64) string {
	if percentage <= 0.25 {
		return "◔" // Quarter filled
	} else if percentage <= 0.50 {
		return "◗" // Half filled
	} else if percentage <= 0.75 {
		return "◕" // Three-quarter filled
	} else {
		return "●" // Fully filled
	}
}
