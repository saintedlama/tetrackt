package ui

import "charm.land/lipgloss/v2"

var logoStyle = lipgloss.NewStyle().
	Background(ColorAccentEnvelope).
	Foreground(ColorGrayDarkest).
	Bold(true).
	Padding(0, 1)

// Logo returns the rendered colorized logo string.
func Logo() string {
	return logoStyle.Render(` /// TeTrackT \\\ `)
}
