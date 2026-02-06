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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
)

// InputMode represents the current input mode

var instrumentPanel = ui.NewInstrumentPanel()

type InputMode int

const (
	TrackMode InputMode = iota
	Oscillator1EditMode
	Envelope1EditMode
	Oscillator2EditMode
	Envelope2EditMode
	MixerEditMode
)

var (
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffa726")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Padding(1, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#d81b60")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true)

	panelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#333333")).
				Padding(0, 2)

	activePanelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00e5ff")).
				Bold(true).
				Padding(0, 2)

	modalBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#ff9800")).
				Padding(0, 2)
)

const (
	minOctave = 1
	maxOctave = 8
)

// model represents the application state
type model struct {
	width       int
	height      int
	sampleRate  beep.SampleRate
	oscillator1 *ui.OscillatorModel
	envelope1   *ui.EnvelopeModel
	oscillator2 *ui.OscillatorModel
	envelope2   *ui.EnvelopeModel
	mixer       *ui.Mixer
	tracker     *ui.TrackerModel

	mode InputMode

	octave       int
	globalVolume float64

	// file dialog
	fileDialog *ui.FileDialogModel
	// current loaded/saved filename (prefill on save)
	currentFilename string
}

// tickMsg is sent to advance playback
type tickMsg time.Time

var noteKeyToName = map[string]audio.Base{
	"1":  "C",
	"!":  "C#",
	"2":  "D",
	"@":  "D#",
	"\"": "D#", // german keyboard layout
	"3":  "E",
	"4":  "F",
	"$":  "F#",
	"5":  "G",
	"%":  "G#",
	"6":  "A",
	"^":  "A#",
	"&":  "A#", // german keyboard layout
	"7":  "B",
}

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
	case tea.KeyMsg:
		// Handle file dialog input first
		if m.fileDialog.IsVisible() {
			var cmd tea.Cmd
			*m.fileDialog, cmd = m.fileDialog.Update(msg)
			return m, cmd
		}

		// Global mode switching
		switch keyStr := msg.String(); keyStr {
		case "s":
			// Open save dialog
			prefill := "song"
			if m.currentFilename != "" {
				prefill = m.currentFilename
			}
			m.fileDialog.Show(ui.ModeSave, prefill)
			return m, nil
		case "l":
			// Open load dialog
			m.fileDialog.Show(ui.ModeLoad, "")
			return m, nil
		case "o":
			switch m.mode {
			case Oscillator1EditMode:
				m.mode = Oscillator2EditMode
			case Oscillator2EditMode:
				m.mode = Oscillator1EditMode
			default:
				m.mode = Oscillator1EditMode
			}

			return m, nil
		case "t":
			m.mode = TrackMode
			return m, nil
		case "e":
			switch m.mode {
			case Envelope1EditMode:
				m.mode = Envelope2EditMode
			case Envelope2EditMode:
				m.mode = Envelope1EditMode
			default:
				m.mode = Envelope1EditMode
			}

			return m, nil
		case "delete":
			// TODO: KeyMsg should be handled by the tracker
			m.tracker.SetNote(audio.Off())

		case "+":
			if m.octave < maxOctave {
				m.octave++
			}

			note := m.tracker.GetNote()
			if newNote, ok := note.Transpose(-1); ok {
				m.tracker.SetNote(newNote)
				m.playNote(newNote)
				return m, nil
			}

			return m, nil
		case "-":
			if m.octave > minOctave {
				m.octave--
			}

			note := m.tracker.CurrentTrack().CurrentRow().Note
			if newNote, ok := note.Transpose(-1); ok {
				m.tracker.SetNote(newNote)
				m.playNote(newNote)
				return m, nil
			}

			return m, nil
		// volume
		case "[", "alt+[": // decrease volume, for german keyboard layout we need to consider the alt+combo
			m.globalVolume -= 0.05
			if m.globalVolume < 0.0 {
				m.globalVolume = 0.0
			}

			m.mixer.GlobalVolume = m.globalVolume
			return m, nil
		case "]", "alt+]": // increase volume, for german keyboard layout we need to consider the alt+combo
			m.globalVolume += 0.05
			if m.globalVolume > 1.0 {
				m.globalVolume = 1.0
			}

			m.mixer.GlobalVolume = m.globalVolume
			return m, nil
		case "tab":
			m.mode = InputMode((int(m.mode) + 1) % 6) // Cycle through 6 modes
			return m, nil
		case "shift+tab":
			m.mode = InputMode((int(m.mode) - 1) % 6) // Cycle through 6 modes
			if m.mode < 0 {
				m.mode += 6
			}
			return m, nil
		case "p", "P":
			// Toggle play/pause
			m.tracker.IsPlaying = !m.tracker.IsPlaying
			m.tracker.LoopToRow = false // normal play toggles off loop mode
			if m.tracker.IsPlaying {
				m.tracker.PlaybackRow = 0

				// TODO: Loop to row is just a special play mode, that does not use 0..numRows range
				if "P" == keyStr {
					m.tracker.LoopToRow = true
					m.tracker.LoopEndRow = m.tracker.CursorRow
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

			if m.mode == TrackMode {
				m.tracker.SetNote(note)
			}

			return m, nil
		}

		if m.mode == Envelope1EditMode {
			var _, cmd = m.envelope1.Update(msg)
			return m, cmd
		}

		if m.mode == Envelope2EditMode {
			var _, cmd = m.envelope2.Update(msg)
			return m, cmd
		}

		// Handle oscillator edit mode
		if m.mode == Oscillator1EditMode {
			var _, cmd = m.oscillator1.Update(msg)
			return m, cmd
		}

		if m.mode == Oscillator2EditMode {
			var _, cmd = m.oscillator2.Update(msg)
			return m, cmd
		}

		if m.mode == MixerEditMode {
			var _, cmd = m.mixer.Update(msg)
			return m, cmd
		}

		if m.mode == TrackMode {
			var _, cmd = m.tracker.Update(msg)
			return m, cmd
		}

	case tickMsg:
		if !m.tracker.IsPlaying {
			return m, nil
		}

		// Play all notes at current playback row
		m.playRowNotes(m.tracker.PlaybackRow)

		// Advance to next row
		m.tracker.PlaybackRow++
		if m.tracker.LoopToRow {
			// Wrap within 0..loopEndRow inclusive
			if m.tracker.PlaybackRow > m.tracker.LoopEndRow {
				m.tracker.PlaybackRow = 0
			}
		} else {
			if m.tracker.PlaybackRow >= m.tracker.NumRows {
				m.tracker.PlaybackRow = 0 // Loop back to start
			}
		}

		// Schedule next tick
		return m, m.tick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// TODO: Could be more generic rendering the chrome and calculate chrome height
		synthViewHeight := lipgloss.Height(m.synthView())

		m.tracker.Viewport = ui.Viewport{
			Height: m.height - (synthViewHeight + 4),
			Width:  m.width,
		}

		return m, nil

	case ui.TrackChanged:
		// Update synth parameters based on current track
		m.envelope1.Envelope = msg.Envelope1
		m.oscillator1.Oscillator = msg.Oscillator1
		m.envelope2.Envelope = msg.Envelope2
		m.oscillator2.Oscillator = msg.Oscillator2
		m.mixer.Mixer = msg.Mixer

	case ui.FileDialogConfirmed:
		// Handle file dialog confirmation
		filename := msg.Filename
		switch m.fileDialog.Mode {
		case ui.ModeSave:
			// Save song
			song := persistence.TracksToSong(m.tracker)
			err := persistence.SaveToFile(filename, song)
			if err != nil {
				m.fileDialog.SetError(fmt.Sprintf("Save failed: %v", err))
			} else {
				m.currentFilename = filename
				m.fileDialog.Hide()
			}
		case ui.ModeLoad:
			// Load song
			song, err := persistence.LoadFromFile(filename)
			if err != nil {
				m.fileDialog.SetError(fmt.Sprintf("Load failed: %v", err))
			} else {
				// Update existing tracker model instead of creating new one
				persistence.SongToTracks(song, m.tracker)
				m.currentFilename = filename
				m.fileDialog.Hide()
			}
		}
		return m, nil

	case ui.FileDialogCancelled:
		// Handle file dialog cancellation
		m.fileDialog.Hide()
		return m, nil

	case ui.OscillatorUpdated:
		// TODO: Refactor to allow updating via a method instead of direct field access
		switch m.mode {
		case Oscillator1EditMode:
			m.tracker.Tracks[m.tracker.CursorTrack].Oscillator1 = msg.Oscillator
		case Oscillator2EditMode:
			m.tracker.Tracks[m.tracker.CursorTrack].Oscillator2 = msg.Oscillator
		}
	case ui.EnvelopeUpdated:
		// TODO: Refactor to allow updating via a method instead of direct field access
		switch m.mode {
		case Envelope1EditMode:
			m.tracker.Tracks[m.tracker.CursorTrack].Envelope1 = msg.Envelope
		case Envelope2EditMode:
			m.tracker.Tracks[m.tracker.CursorTrack].Envelope2 = msg.Envelope
		}
	case ui.MixerUpdated:
		m.tracker.Tracks[m.tracker.CursorTrack].Mixer = msg.Mixer
	}

	return m, nil
}

// tick returns a command that sends a tickMsg after a delay
func (m *model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// playNote plays a note at the given frequency using the current oscillator
func (m *model) playNote(note audio.Note) {
	// TODO: duration should be adjustable
	duration := time.Millisecond * 250

	synth := audio.NewSynth(
		m.sampleRate,
		m.oscillator1.Oscillator,
		m.envelope1.Envelope,
		m.oscillator2.Oscillator,
		m.envelope2.Envelope,
		m.mixer.Mixer)

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
	if row < 0 || row >= m.tracker.NumRows {
		return
	}

	// TODO: duration should be adjustable
	duration := time.Millisecond * 250
	var streamers []beep.Streamer

	// Collect all note generators for this row
	for trackIdx := 0; trackIdx < m.tracker.NumTracks; trackIdx++ {
		track := m.tracker.Tracks[trackIdx]
		trackRow := track.Rows[row]

		// Skip empty notes
		if audio.IsOff(trackRow.Note) {
			continue
		}

		synth := audio.NewSynth(
			m.sampleRate,
			m.oscillator1.Oscillator,
			m.envelope1.Envelope,
			m.oscillator2.Oscillator,
			m.envelope2.Envelope,
			m.mixer.Mixer,
		)

		synthStreamer := synth.Streamer(trackRow.Note, duration)
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
func (m model) View() string {
	// Build header
	var header strings.Builder

	modeStr := "TRACK"
	switch m.mode {
	case Envelope1EditMode:
		modeStr = "ENVELOPE1"
	case Envelope2EditMode:
		modeStr = "ENVELOPE2"
	case MixerEditMode:
		modeStr = "MIXER"
	case Oscillator1EditMode:
		modeStr = "OSCILLATOR1"
	case Oscillator2EditMode:
		modeStr = "OSCILLATOR2"
	}

	playStatus := "STOPPED"
	if m.tracker.IsPlaying {
		if m.tracker.LoopToRow {
			playStatus = fmt.Sprintf("LOOP 0-%d (Row %d)", m.tracker.LoopEndRow, m.tracker.PlaybackRow)
		} else {
			playStatus = fmt.Sprintf("PLAYING (Row %d)", m.tracker.PlaybackRow)
		}
	}

	header.WriteString(infoStyle.Render(fmt.Sprintf("Mode: %s | %s | Track: %d | Row: %d | Octave: %d",
		modeStr, playStatus, m.tracker.CursorTrack, m.tracker.CursorRow, m.octave)))
	header.WriteString("\n\n")

	synthView := m.synthView()
	trackerView := m.tracker.View()

	// Apply border to track view with conditional highlighting
	trackerBorder := panelBorderStyle
	if m.mode == TrackMode {
		trackerBorder = activePanelBorderStyle
	}

	trackerViewWithBorder := trackerBorder.Render(trackerView)
	body := lipgloss.JoinVertical(lipgloss.Left, synthView, trackerViewWithBorder)

	// Footer help
	footer := helpStyle.Render("↑↓←→: Navigate | J: Jump | 1-7: Notes | Shift+1-6: Sharp Notes | +/-: Octave | [/]: Volume | W: Oscillator | E: Envelope | T: Track | p: Play/Pause | P: Loop | S: Save | L: Load | Q: Quit")

	// TODO: More generic modal handling, use commands?
	if m.envelope1.ShowModal && m.mode == Envelope1EditMode {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.envelope1.View())
	}

	if m.envelope2.ShowModal && m.mode == Envelope2EditMode {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.envelope2.View())
	}

	// File dialog modal
	if m.fileDialog.IsVisible() {
		modalView := m.fileDialog.View()
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalView)
	}

	return header.String() + body + "\n" + footer
}

func (m model) synthView() string {
	oscillatorView1 := m.oscillator1.View()
	envelopeView1 := m.envelope1.View()

	oscillatorView2 := m.oscillator2.View()
	envelopeView2 := m.envelope2.View()

	m.mixer.GlobalVolume = m.globalVolume

	// Apply active border to the current mode panel
	oscillator1Border := panelBorderStyle
	envelope1Border := panelBorderStyle
	oscillator2Border := panelBorderStyle
	envelope2Border := panelBorderStyle
	mixerBorder := panelBorderStyle

	switch m.mode {
	case Oscillator1EditMode:
		oscillator1Border = activePanelBorderStyle
	case Envelope1EditMode:
		envelope1Border = activePanelBorderStyle
	case Oscillator2EditMode:
		oscillator2Border = activePanelBorderStyle
	case Envelope2EditMode:
		envelope2Border = activePanelBorderStyle
	case MixerEditMode:
		mixerBorder = activePanelBorderStyle
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		oscillator1Border.Render(oscillatorView1),
		envelope1Border.Render(envelopeView1),
		oscillator2Border.Render(oscillatorView2),
		envelope2Border.Render(envelopeView2),
		mixerBorder.Render(m.mixer.View()),
	)
}

func main() {
	// Initialize synthesizer
	sampleRate := beep.SampleRate(44100)

	// Create pattern with 8 tracks and 64 rows
	tracker := ui.NewTracker(8, 64, 0, 0)
	track := tracker.CurrentTrack()

	p := tea.NewProgram(
		model{
			sampleRate:   sampleRate,
			oscillator1:  ui.NewOscillatorModel(selectedStyle, track.Oscillator1),
			envelope1:    ui.NewEnvelopeModel(selectedStyle, track.Envelope1),
			oscillator2:  ui.NewOscillatorModel(selectedStyle, track.Oscillator2),
			envelope2:    ui.NewEnvelopeModel(selectedStyle, track.Envelope2),
			mixer:        ui.NewMixer(track.Mixer.Balance),
			tracker:      tracker,
			mode:         TrackMode,
			octave:       4,
			globalVolume: 1.0,
			fileDialog:   ui.NewFileDialog(modalBorderStyle),
		},

		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
