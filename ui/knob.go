package ui

import (
	"fmt"
)

func RenderKnob(label string, value float64) string {
	knobChar := percentageToKnob(value)

	knobDisplay := fmt.Sprintf("%s: %s %3d%%", label, knobChar, int(value*100))

	return knobDisplay
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
