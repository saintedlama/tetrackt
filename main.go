package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/notes"
	"github.com/tetrackt/tetrackt/persistence"
	"github.com/tetrackt/tetrackt/render"
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

	speaker   *render.SpeakerSink
	preview   render.PreviewPlayer
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

//go:embed modules/quickstart.json
var embeddedQuickstart []byte

//go:embed modules/chiptune-demo.json
var embeddedDemo []byte

func (m model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Global mode switching
		switch keyStr := msg.String(); keyStr {
		case "?":
			help := ui.NewHelpDialog(m.screens[m.activeScreen].Help())
			return ui.NewDialogModel(help, m, m.width, m.height), nil
		case "ctrl+s":
			// Open save dialog
			prefill := "module"
			if m.currentFilename != "" {
				prefill = m.currentFilename
			}
			d := ui.NewDialogModel(ui.NewFileDialog(ui.ModeSave, prefill), m, m.width, m.height)
			return d, d.Init()
		case "ctrl+l":
			// Open load dialog
			d := ui.NewDialogModel(ui.NewFileDialog(ui.ModeLoad, ""), m, m.width, m.height)
			return d, d.Init()
		case "ctrl+e":
			if m.activeScreen == trackerScreenIdx {
				tm := m.trackerModel()
				cursorTrack := tm.CursorTrack()
				cursorRow := tm.CursorRow()
				row := tm.Tracks[cursorTrack].Rows[cursorRow]
				d := tracker.NewRowEffectsDialog(row, cursorTrack, cursorRow)
				d.FocusForEffect(row.Effect)
				return ui.NewDialogModel(d, m, m.width, m.height), nil
			}
		case "ctrl+t":
			m.activeScreen = (m.activeScreen + 1) % len(m.screens)
			return m, nil
		case "+":
			if m.octave < maxOctave {
				m.octave++
				m.trackerModel().Octave = m.octave
			}
			return m, nil
		case "-":
			if m.octave > minOctave {
				m.octave--
				m.trackerModel().Octave = m.octave
			}
			return m, nil
		case "b", "B":
			// On the synth screen b/B opens the patch bank.
			if m.activeScreen == synthScreenIdx {
				return ui.NewDialogModel(synth.NewSynthPatchBankDialog(m.synth().PatchBankView(), m.octave, m.synth().GetSynth()), m, m.width, m.height), nil
			}
		case "p", "P":
			// Toggle play/pause
			tracker := m.trackerModel()
			tracker.IsPlaying = !tracker.IsPlaying
			if tracker.IsPlaying {
				m.speaker.Clear()
				m.preview.Reset()

				if keyStr == "P" {
					tracker.LoopEndRow = tracker.CursorRow()
					tracker.LoopEndSet = true
					if !tracker.LoopStartSet {
						tracker.LoopStartRow = 0
						tracker.LoopStartSet = true
					}
				}

				loop := tracker.LoopStartSet
				tracker.PlaybackRow = 0
				if loop {
					tracker.PlaybackRow = tracker.LoopStartRow
				}

				cfg := render.RenderConfig{
					SampleRate:   m.sampleRate,
					GlobalVolume: m.globalVolume,
					LoopCount:    1,
				}
				if loop {
					if tracker.LoopEndSet {
						cfg.EndRow = tracker.LoopEndRow + 1
					}
				}
				stream, err := render.RenderToStream(tracker.ToRenderPattern(), cfg, loop)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Playback render failed: %v\n", err)
					tracker.IsPlaying = false
					return m, nil
				}
				if stream == nil {
					tracker.IsPlaying = false
					return m, nil
				}
				m.speaker.Play(stream, 1.0)
				return m, m.tick()
			} else {
				m.speaker.Clear()
			}
		case "ctrl+c":
			if m.dirty {
				return ui.NewDialogModel(ui.NewQuitDialog(), m, m.width, m.height), nil
			}
			m.speaker.Clear()
			return m, tea.Quit
		}

		// Global note playing for synth screen.
		if base, ok := ui.NoteKeys[msg.String()]; ok && m.activeScreen != trackerScreenIdx {
			note := notes.Note{Base: base, Octave: notes.Octave(m.octave)}

			m.playNote(note)
			return m, nil
		}

		// Forward remaining key events to the active screen
		var cmd tea.Cmd
		m.screens[m.activeScreen], cmd = m.screens[m.activeScreen].Update(msg)
		return m, cmd

	case previewTickMsg:
		if m.preview.Tick() {
			return m, m.previewTick()
		}
		return m, nil

	case tickMsg:
		tr := m.trackerModel()
		if !tr.IsPlaying {
			return m, nil
		}
		tr.PlaybackRow++
		if tr.LoopStartSet {
			effectiveEnd := tr.NumRows - 1
			if tr.LoopEndSet {
				effectiveEnd = tr.LoopEndRow
			}
			if tr.PlaybackRow > effectiveEnd {
				tr.PlaybackRow = tr.LoopStartRow
			}
		} else if tr.PlaybackRow >= tr.NumRows {
			tr.IsPlaying = false
			tr.PlaybackRow = 0
			m.speaker.Clear()
			return m, nil
		}
		return m, m.tick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Keep enough vertical budget for:
		// - header/tab area (3)
		// - tracker panel border+padding chrome (4)
		viewportHeight := max(m.height-7, 1)

		m.trackerModel().SetViewport(m.width, viewportHeight)

		return m, nil

	case ui.TrackChanged:
		// Sync synth panels with the newly selected track
		m.synth().ApplyTrackChange(msg)

	case tracker.RowEffectsApplied:
		tm := m.trackerModel()
		if msg.TrackIdx < len(tm.Tracks) && msg.RowIdx < tm.NumRows {
			tm.Tracks[msg.TrackIdx].Rows[msg.RowIdx].Volume = msg.Volume
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
			var err error
			if strings.HasSuffix(strings.ToLower(filename), ".wav") {
				err = render.ExportWAV(m.trackerModel().ToRenderPattern(), filename, render.WavExportOptions{
					SampleRate:   audio.SampleRate(msg.WavSampleRate),
					GlobalVolume: msg.WavExportGain,
					LoopCount:    msg.WavLoopCount,
				})
			} else {
				mod := persistence.TracksToModule(m.trackerModel())
				err = persistence.SaveToFile(filename, mod)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Save failed: %v\n", err)
			} else {
				if !strings.HasSuffix(strings.ToLower(filename), ".wav") {
					m.currentFilename = filename
				}
				m.dirty = false
				if m.quitAfterSave {
					m.speaker.Clear()
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
					fmt.Fprintf(os.Stderr, "Load demo module failed: %v\n", err)
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
		if m.preview.Start(tracker.ToRenderRow(msg.Row), msg.Synth,
			m.trackerModel().BPM.Duration(), m.sampleRate, m.globalVolume) {
			return m, m.previewTick()
		}
		return m, nil

	case synth.OpenPatchBankMsg:
		return ui.NewDialogModel(synth.NewSynthPatchBankDialog(m.synth().PatchBankView(), m.octave, m.synth().GetSynth()), m, m.width, m.height), nil

	case synth.OpenWavetableDialogMsg:
		return ui.NewDialogModel(synth.NewWavetableDialog(msg.BankIdx, msg.EntryIdx), m, m.width, m.height), nil

	case synth.WavetablePickedMsg:
		var cmd tea.Cmd
		m.screens[synthScreenIdx], cmd = m.screens[synthScreenIdx].Update(msg)
		return m, cmd

	case synth.PlayPatchNoteMsg:
		m.playNoteWithSynthPatch(msg.Note, msg.Patch)
		return m, nil

	case synth.PatchSaveRequestedMsg:
		tags := append([]string{"Custom"}, msg.Tags...)
		patch := persistence.SavedPatch{
			Name:  msg.Name,
			Bank:  msg.Bank,
			Tags:  tags,
			Synth: persistence.ToSavedSynth(msg.Synth),
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
		m.speaker.Clear()
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

// previewTick returns a command that sends a previewTickMsg after one sub-tick delay.
func (m *model) previewTick() tea.Cmd {
	trackerModel := m.trackerModel()
	ticks := trackerModel.RowTicks(0)
	duration := trackerModel.BPM.Duration() / time.Duration(ticks)
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return previewTickMsg(t)
	})
}

// tick returns a command that sends a tickMsg after one full row duration.
func (m *model) tick() tea.Cmd {
	return tea.Tick(m.trackerModel().BPM.Duration(), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// playNoteWithSynthPatch plays a note using the given patch's synth parameters.
func (m *model) playNoteWithSynthPatch(note notes.Note, patch synth.SynthPatch) {
	duration := m.trackerModel().BPM.Duration()
	noteSamples := m.sampleRate.N(duration)
	audioPatch := patch.Synth.NewPatch(m.sampleRate, note.Frequency(), noteSamples)
	m.speaker.Play(
		audioPatch,
		m.globalVolume,
	)
}

// bankToPatches converts the patch bank's patches to SynthPatch slice for the UI.
func bankToPatches(bank *persistence.PatchBank) []synth.SynthPatch {
	patches := make([]synth.SynthPatch, len(bank.SynthPatches))
	for i, p := range bank.SynthPatches {
		s := persistence.SynthFromSavedPatch(p)
		s.Meta = audio.Metadata{Bank: p.Bank, Name: p.Name, Tags: p.Tags}
		patches[i] = synth.SynthPatch{
			Name:  p.Name,
			Bank:  p.Bank,
			Tags:  p.Tags,
			Synth: s,
		}
	}
	return patches
}

// playNote plays a note at the given frequency using the current oscillator
func (m *model) playNote(note notes.Note) {
	duration := m.trackerModel().BPM.Duration()
	synth := m.synth().GetSynth()
	noteSamples := m.sampleRate.N(duration)
	patch := synth.NewPatch(m.sampleRate, note.Frequency(), noteSamples)
	m.speaker.Play(
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
	m.currentFilename = "demo module (embedded)"
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
	octaveIndicator := lipgloss.NewStyle().Foreground(common.ColorTextMuted).Render(fmt.Sprintf("Oct:%d", m.octave)) + "  "
	right := lipgloss.JoinHorizontal(lipgloss.Top, mcpIndicator, octaveIndicator, helpHint, "  ", ui.Logo())
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
	track := trackerModel.Tracks[0]

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

	if bank.InputProfile != "" {
		ui.SetInputProfileFromString(bank.InputProfile)
	} else {
		ui.SetInputProfile(ui.InputProfileQWERTY)
	}
	bank.InputProfile = string(ui.CurrentInputProfile())

	synthScreen := synth.NewSynthScreen(track.Synth)
	if len(bank.SynthPatches) > 0 {
		synthScreen.SetUserPatches(bankToPatches(bank))
	}
	speakerSink := render.NewSpeakerSink(sampleRate)

	return model{
		sampleRate: sampleRate,
		speaker:    speakerSink,
		preview:    render.NewPreviewPlayer(speakerSink),
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
