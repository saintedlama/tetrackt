package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle   = lipgloss.NewStyle().Background(lipgloss.Color("198"))
	chromeStyle = baseStyle.Foreground(lipgloss.Color("230"))
	textStyle   = baseStyle.Foreground(lipgloss.Color("18"))
	introChar   = ">"
	outroChar   = "<"
)

// Logo renders the TeTrackT logo in a compact 3x20 layout.
type Logo struct {
	lines  []string
	colors []lipgloss.Color
}

// NewLogo returns the neon-wave logo variant (3 lines, 20 columns).
func NewLogo() *Logo {
	return &Logo{}
}

// View renders the logo with per-line gradient segments.
func (l *Logo) View() string {
	var builder strings.Builder

	builder.WriteString(chromeStyle.Render(strings.Repeat(introChar, 3)))
	builder.WriteString(textStyle.Render(" Te "))
	builder.WriteString(chromeStyle.Render(strings.Repeat(outroChar, 6)))
	builder.WriteString("\n")

	builder.WriteString(chromeStyle.Render(strings.Repeat(introChar, 3)))
	builder.WriteString(textStyle.Render(" Track "))
	builder.WriteString(chromeStyle.Render(strings.Repeat(outroChar, 3)))
	builder.WriteString("\n")

	builder.WriteString(chromeStyle.Render(strings.Repeat(introChar, 3)))
	builder.WriteString(textStyle.Render(" T "))
	builder.WriteString(chromeStyle.Render(strings.Repeat(outroChar, 7)))

	return baseStyle.Padding(1).Render(baseStyle.Render(builder.String()))
}
