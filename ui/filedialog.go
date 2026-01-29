package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// FileDialogMode represents the current state of the file dialog
type FileDialogMode int

const (
	ModeHidden FileDialogMode = iota
	ModeSave
	ModeLoad
)

// FileDialogModel represents the file dialog component state
type FileDialogModel struct {
	Mode           FileDialogMode
	Input          string
	Error          string
	PrefillPath    string
	cursorPosition int
	borderStyle    lipgloss.Style
}

// FileDialogConfirmed is sent when the user confirms the file dialog
type FileDialogConfirmed struct {
	Filename string
}

// FileDialogCancelled is sent when the user cancels the file dialog
type FileDialogCancelled struct{}

// NewFileDialog creates a new file dialog model
func NewFileDialog(borderStyle lipgloss.Style) *FileDialogModel {
	return &FileDialogModel{
		Mode:        ModeHidden,
		Input:       "",
		Error:       "",
		PrefillPath: "",
		borderStyle: borderStyle,
	}
}

// Show displays the file dialog in the specified mode
func (m *FileDialogModel) Show(mode FileDialogMode, prefillPath string) {
	m.Mode = mode
	m.Input = prefillPath
	m.Error = ""
	m.cursorPosition = len(prefillPath)
}

// Hide closes the file dialog
func (m *FileDialogModel) Hide() {
	m.Mode = ModeHidden
	m.Input = ""
	m.Error = ""
	m.cursorPosition = 0
}

// SetError sets an error message to display in the dialog
func (m *FileDialogModel) SetError(err string) {
	m.Error = err
}

// IsVisible returns true if the dialog is currently visible
func (m *FileDialogModel) IsVisible() bool {
	return m.Mode != ModeHidden
}

// Init initializes the file dialog (required by Bubble Tea)
func (m FileDialogModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input for the file dialog
func (m FileDialogModel) Update(msg tea.Msg) (FileDialogModel, tea.Cmd) {
	// Only process messages if dialog is visible
	if !m.IsVisible() {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Confirm and send filename
			filename := m.Input
			if filename == "" {
				m.Error = "Filename cannot be empty"
				return m, nil
			}

			// Auto-append .yaml extension if not present
			if !strings.HasSuffix(filename, ".yaml") {
				filename += ".yaml"
			}

			// Clear dialog state and return confirmation message
			return m, func() tea.Msg {
				return FileDialogConfirmed{Filename: filename}
			}

		case "esc":
			// Cancel and return cancellation message
			return m, func() tea.Msg {
				return FileDialogCancelled{}
			}

		case "backspace":
			// Remove last character
			if len(m.Input) > 0 {
				m.Input = m.Input[:len(m.Input)-1]
				m.cursorPosition = len(m.Input)
			}
			return m, nil

		default:
			// Type into the input field (only printable ASCII characters)
			if len(msg.String()) == 1 && msg.String()[0] >= ' ' && msg.String()[0] <= '~' {
				m.Input += msg.String()
				m.cursorPosition = len(m.Input)
			}
			return m, nil
		}
	}

	return m, nil
}

// View renders the file dialog as a modal overlay
func (m FileDialogModel) View() tea.View {
	if !m.IsVisible() {
		return tea.NewView("")
	}

	var dialogTitle string
	switch m.Mode {
	case ModeSave:
		dialogTitle = "Save Song"
	case ModeLoad:
		dialogTitle = "Load Song"
	default:
		dialogTitle = "File Dialog"
	}

	// Build dialog content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("%s\n\n", dialogTitle))
	content.WriteString(fmt.Sprintf("Filename: %s_\n\n", m.Input))

	// Show error if present
	if m.Error != "" {
		content.WriteString(fmt.Sprintf("Error: %s\n\n", m.Error))
	}

	content.WriteString("[Enter to confirm, Esc to cancel]")

	// Render with border
	return tea.NewView(m.borderStyle.Render(content.String()))
}
