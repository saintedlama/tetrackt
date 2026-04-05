package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/persistence"
	"github.com/tetrackt/tetrackt/ui"

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

var (
	helpStyle = lipgloss.NewStyle().
			Foreground(ui.ColorTextDisabled).
			Padding(1, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(ui.ColorAccentInstrument).
			Foreground(ui.ColorWhite).
			Bold(true)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.ColorBackground).
			Background(ui.ColorAccentPrimary).
			Padding(0, 2)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(ui.ColorTextMuted).
				Background(ui.ColorSurface).
				Padding(0, 2)
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

	instrumentView *ui.InstrumentView // persistent across dialog opens

	octave       int
	globalVolume float64

	// current loaded/saved filename (prefill on save)
	currentFilename string
}

// synth returns the SynthScreen (always screens[synthScreenIdx]).
func (m model) synth() *ui.SynthScreen {
	return m.screens[synthScreenIdx].(*ui.SynthScreen)
}

// tracker returns the TrackerScreen (always screens[trackerScreenIdx]).
func (m model) tracker() *ui.TrackerScreen {
	return m.screens[trackerScreenIdx].(*ui.TrackerScreen)
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
			return ui.NewDialogModel(ui.NewInstrumentDialog(m.instrumentView, m.octave), m, m.width, m.height), nil
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

	case ui.OpenPresetDialogMsg:
		return ui.NewDialogModel(ui.NewEnvelopePresetDialog(), m, m.width, m.height), nil

	case ui.EnvelopePresetSelected:
		tracker := m.trackerModel()
		switch m.synth().ActivePanel {
		case 1: // Env1
			m.synth().Env1().Envelope = msg.Envelope
			tracker.Tracks[tracker.CursorTrack].Envelope1 = msg.Envelope
		case 3: // Env2
			m.synth().Env2().Envelope = msg.Envelope
			tracker.Tracks[tracker.CursorTrack].Envelope2 = msg.Envelope
		}
		return m, nil

	case ui.OscillatorUpdated:
		tracker := m.trackerModel()
		switch m.synth().ActivePanel {
		case 0: // Osc1
			tracker.Tracks[tracker.CursorTrack].Oscillator1 = msg.Oscillator
		case 2: // Osc2
			tracker.Tracks[tracker.CursorTrack].Oscillator2 = msg.Oscillator
		}
	case ui.EnvelopeUpdated:
		tracker := m.trackerModel()
		switch m.synth().ActivePanel {
		case 1: // Env1
			tracker.Tracks[tracker.CursorTrack].Envelope1 = msg.Envelope
		case 3: // Env2
			tracker.Tracks[tracker.CursorTrack].Envelope2 = msg.Envelope
		}
	case ui.MixerUpdated:
		m.trackerModel().Tracks[m.trackerModel().CursorTrack].Mixer = msg.Mixer

	case ui.VolumeChanged:
		m.globalVolume = msg.Volume

	case ui.PlayInstrumentNoteMsg:
		m.playNoteWithInstrument(msg.Note, msg.Instrument)
		return m, nil

	case ui.InstrumentApplied:
		synth := m.synth()
		synth.Osc1().Oscillator = msg.Instrument.Oscillator1
		synth.Env1().Envelope = msg.Instrument.Envelope1
		synth.Osc2().Oscillator = msg.Instrument.Oscillator2
		synth.Env2().Envelope = msg.Instrument.Envelope2
		synth.GetMixer().SetMixer(msg.Instrument.Mixer)

		tracker := m.trackerModel()
		track := &tracker.Tracks[tracker.CursorTrack]
		track.Oscillator1 = msg.Instrument.Oscillator1
		track.Envelope1 = msg.Instrument.Envelope1
		track.Oscillator2 = msg.Instrument.Oscillator2
		track.Envelope2 = msg.Instrument.Envelope2
		track.Mixer = msg.Instrument.Mixer
	}

	return m, nil
}

// tick returns a command that sends a tickMsg after a delay
func (m *model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// playNoteWithInstrument plays a note using the given instrument's parameters.
func (m *model) playNoteWithInstrument(note audio.Note, instr ui.Instrument) {
	duration := time.Millisecond * 250
	synth := audio.NewSynth(
		m.sampleRate,
		instr.Oscillator1,
		instr.Envelope1,
		instr.Oscillator2,
		instr.Envelope2,
		instr.Mixer,
	)
	volumeAdjusted := &effects.Volume{
		Streamer: synth.Streamer(note, duration),
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

	oscillator1, envelope1, oscillator2, envelope2, mixer := m.synth().GetActiveSynthParams()

	synth := audio.NewSynth(
		m.sampleRate,
		oscillator1,
		envelope1,
		oscillator2,
		envelope2,
		mixer)

	synthStreamer := synth.Streamer(note, duration)
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

	synth := m.synth()

	// Collect all note generators for this row
	for trackIdx := 0; trackIdx < tracker.NumTracks; trackIdx++ {
		track := tracker.Tracks[trackIdx]
		trackRow := track.Rows[row]

		// Skip empty notes
		if audio.IsOff(trackRow.Note) {
			continue
		}

		audioSynth := audio.NewSynth(
			m.sampleRate,
			synth.Osc1().Oscillator,
			synth.Env1().Envelope,
			synth.Osc2().Oscillator,
			synth.Env2().Envelope,
			synth.GetMixer().Mixer,
		)

		synthStreamer := audioSynth.Streamer(trackRow.Note, duration)
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
			tabs[i] = tabActiveStyle.Render(s.Title())
		} else {
			tabs[i] = tabInactiveStyle.Render(s.Title())
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	header := tabBar + "\n\n"

	body := m.screens[m.activeScreen].View()
	footer := helpStyle.Render(m.screens[m.activeScreen].Footer())

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

	synthPanels := []ui.Panel{
		ui.NewPanel("Oscillator 1", ui.ColorAccentOscillator, ui.NewOscillatorModel(selectedStyle, track.Oscillator1)),
		ui.NewPanel("Envelope 1", ui.ColorAccentEnvelope, ui.NewEnvelopeModel(selectedStyle, track.Envelope1)),
		ui.NewPanel("Oscillator 2", ui.ColorAccentOscillator, ui.NewOscillatorModel(selectedStyle, track.Oscillator2)),
		ui.NewPanel("Envelope 2", ui.ColorAccentEnvelope, ui.NewEnvelopeModel(selectedStyle, track.Envelope2)),
		ui.NewPanel("Mixer", ui.ColorAccentModulation, ui.NewMixer(track.Mixer.Volume1, track.Mixer.Volume2)),
	}

	p := tea.NewProgram(
		model{
			sampleRate: sampleRate,
			screens: []ui.Screen{
				ui.NewTrackerScreen(tracker),
				ui.NewSynthScreen(synthPanels),
			},
			activeScreen:   trackerScreenIdx,
			instrumentView: ui.NewInstrumentView(selectedStyle),
			octave:         4,
			globalVolume:   1.0,
		},
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
