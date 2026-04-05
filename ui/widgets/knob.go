package widgets

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// RenderKnob renders a knob display showing the label, a character indicating
// the fill level, and the percentage value.
func RenderKnob(label string, value float64) string {
	return fmt.Sprintf("%s: %s %3d%%", label, percentageToKnob(value), int(value*100))
}

// RenderKnobSelected renders a knob, applying selectedStyle when selected is true.
func RenderKnobSelected(label string, value float64, selected bool, selectedStyle lipgloss.Style) string {
	knob := RenderKnob(label, value)
	if selected {
		return selectedStyle.Render(knob)
	}
	return knob
}

// percentageToKnob maps a 0–1 value to a fill character.
func percentageToKnob(percentage float64) string {
	switch {
	case percentage <= 0.25:
		return "◔"
	case percentage <= 0.50:
		return "◗"
	case percentage <= 0.75:
		return "◕"
	default:
		return "●"
	}
}
