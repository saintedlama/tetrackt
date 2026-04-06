package common

import "charm.land/lipgloss/v2"

var (
	onOffOffStyle = lipgloss.NewStyle().Foreground(ColorGray) // disabled / off
	onOffOnStyle  = lipgloss.NewStyle().Foreground(ColorGreen) // active / on
)

// RenderOnOff renders a power symbol styled as on or off.
func RenderOnOff(on bool) string {
	if on {
		return onOffOnStyle.Render("⏻ ")
	}
	return onOffOffStyle.Render("⏼ ")
}
