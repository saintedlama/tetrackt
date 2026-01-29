package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ffff")).
			Padding(0, 1)

	rowNumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	cursorRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff9800")).
			Bold(true)

	playbackRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ffff")).
				Bold(true)

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	cursorCellStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a2a")).
			Foreground(lipgloss.Color("#00e5ff")).
			Padding(0, 1)
)

type Viewport struct {
	Width  int
	Height int
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
}

// Track represents a single track in the pattern
type Track struct {
	number      int
	Oscillator1 audio.Oscillator
	Envelope1   audio.Envelope
	Oscillator2 audio.Oscillator
	Envelope2   audio.Envelope
	Mixer       audio.Mixer
	Rows        []TrackRow
}

// TrackRow represents a single row in a track
type TrackRow struct {
	Note   audio.Note
	Volume int    // 0-64
	Effect string // effect command
}

// NewPattern creates a new pattern with the specified number of tracks and rows
func NewTracker(numTracks, numRows, viewportWidth, viewportHeight int) *TrackerModel {
	tracks := make([]Track, numTracks)
	for i := range numTracks {
		tracks[i] = Track{
			number:      i,
			Oscillator1: audio.Oscillator{Type: audio.Sine},
			Envelope1:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
			Oscillator2: audio.Oscillator{Type: audio.Silent},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
			Mixer:       audio.Mixer{Balance: 0.0},
			Rows:        make([]TrackRow, numRows),
		}
		// Initialize all rows with empty data
		for j := range numRows {
			tracks[i].Rows[j] = TrackRow{
				Note:   audio.Off(),
				Volume: 0,
				Effect: "---",
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
	}
}

func (m *TrackerModel) Init() tea.Cmd {
	return nil
}

func (m *TrackerModel) View() tea.View {
	// Track editor section
	var tracks strings.Builder

	// Track headers
	tracks.WriteString("    ") // Row number space
	for i := 0; i < m.NumTracks; i++ {
		trackHeader := fmt.Sprintf("Track %d", i+1)
		if i == m.CursorTrack {
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
	for i := 0; i < m.NumTracks; i++ {
		tracks.WriteString(strings.Repeat("─", 10))
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
			cellContent := fmt.Sprintf("%-3s %2s %3s", formatNote(trackRow.Note), formatVolume(trackRow.Volume), trackRow.Effect)

			if row == m.CursorRow && trackIdx == m.CursorTrack {
				tracks.WriteString(cursorCellStyle.Render(cellContent))
			} else {
				tracks.WriteString(cellStyle.Render(cellContent))
			}
			tracks.WriteString(" ")
		}
		tracks.WriteString("\n")
	}

	return tea.NewView(tracks.String())
}

type TrackChanged struct {
	Oscillator1 audio.Oscillator
	Envelope1   audio.Envelope
	Oscillator2 audio.Oscillator
	Envelope2   audio.Envelope
	Mixer       audio.Mixer
}

func (m *TrackerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyStr := msg.String()

		// Track mode key handling
		switch keyStr {
		case "left":
			// Move cursor left (previous track)
			if m.CursorTrack > 0 {
				m.CursorTrack--

				// TODO: Extract to common function
				// TODO: solve sync problem between tracker and osc, env, mixer models
				//m.oscillator1.Oscillator = m.pattern.tracks[m.cursorTrack].oscillator1
				currentTrack := m.Tracks[m.CursorTrack]
				cmd = func() tea.Msg {
					return TrackChanged{
						Oscillator1: currentTrack.Oscillator1,
						Envelope1:   currentTrack.Envelope1,
						Oscillator2: currentTrack.Oscillator2,
						Envelope2:   currentTrack.Envelope2,
						Mixer:       currentTrack.Mixer,
					}
				}
			}
		case "right":
			// Move cursor right (next track)
			if m.CursorTrack < m.NumTracks-1 {
				m.CursorTrack++
				// TODO: solve sync problem between tracker and osc, env, mixer models
				//m.oscillator1.Oscillator = m.pattern.tracks[m.cursorTrack].oscillator1
				currentTrack := m.Tracks[m.CursorTrack]
				cmd = func() tea.Msg {
					return TrackChanged{
						Oscillator1: currentTrack.Oscillator1,
						Envelope1:   currentTrack.Envelope1,
						Oscillator2: currentTrack.Oscillator2,
						Envelope2:   currentTrack.Envelope2,
						Mixer:       currentTrack.Mixer,
					}
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
			if m.CursorRow < m.NumRows-1 {
				m.CursorRow++
				// Adjust viewport if needed
				visibleRows := m.visibleRows()
				if m.CursorRow >= m.viewportRow+visibleRows {
					m.viewportRow = m.CursorRow - visibleRows + 1
				}
			}
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

func (m Track) CurrentRow() TrackRow {
	return m.Rows[m.number]
}

func (m *TrackerModel) CurrentTrack() Track {
	return m.Tracks[m.CursorTrack]
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
