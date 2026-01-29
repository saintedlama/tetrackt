package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	leftStyle  = lipgloss.NewStyle().SetString("█").Foreground(lipgloss.Color("#ff5722"))
	rightStyle = lipgloss.NewStyle().SetString("█").Foreground(lipgloss.Color("#2196f3"))
)

type Bar struct {
	minValue float64
	maxValue float64
	Value    float64
	width    int
}

func NewBar(minValue, maxValue, value float64, width int) Bar {
	return Bar{
		minValue: minValue,
		maxValue: maxValue,
		Value:    value,
		width:    width,
	}
}

func (b Bar) View() string {
	filledWidth := int((b.Value - b.minValue) / (b.maxValue - b.minValue) * float64(b.width))
	if filledWidth < 0 {
		filledWidth = 0
	} else if filledWidth > b.width {
		filledWidth = b.width
	}

	var bar strings.Builder
	for i := 0; i < b.width; i++ {
		if i < filledWidth {
			bar.WriteString(leftStyle.Render())
		} else {
			bar.WriteString(rightStyle.Render())
		}
	}

	return bar.String()
}
