package ui

import "charm.land/lipgloss/v2"

var (
	offStyle = lipgloss.NewStyle().Foreground(ColorTextDisabled)
	onStyle  = lipgloss.NewStyle().Foreground(ColorAccentPlay)
)

func RenderOnOff(on bool) string {
	if on {
		return onStyle.Render("⏻ ")
	}

	return offStyle.Render("⏼ ")
}
