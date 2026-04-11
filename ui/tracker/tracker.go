package tracker

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(common.ColorAccentPrimary).
			Padding(0, 1)

	rowNumStyle = lipgloss.NewStyle().
			Foreground(common.ColorTextMuted)

	cursorRowStyle = lipgloss.NewStyle().
			Foreground(common.ColorAccentEnvelope).
			Bold(true)

	playbackRowStyle = lipgloss.NewStyle().
				Foreground(common.ColorAccentPlay).
				Bold(true)

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	cursorCellStyle = lipgloss.NewStyle().
			Background(common.ColorSurface).
			Foreground(common.ColorAccentPrimary).
			Padding(0, 1)
)

type Viewport struct {
	Width  int
	Height int
}

const DefaultBPM = 160
const MinBPM = 40
const MaxBPM = 300
const DefaultSpeed = 6 // sub-ticks per row

// EffectType identifies the per-row playback effect evaluated each sub-tick.
type EffectType int

const (
	EffectNone        EffectType = iota
	EffectVibrato                // Param hi-nibble: speed (1-15 ticks/cycle); lo-nibble: depth (semitones*4, 0-15)
	EffectVolumeSlide            // Param: volume delta per tick in 1/64 units; positive = louder
	EffectNoteCut                // Param: sub-tick at which to silence the note
	EffectNoteDelay              // Param: sub-tick at which to trigger NoteOn
)

// TrackerEffect is a per-row effect command evaluated every sub-tick during playback.
type TrackerEffect struct {
	Type  EffectType
	Param int
}

// TrackerModel represents the state of the tracker pattern editor
type TrackerModel struct {
	Tracks      []Track
	NumRows     int
	NumTracks   int
	CursorTrack int
	CursorRow   int
	IsPlaying   bool
	LoopToRow   bool
	LoopEndRow  int
	PlaybackRow int
	viewportRow int
	Viewport    Viewport
	BPM         int
	Speed       int // sub-ticks per row; 0 treated as DefaultSpeed
}

// BPMDuration returns the duration of one row at the current BPM.
func (m *TrackerModel) BPMDuration() time.Duration {
	bpm := m.BPM
	if bpm <= 0 {
		bpm = DefaultBPM
	}
	return time.Duration(60000/bpm) * time.Millisecond
}

// Track represents a single track in the pattern
type Track struct {
	number int
	Synth  *audio.Synth
	Rows   []TrackRow
}

// TrackRow represents a single row in a track
type TrackRow struct {
	Note       audio.Note
	Volume     int  // 0-64
	Ticks      int  // per-row tick count; 0 = use global Speed
	Continuous bool // synthesise this row as a continuous stream across ticks
	Arpeggio   audio.ArpeggioEffect
	Effect     TrackerEffect
}

// NewTracker creates a new pattern with the specified number of tracks and rows
func NewTracker(numTracks, numRows, viewportWidth, viewportHeight int) *TrackerModel {
	tracks := make([]Track, numTracks)
	for i := range numTracks {
		tracks[i] = Track{
			number: i,
			Synth: audio.NewSynth(
				audio.Oscillator{Type: audio.Sine},
				audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
				audio.Oscillator{Type: audio.Silent},
				audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
				audio.Mixer{Volume1: 1.0, Volume2: 1.0},
				audio.NewFilter(),
				audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Depth: 0, Delay: 0, Dest: audio.ModPitch},
				audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Depth: 0, Delay: 0, Dest: audio.ModVolume},
			),
			Rows: make([]TrackRow, numRows),
		}
		// Initialize all rows with empty data
		for j := range numRows {
			tracks[i].Rows[j] = TrackRow{
				Note:   audio.Off(),
				Volume: 0,
			}
		}
	}
	return &TrackerModel{
		Tracks:      tracks,
		NumRows:     numRows,
		NumTracks:   numTracks,
		CursorTrack: 0,
		IsPlaying:   false,
		LoopToRow:   false,
		PlaybackRow: 0,
		CursorRow:   0,
		viewportRow: 0,
		Viewport:    Viewport{Width: viewportWidth, Height: viewportHeight},
		BPM:         DefaultBPM,
		Speed:       DefaultSpeed,
	}
}

func (m *TrackerModel) Init() tea.Cmd {
	return nil
}

func (m *TrackerModel) View() string {
	// Track editor section
	var tracks strings.Builder

	// Track headers
	tracks.WriteString("    ") // Row number space
	for i := 0; i < m.NumTracks; i++ {
		trackHeader := fmt.Sprintf("Track %d", i+1)
		if i == m.CursorTrack {
			trackHeader = headerStyle.Render(trackHeader)
		} else {
			trackHeader = headerStyle.Foreground(common.ColorGray).Render(trackHeader)
		}
		tracks.WriteString(trackHeader)
		tracks.WriteString("        ")
	}
	tracks.WriteString("\n")

	// Separator
	tracks.WriteString("    ")
	for i := 0; i < m.NumTracks; i++ {
		tracks.WriteString(strings.Repeat("─", 14))
		tracks.WriteString("   ")
	}
	tracks.WriteString("\n")

	endRow := min(m.viewportRow+m.visibleRows(), m.NumRows)

	// Render visible rows
	for row := m.viewportRow; row < endRow; row++ {
		// Row number with playback indicator
		rowNumStr := fmt.Sprintf("%02d ", row)
		if row == m.PlaybackRow && m.IsPlaying {
			tracks.WriteString(playbackRowStyle.Render(rowNumStr))
		} else if row == m.CursorRow {
			tracks.WriteString(cursorRowStyle.Render(rowNumStr))
		} else {
			tracks.WriteString(rowNumStyle.Render(rowNumStr))
		}

		// Track cells
		for trackIdx := 0; trackIdx < m.NumTracks; trackIdx++ {
			trackRow := m.Tracks[trackIdx].Rows[row]
			cellContent := fmt.Sprintf("%-3s %2s %3s %3s", formatNote(trackRow.Note), formatVolume(trackRow.Volume), formatArpeggio(trackRow.Arpeggio), formatEffect(trackRow.Effect))

			if row == m.CursorRow && trackIdx == m.CursorTrack {
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

func (m *TrackerModel) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		keyStr := msg.String()

		// Track mode key handling
		switch keyStr {
		case "left":
			// Move cursor left (previous track)
			if m.CursorTrack > 0 {
				m.CursorTrack--
				currentTrack := m.Tracks[m.CursorTrack]
				cmd = func() tea.Msg {
					return ui.TrackChanged{Synth: currentTrack.Synth}
				}
			}
		case "right":
			// Move cursor right (next track)
			if m.CursorTrack < m.NumTracks-1 {
				m.CursorTrack++
				currentTrack := m.Tracks[m.CursorTrack]
				cmd = func() tea.Msg {
					return ui.TrackChanged{Synth: currentTrack.Synth}
				}
			}
		case "up":
			// Move cursor up (previous row)
			if m.CursorRow > 0 {
				m.CursorRow--
				// Adjust viewport if needed
				if m.CursorRow < m.viewportRow {
					m.viewportRow = m.CursorRow
				}
			}
		case "down":
			// Move cursor down (next row)
			m.MoveCursorDown()
		case "home":
			// Jump to first row
			m.CursorRow = 0
			m.viewportRow = 0
		case "end":
			// Jump to last row
			m.CursorRow = m.NumRows - 1
			visibleRows := m.visibleRows()
			m.viewportRow = max(m.NumRows-visibleRows, 0)
		}
	}

	return m, cmd
}

func (m *TrackerModel) visibleRows() int {
	chromeRows := 4 // header + separator + padding
	return m.Viewport.Height - chromeRows
}

func formatNote(note audio.Note) string {
	if note.Base == audio.BaseOff {
		return "---"
	}

	if len(string(note.Base)) < 2 {
		return fmt.Sprintf("%s-%d", note.Base, note.Octave)
	}

	return fmt.Sprintf("%s%d", note.Base, note.Octave)
}

// formatVolume formats volume value for display
func formatVolume(volume int) string {
	if volume == 0 {
		return ".."
	}
	return fmt.Sprintf("%02d", volume)
}

// formatArpeggio formats an arpeggio effect for display (3 chars).
// Active arp with offsets [0,4,7] shows "A47"; inactive shows "---".
func formatArpeggio(arp audio.ArpeggioEffect) string {
	if !arp.IsActive() {
		return "---"
	}
	o1, o2 := 0, 0
	if len(arp.Offsets) > 1 {
		o1 = arp.Offsets[1] % 10
	}
	if len(arp.Offsets) > 2 {
		o2 = arp.Offsets[2] % 10
	}
	return fmt.Sprintf("A%d%d", o1, o2)
}

// formatEffect formats a TrackerEffect for display (3 chars).
// Shows a letter code and packed parameter: V27 = vibrato speed 2 depth 7,
// S+4 = volume slide +4/tick, C03 = note cut at tick 3, D05 = note delay to tick 5.
// Inactive effect shows "---".
func formatEffect(e TrackerEffect) string {
	switch e.Type {
	case EffectVibrato:
		speed := (e.Param >> 4) & 0xF
		depth := e.Param & 0xF
		return fmt.Sprintf("V%X%X", speed, depth)
	case EffectVolumeSlide:
		if e.Param >= 0 {
			return fmt.Sprintf("S+%d", e.Param)
		}
		return fmt.Sprintf("S%d", e.Param)
	case EffectNoteCut:
		return fmt.Sprintf("C%02d", e.Param)
	case EffectNoteDelay:
		return fmt.Sprintf("D%02d", e.Param)
	}
	return "---"
}

func (m Track) CurrentRow() TrackRow {
	return m.Rows[m.number]
}

func (m *TrackerModel) CurrentTrack() Track {
	return m.Tracks[m.CursorTrack]
}

// MoveCursorDown advances the cursor one row, clamping at the last row
// and scrolling the viewport if necessary.
func (m *TrackerModel) MoveCursorDown() {
	if m.CursorRow < m.NumRows-1 {
		m.CursorRow++
		visibleRows := m.visibleRows()
		if m.CursorRow >= m.viewportRow+visibleRows {
			m.viewportRow = m.CursorRow - visibleRows + 1
		}
	}
}

func (m *TrackerModel) SetNote(note audio.Note) TrackRow {
	trackCell := &m.Tracks[m.CursorTrack].Rows[m.CursorRow]
	trackCell.Note = note

	return *trackCell
}

func (m *TrackerModel) GetNote() audio.Note {
	trackCell := &m.Tracks[m.CursorTrack].Rows[m.CursorRow]
	return trackCell.Note
}
