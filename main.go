package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/persistence"
	"github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
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
	sampleRate   beep.SampleRate
	screens      []ui.Screen
	activeScreen int

	synthPresetView *ui.SynthPresetView // persistent across dialog opens

	octave       int
	globalVolume float64

	// current loaded/saved filename (prefill on save)
	currentFilename string
}

// synth returns the SynthScreen (always screens[synthScreenIdx]).
func (m model) synth() *ui.SynthScreen {
	return m.screens[synthScreenIdx].(*ui.SynthScreen)
}

// trackerModel returns the TrackerModel owned by the TrackerScreen.
func (m model) trackerModel() *ui.TrackerModel {
	return m.screens[trackerScreenIdx].(*ui.TrackerScreen).Tracker
}

// tickMsg is sent to advance playback
type tickMsg time.Time

var noteKeyToName = ui.NoteKeys

func (m model) Init() tea.Cmd {
	// Initialize speaker with sample rate
	sampleRate := m.sampleRate
	buffersize := sampleRate.N(time.Millisecond * 250)

	speaker.Init(sampleRate, buffersize)

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
			return ui.NewDialogModel(ui.NewSynthPresetsDialog(m.synthPresetView, m.octave), m, m.width, m.height), nil
		case "t":
			m.activeScreen = (m.activeScreen + 1) % len(m.screens)
			return m, nil
		case "+":
			if m.octave < maxOctave {
				m.octave++
			}

			note := m.trackerModel().GetNote()
			if newNote, ok := note.Transpose(-1); ok {
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
			if newNote, ok := note.Transpose(-1); ok {
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
			speaker.Clear()
			return m, tea.Quit
		}

		// Global note playing (available in any mode)
		if base, ok := noteKeyToName[msg.String()]; ok {
			note := audio.Note{Base: base, Octave: audio.Octave(m.octave)}
			m.playNote(note)

			if m.activeScreen == trackerScreenIdx {
				m.trackerModel().SetNote(note)
			}

			return m, nil
		}

		// Forward remaining key events to the active screen
		var cmd tea.Cmd
		m.screens[m.activeScreen], cmd = m.screens[m.activeScreen].Update(msg)
		return m, cmd

	case tickMsg:
		tracker := m.trackerModel()
		if !tracker.IsPlaying {
			return m, nil
		}

		// Play all notes at current playback row
		m.playRowNotes(tracker.PlaybackRow)

		// Advance to next row
		tracker.PlaybackRow++
		if tracker.LoopToRow {
			// Wrap within 0..loopEndRow inclusive
			if tracker.PlaybackRow > tracker.LoopEndRow {
				tracker.PlaybackRow = 0
			}
		} else {
			if tracker.PlaybackRow >= tracker.NumRows {
				tracker.PlaybackRow = 0 // Loop back to start
			}
		}

		// Schedule next tick
		return m, m.tick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.trackerModel().Viewport = ui.Viewport{
			Height: m.height - 7, // tab bar (1) + blank line (1) + newline (1) + footer padding+text (2) + panel border (2)
			Width:  m.width,
		}

		return m, nil

	case ui.TrackChanged:
		// Sync synth panels with the newly selected track
		m.synth().ApplyTrackChange(msg)

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

	case ui.VolumeChanged:
		m.globalVolume = msg.Volume

	case ui.PlaySynthPresetNoteMsg:
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

// tick returns a command that sends a tickMsg after a delay
func (m *model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// playNoteWithSynthPreset plays a note using the given synth preset's parameters.
func (m *model) playNoteWithSynthPreset(note audio.Note, preset ui.SynthPreset) {
	duration := time.Millisecond * 250
	volumeAdjusted := &effects.Volume{
		Streamer: preset.Synth.Streamer(m.sampleRate, note, duration),
		Base:     2,
		Volume:   volumeToDecibels(m.globalVolume),
		Silent:   m.globalVolume == 0,
	}
	speaker.Play(volumeAdjusted)
}

// playNote plays a note at the given frequency using the current oscillator
func (m *model) playNote(note audio.Note) {
	// TODO: duration should be adjustable
	duration := time.Millisecond * 250

	synth := m.synth().GetSynth()

	synthStreamer := synth.Streamer(m.sampleRate, note, duration)
	volumeAdjusted := &effects.Volume{
		Streamer: synthStreamer,
		Base:     2,
		Volume:   volumeToDecibels(m.globalVolume),
		Silent:   m.globalVolume == 0,
	}

	// Clear previous sound and play the new note
	speaker.Play(volumeAdjusted)
}

// playRowNotes plays all notes in the specified row across all tracks
func (m *model) playRowNotes(row int) {
	tracker := m.trackerModel()
	if row < 0 || row >= tracker.NumRows {
		return
	}

	// TODO: duration should be adjustable
	duration := time.Millisecond * 250
	var streamers []beep.Streamer

	// Collect all note generators for this row
	for trackIdx := 0; trackIdx < tracker.NumTracks; trackIdx++ {
		track := tracker.Tracks[trackIdx]
		trackRow := track.Rows[row]

		// Skip empty notes
		if audio.IsOff(trackRow.Note) {
			continue
		}

		synthStreamer := track.Synth.Streamer(m.sampleRate, trackRow.Note, duration)
		streamers = append(streamers, synthStreamer)
	}

	// If we have any notes to play, mix and play them
	if len(streamers) > 0 {
		mixed := beep.Mix(streamers...)

		// global vol
		volumeAdjusted := &effects.Volume{
			Streamer: mixed,
			Base:     2,
			Volume:   volumeToDecibels(m.globalVolume),
			Silent:   m.globalVolume == 0,
		}

		speaker.Play(volumeAdjusted)
	}
}

func volumeToDecibels(volume float64) float64 {
	if volume <= 0 {
		return -999
	}
	return math.Log2(volume) * 6
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
	sampleRate := beep.SampleRate(44100)

	// Create pattern with 8 tracks and 64 rows
	tracker := ui.NewTracker(8, 64, 0, 0)
	track := tracker.CurrentTrack()

	p := tea.NewProgram(
		model{
			sampleRate: sampleRate,
			screens: []ui.Screen{
				ui.NewTrackerScreen(tracker),
				ui.NewSynthScreen(track.Synth),
			},
			activeScreen:    trackerScreenIdx,
			synthPresetView: ui.NewSynthPresetView(),
			octave:          4,
			globalVolume:    1.0,
		},
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
