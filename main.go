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
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
)

// Track represents a single track in the pattern
type Track struct {
	number      int
	oscillator1 audio.OscillatorType
	envelope1   audio.Envelope
	oscillator2 audio.OscillatorType
	envelope2   audio.Envelope
	mixer       float64
	rows        []TrackRow
}

// TrackRow represents a single row in a track
type TrackRow struct {
	note   string // e.g., "C-4", "D#5", "---" for empty
	volume int    // 0-64
	effect string // effect command
}

// Pattern represents the pattern editor with multiple tracks
type Pattern struct {
	tracks    []Track
	numRows   int
	numTracks int
}

// NewPattern creates a new pattern with the specified number of tracks and rows
func NewPattern(numTracks, numRows int) *Pattern {
	tracks := make([]Track, numTracks)
	for i := range numTracks {
		tracks[i] = Track{
			number:      i,
			oscillator1: audio.Sine,
			envelope1:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
			oscillator2: audio.Sine,
			envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
			rows:        make([]TrackRow, numRows),
		}
		// Initialize all rows with empty data
		for j := range numRows {
			tracks[i].rows[j] = TrackRow{
				note:   "---",
				volume: 0,
				effect: "---",
			}
		}
	}
	return &Pattern{
		tracks:    tracks,
		numRows:   numRows,
		numTracks: numTracks,
	}
}

// InputMode represents the current input mode
type InputMode int

const (
	TrackMode InputMode = iota
	Envelope1EditMode
	Oscillator1EditMode
	Envelope2EditMode
	Oscillator2EditMode
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

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ffff")).
			Padding(0, 1)

	rowNumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	cursorRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff9800")).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#d81b60")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true)

	defaultStyle = lipgloss.NewStyle()

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	cursorCellStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a2a")).
			Foreground(lipgloss.Color("#00e5ff")).
			Padding(0, 1)

	playbackRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ffff")).
				Bold(true)

	panelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#333333")).
				Padding(1, 2)

	activePanelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00e5ff")).
				Bold(true).
				Padding(1, 2)

	modalBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#ff9800")).
				Padding(1, 2)
)

const (
	minOctave = 1
	maxOctave = 8
)

// model represents the application state
type model struct {
	width        int
	height       int
	synth        *audio.Synth
	oscillator1  ui.OscillatorModel
	envelope1    *ui.EnvelopeModel
	oscillator2  ui.OscillatorModel
	envelope2    *ui.EnvelopeModel
	mixer        ui.Mixer
	pattern      *Pattern
	cursorTrack  int
	cursorRow    int
	viewportRow  int // Top row visible in the viewport
	mode         InputMode
	jumpInput    string
	isPlaying    bool
	playbackRow  int
	octave       int
	globalVolume float64 
	// loop-to-row mode: loops rows 0..loopEndRow (inclusive)
	loopToRow  bool
	loopEndRow int
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

// Init initializes the application
func (m model) Init() tea.Cmd {
	// Initialize speaker with sample rate
	sampleRate := m.synth.SampleRate
	buffersize := sampleRate.N(time.Millisecond * 300)

	speaker.Init(sampleRate, buffersize)

	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global mode switching
		switch msg.String() {
		case "w":
			m.mode = Oscillator1EditMode
			return m, nil
		case "t":
			m.mode = TrackMode
			return m, nil
		case "e":
			m.mode = Envelope1EditMode
			return m, nil
		case "+":
			// change octave for current note
			if m.mode == TrackMode {
				note := m.pattern.tracks[m.cursorTrack].rows[m.cursorRow].note
				if note != "---" && note != "" {
					if newNote, freq, ok := changeNoteOctave(note, 1); ok {
						m.pattern.tracks[m.cursorTrack].rows[m.cursorRow].note = newNote
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
				note := m.pattern.tracks[m.cursorTrack].rows[m.cursorRow].note
				if note != "---" && note != "" {
					if newNote, freq, ok := changeNoteOctave(note, -1); ok {
						m.pattern.tracks[m.cursorTrack].rows[m.cursorRow].note = newNote
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
		case "[":
			if m.globalVolume > 0.0 {
				m.globalVolume -= 0.05
				if m.globalVolume < 0.0 {
					m.globalVolume = 0.0
				}
			}
			return m, nil
		case "]":
			if m.globalVolume < 1.0 {
				m.globalVolume += 0.05
				if m.globalVolume > 1.0 {
					m.globalVolume = 1.0
				}
			}
			return m, nil
		// TODO: Use int based logic with modes for cycling!
		case "tab":
			// Cycle through Oscillator, Envelope, Track modes
			switch m.mode {
			case Oscillator1EditMode:
				m.mode = Envelope1EditMode
			case Envelope1EditMode:
				m.mode = Oscillator2EditMode
			case Oscillator2EditMode:
				m.mode = Envelope2EditMode
			case Envelope2EditMode:
				m.mode = MixerEditMode
			case MixerEditMode:
				m.mode = TrackMode
			default:
				m.mode = Oscillator1EditMode
			}
			return m, nil
		case "shift+tab":
			// Reverse cycle through Oscillator, Envelope, Track modes
			switch m.mode {
			case Oscillator1EditMode:
				m.mode = TrackMode
			case Envelope1EditMode:
				m.mode = Oscillator1EditMode
			case Oscillator2EditMode:
				m.mode = Envelope1EditMode
			case Envelope2EditMode:
				m.mode = Oscillator2EditMode
			case MixerEditMode:
				m.mode = Envelope2EditMode
			default:
				m.mode = Envelope1EditMode
			}
			return m, nil
		case "q", "ctrl+c":
			speaker.Clear()
			return m, tea.Quit
		}

		// Global note playing (available in any mode)
		if base, ok := noteKeyToName[msg.String()]; ok {
			noteName := fmt.Sprintf("%s-%d", base, m.octave)
			freq := noteFrequency(base, m.octave)
			m.playNote(freq)
			// Also set note in track if in track mode
			if m.mode == TrackMode {
				m.pattern.tracks[m.cursorTrack].rows[m.cursorRow].note = noteName
			}
			return m, nil
		}

		if m.mode == Envelope1EditMode {
			m.envelope1.Update(msg)
			track := &m.pattern.tracks[m.cursorTrack]
			track.envelope1 = audio.Envelope{
				Attack:  m.envelope1.Attack.Value,
				Decay:   m.envelope1.Decay.Value,
				Sustain: m.envelope1.Sustain.Value,
				Release: m.envelope1.Release.Value,
			}

			return m, nil
		}

		if m.mode == Envelope2EditMode {
			m.envelope2.Update(msg)
			track := &m.pattern.tracks[m.cursorTrack]
			track.envelope2 = audio.Envelope{
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
			track := &m.pattern.tracks[m.cursorTrack]
			track.oscillator1 = m.oscillator1.Oscillator

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
			track := &m.pattern.tracks[m.cursorTrack]
			track.oscillator2 = m.oscillator2.Oscillator

			return m, nil
		}

		if m.mode == MixerEditMode {
			m.mixer.Update(msg)
			track := &m.pattern.tracks[m.cursorTrack]
			track.mixer = m.mixer.MixBalance.Value

			return m, nil
		}

		keyStr := msg.String()

		// Track mode key handling
		switch keyStr {
		case "p", "P":
			// Toggle play/pause
			m.isPlaying = !m.isPlaying
			m.loopToRow = false // normal play toggles off loop mode
			if m.isPlaying {
				m.playbackRow = 0

				if "P" == keyStr {
					m.loopToRow = true
					m.loopEndRow = m.cursorRow
				}
				return m, m.tick()
			} else {
				//speaker.Clear()
			}
		case "e":
			// Enter envelope edit mode
			m.mode = Envelope1EditMode
		case "left":
			// Move cursor left (previous track)
			if m.cursorTrack > 0 {
				m.cursorTrack--
				m.oscillator1.Oscillator = m.pattern.tracks[m.cursorTrack].oscillator1
			}
		case "right":
			// Move cursor right (next track)
			if m.cursorTrack < m.pattern.numTracks-1 {
				m.cursorTrack++
				m.oscillator1.Oscillator = m.pattern.tracks[m.cursorTrack].oscillator1
			}
		case "up":
			// Move cursor up (previous row)
			if m.cursorRow > 0 {
				m.cursorRow--
				// Adjust viewport if needed
				if m.cursorRow < m.viewportRow {
					m.viewportRow = m.cursorRow
				}
			}
		case "down":
			// Move cursor down (next row)
			if m.cursorRow < m.pattern.numRows-1 {
				m.cursorRow++
				// Adjust viewport if needed
				instrumentHeight := m.instrumentHeight()
				visibleRows := m.visibleRows(instrumentHeight)
				if m.cursorRow >= m.viewportRow+visibleRows {
					m.viewportRow = m.cursorRow - visibleRows + 1
				}
			}
		case "home":
			// Jump to first row
			m.cursorRow = 0
			m.viewportRow = 0
		case "end":
			// Jump to last row
			m.cursorRow = m.pattern.numRows - 1
			visibleRows := m.visibleRows(m.instrumentHeight())
			m.viewportRow = m.pattern.numRows - visibleRows
			if m.viewportRow < 0 {
				m.viewportRow = 0
			}
		}

	case tickMsg:
		if !m.isPlaying {
			return m, nil
		}

		// Play all notes at current playback row
		m.playRowNotes(m.playbackRow)

		// Advance to next row
		m.playbackRow++
		if m.loopToRow {
			// Wrap within 0..loopEndRow inclusive
			if m.playbackRow > m.loopEndRow {
				m.playbackRow = 0
			}
		} else {
			if m.playbackRow >= m.pattern.numRows {
				m.playbackRow = 0 // Loop back to start
			}
		}

		// Schedule next tick
		return m, m.tick()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// tick returns a command that sends a tickMsg after a delay
func (m *model) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// playRowNotes plays all notes in the specified row across all tracks
func (m *model) playRowNotes(row int) {
	if row < 0 || row >= m.pattern.numRows {
		return
	}

	var generators []beep.Streamer

	// Collect all note generators for this row
	for trackIdx := 0; trackIdx < m.pattern.numTracks; trackIdx++ {
		trackRow := m.pattern.tracks[trackIdx].rows[row]

		// Skip empty notes
		if trackRow.note == "---" || trackRow.note == "" {
			continue
		}

		// Parse note to frequency (simple mapping for now)
		freq := m.noteToFrequency(trackRow.note)
		if freq > 0 {
			inst := m.pattern.tracks[trackIdx].oscillator1
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
	oscillatorType1 := m.pattern.tracks[m.cursorTrack].oscillator1
	oscillator1 := m.synth.NewOscillator(oscillatorType1, frequency)
	envelope1 := m.pattern.tracks[m.cursorTrack].envelope1

	oscillatorType2 := m.pattern.tracks[m.cursorTrack].oscillator2
	oscillator2 := m.synth.NewOscillator(oscillatorType2, frequency)
	envelope2 := m.pattern.tracks[m.cursorTrack].envelope2

	duration := m.synth.SampleRate.N(time.Millisecond * 300)

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

	v := (m.mixer.MixBalance.Value - 0.5) * 2 // Scale to -1 to 1
	mix1 := &effects.Volume{Streamer: streamer1, Base: 2, Volume: -v, Silent: v == 1}
	mix2 := &effects.Volume{Streamer: streamer2, Base: 2, Volume: v, Silent: v == -1}

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
	case Oscillator1EditMode:
		modeStr = "OSCILLATOR1"
	}

	playStatus := "STOPPED"
	if m.isPlaying {
		if m.loopToRow {
			playStatus = fmt.Sprintf("LOOP 0-%d (Row %d)", m.loopEndRow, m.playbackRow)
		} else {
			playStatus = fmt.Sprintf("PLAYING (Row %d)", m.playbackRow)
		}
	}
	currentInst := m.pattern.tracks[m.cursorTrack].oscillator1
	header.WriteString(infoStyle.Render(fmt.Sprintf("Oscillator: %s | Instrument: %s | Mode: %s | %s | Track: %d | Row: %d | Octave: %d",
		m.oscillator1.Oscillator, currentInst, modeStr, playStatus, m.cursorTrack, m.cursorRow, m.octave)))
	header.WriteString("\n\n")

	instView := m.synthView()
	trackView := m.trackView()

	// Apply border to track view with conditional highlighting
	trackBorder := panelBorderStyle
	if m.mode == TrackMode {
		trackBorder = activePanelBorderStyle
	}
	trackViewWithBorder := trackBorder.Render(trackView)

	body := lipgloss.JoinVertical(lipgloss.Left, instView, trackViewWithBorder)

	// Footer help
	footer := helpStyle.Render("↑↓←→: Navigate | J: Jump | 1-7: Notes | +/-: Octave | [/]: Volume | W: Oscillator | E: Envelope | T: Track | p: Play/Pause | P: Loop | Q: Quit")

	// TODO: More generic modal handling
	if m.envelope1.ShowModal && m.mode == Envelope1EditMode {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.envelope1.View())
	}

	if m.envelope2.ShowModal && m.mode == Envelope2EditMode {
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.envelope2.View())
	}

	return header.String() + body + "\n" + footer
}

func (m model) trackView() string {
	// Track editor section
	var tracks strings.Builder

	// Track headers
	tracks.WriteString("    ") // Row number space
	for i := 0; i < m.pattern.numTracks; i++ {
		trackHeader := fmt.Sprintf("Track %d", i+1)
		if i == m.cursorTrack {
			trackHeader = headerStyle.Render(trackHeader)
		} else {
			trackHeader = headerStyle.Foreground(lipgloss.Color("#555555")).Render(trackHeader)
		}
		tracks.WriteString(trackHeader)
		tracks.WriteString("    ")
	}
	tracks.WriteString("\n")

	// Separator
	tracks.WriteString("    ")
	for i := 0; i < m.pattern.numTracks; i++ {
		tracks.WriteString(strings.Repeat("─", 10))
		tracks.WriteString("   ")
	}
	tracks.WriteString("\n")

	endRow := min(m.viewportRow+m.visibleRows(m.instrumentHeight()), m.pattern.numRows)

	// Render visible rows
	for row := m.viewportRow; row < endRow; row++ {
		// Row number with playback indicator
		rowNumStr := fmt.Sprintf("%02d ", row)
		if row == m.playbackRow && m.isPlaying {
			tracks.WriteString(playbackRowStyle.Render(rowNumStr))
		} else if row == m.cursorRow {
			tracks.WriteString(cursorRowStyle.Render(rowNumStr))
		} else {
			tracks.WriteString(rowNumStyle.Render(rowNumStr))
		}

		// Track cells
		for trackIdx := 0; trackIdx < m.pattern.numTracks; trackIdx++ {
			trackRow := m.pattern.tracks[trackIdx].rows[row]
			cellContent := fmt.Sprintf("%-3s %2s %3s", trackRow.note, formatVolume(trackRow.volume), trackRow.effect)

			if row == m.cursorRow && trackIdx == m.cursorTrack {
				tracks.WriteString(cursorCellStyle.Render(cellContent))
			} else {
				tracks.WriteString(cellStyle.Render(cellContent))
			}
			tracks.WriteString(" ")
		}
		tracks.WriteString("\n")
	}

	return tracks.String()
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

func (m model) instrumentHeight() int {
	instView := m.synthView()
	return countLines(instView) + 2 // +2 for panel border padding
}

func (m model) visibleRows(instrumentHeight int) int {
	available := m.height - instrumentHeight - 8 // Leave space for header and footer and borders
	if available < 5 {
		return 5
	}
	return available
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// formatVolume formats volume value for display
func formatVolume(volume int) string {
	if volume == 0 {
		return ".."
	}
	return fmt.Sprintf("%02d", volume)
}

func main() {
	// Initialize synthesizer
	sampleRate := beep.SampleRate(44100)
	synth := audio.NewSynth(sampleRate)

	// Create pattern with 8 tracks and 64 rows
	pattern := NewPattern(8, 64)

	cursorTrack := 0
	track := pattern.tracks[cursorTrack]

	p := tea.NewProgram(
		model{
			synth:        synth,
			oscillator1:  ui.NewOscillatorModel(selectedStyle, track.oscillator1),
			envelope1:    ui.NewEnvelopeModel(selectedStyle, track.envelope1),
			oscillator2:  ui.NewOscillatorModel(selectedStyle, track.oscillator2),
			envelope2:    ui.NewEnvelopeModel(selectedStyle, track.envelope2),
			mixer:        ui.NewMixer(),
			pattern:      pattern,
			cursorTrack:  cursorTrack,
			cursorRow:    0,
			viewportRow:  0,
			mode:         TrackMode,
			jumpInput:    "",
			isPlaying:    false,
			playbackRow:  0,
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
