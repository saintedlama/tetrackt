package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/ui/common"
)

// RenderPanel renders content inside a rounded border with an optional title
// embedded in the top border line using the lipgloss v2 compositor.
// titleColor is the foreground color for the title text.
// active controls whether the active or inactive border color is used.
func RenderPanel(title string, titleColor color.Color, content string, active bool) string {
	borderColor := common.ColorBorder
	if active {
		borderColor = common.ColorBorderActive
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2, 1, 2)

	bordered := borderStyle.Render(content)

	if title == "" {
		return bordered
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Background(common.ColorSurface)
	titleRendered := titleStyle.Render(" " + title + " ")

	w := lipgloss.Width(bordered)
	h := lipgloss.Height(bordered)

	bgLayer := lipgloss.NewLayer(bordered).Z(0)
	titleLayer := lipgloss.NewLayer(titleRendered).X(2).Y(0).Z(1)

	return lipgloss.NewCanvas(w, h).
		Compose(lipgloss.NewCompositor(bgLayer, titleLayer)).
		Render()
}
