package ui

import "charm.land/lipgloss/v2"

var (
	offStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	onStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

func RenderOnOff(on bool) string {
	if on {
		return onStyle.Render("⏻ ")
	}

	return offStyle.Render("⏼ ")
}
