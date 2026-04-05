package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// FileDialogMode represents whether the dialog is saving or loading.
type FileDialogMode int

const (
	ModeSave FileDialogMode = iota
	ModeLoad
)

// FileDialogModel represents the file dialog component state.
// It is always visible when instantiated; use NewDialogModel to overlay it.
type FileDialogModel struct {
	Mode           FileDialogMode
	Input          string
	Error          string
	cursorPosition int
}

// FileDialogConfirmed is the result payload when the user confirms the file dialog.
type FileDialogConfirmed struct {
	Filename string
	Mode     FileDialogMode
}

// NewFileDialog creates a file dialog in the given mode with an optional prefill.
func NewFileDialog(mode FileDialogMode, prefill string) *FileDialogModel {
	return &FileDialogModel{
		Mode:           mode,
		Input:          prefill,
		Error:          "",
		cursorPosition: len(prefill),
	}
}

// SetError sets an error message to display in the dialog.
func (m *FileDialogModel) SetError(err string) {
	m.Error = err
}

// Init initializes the file dialog (required by Bubble Tea).
func (m *FileDialogModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input for the file dialog.
func (m *FileDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			filename := m.Input
			if filename == "" {
				m.Error = "Filename cannot be empty"
				return m, nil
			}

			if !strings.HasSuffix(filename, ".yaml") {
				filename += ".yaml"
			}

			mode := m.Mode
			return m, func() tea.Msg {
				return CloseDialogMsg{Payload: FileDialogConfirmed{Filename: filename, Mode: mode}}
			}

		case "esc":
			return m, func() tea.Msg {
				return CloseDialogMsg{}
			}

		case "backspace":
			if len(m.Input) > 0 {
				m.Input = m.Input[:len(m.Input)-1]
				m.cursorPosition = len(m.Input)
			}
			return m, nil

		default:
			if len(msg.String()) == 1 && msg.String()[0] >= ' ' && msg.String()[0] <= '~' {
				m.Input += msg.String()
				m.cursorPosition = len(m.Input)
			}
			return m, nil
		}
	}

	return m, nil
}

// View renders the file dialog content (border is added by dialogModel).
func (m *FileDialogModel) View() tea.View {
	var dialogTitle string
	switch m.Mode {
	case ModeSave:
		dialogTitle = "Save Song"
	case ModeLoad:
		dialogTitle = "Load Song"
	default:
		dialogTitle = "File Dialog"
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("%s\n\n", dialogTitle))
	content.WriteString(fmt.Sprintf("Filename: %s_\n\n", m.Input))

	if m.Error != "" {
		content.WriteString(fmt.Sprintf("Error: %s\n\n", m.Error))
	}

	content.WriteString("[Enter to confirm, Esc to cancel]")

	return tea.NewView(content.String())
}
