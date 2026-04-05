package ui

import tea "charm.land/bubbletea/v2"

// Component is implemented by UI sub-components that compose as strings.
// Unlike tea.Model, View returns a plain string rather than a tea.View,
// making it suitable for lipgloss-based string composition.
type Component interface {
	Init() tea.Cmd
	Update(tea.Msg) (Component, tea.Cmd)
	View() string
}
