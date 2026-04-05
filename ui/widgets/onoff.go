package widgets

import "charm.land/lipgloss/v2"

var (
	onOffOffStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#5a5a5a")) // disabled / off
	onOffOnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00c853")) // active / on
)

// RenderOnOff renders a power symbol styled as on or off.
func RenderOnOff(on bool) string {
	if on {
		return onOffOnStyle.Render("⏻ ")
	}
	return onOffOffStyle.Render("⏼ ")
}
