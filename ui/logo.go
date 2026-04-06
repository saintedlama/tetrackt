package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/ui/common"
)

var arrowStyle = lipgloss.NewStyle().
	Background(common.ColorGrayDarkest).
	Foreground(common.ColorAccentEnvelope).
	Bold(true)

var logoStyle = lipgloss.NewStyle().
	Background(common.ColorAccentEnvelope).
	Foreground(common.ColorGrayDarkest).
	Bold(true).
	Padding(0, 1)

// Logo returns the rendered colorized logo string.
func Logo() string {
	var sb strings.Builder
	sb.WriteString(arrowStyle.Render(" ««« "))
	sb.WriteString(logoStyle.Render("TeTrackT"))
	sb.WriteString(arrowStyle.Render(" »»» "))

	return sb.String()
}
