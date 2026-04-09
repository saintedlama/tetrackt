package common

import (
	"fmt"
	"time"
)

// RenderKnob renders a knob display showing the label, a character indicating
// the fill level, and the percentage value.
func RenderKnob(label string, value float64) string {
	return fmt.Sprintf("%s: %s %3d%%", label, percentageToKnob(value), int(value*100))
}

// RenderKnobSelected renders a knob, applying StyleSelected when selected is true.
func RenderKnobSelected(label string, value float64, selected bool) string {
	knob := RenderKnob(label, value)
	if selected {
		return StyleSelected.Render(knob)
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

// RenderKnobDuration renders a knob display for a time.Duration, showing
// the label, a fill indicator (scaled to a 2000 ms maximum), and the value in ms.
func RenderKnobDuration(label string, d time.Duration) string {
	ms := d.Milliseconds()
	normalised := float64(ms) / 2000.0
	if normalised > 1.0 {
		normalised = 1.0
	}
	return fmt.Sprintf("%s: %s %4dms", label, percentageToKnob(normalised), ms)
}

// RenderKnobDurationSelected renders a duration knob, applying StyleSelected when selected is true.
func RenderKnobDurationSelected(label string, d time.Duration, selected bool) string {
	knob := RenderKnobDuration(label, d)
	if selected {
		return StyleSelected.Render(knob)
	}
	return knob
}
