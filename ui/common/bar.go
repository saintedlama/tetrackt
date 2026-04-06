package common

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	// barFilledStyle matches ColorAccentEnvelope (#ff7700) from styles.go.
	barFilledStyle = lipgloss.NewStyle().SetString("▪").Foreground(ColorAccentEnvelope)
	// barEmptyStyle matches ColorGrayMedium (#3c3c3c) from styles.go.
	barEmptyStyle = lipgloss.NewStyle().SetString("▫").Foreground(ColorGrayMedium)
)

// Bar is a horizontal fill indicator that visualises a value within a range.
type Bar struct {
	minValue float64
	maxValue float64
	Value    float64
	width    int
}

// NewBar creates a Bar with the given range, initial value, and display width.
func NewBar(minValue, maxValue, value float64, width int) Bar {
	return Bar{
		minValue: minValue,
		maxValue: maxValue,
		Value:    value,
		width:    width,
	}
}

// View renders the bar as a coloured string of block characters.
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
			bar.WriteString(barFilledStyle.Render())
		} else {
			bar.WriteString(barEmptyStyle.Render())
		}
	}

	return bar.String()
}
