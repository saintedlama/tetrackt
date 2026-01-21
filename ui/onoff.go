package ui

import "github.com/charmbracelet/lipgloss"

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
