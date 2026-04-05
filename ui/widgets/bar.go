package widgets

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	// Left (filled) side of the bar — orange accent, matching envelope colour.
	barLeftStyle = lipgloss.NewStyle().SetString("█").Foreground(lipgloss.Color("#ff7700"))
	// Right (empty) side of the bar — cyan accent, matching oscillator colour.
	barRightStyle = lipgloss.NewStyle().SetString("█").Foreground(lipgloss.Color("#00d4e8"))
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
			bar.WriteString(barLeftStyle.Render())
		} else {
			bar.WriteString(barRightStyle.Render())
		}
	}

	return bar.String()
}
