package ui

import (
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/ui/common"
)

// FileDialogMode represents whether the dialog is saving or loading.
type FileDialogMode int

const (
	ModeSave FileDialogMode = iota
	ModeLoad
)

// FileDialogConfirmed is the result payload when the user confirms the file dialog.
type FileDialogConfirmed struct {
	Filename string
	Mode     FileDialogMode
}

var (
	fdTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccentPrimary)
	fdHelpStyle  = lipgloss.NewStyle().Foreground(common.ColorTextMuted)
	fdErrStyle   = lipgloss.NewStyle().Foreground(common.ColorAccentWarning)
	fdLabelStyle = lipgloss.NewStyle().Foreground(common.ColorTextMuted)
)

// FileDialogModel is the file browser dialog for loading and saving modules.
// In Load mode the filepicker is used directly; in Save mode a filename
// textinput is shown below the filepicker and Tab switches focus between them.
type FileDialogModel struct {
	Mode      FileDialogMode
	ErrMsg    string
	fp        filepicker.Model
	input     textinput.Model // save mode only
	focusPane int             // 0=filepicker, 1=textinput; save mode only
}

// NewFileDialog creates a file dialog in the given mode with an optional prefill.
// prefill sets the initial filename value (save mode).
func NewFileDialog(mode FileDialogMode, prefill string) *FileDialogModel {
	fp := filepicker.New()
	fp.CurrentDirectory = "."
	fp.ShowPermissions = false
	fp.ShowSize = false
	fp.ShowHidden = false
	fp.SetHeight(10)

	// Remove esc from the Back binding — the dialog itself intercepts esc to close.
	fp.KeyMap.Back = key.NewBinding(
		key.WithKeys("h", "backspace", "left"),
		key.WithHelp("h/←", "back"),
	)

	// Make directories visually distinct.
	fp.Styles.Directory = lipgloss.NewStyle().
		Foreground(common.ColorAccentPrimary).
		Bold(true)

	if mode == ModeLoad {
		// Only show .json files when loading — the user must pick an existing file.
		fp.AllowedTypes = []string{".json"}
		fp.FileAllowed = true
		fp.DirAllowed = false
	} else {
		// Save mode: show all files for context; user types the filename below.
		// DirAllowed=false so directories are shown for navigation but not selectable.
		fp.AllowedTypes = nil
		fp.FileAllowed = true
		fp.DirAllowed = false
	}

	m := &FileDialogModel{
		Mode: mode,
		fp:   fp,
	}

	if mode == ModeSave {
		ti := textinput.New()
		ti.Placeholder = "module"
		ti.CharLimit = 200
		ti.SetWidth(40)
		if prefill != "" {
			ti.SetValue(prefill)
		}
		m.input = ti
	}

	return m
}

// InputValue returns the current filename input value (save mode).
func (m *FileDialogModel) InputValue() string {
	return m.input.Value()
}

// FocusInput switches focus to the filename text input (save mode only).
func (m *FileDialogModel) FocusInput() {
	if m.Mode == ModeSave {
		m.focusPane = 1
		m.input.Focus()
	}
}

// Init initialises the filepicker (loads the initial directory listing).
func (m *FileDialogModel) Init() tea.Cmd {
	return m.fp.Init()
}

func (m *FileDialogModel) Update(rawMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := rawMsg.(type) {
	case tea.WindowSizeMsg:
		fpHeight := msg.Height - 12
		if fpHeight < 4 {
			fpHeight = 4
		}
		if m.Mode == ModeSave {
			fpHeight -= 3 // leave room for the filename row
		}
		m.fp.SetHeight(fpHeight)
		inputWidth := msg.Width - 20
		if inputWidth < 10 {
			inputWidth = 10
		}
		m.input.SetWidth(inputWidth)
		return m, nil

	case tea.KeyPressMsg:
		m.ErrMsg = ""
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseDialogMsg{} }
		case "tab":
			if m.Mode == ModeSave {
				m.focusPane = (m.focusPane + 1) % 2
				if m.focusPane == 1 {
					m.input.Focus()
				} else {
					m.input.Blur()
				}
				return m, nil
			}
		case "enter":
			if m.Mode == ModeSave && m.focusPane == 1 {
				return m, m.confirmSave()
			}
			// else: fall through to filepicker (navigates dirs / selects file)
		}
	}

	var cmd tea.Cmd

	// Textinput focused (save mode) — forward all messages to it.
	if m.Mode == ModeSave && m.focusPane == 1 {
		m.input, cmd = m.input.Update(rawMsg)
		return m, cmd
	}

	// Filepicker focused (or load mode).
	m.fp, cmd = m.fp.Update(rawMsg)

	if didSelect, path := m.fp.DidSelectFile(rawMsg); didSelect {
		if m.Mode == ModeLoad {
			mode := m.Mode
			return m, func() tea.Msg {
				return CloseDialogMsg{Payload: FileDialogConfirmed{Filename: path, Mode: mode}}
			}
		}
		// Save mode: prefill textinput with the selected filename and focus it.
		m.input.SetValue(filepath.Base(path))
		m.focusPane = 1
		m.input.Focus()
		return m, cmd
	}

	if didDisabled, _ := m.fp.DidSelectDisabledFile(rawMsg); didDisabled {
		m.ErrMsg = "Only .json files are supported"
	}

	return m, cmd
}

func (m *FileDialogModel) confirmSave() tea.Cmd {
	filename := strings.TrimSpace(m.input.Value())
	if filename == "" {
		m.ErrMsg = "Filename cannot be empty"
		return nil
	}
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}
	fullPath := filepath.Join(m.fp.CurrentDirectory, filename)
	mode := m.Mode
	return func() tea.Msg {
		return CloseDialogMsg{Payload: FileDialogConfirmed{Filename: fullPath, Mode: mode}}
	}
}

// View renders the file browser dialog content (border is added by dialogModel).
func (m *FileDialogModel) View() tea.View {
	var sb strings.Builder

	// Title
	title := "Load Module"
	if m.Mode == ModeSave {
		title = "Save Module"
	}
	sb.WriteString(fdTitleStyle.Render(title))
	sb.WriteString("\n\n")

	// Filepicker
	sb.WriteString(m.fp.View())

	// Save mode: filename input row below the picker
	if m.Mode == ModeSave {
		sb.WriteString("\n")
		sb.WriteString(fdLabelStyle.Render("Filename: "))
		sb.WriteString(m.input.View())
		if m.ErrMsg != "" {
			sb.WriteString("  ")
			sb.WriteString(fdErrStyle.Render(m.ErrMsg))
		}
	} else if m.ErrMsg != "" {
		sb.WriteString("\n")
		sb.WriteString(fdErrStyle.Render(m.ErrMsg))
	}

	// Help footer
	sb.WriteString("\n\n")
	var help string
	switch {
	case m.Mode == ModeSave && m.focusPane == 1:
		help = "Enter: save  Tab: browse files  Esc: cancel"
	case m.Mode == ModeSave:
		help = "Tab: edit filename  ↑↓: navigate  Enter: open/select  Esc: cancel"
	default:
		help = "↑↓: navigate  Enter: load  Esc: cancel"
	}
	sb.WriteString(fdHelpStyle.Render(help))

	return tea.NewView(sb.String())
}
