package ui

import tea "charm.land/bubbletea/v2"

// Screen represents a full-screen view in the application.
// Each screen owns its own navigation state and renders to the full terminal area
// below the persistent header.
type Screen interface {
	Update(tea.Msg) (Screen, tea.Cmd)
	View() string
	Footer() string
	Title() string
}
