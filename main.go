package main

import (
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

	synthPresetView *synth.SynthPresetView // persistent across dialog opens

	octave       int
	globalVolume float64

	// current loaded/saved filename (prefill on save)
	currentFilename string

	player player.Player
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

var noteKeyToName = ui.NoteKeys

func (m model) Init() tea.Cmd {
	// Initialize speaker with sample rate
	player.Init(m.sampleRate)

	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Global mode switching
		switch keyStr := msg.String(); keyStr {
		case "s":
			// Open save dialog
			prefill := "song"
			if m.currentFilename != "" {
				prefill = m.currentFilename
			}
			return ui.NewDialogModel(ui.NewFileDialog(ui.ModeSave, prefill), m, m.width, m.height), nil
		case "l":
			// Open load dialog
			return ui.NewDialogModel(ui.NewFileDialog(ui.ModeLoad, ""), m, m.width, m.height), nil
		case "i":
			return ui.NewDialogModel(synth.NewSynthPresetsDialog(m.synthPresetView, m.octave), m, m.width, m.height), nil
		case "e":
			if m.activeScreen == trackerScreenIdx {
				tm := m.trackerModel()
				row := tm.Tracks[tm.CursorTrack].Rows[tm.CursorRow]
				return ui.NewDialogModel(tracker.NewRowEffectsDialog(row, tm.CursorTrack, tm.CursorRow), m, m.width, m.height), nil
			}
		case "t":
			m.activeScreen = (m.activeScreen + 1) % len(m.screens)
			return m, nil
		case "+":
			if m.octave < maxOctave {
				m.octave++
			}

			note := m.trackerModel().GetNote()
			if newNote, ok := note.Transpose(-12); ok {
				m.trackerModel().SetNote(newNote)
				m.playNote(newNote)
				return m, nil
			}

			return m, nil
		case "-":
			if m.octave > minOctave {
				m.octave--
			}

			note := m.trackerModel().CurrentTrack().CurrentRow().Note
			if newNote, ok := note.Transpose(-12); ok {
				m.trackerModel().SetNote(newNote)
				m.playNote(newNote)
				return m, nil
			}

			return m, nil
		case "p", "P":
			// Toggle play/pause
			tracker := m.trackerModel()
			tracker.IsPlaying = !tracker.IsPlaying
			tracker.LoopToRow = false // normal play toggles off loop mode
			if tracker.IsPlaying {
				tracker.PlaybackRow = 0
				m.player.Reset()

				// TODO: Loop to row is just a special play mode, that does not use 0..numRows range
				if "P" == keyStr {
					tracker.LoopToRow = true
					tracker.LoopEndRow = tracker.CursorRow
				}

				// TODO: Refactor to have a play command returned from tracker.Update
				return m, m.tick()
			} else {
				//speaker.Clear()
			}
		case "q", "ctrl+c":
			m.player.Clear()
			return m, tea.Quit
		}

		// Global note playing (available in any mode)
		if base, ok := noteKeyToName[msg.String()]; ok {
			note := audio.Note{Base: base, Octave: audio.Octave(m.octave)}

			if m.activeScreen == trackerScreenIdx {
				tr := m.trackerModel()
				tr.SetNote(note)
				row := tr.Tracks[tr.CursorTrack].Rows[tr.CursorRow]
				if m.player.StartPreview(note, row.Arpeggio, tr.Tracks[tr.CursorTrack].Synth,
					tr.BPMDuration(), m.sampleRate, m.globalVolume, tr.Speed) {
					return m, m.previewTick()
				}
				return m, nil
			}

			m.playNote(note)
			return m, nil
		}

		// Forward remaining key events to the active screen
		var cmd tea.Cmd
		m.screens[m.activeScreen], cmd = m.screens[m.activeScreen].Update(msg)
		return m, cmd

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

		m.trackerModel().Viewport = tracker.Viewport{
			Height: m.height - 7, // tab bar (1) + blank line (1) + newline (1) + footer padding+text (2) + panel border (2)
			Width:  m.width,
		}

		return m, nil

	case ui.TrackChanged:
		// Sync synth panels with the newly selected track
		m.synth().ApplyTrackChange(msg)

	case tracker.RowEffectsApplied:
		tm := m.trackerModel()
		if msg.TrackIdx < len(tm.Tracks) && msg.RowIdx < tm.NumRows {
			tm.Tracks[msg.TrackIdx].Rows[msg.RowIdx].Arpeggio = msg.Arpeggio
			tm.Tracks[msg.TrackIdx].Rows[msg.RowIdx].Ticks = msg.Ticks
			tm.Tracks[msg.TrackIdx].Rows[msg.RowIdx].Continuous = msg.Continuous
		}
		return m, nil

	case ui.FileDialogConfirmed:
		// Handle file dialog confirmation
		filename := msg.Filename
		switch msg.Mode {
		case ui.ModeSave:
			// Save song
			song := persistence.TracksToSong(m.trackerModel())
			err := persistence.SaveToFile(filename, song)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Save failed: %v\n", err)
			} else {
				m.currentFilename = filename
			}
		case ui.ModeLoad:
			// Load song
			song, err := persistence.LoadFromFile(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Load failed: %v\n", err)
			} else {
				// Update existing tracker model instead of creating new one
				persistence.SongToTracks(song, m.trackerModel())
				m.currentFilename = filename
			}
		}
		return m, nil

	case tracker.VolumeChanged:
		m.globalVolume = msg.Volume

	case tracker.BPMChanged:
		// BPM is already updated on the TrackerModel; nothing else to do here.

	case synth.PlaySynthPresetNoteMsg:
		m.playNoteWithSynthPreset(msg.Note, msg.Preset)
		return m, nil

	case ui.SynthUpdated:
		var cmd1, cmd2 tea.Cmd
		m.screens[synthScreenIdx], cmd1 = m.screens[synthScreenIdx].Update(msg)
		m.screens[trackerScreenIdx], cmd2 = m.screens[trackerScreenIdx].Update(msg)
		return m, tea.Batch(cmd1, cmd2)
	}

	return m, nil
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

// playNoteWithSynthPreset plays a note using the given synth preset's parameters.
func (m *model) playNoteWithSynthPreset(note audio.Note, preset synth.SynthPreset) {
	duration := m.trackerModel().BPMDuration()
	noteSamples := m.sampleRate.N(duration)
	patch := preset.Synth.NewPatch(m.sampleRate, note.Frequency(), noteSamples)
	m.player.Play(
		patch,
		m.globalVolume,
	)
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

	logoStr := ui.Logo()
	spacerWidth := m.width - lipgloss.Width(tabBar) - lipgloss.Width(logoStr)
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	header := tabBar + strings.Repeat(" ", spacerWidth) + logoStr + "\n\n"

	body := m.screens[m.activeScreen].View()
	footer := common.StyleHelp.Render(m.screens[m.activeScreen].Footer())

	v := tea.NewView(header + body + "\n" + footer)
	v.AltScreen = true
	return v
}

func main() {
	// Initialize synthesizer
	sampleRate := audio.SampleRate(44100)

	// Create pattern with 8 tracks and 64 rows
	trackerModel := tracker.NewTracker(8, 64, 0, 0)
	track := trackerModel.CurrentTrack()

	p := tea.NewProgram(
		model{
			sampleRate: sampleRate,
			screens: []ui.Screen{
				tracker.NewTrackerScreen(trackerModel),
				synth.NewSynthScreen(track.Synth),
			},
			activeScreen:    trackerScreenIdx,
			synthPresetView: synth.NewSynthPresetView(),
			octave:          4,
			globalVolume:    1.0,
		},
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
