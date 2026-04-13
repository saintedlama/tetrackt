package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/persistence"
	"github.com/tetrackt/tetrackt/player"
	"github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
	"github.com/tetrackt/tetrackt/ui/synth"
	"github.com/tetrackt/tetrackt/ui/tracker"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	trackerScreenIdx = 0
	synthScreenIdx   = 1
)

const (
	minOctave = 1
	maxOctave = 8
)

// model represents the application state
type model struct {
	width        int
	height       int
	sampleRate   audio.SampleRate
	screens      []ui.Screen
	activeScreen int

	bank *persistence.PatchBank // user patch bank (~/.tetrackt)

	octave       int
	globalVolume float64

	// current loaded/saved filename (prefill on save)
	currentFilename string

	// dirty is true when there are unsaved changes.
	dirty bool
	// quitAfterSave signals that the app should quit once the next save completes.
	quitAfterSave bool

	player    player.Player
	mcpActive bool
}

// synth returns the SynthScreen (always screens[synthScreenIdx]).
func (m model) synth() *synth.SynthScreen {
	return m.screens[synthScreenIdx].(*synth.SynthScreen)
}

// trackerModel returns the TrackerModel owned by the TrackerScreen.
func (m model) trackerModel() *tracker.TrackerModel {
	return m.screens[trackerScreenIdx].(*tracker.TrackerScreen).Tracker
}

// tickMsg is sent to advance playback
type tickMsg time.Time

// previewTickMsg is sent to advance arp preview one sub-tick
type previewTickMsg time.Time

type openHelpMsg struct{}
type openSaveDialogMsg struct{}
type openLoadDialogMsg struct{}
type openPatchBankDialogMsg struct{}
type requestQuitMsg struct{}

//go:embed modules/quickstart.json
var embeddedQuickstart []byte

//go:embed modules/chiptune-demo.json
var embeddedDemo []byte

func (m model) Init() tea.Cmd {
	// Initialize speaker with sample rate
	player.Init(m.sampleRate)

	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		handled, globalCmd := ui.DispatchShortcutSections(msg, (&m).globalShortcutSections())
		if handled {
			return m, globalCmd
		}

		// Global note playing for synth screen.
		if base, ok := ui.NoteKeys[msg.String()]; ok && m.activeScreen != trackerScreenIdx {
			note := audio.Note{Base: base, Octave: audio.Octave(m.octave)}

			m.playNote(note)
			return m, nil
		}

		// Forward remaining key events to the active screen
		var screenCmd tea.Cmd
		m.screens[m.activeScreen], screenCmd = m.screens[m.activeScreen].Update(msg)
		return m, screenCmd

	case openHelpMsg:
		help := ui.NewHelpDialog(m.helpSections())
		return ui.NewDialogModel(help, m, m.width, m.height), nil

	case openSaveDialogMsg:
		prefill := "module"
		if m.currentFilename != "" {
			prefill = m.currentFilename
		}
		d := ui.NewDialogModel(ui.NewFileDialog(ui.ModeSave, prefill), m, m.width, m.height)
		return d, d.Init()

	case openLoadDialogMsg:
		d := ui.NewDialogModel(ui.NewFileDialog(ui.ModeLoad, ""), m, m.width, m.height)
		return d, d.Init()

	case openPatchBankDialogMsg:
		if m.activeScreen == synthScreenIdx {
			return ui.NewDialogModel(synth.NewSynthPatchBankDialog(m.synth().PatchBankView(), m.octave, m.synth().GetSynth()), m, m.width, m.height), nil
		}
		return m, nil

	case requestQuitMsg:
		if m.dirty {
			return ui.NewDialogModel(ui.NewQuitDialog(), m, m.width, m.height), nil
		}
		m.player.Clear()
		return m, tea.Quit

	case previewTickMsg:
		if m.player.TickPreview() {
			return m, m.previewTick()
		}
		return m, nil

	case tickMsg:
		tr := m.trackerModel()
		if !tr.IsPlaying {
			return m, nil
		}
		m.player.Tick(tr, m.sampleRate, m.globalVolume)
		return m, m.tick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Keep enough vertical budget for:
		// - header/tab area (3)
		// - tracker panel border+padding chrome (4)
		viewportHeight := m.height - 7
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		m.trackerModel().Viewport = tracker.Viewport{
			Height: viewportHeight,
			Width:  m.width,
		}

		return m, nil

	case ui.TrackChanged:
		// Sync synth panels with the newly selected track
		m.synth().ApplyTrackChange(msg)

	case tracker.OpenRowEffectsMsg:
		if m.activeScreen == trackerScreenIdx {
			d := tracker.NewRowEffectsDialog(msg.Row, msg.TrackIdx, msg.RowIdx)
			d.FocusForEffect(msg.Row.Effect)
			return ui.NewDialogModel(d, m, m.width, m.height), nil
		}

	case tracker.RowEffectsApplied:
		tm := m.trackerModel()
		if msg.TrackIdx < len(tm.Tracks) && msg.RowIdx < tm.NumRows {
			tm.Tracks[msg.TrackIdx].Rows[msg.RowIdx].Arpeggio = msg.Arpeggio
			tm.Tracks[msg.TrackIdx].Rows[msg.RowIdx].Ticks = msg.Ticks
			tm.Tracks[msg.TrackIdx].Rows[msg.RowIdx].Continuous = msg.Continuous
			tm.Tracks[msg.TrackIdx].Rows[msg.RowIdx].Effect = msg.Effect
		}
		m.dirty = true
		return m, nil

	case ui.FileDialogConfirmed:
		// Handle file dialog confirmation
		filename := msg.Filename
		switch msg.Mode {
		case ui.ModeSave:
			// Save module
			mod := persistence.TracksToModule(m.trackerModel())
			err := persistence.SaveToFile(filename, mod)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Save failed: %v\n", err)
			} else {
				m.currentFilename = filename
				m.dirty = false
				if m.quitAfterSave {
					m.player.Clear()
					return m, tea.Quit
				}
			}
		case ui.ModeLoad:
			// Load module
			switch filename {
			case ui.EmbeddedQuickstartToken:
				if err := m.loadEmbeddedQuickstart(); err != nil {
					fmt.Fprintf(os.Stderr, "Load quickstart failed: %v\n", err)
				}
			case ui.EmbeddedDemoToken:
				if err := m.loadEmbeddedDemo(); err != nil {
					fmt.Fprintf(os.Stderr, "Load demo song failed: %v\n", err)
				}
			default:
				mod, err := persistence.LoadFromFile(filename)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Load failed: %v\n", err)
				} else {
					// Update existing tracker model instead of creating new one
					persistence.ModuleToTracks(mod, m.trackerModel())
					m.currentFilename = filename
					m.dirty = false
				}
			}
		}
		return m, nil

	case tracker.VolumeChanged:
		m.globalVolume = msg.Volume
		m.dirty = true

	case tracker.BPMChanged:
		// BPM is already updated on the TrackerModel; nothing else to do here.
		m.dirty = true

	case tracker.TrackerEdited:
		m.dirty = true
		return m, nil

	case tracker.TrackerNoteEntered:
		m.dirty = true
		speed := msg.Speed
		if speed <= 0 {
			speed = tracker.DefaultSpeed
		}
		if m.player.StartPreview(msg.Note, msg.Row.Arpeggio, msg.Synth,
			m.trackerModel().BPMDuration(), m.sampleRate, m.globalVolume, speed) {
			return m, m.previewTick()
		}
		return m, nil

	case synth.OpenPatchBankMsg:
		return ui.NewDialogModel(synth.NewSynthPatchBankDialog(m.synth().PatchBankView(), m.octave, m.synth().GetSynth()), m, m.width, m.height), nil

	case synth.PlayPatchNoteMsg:
		m.playNoteWithSynthPatch(msg.Note, msg.Patch)
		return m, nil

	case synth.PatchSaveRequestedMsg:
		tags := append([]string{"Custom"}, msg.Tags...)
		patch := persistence.SavedPatch{
			Name:     msg.Name,
			Category: msg.Category,
			Tags:     tags,
			Synth:    persistence.ToSavedSynth(msg.Synth),
		}
		m.bank.SynthPatches = append(m.bank.SynthPatches, patch)
		if err := m.bank.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "patch bank save failed: %v\n", err)
		}
		m.synth().SetUserPatches(bankToPatches(m.bank))
		return m, nil

	case synth.PatchDeleteRequestedMsg:
		patches := m.bank.SynthPatches[:0]
		for _, p := range m.bank.SynthPatches {
			if !(p.IsCustom() && p.Name == msg.PatchName) {
				patches = append(patches, p)
			}
		}
		m.bank.SynthPatches = patches
		if err := m.bank.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "patch bank save failed: %v\n", err)
		}
		m.synth().SetUserPatches(bankToPatches(m.bank))
		return m, nil

	case synth.PatchRenameRequestedMsg:
		for i := range m.bank.SynthPatches {
			if m.bank.SynthPatches[i].IsCustom() && m.bank.SynthPatches[i].Name == msg.OldName {
				m.bank.SynthPatches[i].Name = msg.NewName
				break
			}
		}
		if err := m.bank.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "patch bank save failed: %v\n", err)
		}
		m.synth().SetUserPatches(bankToPatches(m.bank))
		return m, nil

	case ui.QuitDiscardMsg:
		m.player.Clear()
		return m, tea.Quit

	case ui.QuitSaveMsg:
		prefill := "module"
		if m.currentFilename != "" {
			prefill = m.currentFilename
		}
		m.quitAfterSave = true
		d := ui.NewDialogModel(ui.NewFileDialog(ui.ModeSave, prefill), m, m.width, m.height)
		return d, d.Init()

	case ui.SynthUpdated:
		var cmd1, cmd2 tea.Cmd
		m.screens[synthScreenIdx], cmd1 = m.screens[synthScreenIdx].Update(msg)
		m.screens[trackerScreenIdx], cmd2 = m.screens[trackerScreenIdx].Update(msg)
		m.dirty = true
		return m, tea.Batch(cmd1, cmd2)

	case mcpRequestMsg:
		result, err := msg.apply(&m)
		msg.reply <- mcpResponse{result: result, err: err}
		return m, nil
	}

	return m, nil
}

func (m *model) globalShortcutSections() []ui.ShortcutSection {
	return []ui.ShortcutSection{
		{
			Title: "Global",
			Shortcuts: []ui.Shortcut{
				{
					Keys:        []string{"?"},
					Description: "Open help",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						return true, func() tea.Msg { return openHelpMsg{} }
					},
				},
				{
					Keys:        []string{"ctrl+t"},
					Description: "Toggle Tracker / Synth screen",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						m.activeScreen = (m.activeScreen + 1) % len(m.screens)
						return true, nil
					},
				},
				{},
				{
					Keys:        []string{"p", "P"},
					KeyLabel:    "p / P",
					Description: "Play / Pause from row 0, Shifted loops to current row",
					Action: func(k tea.KeyPressMsg) (bool, tea.Cmd) {
						tm := m.trackerModel()
						tm.IsPlaying = !tm.IsPlaying
						tm.LoopToRow = false
						if tm.IsPlaying {
							tm.PlaybackRow = 0
							m.player.Reset()
							if k.String() == "P" {
								tm.LoopToRow = true
								tm.LoopEndRow = tm.CursorRow
							}
							return true, m.tick()
						}
						return true, nil
					},
				},
				{},
				{
					Keys:        []string{"ctrl+s"},
					Description: "Save module",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						return true, func() tea.Msg { return openSaveDialogMsg{} }
					},
				},
				{
					Keys:        []string{"ctrl+l"},
					Description: "Load module",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						return true, func() tea.Msg { return openLoadDialogMsg{} }
					},
				},
				{},
				{
					Keys:        []string{"+"},
					Description: "Octave up",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						if m.octave < maxOctave {
							m.octave++
							m.trackerModel().Octave = m.octave
						}
						return true, nil
					},
				},
				{
					Keys:        []string{"-"},
					Description: "Octave down",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						if m.octave > minOctave {
							m.octave--
							m.trackerModel().Octave = m.octave
						}
						return true, nil
					},
				},
				{
					Keys:        []string{"b", "B"},
					KeyLabel:    "b / B",
					Description: "Open patch bank on Synth screen",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						if m.activeScreen != synthScreenIdx {
							return false, nil
						}
						return true, func() tea.Msg { return openPatchBankDialogMsg{} }
					},
				},
				{KeyLabel: "Input profile", Description: "QWERTY default, QWERTZ via .tetrackt file"},
				{},
				{
					Keys:        []string{"ctrl+c"},
					Description: "Quit",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						return true, func() tea.Msg { return requestQuitMsg{} }
					},
				},
			},
		},
	}
}

func (m *model) helpSections() []ui.HelpSection {
	sections := ui.HelpSectionsFromShortcutSections(m.globalShortcutSections())
	sections = append(sections, m.screens[m.activeScreen].Help()...)
	return sections
}

// previewTick returns a command that sends a previewTickMsg after one sub-tick delay.
func (m *model) previewTick() tea.Cmd {
	trackerModel := m.trackerModel()
	speed := trackerModel.Speed
	if speed <= 0 {
		speed = tracker.DefaultSpeed
	}
	duration := trackerModel.BPMDuration() / time.Duration(speed)
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return previewTickMsg(t)
	})
}

// tick returns a command that sends a tickMsg after one sub-tick delay.
func (m *model) tick() tea.Cmd {
	trackerModel := m.trackerModel()
	speed := trackerModel.Speed
	if speed <= 0 {
		speed = tracker.DefaultSpeed
	}
	duration := trackerModel.BPMDuration() / time.Duration(speed)
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// playNoteWithSynthPatch plays a note using the given patch's synth parameters.
func (m *model) playNoteWithSynthPatch(note audio.Note, patch synth.SynthPatch) {
	duration := m.trackerModel().BPMDuration()
	noteSamples := m.sampleRate.N(duration)
	audioPatch := patch.Synth.NewPatch(m.sampleRate, note.Frequency(), noteSamples)
	m.player.Play(
		audioPatch,
		m.globalVolume,
	)
}

// bankToPatches converts the patch bank's patches to SynthPatch slice for the UI.
func bankToPatches(bank *persistence.PatchBank) []synth.SynthPatch {
	patches := make([]synth.SynthPatch, len(bank.SynthPatches))
	for i, p := range bank.SynthPatches {
		patches[i] = synth.SynthPatch{
			Name:     p.Name,
			Category: p.Category,
			Tags:     p.Tags,
			Synth:    persistence.SynthFromSavedPatch(p),
		}
	}
	return patches
}

// playNote plays a note at the given frequency using the current oscillator
func (m *model) playNote(note audio.Note) {
	duration := m.trackerModel().BPMDuration()
	synth := m.synth().GetSynth()
	noteSamples := m.sampleRate.N(duration)
	patch := synth.NewPatch(m.sampleRate, note.Frequency(), noteSamples)
	m.player.Play(
		patch,
		m.globalVolume,
	)
}

// loadEmbeddedQuickstart restores the built-in quickstart module from the
// executable and applies it to the current tracker model.
func (m *model) loadEmbeddedQuickstart() error {
	mod, err := persistence.LoadFromBytes(embeddedQuickstart)
	if err != nil {
		return err
	}
	persistence.ModuleToTracks(mod, m.trackerModel())
	m.currentFilename = "quickstart (embedded)"
	m.dirty = false
	return nil
}

func (m *model) loadEmbeddedDemo() error {
	mod, err := persistence.LoadFromBytes(embeddedDemo)
	if err != nil {
		return err
	}
	persistence.ModuleToTracks(mod, m.trackerModel())
	m.currentFilename = "demo song (embedded)"
	m.dirty = false
	return nil
}

// View renders the UI
func (m model) View() tea.View {
	// Tab bar
	tabs := make([]string, len(m.screens))
	for i, s := range m.screens {
		if i == m.activeScreen {
			tabs[i] = common.StyleTabActive.Render(s.Title())
		} else {
			tabs[i] = common.StyleTabInactive.Render(s.Title())
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	helpHint := lipgloss.NewStyle().
		Foreground(common.ColorTextDisabled).
		Render("? - Help")
	var mcpIndicator string
	if m.mcpActive {
		mcpIndicator = lipgloss.NewStyle().Foreground(common.ColorCyan).Render("● MCP") + "  "
	}
	right := lipgloss.JoinHorizontal(lipgloss.Top, mcpIndicator, helpHint, "  ", ui.Logo())
	spacerWidth := m.width - lipgloss.Width(tabBar) - lipgloss.Width(right)
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	header := tabBar + strings.Repeat(" ", spacerWidth) + right + "\n\n"

	body := m.screens[m.activeScreen].View()

	v := tea.NewView(header + body)
	v.AltScreen = true
	return v
}

func main() {
	mcpMode := flag.Bool("mcp", false, "start MCP server")
	mcpAddress := flag.String("mcp-address", "127.0.0.1:8347", "MCP bind address")
	flag.Parse()

	// Initialize synthesizer
	sampleRate := audio.SampleRate(44100)

	// Create pattern with 8 tracks and 64 rows
	trackerModel := tracker.NewTracker(8, 64, 0, 0)
	track := trackerModel.CurrentTrack()

	p := tea.NewProgram(
		newModel(sampleRate, trackerModel, track, *mcpMode),
	)

	if *mcpMode {
		go func(address string) {
			if err := runMCPServer(address, newMCPUIBridge(p)); err != nil {
				fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			}
		}(*mcpAddress)
	}

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// newModel constructs the application model, loading the user patch bank from disk.
func newModel(sampleRate audio.SampleRate, trackerModel *tracker.TrackerModel, track tracker.Track, mcpActive bool) model {
	trackerModel.Octave = 4

	bank, err := persistence.LoadPatchBank()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load patch bank: %v\n", err)
		bank = &persistence.PatchBank{Version: 1}
	}

	synthScreen := synth.NewSynthScreen(track.Synth)
	if len(bank.SynthPatches) > 0 {
		synthScreen.SetUserPatches(bankToPatches(bank))
	}

	return model{
		sampleRate: sampleRate,
		screens: []ui.Screen{
			tracker.NewTrackerScreen(trackerModel),
			synthScreen,
		},
		activeScreen: trackerScreenIdx,
		bank:         bank,
		octave:       4,
		globalVolume: 1.0,
		mcpActive:    mcpActive,
	}
}
