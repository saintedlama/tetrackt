package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/ui/common"
)

// QuitDiscardMsg is the result payload when the user chooses to quit without saving.
type QuitDiscardMsg struct{}

// QuitSaveMsg is the result payload when the user chooses to save before quitting.
type QuitSaveMsg struct{}

var (
	quitTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(common.ColorAccentWarning)

	quitKeyStyle  = lipgloss.NewStyle().Foreground(common.ColorAccentWarning).Bold(true)
	quitHelpStyle = lipgloss.NewStyle().Foreground(common.ColorTextMuted)
)

// QuitDialog is a simple confirmation dialog shown when quitting with unsaved changes.
// Keys: S = save & quit, Y/Enter/Ctrl+C = quit without saving, N/Esc = cancel.
type QuitDialog struct{}

func NewQuitDialog() *QuitDialog { return &QuitDialog{} }

func (d *QuitDialog) Init() tea.Cmd { return nil }

func (d *QuitDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "s":
			return d, func() tea.Msg { return CloseDialogMsg{Payload: QuitSaveMsg{}} }
		case "y", "enter", "ctrl+c":
			return d, func() tea.Msg { return CloseDialogMsg{Payload: QuitDiscardMsg{}} }
		case "n", "esc":
			return d, func() tea.Msg { return CloseDialogMsg{} }
		}
	}
	return d, nil
}

func (d *QuitDialog) View() tea.View {
	title := quitTitleStyle.Render("Unsaved changes")

	row := func(key, desc string) string {
		return quitKeyStyle.Render("["+key+"]") + " " + quitHelpStyle.Render(desc)
	}

	content := title + "\n\n" +
		row("S", "Save & quit") + "\n" +
		row("Y", "Quit without saving") + "\n" +
		row("N", "Cancel")

	return tea.NewView(content)
}
