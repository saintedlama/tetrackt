package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var dialogBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#ff9800")).
	Padding(0, 2)

// CloseDialogMsg signals that the dialog should be closed.
// If Payload is non-nil it is forwarded to the background model as a result.
// A nil Payload means the dialog was cancelled.
type CloseDialogMsg struct{ Payload tea.Msg }

// dialogModel wraps a dialog content model on top of a background model.
// All messages are forwarded to the dialog; CloseDialogMsg dismisses the dialog
// and optionally forwards the result payload to the background model.
type dialogModel struct {
	dialog tea.Model
	main   tea.Model
	width  int
	height int
}

// NewDialogModel creates a dialogModel that renders dialog centered over main.
func NewDialogModel(dialog, main tea.Model, width, height int) dialogModel {
	return dialogModel{
		dialog: dialog,
		main:   main,
		width:  width,
		height: height,
	}
}

func (m dialogModel) Init() tea.Cmd {
	return m.dialog.Init()
}

func (m dialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CloseDialogMsg:
		if msg.Payload != nil {
			var cmd tea.Cmd
			m.main, cmd = m.main.Update(msg.Payload)
			return m.main, cmd
		}
		return m.main, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.main, _ = m.main.Update(msg)
		return m, nil
	}

	var cmd tea.Cmd
	m.dialog, cmd = m.dialog.Update(msg)
	return m, cmd
}

func (m dialogModel) View() tea.View {
	bg := m.main.View().Content
	dialogContent := dialogBorderStyle.Render(m.dialog.View().Content)

	v := tea.NewView(OverlayCenter(m.width, m.height, dialogContent, bg, WithDim()))
	v.AltScreen = true
	return v
}
