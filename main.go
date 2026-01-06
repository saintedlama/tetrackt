package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tetrackt/tetrackt/audio"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

// Envelope represents ADSR envelope parameters
type Envelope struct {
	Attack  int // 0-100 (percentage of samples)
	Decay   int // 0-100 (percentage of samples)
	Sustain int // 0-100 (percentage of peak)
	Release int // 0-100 (percentage of samples)
}

// Instrument represents a complete instrument with waveform and envelope
type Instrument struct {
	Waveform audio.WaveformType
	Envelope Envelope
}

// NewInstrument creates a new instrument with default envelope
func NewInstrument(waveform audio.WaveformType) *Instrument {
	return &Instrument{
		Waveform: waveform,
		Envelope: Envelope{
			Attack:  0,
			Decay:   0,
			Sustain: 100,
			Release: 0,
		},
	}
}

// Track represents a single track in the pattern
type Track struct {
	number     int
	instrument *Instrument
	rows       []TrackRow
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
			number:     i,
			instrument: NewInstrument(audio.Sine),
			rows:       make([]TrackRow, numRows),
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
	JumpMode
	EnvelopeEditMode
	WaveformEditMode
)

// EnvelopeEditField represents which envelope parameter is being edited
type EnvelopeEditField int

const (
	EnvelopeAttack EnvelopeEditField = iota
	EnvelopeDecay
	EnvelopeSustain
	EnvelopeRelease
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(1, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Padding(0, 1)

	rowNumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))

	cursorRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("126")).
			Bold(true)

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	cursorCellStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("240")).
			Padding(0, 1)

	playbackRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true)

	panelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2)
)

// model represents the application state
type model struct {
	width             int
	height            int
	synth             *audio.Synth
	waveform          audio.WaveformType
	waveformList      []audio.WaveformType
	waveformIdx       int
	pattern           *Pattern
	cursorTrack       int
	cursorRow         int
	viewportRow       int // Top row visible in the viewport
	mode              InputMode
	jumpInput         string
	isPlaying         bool
	playbackRow       int
	envelopeField     EnvelopeEditField
	envelopeEditInput string
}

// tickMsg is sent to advance playback
type tickMsg time.Time

// noteFrequencies maps keys to frequencies for C major scale (C4 to B4)
var noteFrequencies = map[string]float64{
	"1": 261.63, // C4
	"2": 293.66, // D4
	"3": 329.63, // E4
	"4": 349.23, // F4
	"5": 392.00, // G4
	"6": 440.00, // A4
	"7": 493.88, // B4
}

// noteNames maps keys to note names for display in track editor
var noteNames = map[string]string{
	"1": "C-4",
	"2": "D-4",
	"3": "E-4",
	"4": "F-4",
	"5": "G-4",
	"6": "A-4",
	"7": "B-4",
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
			m.mode = WaveformEditMode
			return m, nil
		case "t":
			m.mode = TrackMode
			return m, nil
		case "e":
			m.mode = EnvelopeEditMode
			return m, nil
		}

		// Global note playing (available in any mode)
		if freq, ok := noteFrequencies[msg.String()]; ok {
			m.playNote(freq)
			// Also set note in track if in track mode
			if m.mode == TrackMode {
				if noteName, ok := noteNames[msg.String()]; ok {
					m.pattern.tracks[m.cursorTrack].rows[m.cursorRow].note = noteName
				}
			}
			return m, nil
		}

		// Handle envelope edit mode
		if m.mode == EnvelopeEditMode {
			switch msg.String() {
			case "up":
				// Move to previous envelope field
				m.envelopeField = (m.envelopeField - 1 + 4) % 4
			case "down":
				// Move to next envelope field
				m.envelopeField = (m.envelopeField + 1) % 4
			case "left":
				// Decrease value by 1
				m.adjustEnvelopeValue(-1)
			case "shift+left":
				// Decrease value by 10
				m.adjustEnvelopeValue(-10)
			case "right":
				// Increase value by 1
				m.adjustEnvelopeValue(1)
			case "shift+right":
				// Increase value by 10
				m.adjustEnvelopeValue(10)
			case "esc":
				m.mode = TrackMode
			}
			return m, nil
		}

		// Handle waveform edit mode
		if m.mode == WaveformEditMode {
			switch msg.String() {
			case "up", "left":
				// Move to previous waveform
				m.waveformIdx = (m.waveformIdx - 1 + len(m.waveformList)) % len(m.waveformList)
				m.waveform = m.waveformList[m.waveformIdx]
				m.setTrackInstrument(m.cursorTrack, m.waveform)
			case "down", "right":
				// Move to next waveform
				m.waveformIdx = (m.waveformIdx + 1) % len(m.waveformList)
				m.waveform = m.waveformList[m.waveformIdx]
				m.setTrackInstrument(m.cursorTrack, m.waveform)
			case "esc":
				m.mode = TrackMode
			}
			return m, nil
		}

		// Handle jump mode separately
		if m.mode == JumpMode {
			switch msg.String() {
			case "enter":
				// Parse jump input and jump to track
				if trackNum, err := strconv.Atoi(m.jumpInput); err == nil {
					if trackNum >= 0 && trackNum < m.pattern.numTracks {
						m.cursorTrack = trackNum
						m.waveform = m.ensureTrackInstrument(m.cursorTrack)
					}
				}
				m.mode = TrackMode
				m.jumpInput = ""
			case "esc":
				m.mode = TrackMode
				m.jumpInput = ""
			case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
				m.jumpInput += msg.String()
			case "backspace":
				if len(m.jumpInput) > 0 {
					m.jumpInput = m.jumpInput[:len(m.jumpInput)-1]
				}
			}
			return m, nil
		}

		// Track mode key handling
		switch msg.String() {
		case "q", "ctrl+c":
			speaker.Clear()
			return m, tea.Quit
		case "space", " ":
			// Toggle play/pause
			m.isPlaying = !m.isPlaying
			if m.isPlaying {
				m.playbackRow = 0
				return m, m.tick()
			} else {
				speaker.Clear()
			}
		case "j":
			// Enter jump mode
			m.mode = JumpMode
			m.jumpInput = ""
		case "e":
			// Enter envelope edit mode
			m.mode = EnvelopeEditMode
			m.envelopeField = EnvelopeAttack
			m.envelopeEditInput = ""
		case "left":
			// Move cursor left (previous track)
			if m.cursorTrack > 0 {
				m.cursorTrack--
				m.waveform = m.ensureTrackInstrument(m.cursorTrack)
			}
		case "right":
			// Move cursor right (next track)
			if m.cursorTrack < m.pattern.numTracks-1 {
				m.cursorTrack++
				m.waveform = m.ensureTrackInstrument(m.cursorTrack)
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
		if m.playbackRow >= m.pattern.numRows {
			m.playbackRow = 0 // Loop back to start
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
	return tea.Tick(time.Millisecond*150, func(t time.Time) tea.Msg {
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
			inst := m.ensureTrackInstrument(trackIdx)
			gen := m.synth.NewWaveform(inst, freq)
			generators = append(generators, gen)
		}
	}

	// If we have any notes to play, mix and play them
	if len(generators) > 0 {
		mixed := beep.Mix(generators...)
		duration := beep.SampleRate(44100).N(time.Millisecond * 150)
		limited := beep.Take(duration, mixed)

		speaker.Clear()
		speaker.Play(limited)
	}
}

// noteToFrequency converts a note name to frequency
func (m *model) noteToFrequency(note string) float64 {
	// Map note names to frequencies
	noteFreqMap := map[string]float64{
		"C-4": 261.63,
		"D-4": 293.66,
		"E-4": 329.63,
		"F-4": 349.23,
		"G-4": 392.00,
		"A-4": 440.00,
		"B-4": 493.88,
	}

	if freq, ok := noteFreqMap[note]; ok {
		return freq
	}
	return 0
}

// playNote plays a note at the given frequency using the current waveform
func (m *model) playNote(frequency float64) {
	inst := m.ensureTrackInstrument(m.cursorTrack)
	waveform := m.synth.NewWaveform(inst, frequency)

	// TODO: Calculate the number of samples for the note duration
	envelope := m.pattern.tracks[m.cursorTrack].instrument.Envelope

	duration := m.synth.SampleRate.N(time.Millisecond * 300)

	adsr := m.synth.NewADSREnvelope(
		waveform,
		duration,
		float64(envelope.Attack)/100.0,
		float64(envelope.Decay)/100.0,
		float64(envelope.Sustain)/100.0,
		float64(envelope.Release)/100.0,
	)

	// Take only 0.3 seconds of the generated tone

	limited := beep.Take(duration, adsr)

	// Clear previous sound and play the new note
	speaker.Clear()
	speaker.Play(limited)
}

// View renders the UI
func (m model) View() string {
	// Build header
	var header strings.Builder
	header.WriteString(titleStyle.Render("🎵 Tetrackt - Music Tracker"))
	header.WriteString("\n")

	modeStr := "NORMAL"
	if m.mode == JumpMode {
		modeStr = fmt.Sprintf("JUMP: %s", m.jumpInput)
	} else if m.mode == EnvelopeEditMode {
		modeStr = "ENVELOPE"
	} else if m.mode == WaveformEditMode {
		modeStr = "WAVEFORM"
	}
	playStatus := "STOPPED"
	if m.isPlaying {
		playStatus = fmt.Sprintf("PLAYING (Row %d)", m.playbackRow)
	}
	currentInst := m.trackInstrumentValue(m.cursorTrack)
	header.WriteString(infoStyle.Render(fmt.Sprintf("Waveform: %s | Instrument: %s | Mode: %s | %s | Track: %d | Row: %d",
		m.waveform, currentInst, modeStr, playStatus, m.cursorTrack, m.cursorRow)))
	header.WriteString("\n\n")

	instView := m.synthView()
	trackView := m.trackView()

	body := lipgloss.JoinVertical(lipgloss.Left, instView, trackView)

	// Footer help
	footer := helpStyle.Render("↑↓←→: Navigate | J: Jump | 1-7: Notes | W: Waveform (↑↓←→ select) | E: Envelope (↑↓ select, ←→ adjust) | T: Track | Space: Play/Pause | Q: Quit")

	return header.String() + body + "\n" + footer
}

func (m model) trackView() string {
	// Track editor section
	var tracks strings.Builder

	// Track headers
	tracks.WriteString("    ") // Row number space
	for i := 0; i < m.pattern.numTracks; i++ {
		trackHeader := fmt.Sprintf("Track %d", i)
		if i == m.cursorTrack {
			trackHeader = headerStyle.Render(trackHeader)
		} else {
			trackHeader = headerStyle.Copy().Foreground(lipgloss.Color("244")).Render(trackHeader)
		}
		tracks.WriteString(trackHeader)
		tracks.WriteString("  ")
	}
	tracks.WriteString("\n")

	// Separator
	tracks.WriteString("    ")
	for i := 0; i < m.pattern.numTracks; i++ {
		tracks.WriteString(strings.Repeat("─", 10))
		tracks.WriteString("  ")
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

func (m model) waveformView() string {
	var waveformView strings.Builder
	waveformView.WriteString("Waveform:\n")

	currentInst := m.trackInstrumentValue(m.cursorTrack)
	for _, wf := range m.waveformList {
		if wf == currentInst {
			waveformView.WriteString(selectedStyle.Render(fmt.Sprintf("%s", wf)))
			waveformView.WriteString("\n")
		} else {
			waveformView.WriteString(fmt.Sprintf("%s\n", wf))
		}
	}

	return waveformView.String()
}

// percentageToKnob converts a percentage value to a knob character
func percentageToKnob(percentage int) string {
	if percentage <= 25 {
		return "◔" // Quarter filled
	} else if percentage <= 50 {
		return "◗" // Half filled
	} else if percentage <= 75 {
		return "◕" // Three-quarter filled
	} else {
		return "●" // Fully filled
	}
}

// renderKnobLine renders a single knob line with label, knob, and percentage
func (m model) renderKnobLine(field EnvelopeEditField, value int, label string) string {
	knobChar := percentageToKnob(value)
	isSelected := m.mode == EnvelopeEditMode && m.envelopeField == field

	knobDisplay := fmt.Sprintf("%s: %s %3d%%", label, knobChar, value)

	if isSelected {
		return selectedStyle.Render(knobDisplay)
	}
	return knobDisplay
}

func (m model) envelopeView() string {
	// Show envelope
	env := m.pattern.tracks[m.cursorTrack].instrument.Envelope
	envText := "ADSR Envelope:\n"

	envText += m.renderKnobLine(EnvelopeAttack, env.Attack, "A") + "\n"
	envText += m.renderKnobLine(EnvelopeDecay, env.Decay, "D") + "\n"
	envText += m.renderKnobLine(EnvelopeSustain, env.Sustain, "S") + "\n"
	envText += m.renderKnobLine(EnvelopeRelease, env.Release, "R")

	return envText
}

func (m model) synthView() string {
	waveformView := m.waveformView()
	envelopeView := m.envelopeView()

	return lipgloss.JoinHorizontal(lipgloss.Top,
		panelBorderStyle.Render(waveformView),
		panelBorderStyle.Render(envelopeView),
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

// setTrackInstrument assigns an instrument waveform to a track
func (m *model) setTrackInstrument(trackIdx int, wf audio.WaveformType) {
	if trackIdx < 0 || trackIdx >= m.pattern.numTracks {
		return
	}
	m.pattern.tracks[trackIdx].instrument.Waveform = wf
}

// ensureTrackInstrument returns the track instrument waveform, defaulting to sine
func (m *model) ensureTrackInstrument(trackIdx int) audio.WaveformType {
	if trackIdx < 0 || trackIdx >= m.pattern.numTracks {
		return audio.Sine
	}
	track := &m.pattern.tracks[trackIdx]
	if track.instrument == nil {
		track.instrument = NewInstrument(audio.Sine)
	}
	return track.instrument.Waveform
}

// trackInstrumentValue returns the track instrument waveform without mutating
func (m model) trackInstrumentValue(trackIdx int) audio.WaveformType {
	if trackIdx < 0 || trackIdx >= m.pattern.numTracks {
		return audio.Sine
	}
	track := m.pattern.tracks[trackIdx]
	if track.instrument == nil {
		return audio.Sine
	}
	return track.instrument.Waveform
}

func main() {
	// Initialize synthesizer
	sampleRate := beep.SampleRate(44100)
	synth := audio.NewSynth(sampleRate)

	// Create list of available waveforms
	waveforms := []audio.WaveformType{
		audio.Sine,
		audio.Square,
		audio.Triangle,
		audio.Sawtooth,
		audio.SawtoothReverse,
		audio.Noise,
	}

	// Create pattern with 8 tracks and 64 rows
	pattern := NewPattern(8, 64)

	p := tea.NewProgram(
		model{
			synth:             synth,
			waveform:          audio.Sine,
			waveformList:      waveforms,
			waveformIdx:       0,
			pattern:           pattern,
			cursorTrack:       0,
			cursorRow:         0,
			viewportRow:       0,
			mode:              TrackMode,
			jumpInput:         "",
			isPlaying:         false,
			playbackRow:       0,
			envelopeField:     EnvelopeAttack,
			envelopeEditInput: "",
		},
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// adjustEnvelopeValue adjusts the current envelope field by a delta value
func (m *model) adjustEnvelopeValue(delta int) {
	env := &m.pattern.tracks[m.cursorTrack].instrument.Envelope
	var currentValue *int

	switch m.envelopeField {
	case EnvelopeAttack:
		currentValue = &env.Attack
	case EnvelopeDecay:
		currentValue = &env.Decay
	case EnvelopeSustain:
		currentValue = &env.Sustain
	case EnvelopeRelease:
		currentValue = &env.Release
	}

	if currentValue != nil {
		newValue := *currentValue + delta

		// For A, D, R: prevent increases that would make A+D+R exceed 100
		if m.envelopeField != EnvelopeSustain && delta > 0 {
			otherSum := env.Attack + env.Decay + env.Release - *currentValue
			if newValue+otherSum > 100 {
				return // block the increase
			}
		}

		// Clamp value between 0 and 100
		if newValue < 0 {
			newValue = 0
		} else if newValue > 100 {
			newValue = 100
		}
		*currentValue = newValue
	}
}
