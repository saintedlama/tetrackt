package ui

import (
	"path/filepath"
	"strconv"
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

// EmbeddedQuickstartToken is a virtual filename sent when the user selects
// the built-in quickstart entry in the load dialog.
const EmbeddedQuickstartToken = "__embedded_quickstart__"

// EmbeddedDemoToken is a virtual filename sent when the user selects
// the built-in demo entry in the load dialog.
const EmbeddedDemoToken = "__embedded_demo__"

// FileDialogConfirmed is the result payload when the user confirms the file dialog.
type FileDialogConfirmed struct {
	Filename      string
	Mode          FileDialogMode
	WavSampleRate int
	WavLoopCount  int
	WavExportGain float64
}

var (
	fdTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccentPrimary)
	fdHelpStyle  = lipgloss.NewStyle().Foreground(common.ColorTextMuted)
	fdErrStyle   = lipgloss.NewStyle().Foreground(common.ColorAccentWarning)
	fdLabelStyle = lipgloss.NewStyle().Foreground(common.ColorTextMuted)
)

const (
	loadPaneFiles = iota
	loadPaneBuiltIn
)

// FileDialogModel is the file browser dialog for loading and saving modules.
// In Load mode the filepicker is used directly; in Save mode a filename
// textinput is shown below the filepicker and Tab switches focus between them.
type FileDialogModel struct {
	Mode             FileDialogMode
	ErrMsg           string
	fp               filepicker.Model
	input            textinput.Model // save mode only
	wavSampleRate    textinput.Model // save mode only
	wavLoopCount     textinput.Model // save mode only
	wavExportGain    textinput.Model // save mode only
	focusPane        int             // save mode only: 0=filepicker, 1=filename, 2=sample rate, 3=loops, 4=gain
	loadPane         int             // load mode only: built-in tab or files tab
	loadBuiltinFocus int             // load mode only: 0=Quickstart, 1=Demo
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
		Foreground(common.ColorAccentModulation).
		Bold(true)

	if mode == ModeLoad {
		// Only show .json files when loading.
		fp.AllowedTypes = []string{".json"}
		fp.FileAllowed = true
		// Keep directory entries selectable so users can browse to module folders.
		fp.DirAllowed = true
	} else {
		// Save mode: show all files for context; user types the filename below.
		// DirAllowed=false so directories are shown for navigation but not selectable.
		fp.AllowedTypes = nil
		fp.FileAllowed = true
		fp.DirAllowed = false
	}

	m := &FileDialogModel{
		Mode:             mode,
		fp:               fp,
		loadPane:         loadPaneFiles,
		loadBuiltinFocus: 0,
	}
	if mode == ModeLoad {
		m.loadPane = loadPaneFiles
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

		sr := textinput.New()
		sr.CharLimit = 5
		sr.SetValue("44100")
		m.wavSampleRate = sr

		loops := textinput.New()
		loops.CharLimit = 2
		loops.SetValue("1")
		m.wavLoopCount = loops

		gain := textinput.New()
		gain.CharLimit = 4
		gain.SetValue("1.0")
		m.wavExportGain = gain
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
		m.wavSampleRate.SetWidth(8)
		m.wavLoopCount.SetWidth(4)
		m.wavExportGain.SetWidth(6)
		return m, nil

	case tea.KeyPressMsg:
		m.ErrMsg = ""

		if m.Mode == ModeLoad {
			switch msg.String() {
			case "esc":
				return m, func() tea.Msg { return CloseDialogMsg{} }
			case "tab", "shift+tab":
				if m.loadPane == loadPaneBuiltIn {
					m.loadPane = loadPaneFiles
				} else {
					m.loadPane = loadPaneBuiltIn
				}
				return m, nil
			case "enter":
				if m.loadPane == loadPaneBuiltIn {
					mode := m.Mode
					token := EmbeddedQuickstartToken
					if m.loadBuiltinFocus == 1 {
						token = EmbeddedDemoToken
					}
					return m, func() tea.Msg {
						return CloseDialogMsg{Payload: FileDialogConfirmed{Filename: token, Mode: mode}}
					}
				}
			case "up":
				if m.loadPane == loadPaneBuiltIn && m.loadBuiltinFocus > 0 {
					m.loadBuiltinFocus--
					return m, nil
				}
			case "down":
				if m.loadPane == loadPaneBuiltIn && m.loadBuiltinFocus < 1 {
					m.loadBuiltinFocus++
					return m, nil
				}
			}

			if m.loadPane == loadPaneBuiltIn {
				return m, nil
			}
		}

		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseDialogMsg{} }
		case "tab":
			if m.Mode == ModeSave {
				m.focusSavePane((m.focusPane + 1) % m.saveFocusCount())
				return m, nil
			}
		case "shift+tab":
			if m.Mode == ModeSave {
				nextPane := m.focusPane - 1
				if nextPane < 0 {
					nextPane = m.saveFocusCount() - 1
				}
				m.focusSavePane(nextPane)
				return m, nil
			}
		case "enter":
			if m.Mode == ModeLoad && m.loadBuiltinFocus == 0 {
				mode := m.Mode
				return m, func() tea.Msg {
					return CloseDialogMsg{Payload: FileDialogConfirmed{Filename: EmbeddedQuickstartToken, Mode: mode}}
				}
			}
			if m.Mode == ModeLoad && m.loadBuiltinFocus == 1 {
				mode := m.Mode
				return m, func() tea.Msg {
					return CloseDialogMsg{Payload: FileDialogConfirmed{Filename: EmbeddedDemoToken, Mode: mode}}
				}
			}
			if m.Mode == ModeSave && m.focusPane > 0 {
				return m, m.confirmSave()
			}
			// else: fall through to filepicker (navigates dirs / selects file)
		}
	}

	var cmd tea.Cmd

	// Textinput focused (save mode) — forward all messages to it.
	if m.Mode == ModeSave && m.focusPane > 0 {
		switch m.focusPane {
		case 1:
			m.input, cmd = m.input.Update(rawMsg)
		case 2:
			m.wavSampleRate, cmd = m.wavSampleRate.Update(rawMsg)
		case 3:
			m.wavLoopCount, cmd = m.wavLoopCount.Update(rawMsg)
		case 4:
			m.wavExportGain, cmd = m.wavExportGain.Update(rawMsg)
		}
		return m, cmd
	}

	m.fp, cmd = m.fp.Update(rawMsg)

	if didSelect, path := m.fp.DidSelectFile(rawMsg); didSelect {
		if m.Mode == ModeLoad {
			if !strings.HasSuffix(strings.ToLower(path), ".json") {
				m.ErrMsg = "Only .json files are supported"
				return m, cmd
			}
			mode := m.Mode
			return m, func() tea.Msg {
				return CloseDialogMsg{Payload: FileDialogConfirmed{Filename: path, Mode: mode}}
			}
		}
		// Save mode: prefill textinput with the selected filename and focus it.
		m.input.SetValue(filepath.Base(path))
		m.focusSavePane(1)
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
	lowerFilename := strings.ToLower(filename)
	if !strings.HasSuffix(lowerFilename, ".json") && !strings.HasSuffix(lowerFilename, ".wav") {
		filename += ".json"
		lowerFilename = strings.ToLower(filename)
	}

	confirmed := FileDialogConfirmed{Filename: filepath.Join(m.fp.CurrentDirectory, filename), Mode: m.Mode}
	if strings.HasSuffix(lowerFilename, ".wav") {
		sampleRate, loopCount, exportGain, errMsg := m.wavOptions()
		if errMsg != "" {
			m.ErrMsg = errMsg
			return nil
		}
		confirmed.WavSampleRate = sampleRate
		confirmed.WavLoopCount = loopCount
		confirmed.WavExportGain = exportGain
	}

	return func() tea.Msg {
		return CloseDialogMsg{Payload: confirmed}
	}
}

func (m *FileDialogModel) isWAVTarget() bool {
	if m.Mode != ModeSave {
		return false
	}
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(m.input.Value())), ".wav")
}

func (m *FileDialogModel) saveFocusCount() int {
	if m.isWAVTarget() {
		return 5
	}
	return 2
}

func (m *FileDialogModel) focusSavePane(pane int) {
	m.focusPane = pane
	m.input.Blur()
	m.wavSampleRate.Blur()
	m.wavLoopCount.Blur()
	m.wavExportGain.Blur()
	switch pane {
	case 1:
		m.input.Focus()
	case 2:
		m.wavSampleRate.Focus()
	case 3:
		m.wavLoopCount.Focus()
	case 4:
		m.wavExportGain.Focus()
	}
}

func (m *FileDialogModel) wavOptions() (int, int, float64, string) {
	sampleRateText := strings.TrimSpace(m.wavSampleRate.Value())
	loopCountText := strings.TrimSpace(m.wavLoopCount.Value())
	exportGainText := strings.TrimSpace(m.wavExportGain.Value())

	sampleRate, err := strconv.Atoi(sampleRateText)
	if err != nil || (sampleRate != 22050 && sampleRate != 44100 && sampleRate != 48000) {
		return 0, 0, 0, "Sample rate must be 22050, 44100, or 48000"
	}
	loopCount, err := strconv.Atoi(loopCountText)
	if err != nil || loopCount < 1 || loopCount > 64 {
		return 0, 0, 0, "Loop count must be between 1 and 64"
	}
	exportGain, err := strconv.ParseFloat(exportGainText, 64)
	if err != nil || exportGain < 0 || exportGain > 1.5 {
		return 0, 0, 0, "Export gain must be between 0.0 and 1.5"
	}
	return sampleRate, loopCount, exportGain, ""
}

// View renders the file browser dialog content (border is added by dialogModel).
func (m *FileDialogModel) View() tea.View {
	var sb strings.Builder
	renderSaveLabel := func(label string, active bool) string {
		if active {
			return fdTitleStyle.Render(label)
		}
		return fdLabelStyle.Render(label)
	}

	// Title
	title := "Load Module"
	if m.Mode == ModeSave {
		title = "Save Module"
	}
	sb.WriteString(fdTitleStyle.Render(title))
	sb.WriteString("\n\n")

	// Tab bar (load mode)
	if m.Mode == ModeLoad {
		filesTab := common.StyleTabInactive.Render("Files")
		builtInTab := common.StyleTabInactive.Render("Built-in")
		if m.loadPane == loadPaneBuiltIn {
			builtInTab = common.StyleTabActive.Render("Built-in")
		} else {
			filesTab = common.StyleTabActive.Render("Files")
		}
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, filesTab, builtInTab))
		sb.WriteString("\n\n")

		if m.loadPane == loadPaneFiles {
			sb.WriteString(m.fp.View())
		} else {
			quickstartLabel := "Quickstart"
			if m.loadBuiltinFocus == 0 {
				quickstartLabel = common.StyleSelected.Render(quickstartLabel)
			}
			sb.WriteString(quickstartLabel)
			sb.WriteString("\n")

			demoLabel := "Demo Song"
			if m.loadBuiltinFocus == 1 {
				demoLabel = common.StyleSelected.Render(demoLabel)
			}
			sb.WriteString(demoLabel)
		}
	} else {
		sb.WriteString(m.fp.View())
	}

	// Save mode: filename input row below the picker
	if m.Mode == ModeSave {
		sb.WriteString("\n")
		sb.WriteString(renderSaveLabel("Filename: ", m.focusPane == 1))
		sb.WriteString(m.input.View())
		if m.isWAVTarget() {
			sb.WriteString("\n\n")
			sb.WriteString(fdLabelStyle.Render("WAV export options"))
			sb.WriteString("\n")
			sb.WriteString(renderSaveLabel("Sample rate: ", m.focusPane == 2))
			sb.WriteString(m.wavSampleRate.View())
			sb.WriteString("  ")
			sb.WriteString(renderSaveLabel("Loops: ", m.focusPane == 3))
			sb.WriteString(m.wavLoopCount.View())
			sb.WriteString("  ")
			sb.WriteString(renderSaveLabel("Gain: ", m.focusPane == 4))
			sb.WriteString(m.wavExportGain.View())
		}
		if m.ErrMsg != "" {
			sb.WriteString("\n")
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
	case m.Mode == ModeSave && m.focusPane > 0:
		help = "Enter: save  Tab: next field  Shift+Tab: previous field  Esc: cancel"
	case m.Mode == ModeSave:
		help = "Tab: edit fields  ↑↓: navigate  Enter: open/select  Esc: cancel"
	case m.Mode == ModeLoad && m.loadPane == loadPaneBuiltIn:
		help = "Tab: files  ↑↓: choose song  Enter: load song  Esc: cancel"
	default:
		help = "Tab: built-in  ↑↓: navigate files  Enter: load file  Esc: cancel"
	}
	sb.WriteString(fdHelpStyle.Render(help))

	return tea.NewView(sb.String())
}
