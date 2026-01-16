package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
)

// SavedTrackRow is the YAML-serializable form of TrackRow
type SavedTrackRow struct {
	Note   string `yaml:"note"`
	Volume int    `yaml:"volume"`
	Effect string `yaml:"effect"`
}

// SavedTrack is the YAML-serializable form of Track
type SavedTrack struct {
	Oscillator1 string          `yaml:"oscillator1"`
	Envelope1   audio.Envelope  `yaml:"envelope1"`
	Oscillator2 string          `yaml:"oscillator2"`
	Envelope2   audio.Envelope  `yaml:"envelope2"`
	Mixer       float64         `yaml:"mixer"`
	Rows        []SavedTrackRow `yaml:"rows"`
}

// SavedSong is the complete song structure for YAML serialization
type SavedSong struct {
	NumRows   int          `yaml:"num_rows"`
	NumTracks int          `yaml:"num_tracks"`
	Tracks    []SavedTrack `yaml:"tracks"`
}

// InputMode represents the current input mode
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
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00e5ff")).
			Background(lipgloss.Color("#0f0f0f")).
			Padding(0, 1)

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
	synth       *audio.Synth
	oscillator1 ui.OscillatorModel
	envelope1   *ui.EnvelopeModel
	oscillator2 ui.OscillatorModel
	envelope2   *ui.EnvelopeModel
	mixer       ui.Mixer
	tracker     *ui.TrackerModel

	mode InputMode

	octave       int
	globalVolume float64

	// file dialog state
	fileDialogMode  int // 0: none, 1: save, 2: load
	fileDialogInput string
	fileDialogError string
	// current loaded/saved filename (prefill on save)
	currentFilename string
}

// tickMsg is sent to advance playback
type tickMsg time.Time

// Base note data (octave-agnostic)
var noteBaseFrequencies = map[string]float64{
	"C": 261.63, // C4
	"D": 293.66, // D4
	"E": 329.63, // E4
	"F": 349.23, // F4
	"G": 392.00, // G4
	"A": 440.00, // A4
	"B": 493.88, // B4
}

var noteKeyToName = map[string]string{
	"1": "C",
	"2": "D",
	"3": "E",
	"4": "F",
	"5": "G",
	"6": "A",
	"7": "B",
}

// tracksToSong converts the runtime Pattern to a SavedSong for YAML serialization
func tracksToSong(p *ui.TrackerModel) SavedSong {
	saved := SavedSong{
		NumRows:   p.NumRows,
		NumTracks: p.NumTracks,
		Tracks:    make([]SavedTrack, p.NumTracks),
	}

	for i, track := range p.Tracks {
		rows := make([]SavedTrackRow, len(track.Rows))
		for j, row := range track.Rows {
			rows[j] = SavedTrackRow{
				Note:   row.Note,
				Volume: row.Volume,
				Effect: row.Effect,
			}
		}
		saved.Tracks[i] = SavedTrack{
			Oscillator1: string(track.Oscillator1),
			Envelope1:   track.Envelope1,
			Oscillator2: string(track.Oscillator2),
			Envelope2:   track.Envelope2,
			Mixer:       track.Mixer,
			Rows:        rows,
		}
	}
	return saved
}

// TODO: This should NOT create a new model but new tracks inside the tracker model!

// songToTracks converts a SavedSong back to a runtime Pattern
func songToTracks(saved SavedSong) *ui.TrackerModel {
	p := ui.NewTracker(saved.NumTracks, saved.NumRows, 0, 0)
	for i, savedTrack := range saved.Tracks {
		track := &p.Tracks[i]
		track.Oscillator1 = audio.OscillatorType(savedTrack.Oscillator1)
		track.Envelope1 = savedTrack.Envelope1
		track.Oscillator2 = audio.OscillatorType(savedTrack.Oscillator2)
		track.Envelope2 = savedTrack.Envelope2
		track.Mixer = savedTrack.Mixer
		for j, row := range savedTrack.Rows {
			if j < len(track.Rows) {
				track.Rows[j] = ui.TrackRow{
					Note:   row.Note,
					Volume: row.Volume,
					Effect: row.Effect,
				}
			}
		}
	}
	return p
}

// saveSongToFile writes the pattern as YAML
func saveSongToFile(p *ui.TrackerModel, filename string) error {
	song := tracksToSong(p)
	data, err := yaml.Marshal(song)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// loadSongFromFile reads a YAML file and returns a Pattern
func loadSongFromFile(filename string) (*ui.TrackerModel, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var saved SavedSong
	err = yaml.Unmarshal(data, &saved)
	if err != nil {
		return nil, err
	}
	return songToTracks(saved), nil
}

// Init initializes the application
func (m model) Init() tea.Cmd {
	// Initialize speaker with sample rate
	sampleRate := m.synth.SampleRate
	buffersize := sampleRate.N(time.Millisecond * 250)

	speaker.Init(sampleRate, buffersize)

	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle file dialog input first
		if m.fileDialogMode > 0 {
			switch msg.String() {
			case "enter":
				if m.fileDialogMode == 1 {
					// Save
					filename := m.fileDialogInput
					if !strings.HasSuffix(filename, ".yaml") {
						filename += ".yaml"
					}
					err := saveSongToFile(m.tracker, filename)
					if err != nil {
						m.fileDialogError = fmt.Sprintf("Save failed: %v", err)
					} else {
						m.currentFilename = filename
						m.fileDialogMode = 0
						m.fileDialogInput = ""
						m.fileDialogError = ""
					}
				} else if m.fileDialogMode == 2 {
					// Load
					filename := m.fileDialogInput
					if !strings.HasSuffix(filename, ".yaml") {
						filename += ".yaml"
					}
					p, err := loadSongFromFile(filename)
					if err != nil {
						m.fileDialogError = fmt.Sprintf("Load failed: %v", err)
					} else {
						// TODO: This should NOT create a new model but new tracks inside the tracker model!
						m.tracker = p
						m.currentFilename = filename
						m.fileDialogMode = 0
						m.fileDialogInput = ""
						m.fileDialogError = ""
					}
				}
				return m, nil
			case "esc":
				m.fileDialogMode = 0
				m.fileDialogInput = ""
				m.fileDialogError = ""
				return m, nil
			case "backspace":
				if len(m.fileDialogInput) > 0 {
					m.fileDialogInput = m.fileDialogInput[:len(m.fileDialogInput)-1]
				}
				return m, nil
			default:
				// Type into the input field
				if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
					m.fileDialogInput += msg.String()
				}
				return m, nil
			}
		}

		// Global mode switching
		switch keyStr := msg.String(); keyStr {
		case "s":
			// Open save dialog
			m.fileDialogMode = 1
			if m.currentFilename != "" {
				m.fileDialogInput = m.currentFilename
			} else {
				m.fileDialogInput = "song"
			}
			m.fileDialogError = ""
			return m, nil
		case "l":
			// Open load dialog
			m.fileDialogMode = 2
			m.fileDialogInput = ""
			m.fileDialogError = ""
			return m, nil
		case "o":
			if m.mode == Oscillator1EditMode {
				m.mode = Oscillator2EditMode
			} else if m.mode == Oscillator2EditMode {
				m.mode = Oscillator1EditMode
			} else {
				m.mode = Oscillator1EditMode
			}

			return m, nil
		case "t":
			m.mode = TrackMode
			return m, nil
		case "e":
			if m.mode == Envelope1EditMode {
				m.mode = Envelope2EditMode
			} else if m.mode == Envelope2EditMode {
				m.mode = Envelope1EditMode
			} else {
				m.mode = Envelope1EditMode
			}

			return m, nil
		case "+":
			// change octave for current note
			if m.mode == TrackMode {
				note := m.tracker.CurrentTrack().CurrentRow().Note
				if note != "---" && note != "" {
					if newNote, freq, ok := changeNoteOctave(note, 1); ok {
						trackRow := m.tracker.CurrentTrack().CurrentRow()
						trackRow.Note = newNote

						m.playNote(freq)
						return m, nil
					}
				}
			}

			if m.octave < maxOctave {
				m.octave++
			}
			return m, nil
		case "-":
			// change octave for current note
			if m.mode == TrackMode {
				note := m.tracker.CurrentTrack().CurrentRow().Note
				if note != "---" && note != "" {
					if newNote, freq, ok := changeNoteOctave(note, -1); ok {
						trackRow := m.tracker.CurrentTrack().CurrentRow()
						trackRow.Note = newNote
						m.playNote(freq)
						return m, nil
					}
				}
			}

			if m.octave > minOctave {
				m.octave--
			}
			return m, nil
		// volume
		case "[", "alt+[": // decrease volume, for german keyboard layout we need to consider the alt+combo
			m.globalVolume -= 0.05
			if m.globalVolume < 0.0 {
				m.globalVolume = 0.0
			}

			// TODO: Refactor to avoid syncing state with models
			m.mixer.GlobalVolume = m.globalVolume
			return m, nil
		case "]", "alt+]": // increase volume, for german keyboard layout we need to consider the alt+combo
			m.globalVolume += 0.05
			if m.globalVolume > 1.0 {
				m.globalVolume = 1.0
			}

			// TODO: Refactor to avoid syncing state with models
			m.mixer.GlobalVolume = m.globalVolume
			return m, nil
		// TODO: Use int based logic with modes for cycling!
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
			freq := noteFrequency(base, m.octave)
			m.playNote(freq)

			return m, nil
		}

		if m.mode == Envelope1EditMode {
			m.envelope1.Update(msg)
			track := m.tracker.CurrentTrack()
			track.Envelope1 = audio.Envelope{
				Attack:  m.envelope1.Attack.Value,
				Decay:   m.envelope1.Decay.Value,
				Sustain: m.envelope1.Sustain.Value,
				Release: m.envelope1.Release.Value,
			}

			return m, nil
		}

		if m.mode == Envelope2EditMode {
			m.envelope2.Update(msg)
			track := m.tracker.CurrentTrack()
			track.Envelope2 = audio.Envelope{
				Attack:  m.envelope2.Attack.Value,
				Decay:   m.envelope2.Decay.Value,
				Sustain: m.envelope2.Sustain.Value,
				Release: m.envelope2.Release.Value,
			}

			return m, nil
		}

		// Handle oscillator edit mode
		if m.mode == Oscillator1EditMode {
			switch msg.String() {
			case "esc":
				m.mode = TrackMode
				return m, nil
			}

			m.oscillator1 = m.oscillator1.Update(msg)
			// TODO: This seems weird - Explose if passing an OnChange callback be better?
			track := m.tracker.CurrentTrack()
			track.Oscillator1 = m.oscillator1.Oscillator

			return m, nil
		}

		if m.mode == Oscillator2EditMode {
			switch msg.String() {
			case "esc":
				m.mode = TrackMode
				return m, nil
			}

			m.oscillator2 = m.oscillator2.Update(msg)
			// TODO: This seems weird - Explose if passing an OnChange callback be better?
			track := m.tracker.CurrentTrack()
			track.Oscillator2 = m.oscillator2.Oscillator

			return m, nil
		}

		if m.mode == MixerEditMode {
			m.mixer.Update(msg)
			track := m.tracker.CurrentTrack()
			track.Mixer = m.mixer.MixBalance

			return m, nil
		}

		if m.mode == TrackMode {
			m.tracker.Update(msg)

			return m, nil
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

		synthViewHeight := lipgloss.Height(m.synthView())

		m.tracker.ViewportHeight = m.height - synthViewHeight - 4
		m.tracker.ViewportWidth = m.width

		return m, nil
	}

	return m, nil
}

// tick returns a command that sends a tickMsg after a delay
func (m *model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*250, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// playRowNotes plays all notes in the specified row across all tracks
func (m *model) playRowNotes(row int) {
	if row < 0 || row >= m.tracker.NumRows {
		return
	}

	var generators []beep.Streamer

	// Collect all note generators for this row
	for trackIdx := 0; trackIdx < m.tracker.NumTracks; trackIdx++ {
		trackRow := m.tracker.Tracks[trackIdx].Rows[row]

		// Skip empty notes
		if trackRow.Note == "---" || trackRow.Note == "" {
			continue
		}

		// Parse note to frequency (simple mapping for now)
		freq := m.noteToFrequency(trackRow.Note)
		if freq > 0 {
			inst := m.tracker.Tracks[trackIdx].Oscillator1
			gen := m.synth.NewOscillator(inst, freq)
			generators = append(generators, gen)
		}
	}

	// If we have any notes to play, mix and play them
	if len(generators) > 0 {
		mixed := beep.Mix(generators...)

		// global vol
		volumeAdjusted := &effects.Volume{
			Streamer: mixed,
			Base:     2,
			Volume:   volumeToDecibels(m.globalVolume),
			Silent:   m.globalVolume == 0,
		}

		duration := beep.SampleRate(44100).N(time.Millisecond * 150)
		limited := beep.Take(duration, volumeAdjusted)

		//speaker.Clear()
		speaker.Play(limited)
	}
}

func volumeToDecibels(volume float64) float64 {
	if volume <= 0 {
		return -999
	}
	return math.Log2(volume) * 6
}

func changeNoteOctave(note string, delta int) (string, float64, bool) {
	parts := strings.Split(note, "-")
	if len(parts) != 2 {
		return "", 0, false
	}
	base := parts[0]
	octave, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, false
	}

	newOctave := octave + delta
	if newOctave < minOctave || newOctave > maxOctave {
		return "", 0, false
	}

	newNote := fmt.Sprintf("%s-%d", base, newOctave)
	freq := noteFrequency(base, newOctave)
	return newNote, freq, true
}

// noteFrequency returns the frequency for a base note name (C..B) at an octave.
func noteFrequency(base string, octave int) float64 {
	baseFreq, ok := noteBaseFrequencies[base]
	if !ok {
		return 0
	}
	// Reference octave is 4 in noteBaseFrequencies
	offset := float64(octave - 4)
	return baseFreq * math.Pow(2, offset)
}

// noteToFrequency converts a note name like "C-4" to frequency.
func (m *model) noteToFrequency(note string) float64 {
	parts := strings.Split(note, "-")
	if len(parts) != 2 {
		return 0
	}
	base := parts[0]
	oct, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return noteFrequency(base, oct)
}

// playNote plays a note at the given frequency using the current oscillator
func (m *model) playNote(frequency float64) {
	// TODO: This is synth arrangement coupled with playback functionality. Refactor to a synth method in audio that takes oscillator type, envelope, frequency and combines this into a playable note or streamer?
	oscillatorType1 := m.tracker.CurrentTrack().Oscillator1
	oscillator1 := m.synth.NewOscillator(oscillatorType1, frequency)
	envelope1 := m.tracker.CurrentTrack().Envelope1

	oscillatorType2 := m.tracker.CurrentTrack().Oscillator2
	oscillator2 := m.synth.NewOscillator(oscillatorType2, frequency)
	envelope2 := m.tracker.CurrentTrack().Envelope2

	duration := m.synth.SampleRate.N(time.Millisecond * 250)

	streamer1 := m.synth.NewADSREnvelope(
		oscillator1,
		duration, audio.Envelope{
			Attack:  envelope1.Attack,
			Decay:   envelope1.Decay,
			Sustain: envelope1.Sustain,
			Release: envelope1.Release,
		},
	)

	streamer2 := m.synth.NewADSREnvelope(
		oscillator2,
		duration, audio.Envelope{
			Attack:  envelope2.Attack,
			Decay:   envelope2.Decay,
			Sustain: envelope2.Sustain,
			Release: envelope2.Release,
		},
	)

	v := (m.mixer.MixBalance - 0.5) * 2 // Scale to -1 to 1
	mix1 := &effects.Volume{Streamer: streamer1, Base: 2, Volume: -v, Silent: v >= 1}
	mix2 := &effects.Volume{Streamer: streamer2, Base: 2, Volume: v, Silent: v <= -1}

	mixed := beep.Mix(mix1, mix2)

	volumeAdjusted := &effects.Volume{
		Streamer: mixed,
		Base:     2,
		Volume:   volumeToDecibels(m.globalVolume),
		Silent:   m.globalVolume == 0,
	}

	// Take only 0.3 seconds of the generated tone
	limited := beep.Take(duration, volumeAdjusted)

	// Clear previous sound and play the new note
	//speaker.Clear()
	speaker.Play(limited)
}

// View renders the UI
func (m model) View() string {
	// Build header
	var header strings.Builder
	header.WriteString(titleStyle.Render("TeTrackT - Music Tracker"))
	header.WriteString("\n")

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

	instView := m.synthView()
	trackView := m.tracker.View()

	// Apply border to track view with conditional highlighting
	trackBorder := panelBorderStyle
	if m.mode == TrackMode {
		trackBorder = activePanelBorderStyle
	}
	trackViewWithBorder := trackBorder.Render(trackView)

	body := lipgloss.JoinVertical(lipgloss.Left, instView, trackViewWithBorder)

	// Footer help
	footer := helpStyle.Render("↑↓←→: Navigate | J: Jump | 1-7: Notes | +/-: Octave | [/]: Volume | W: Oscillator | E: Envelope | T: Track | p: Play/Pause | P: Loop | S: Save | L: Load | Q: Quit")

	// TODO: More generic modal handling
	if m.envelope1.ShowModal && m.mode == Envelope1EditMode {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.envelope1.View())
	}

	if m.envelope2.ShowModal && m.mode == Envelope2EditMode {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.envelope2.View())
	}

	// File dialog modal
	if m.fileDialogMode > 0 {
		dialogTitle := "Save Song"
		if m.fileDialogMode == 2 {
			dialogTitle = "Load Song"
		}
		dialogContent := fmt.Sprintf("%s\n\nFilename: %s_\n\n", dialogTitle, m.fileDialogInput)
		if m.fileDialogError != "" {
			dialogContent += fmt.Sprintf("Error: %s\n", m.fileDialogError)
		}
		dialogContent += "[Enter to confirm, Esc to cancel]"
		modalView := modalBorderStyle.Render(dialogContent)
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
	synth := audio.NewSynth(sampleRate)

	// Create pattern with 8 tracks and 64 rows
	tracker := ui.NewTracker(8, 64, 0, 0)
	track := tracker.CurrentTrack()

	p := tea.NewProgram(
		model{
			synth:        synth,
			oscillator1:  ui.NewOscillatorModel(selectedStyle, track.Oscillator1),
			envelope1:    ui.NewEnvelopeModel(selectedStyle, track.Envelope1),
			oscillator2:  ui.NewOscillatorModel(selectedStyle, track.Oscillator2),
			envelope2:    ui.NewEnvelopeModel(selectedStyle, track.Envelope2),
			mixer:        ui.NewMixer(track.Mixer),
			tracker:      tracker,
			mode:         TrackMode,
			octave:       4,
			globalVolume: 1.0,
		},

		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
