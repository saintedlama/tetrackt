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

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
)

// InputMode represents the current input mode
type InputMode int

const (
	TrackMode InputMode = iota
	Oscillator1EditMode
	Envelope1EditMode
	Oscillator2EditMode
	Envelope2EditMode
	MixerEditMode
	InstrumentMode
)

var (
	infoStyle = lipgloss.NewStyle().
			Foreground(ui.ColorAccentEnvelope).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(ui.ColorTextDisabled).
			Padding(1, 1)

	selectedStyle = lipgloss.NewStyle().
			Background(ui.ColorAccentInstrument).
			Foreground(ui.ColorWhite).
			Bold(true)
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
	synthPanels []ui.Panel
	tracker     *ui.TrackerModel

	mode InputMode

	octave       int
	globalVolume float64

	// current loaded/saved filename (prefill on save)
	currentFilename string
}

// Accessors for synth panel children (synthPanels order: osc1, env1, osc2, env2, mixer, instrument)
func (m model) osc1() *ui.OscillatorModel  { return m.synthPanels[0].Child.(*ui.OscillatorModel) }
func (m model) env1() *ui.EnvelopeModel    { return m.synthPanels[1].Child.(*ui.EnvelopeModel) }
func (m model) osc2() *ui.OscillatorModel  { return m.synthPanels[2].Child.(*ui.OscillatorModel) }
func (m model) env2() *ui.EnvelopeModel    { return m.synthPanels[3].Child.(*ui.EnvelopeModel) }
func (m model) mixer() *ui.Mixer           { return m.synthPanels[4].Child.(*ui.Mixer) }
func (m model) instr() *ui.InstrumentView  { return m.synthPanels[5].Child.(*ui.InstrumentView) }

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

			m.mixer().GlobalVolume = m.globalVolume
			return m, nil
		case "]", "alt+]": // increase volume, for german keyboard layout we need to consider the alt+combo
			m.globalVolume += 0.05
			if m.globalVolume > 1.0 {
				m.globalVolume = 1.0
			}

			m.mixer().GlobalVolume = m.globalVolume
			return m, nil
		case "tab":
			m.mode = InputMode((int(m.mode) + 1) % 7) // Cycle through 7 modes
			return m, nil
		case "shift+tab":
			m.mode = InputMode((int(m.mode) - 1) % 7) // Cycle through 7 modes
			if m.mode < 0 {
				m.mode += 7
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

		if m.mode == TrackMode {
			var _, cmd = m.tracker.Update(msg)
			return m, cmd
		}

		// Route to the active synth panel (modes 1-6 map to synthPanels[0-5])
		idx := int(m.mode) - 1
		var cmd tea.Cmd
		m.synthPanels[idx], cmd = m.synthPanels[idx].Update(msg)
		return m, cmd

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
		m.env1().Envelope = msg.Envelope1
		m.osc1().Oscillator = msg.Oscillator1
		m.env2().Envelope = msg.Envelope2
		m.osc2().Oscillator = msg.Oscillator2
		m.mixer().Mixer = msg.Mixer
		m.instr().CurrentTrackNum = m.tracker.CursorTrack

	case ui.FileDialogConfirmed:
		// Handle file dialog confirmation
		filename := msg.Filename
		switch msg.Mode {
		case ui.ModeSave:
			// Save song
			song := persistence.TracksToSong(m.tracker)
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
				persistence.SongToTracks(song, m.tracker)
				m.currentFilename = filename
			}
		}
		return m, nil

	case ui.OpenPresetDialogMsg:
		return ui.NewDialogModel(ui.NewEnvelopePresetDialog(), m, m.width, m.height), nil

	case ui.EnvelopePresetSelected:
		switch m.mode {
		case Envelope1EditMode:
			m.env1().Envelope = msg.Envelope
			m.tracker.Tracks[m.tracker.CursorTrack].Envelope1 = msg.Envelope
		case Envelope2EditMode:
			m.env2().Envelope = msg.Envelope
			m.tracker.Tracks[m.tracker.CursorTrack].Envelope2 = msg.Envelope
		}
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

	case ui.InstrumentApplied:
		m.osc1().Oscillator = msg.Instrument.Oscillator1
		m.env1().Envelope = msg.Instrument.Envelope1
		m.osc2().Oscillator = msg.Instrument.Oscillator2
		m.env2().Envelope = msg.Instrument.Envelope2
		m.mixer().Mixer = msg.Instrument.Mixer

		track := &m.tracker.Tracks[m.tracker.CursorTrack]
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

// playNote plays a note at the given frequency using the current oscillator
func (m *model) playNote(note audio.Note) {
	// TODO: duration should be adjustable
	duration := time.Millisecond * 250

	oscillator1 := m.osc1().Oscillator
	envelope1 := m.env1().Envelope
	oscillator2 := m.osc2().Oscillator
	envelope2 := m.env2().Envelope
	mixer := m.mixer().Mixer

	if m.mode == InstrumentMode {
		if preset := m.instr().GetPreset(m.instr().SelectedPreset); preset != nil {
			oscillator1 = preset.Oscillator1
			envelope1 = preset.Envelope1
			oscillator2 = preset.Oscillator2
			envelope2 = preset.Envelope2
			mixer = preset.Mixer
		}
	}

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
			m.osc1().Oscillator,
			m.env1().Envelope,
			m.osc2().Oscillator,
			m.env2().Envelope,
			m.mixer().Mixer,
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
func (m model) View() tea.View {
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

	trackerViewWithBorder := ui.RenderPanel("Tracker", ui.ColorAccentPrimary, trackerView, m.mode == TrackMode)
	body := lipgloss.JoinVertical(lipgloss.Left, synthView, trackerViewWithBorder)

	// Footer help
	footer := helpStyle.Render("↑↓←→: Navigate | J: Jump | 1-7: Notes | Shift+1-6: Sharp Notes | +/-: Octave | [/]: Volume | W: Oscillator | E: Envelope | T: Track | p: Play/Pause | P: Loop | S: Save | L: Load | Q: Quit")

	v := tea.NewView(header.String() + body + "\n" + footer)
	v.AltScreen = true
	return v
}

func (m model) synthView() string {
	// Pre-render child views to compute MaxHeight for the instrument panel
	childViews := make([]string, len(m.synthPanels))
	for i, p := range m.synthPanels {
		if i != 5 {
			childViews[i] = p.Child.View()
		}
	}
	maxH := 0
	for i, v := range childViews {
		if i != 5 {
			maxH = max(maxH, lipgloss.Height(v))
		}
	}
	m.instr().MaxHeight = maxH
	m.mixer().GlobalVolume = m.globalVolume
	childViews[5] = m.instr().View()

	panelViews := make([]string, len(m.synthPanels))
	for i, p := range m.synthPanels {
		active := m.mode != TrackMode && i == int(m.mode)-1
		panelViews[i] = ui.RenderPanel(p.Title, p.Color, childViews[i], active)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, panelViews...)
}

func maxInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}

	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}

	return maxValue
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
			synthPanels: []ui.Panel{
				ui.NewPanel("Oscillator 1", ui.ColorAccentOscillator, ui.NewOscillatorModel(selectedStyle, track.Oscillator1)),
				ui.NewPanel("Envelope 1", ui.ColorAccentEnvelope, ui.NewEnvelopeModel(selectedStyle, track.Envelope1)),
				ui.NewPanel("Oscillator 2", ui.ColorAccentOscillator, ui.NewOscillatorModel(selectedStyle, track.Oscillator2)),
				ui.NewPanel("Envelope 2", ui.ColorAccentEnvelope, ui.NewEnvelopeModel(selectedStyle, track.Envelope2)),
				ui.NewPanel("Mixer", ui.ColorAccentModulation, ui.NewMixer(track.Mixer.Balance)),
				ui.NewPanel("Instruments", ui.ColorAccentInstrument, ui.NewInstrumentView(selectedStyle)),
			},
			tracker:      tracker,
			mode:         TrackMode,
			octave:       4,
			globalVolume: 1.0,
		},
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
