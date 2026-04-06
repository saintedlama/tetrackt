package ui

import (
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/ui/common"
)

var logoStyle = lipgloss.NewStyle().
	Background(common.ColorAccentEnvelope).
	Foreground(common.ColorGrayDarkest).
	Bold(true).
	Padding(0, 1)

// Logo returns the rendered colorized logo string.
func Logo() string {
	return logoStyle.Render(` /// TeTrackT \\\ `)
}
