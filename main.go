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
	"github.com/gopxl/beep/v2/speaker"
)

// Envelope represents ADSR envelope parameters
type Envelope struct {
	Attack  ui.PercentageKnob
	Decay   ui.PercentageKnob
	Sustain ui.PercentageKnob
	Release ui.PercentageKnob
}

// Instrument represents a complete instrument with oscilator and envelope
type Instrument struct {
	Oscilator audio.OscilatorType
	Envelope  Envelope
}

// NewInstrument creates a new instrument with default envelope
func NewInstrument(oscilator audio.OscilatorType) *Instrument {
	return &Instrument{
		Oscilator: oscilator,
		Envelope: Envelope{
			Attack:  ui.NewPercentageKnob("A", 0.0, false, defaultStyle, selectedStyle),
			Decay:   ui.NewPercentageKnob("D", 0.0, false, defaultStyle, selectedStyle),
			Sustain: ui.NewPercentageKnob("S", 1.0, false, defaultStyle, selectedStyle),
			Release: ui.NewPercentageKnob("R", 0.0, false, defaultStyle, selectedStyle),
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
	OscilatorEditMode
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
	width           int
	height          int
	synth           *audio.Synth
	oscilator       ui.OscilatorModel
	pattern         *Pattern
	cursorTrack     int
	cursorRow       int
	viewportRow     int // Top row visible in the viewport
	mode            InputMode
	jumpInput       string
	isPlaying       bool
	playbackRow     int
	envelopeField   EnvelopeEditField
	showPresetModal bool
	presetIndex     int
	octave          int
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

type envelopePreset struct {
	Name    string
	Type    string
	Attack  float64
	Decay   float64
	Sustain float64
	Release float64
}

var envelopePresets = []envelopePreset{
	{Name: "Off", Type: "Utility", Attack: 0.00, Decay: 0.00, Sustain: 1.00, Release: 0.00},
	{Name: "Pluck Clean", Type: "Pluck", Attack: 0.01, Decay: 0.12, Sustain: 0.00, Release: 0.10},
	{Name: "Soft Pad", Type: "Pad", Attack: 0.30, Decay: 0.30, Sustain: 0.80, Release: 0.35},
	{Name: "Bright Lead", Type: "Lead", Attack: 0.02, Decay: 0.10, Sustain: 0.60, Release: 0.12},
	{Name: "Organ Hold", Type: "Organ", Attack: 0.00, Decay: 0.00, Sustain: 1.00, Release: 0.08},
	{Name: "Perc Hit", Type: "Percussion", Attack: 0.00, Decay: 0.18, Sustain: 0.00, Release: 0.20},
	{Name: "Bass Pluck", Type: "Bass", Attack: 0.01, Decay: 0.15, Sustain: 0.10, Release: 0.08},
	{Name: "Piano", Type: "Keys", Attack: 0.02, Decay: 0.40, Sustain: 0.20, Release: 0.15},
	{Name: "Brass", Type: "Brass", Attack: 0.12, Decay: 0.25, Sustain: 0.70, Release: 0.20},
	{Name: "Warm Strings", Type: "Strings", Attack: 0.50, Decay: 0.20, Sustain: 0.80, Release: 0.25},
	{Name: "Slow Swell", Type: "Pad", Attack: 0.65, Decay: 0.10, Sustain: 0.90, Release: 0.20},
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
			m.mode = OscilatorEditMode
			return m, nil
		case "t":
			m.mode = TrackMode
			return m, nil
		case "e":
			m.mode = EnvelopeEditMode
			return m, nil
		case "+":
			if m.octave < maxOctave {
				m.octave++
			}
			return m, nil
		case "-":
			if m.octave > minOctave {
				m.octave--
			}
			return m, nil
		case "tab":
			// Cycle through Oscilator, Envelope, Track modes
			switch m.mode {
			case OscilatorEditMode:
				m.mode = EnvelopeEditMode
			case EnvelopeEditMode:
				m.mode = TrackMode
			default:
				m.mode = OscilatorEditMode
			}
			return m, nil
		case "shift+tab":
			// Reverse cycle through Oscilator, Envelope, Track modes
			switch m.mode {
			case OscilatorEditMode:
				m.mode = TrackMode
			case EnvelopeEditMode:
				m.mode = OscilatorEditMode
			default:
				m.mode = EnvelopeEditMode
			}
			return m, nil
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

		// Handle envelope edit mode
		if m.mode == EnvelopeEditMode {
			// If preset modal is open, handle modal navigation
			if m.showPresetModal {
				switch msg.String() {
				case "up":
					m.presetIndex = (m.presetIndex - 1 + len(envelopePresets)) % len(envelopePresets)
				case "down":
					m.presetIndex = (m.presetIndex + 1) % len(envelopePresets)
				case "enter":
					preset := envelopePresets[m.presetIndex]
					env := &m.pattern.tracks[m.cursorTrack].instrument.Envelope
					env.Attack.Value = preset.Attack
					env.Decay.Value = preset.Decay
					env.Sustain.Value = preset.Sustain
					env.Release.Value = preset.Release
					m.showPresetModal = false
				case "esc", "p":
					m.showPresetModal = false
				}
				return m, nil
			}

			switch msg.String() {
			case "p":
				m.showPresetModal = true
				m.presetIndex = 0
				return m, nil
			case "up":
				// Move to previous envelope field
				m.envelopeField = (m.envelopeField - 1 + 4) % 4
			case "down":
				// Move to next envelope field
				m.envelopeField = (m.envelopeField + 1) % 4
			case "left":
				// Decrease value by 1%
				m.adjustEnvelopeValue(-0.01)
			case "shift+left":
				// Decrease value by 10%
				m.adjustEnvelopeValue(-0.10)
			case "right":
				// Increase value by 1%
				m.adjustEnvelopeValue(0.01)
			case "shift+right":
				// Increase value by 10%
				m.adjustEnvelopeValue(0.10)
			case "esc":
				m.mode = TrackMode
			}
			return m, nil
		}

		// Handle oscilator edit mode
		if m.mode == OscilatorEditMode {
			switch msg.String() {
			case "esc":
				m.mode = TrackMode
				return m, nil
			}

			m.oscilator, _ = m.oscilator.Update(msg)
			// TODO: This seems weird - Explose if passing an OnChange callback be better?
			track := &m.pattern.tracks[m.cursorTrack]
			track.instrument.Oscilator = m.oscilator.Oscilator

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
						// TODO: This is a rather clunky way to sync oscilator with track instrument
						m.oscilator.Oscilator = m.ensureTrackInstrument(m.cursorTrack)
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
		case "left":
			// Move cursor left (previous track)
			if m.cursorTrack > 0 {
				m.cursorTrack--
				m.oscilator.Oscilator = m.ensureTrackInstrument(m.cursorTrack)
			}
		case "right":
			// Move cursor right (next track)
			if m.cursorTrack < m.pattern.numTracks-1 {
				m.cursorTrack++
				m.oscilator.Oscilator = m.ensureTrackInstrument(m.cursorTrack)
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
			gen := m.synth.NewOscilator(inst, freq)
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

// playNote plays a note at the given frequency using the current oscilator
func (m *model) playNote(frequency float64) {
	inst := m.ensureTrackInstrument(m.cursorTrack)
	oscilator := m.synth.NewOscilator(inst, frequency)

	// TODO: Calculate the number of samples for the note duration
	envelope := m.pattern.tracks[m.cursorTrack].instrument.Envelope

	duration := m.synth.SampleRate.N(time.Millisecond * 300)

	adsr := m.synth.NewADSREnvelope(
		oscilator,
		duration,
		envelope.Attack.Value,
		envelope.Decay.Value,
		envelope.Sustain.Value,
		envelope.Release.Value,
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

	modeStr := "TRACK"
	switch m.mode {
	case JumpMode:
		modeStr = fmt.Sprintf("JUMP: %s", m.jumpInput)
	case EnvelopeEditMode:
		modeStr = "ENVELOPE"
	case OscilatorEditMode:
		modeStr = "OSCILATOR"
	}
	playStatus := "STOPPED"
	if m.isPlaying {
		playStatus = fmt.Sprintf("PLAYING (Row %d)", m.playbackRow)
	}
	currentInst := m.trackInstrumentValue(m.cursorTrack)
	header.WriteString(infoStyle.Render(fmt.Sprintf("Oscilator: %s | Instrument: %s | Mode: %s | %s | Track: %d | Row: %d | Octave: %d",
		m.oscilator, currentInst, modeStr, playStatus, m.cursorTrack, m.cursorRow, m.octave)))
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
	footer := helpStyle.Render("↑↓←→: Navigate | J: Jump | 1-7: Notes | +/-: Octave | W: Oscilator (↑↓←→ select) | E: Envelope (↑↓ select, ←→ adjust) | T: Track | Space: Play/Pause | Q: Quit")

	// Optional modal overlay for envelope presets
	if m.showPresetModal {
		modal := m.renderPresetModal()
		width := lipgloss.Width(modal)
		height := lipgloss.Height(modal)

		body = lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars("TETRACKT"), lipgloss.WithWhitespaceBackground(lipgloss.Color("236")))
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

func (m model) envelopeView() string {
	// Show envelope
	env := m.pattern.tracks[m.cursorTrack].instrument.Envelope

	envView := strings.Builder{}
	envView.WriteString("ADSR Envelope:\n")

	envView.WriteString(env.Attack.View() + "\n")
	envView.WriteString(env.Decay.View() + "\n")
	envView.WriteString(env.Sustain.View() + "\n")
	envView.WriteString(env.Release.View() + "\n")

	return envView.String()
}

func (m model) renderPresetModal() string {
	var b strings.Builder
	b.WriteString("Envelope Presets (Enter to apply, Esc to cancel)\n")

	for idx, p := range envelopePresets {
		line := fmt.Sprintf("%s  A:%3d%% D:%3d%% S:%3d%% R:%3d%%",
			p.Name,
			int(p.Attack*100),
			int(p.Decay*100),
			int(p.Sustain*100),
			int(p.Release*100),
		)

		if idx == m.presetIndex {
			line = selectedStyle.Render(line)
		} else {
			line = defaultStyle.Render(line)
		}

		b.WriteString(line + "\n")
	}

	return modalBorderStyle.Render(b.String())
}

func (m model) synthView() string {
	oscilatorView := m.oscilator.View()
	envelopeView := m.envelopeView()

	// Apply active border to the current mode panel
	oscilatorBorder := panelBorderStyle
	envelopeBorder := panelBorderStyle

	switch m.mode {
	case OscilatorEditMode:
		oscilatorBorder = activePanelBorderStyle
	case EnvelopeEditMode:
		envelopeBorder = activePanelBorderStyle
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		oscilatorBorder.Render(oscilatorView),
		envelopeBorder.Render(envelopeView),
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

// setTrackInstrument assigns an instrument oscilator to a track
func (m *model) setTrackInstrument(trackIdx int, osc audio.OscilatorType) {
	if trackIdx < 0 || trackIdx >= m.pattern.numTracks {
		return
	}
	m.pattern.tracks[trackIdx].instrument.Oscilator = osc
}

// ensureTrackInstrument returns the track instrument oscilator, defaulting to sine
func (m *model) ensureTrackInstrument(trackIdx int) audio.OscilatorType {
	if trackIdx < 0 || trackIdx >= m.pattern.numTracks {
		return audio.Sine
	}
	track := &m.pattern.tracks[trackIdx]
	if track.instrument == nil {
		track.instrument = NewInstrument(audio.Sine)
	}
	return track.instrument.Oscilator
}

// trackInstrumentValue returns the track instrument oscilator without mutating
func (m model) trackInstrumentValue(trackIdx int) audio.OscilatorType {
	if trackIdx < 0 || trackIdx >= m.pattern.numTracks {
		return audio.Sine
	}
	track := m.pattern.tracks[trackIdx]
	if track.instrument == nil {
		return audio.Sine
	}
	return track.instrument.Oscilator
}

func main() {
	// Initialize synthesizer
	sampleRate := beep.SampleRate(44100)
	synth := audio.NewSynth(sampleRate)

	// Create pattern with 8 tracks and 64 rows
	pattern := NewPattern(8, 64)

	p := tea.NewProgram(
		model{
			synth:         synth,
			oscilator:     ui.NewOscilatorModel(selectedStyle),
			pattern:       pattern,
			cursorTrack:   0,
			cursorRow:     0,
			viewportRow:   0,
			mode:          TrackMode,
			jumpInput:     "",
			isPlaying:     false,
			playbackRow:   0,
			envelopeField: EnvelopeAttack,
			octave:        4,
		},
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// adjustEnvelopeValue adjusts the current envelope field by a delta value
func (m *model) adjustEnvelopeValue(delta float64) {
	env := &m.pattern.tracks[m.cursorTrack].instrument.Envelope
	var currentValue *float64

	switch m.envelopeField {
	case EnvelopeAttack:
		currentValue = &env.Attack.Value
	case EnvelopeDecay:
		currentValue = &env.Decay.Value
	case EnvelopeSustain:
		currentValue = &env.Sustain.Value
	case EnvelopeRelease:
		currentValue = &env.Release.Value
	}

	if currentValue != nil {
		newValue := *currentValue + delta

		// For A, D, R: prevent increases that would make A+D+R exceed 100
		if m.envelopeField != EnvelopeSustain && delta > 0 {
			otherSum := env.Attack.Value + env.Decay.Value + env.Release.Value - *currentValue
			if newValue+otherSum > 100 {
				return // block the increase
			}
		}

		// Clamp value between 0 and 100
		if newValue < 0 {
			newValue = 0
		} else if newValue > 1.0 {
			newValue = 1.0
		}

		*currentValue = newValue
	}
}
