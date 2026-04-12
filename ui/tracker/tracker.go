package tracker

import (
	"fmt"
	"strconv"
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

	cellStyle = lipgloss.NewStyle()

	cursorCellStyle = lipgloss.NewStyle().
			Background(common.ColorSurface).
			Foreground(common.ColorAccentPrimary)
)

type Viewport struct {
	Width  int
	Height int
}

const DefaultBPM = 160
const MinBPM = 40
const MaxBPM = 300
const DefaultSpeed = 6 // sub-ticks per row
const DefaultEditStep = 1

// TrackerEditMode controls whether typing edits cells or only navigates.
type TrackerEditMode int

const (
	NavigateMode TrackerEditMode = iota
	EditMode
)

// TrackerColumn identifies the currently focused in-grid subcolumn.
type TrackerColumn int

const (
	ColumnNote TrackerColumn = iota
	ColumnVolume
	ColumnArpeggio
	ColumnEffect
	ColumnParam
	trackerColumnCount
)

// TrackerEdited is emitted when the user mutates tracker pattern data.
type TrackerEdited struct{}

// TrackerNoteEntered is emitted when a note is entered in the tracker grid.
type TrackerNoteEntered struct {
	Note  audio.Note
	Row   TrackRow
	Synth *audio.Synth
	Speed int
}

type trackerSelection struct {
	Active      bool
	AnchorRow   int
	AnchorTrack int
	EndRow      int
	EndTrack    int
}

type trackerClipboard struct {
	HasData bool
	Cells   [][]TrackRow
}

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
	Octave      int
	Mode        TrackerEditMode
	CursorCol   TrackerColumn
	EditStep    int
	selection   trackerSelection
	clipboard   trackerClipboard
	nibbleHi    *int
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
	// Optional informational metadata for synth patches selected from patch bank.
	PatchName     string
	PatchCategory string
	PatchTags     []string
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
		Octave:      4,
		Mode:        NavigateMode,
		CursorCol:   ColumnNote,
		EditStep:    DefaultEditStep,
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
		rowNumStr := fmt.Sprintf("%02d  ", row)
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
			parts := []string{
				fmt.Sprintf("%-3s", formatNote(trackRow.Note)),
				formatVolume(trackRow.Volume),
				formatArpeggio(trackRow.Arpeggio),
				formatEffectType(trackRow.Effect.Type),
				formatEffectParam(trackRow.Effect.Param),
			}

			selected := m.isSelected(row, trackIdx)
			for colIdx, part := range parts {
				styled := cellStyle.Render(part)
				if selected {
					styled = common.StyleSelected.Render(part)
				}
				if row == m.CursorRow && trackIdx == m.CursorTrack && TrackerColumn(colIdx) == m.CursorCol {
					styled = cursorCellStyle.Render(part)
				}
				tracks.WriteString(styled)
				if colIdx < len(parts)-1 {
					tracks.WriteString(" ")
				}
			}
			tracks.WriteString("  ")
		}
		tracks.WriteString("\n")
	}

	mode := "NAV"
	if m.Mode == EditMode {
		mode = "EDIT"
	}
	tracks.WriteString("\n")
	tracks.WriteString(fmt.Sprintf("Mode: %s  Step: %d", mode, m.EditStep))

	return tracks.String()
}

func (m *TrackerModel) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		keyStr := msg.String()
		if m.handleGlobalEditingKey(keyStr) {
			return m, func() tea.Msg { return TrackerEdited{} }
		}

		trackChanged := false
		edited := false
		noteEntered := (*TrackerNoteEntered)(nil)

		// Track mode key handling
		switch keyStr {
		case "space":
			m.toggleMode()
			m.clearNibbleBuffer()
			return m, nil
		case "tab":
			m.moveSubcolumn(1)
			m.clearNibbleBuffer()
			return m, nil
		case "shift+tab":
			m.moveSubcolumn(-1)
			m.clearNibbleBuffer()
			return m, nil
		case "left":
			// Move cursor left (previous track)
			m.clearSelection()
			m.clearNibbleBuffer()
			if m.CursorTrack > 0 {
				m.CursorTrack--
				trackChanged = true
			}
		case "right":
			// Move cursor right (next track)
			m.clearSelection()
			m.clearNibbleBuffer()
			if m.CursorTrack < m.NumTracks-1 {
				m.CursorTrack++
				trackChanged = true
			}
		case "up":
			// Move cursor up (previous row)
			m.clearSelection()
			m.clearNibbleBuffer()
			if m.CursorRow > 0 {
				m.CursorRow--
				// Adjust viewport if needed
				if m.CursorRow < m.viewportRow {
					m.viewportRow = m.CursorRow
				}
			}
		case "down":
			// Move cursor down (next row)
			m.clearSelection()
			m.clearNibbleBuffer()
			m.MoveCursorDown()
		case "shift+up":
			m.startSelection()
			m.clearNibbleBuffer()
			if m.CursorRow > 0 {
				m.CursorRow--
				if m.CursorRow < m.viewportRow {
					m.viewportRow = m.CursorRow
				}
			}
			m.extendSelection()
		case "shift+down":
			m.startSelection()
			m.clearNibbleBuffer()
			m.MoveCursorDown()
			m.extendSelection()
		case "shift+left":
			m.startSelection()
			m.clearNibbleBuffer()
			if m.CursorTrack > 0 {
				m.CursorTrack--
				trackChanged = true
			}
			m.extendSelection()
		case "shift+right":
			m.startSelection()
			m.clearNibbleBuffer()
			if m.CursorTrack < m.NumTracks-1 {
				m.CursorTrack++
				trackChanged = true
			}
			m.extendSelection()
		case "home":
			// Jump to first row
			m.clearSelection()
			m.clearNibbleBuffer()
			m.CursorRow = 0
			m.viewportRow = 0
		case "end":
			// Jump to last row
			m.clearSelection()
			m.clearNibbleBuffer()
			m.CursorRow = m.NumRows - 1
			visibleRows := m.visibleRows()
			m.viewportRow = max(m.NumRows-visibleRows, 0)
		case "pgup":
			m.clearSelection()
			m.clearNibbleBuffer()
			jump := max(1, m.visibleRows())
			m.CursorRow = max(0, m.CursorRow-jump)
			if m.CursorRow < m.viewportRow {
				m.viewportRow = m.CursorRow
			}
		case "pgdown":
			m.clearSelection()
			m.clearNibbleBuffer()
			jump := max(1, m.visibleRows())
			m.CursorRow = min(m.NumRows-1, m.CursorRow+jump)
			if m.CursorRow >= m.viewportRow+m.visibleRows() {
				m.viewportRow = max(0, m.CursorRow-m.visibleRows()+1)
			}
		default:
			if m.Mode == EditMode {
				if msg, didEdit := m.handleEditInput(keyStr); didEdit {
					edited = true
					noteEntered = msg
				}
			}
		}

		if noteEntered != nil {
			return m, func() tea.Msg { return *noteEntered }
		}
		if trackChanged {
			currentTrack := m.Tracks[m.CursorTrack]
			cmd = func() tea.Msg {
				return ui.TrackChanged{
					Synth:         currentTrack.Synth,
					PatchName:     currentTrack.PatchName,
					PatchCategory: currentTrack.PatchCategory,
					PatchTags:     append([]string(nil), currentTrack.PatchTags...),
				}
			}
		}
		if edited {
			if cmd != nil {
				return m, tea.Batch(cmd, func() tea.Msg { return TrackerEdited{} })
			}
			return m, func() tea.Msg { return TrackerEdited{} }
		}
	}

	return m, cmd
}

func (m *TrackerModel) toggleMode() {
	if m.Mode == EditMode {
		m.Mode = NavigateMode
		return
	}
	m.Mode = EditMode
}

func (m *TrackerModel) handleGlobalEditingKey(key string) bool {
	switch key {
	case "ctrl+a":
		m.selection = trackerSelection{
			Active:      true,
			AnchorRow:   0,
			AnchorTrack: 0,
			EndRow:      m.NumRows - 1,
			EndTrack:    m.NumTracks - 1,
		}
		m.clearNibbleBuffer()
		return false
	case "ctrl+c":
		m.copySelectionToClipboard()
		m.clearNibbleBuffer()
		return false
	case "ctrl+x":
		m.copySelectionToClipboard()
		m.clearSelectedCells()
		m.clearNibbleBuffer()
		return true
	case "alt+up", "f8":
		if m.transposeSelection(1) {
			m.clearNibbleBuffer()
			return true
		}
	case "alt+down", "f7":
		if m.transposeSelection(-1) {
			m.clearNibbleBuffer()
			return true
		}
	case "alt+shift+up", "shift+f8":
		if m.transposeSelection(12) {
			m.clearNibbleBuffer()
			return true
		}
	case "alt+shift+down", "shift+f7":
		if m.transposeSelection(-12) {
			m.clearNibbleBuffer()
			return true
		}
	case "ctrl+v":
		if m.pasteClipboard() {
			m.clearNibbleBuffer()
			return true
		}
	case "insert":
		m.insertTrackSpace()
		m.clearNibbleBuffer()
		return true
	case "shift+insert":
		m.insertGlobalRowSpace()
		m.clearNibbleBuffer()
		return true
	}
	return false
}

func (m *TrackerModel) handleEditInput(key string) (*TrackerNoteEntered, bool) {
	row := &m.Tracks[m.CursorTrack].Rows[m.CursorRow]
	switch m.CursorCol {
	case ColumnNote:
		if base, ok := ui.NoteKeys[key]; ok {
			note := audio.Note{Base: base, Octave: audio.Octave(m.Octave)}
			row.Note = note
			entered := &TrackerNoteEntered{Note: note, Row: *row, Synth: m.Tracks[m.CursorTrack].Synth, Speed: m.Speed}
			m.advanceByEditStep()
			m.clearNibbleBuffer()
			return entered, true
		}
	case ColumnVolume:
		if v, ok := parseHexNibble(key); ok {
			val := m.pushNibble(v)
			if val != nil {
				row.Volume = min(64, *val)
				m.advanceByEditStep()
				return nil, true
			}
		}
	case ColumnArpeggio:
		if v, ok := parseHexNibble(key); ok {
			val := m.pushNibble(v)
			if val != nil {
				o1 := (*val >> 4) & 0xF
				o2 := *val & 0xF
				row.Arpeggio = audio.ArpeggioEffect{Offsets: []int{0, o1, o2}}
				row.Continuous = true
				m.advanceByEditStep()
				return nil, true
			}
		}
	case ColumnEffect:
		if v, ok := parseHexNibble(key); ok {
			if v <= int(EffectNoteDelay) {
				row.Effect.Type = EffectType(v)
				m.clearNibbleBuffer()
				return nil, true
			}
		}
	case ColumnParam:
		if v, ok := parseHexNibble(key); ok {
			val := m.pushNibble(v)
			if val != nil {
				row.Effect.Param = *val
				m.advanceByEditStep()
				return nil, true
			}
		}
	}

	if key == "delete" {
		m.clearCurrentCellField(row)
		m.clearNibbleBuffer()
		return nil, true
	}

	return nil, false
}

func (m *TrackerModel) clearCurrentCellField(row *TrackRow) {
	switch m.CursorCol {
	case ColumnNote:
		row.Note = audio.Off()
	case ColumnVolume:
		row.Volume = 0
	case ColumnArpeggio:
		row.Arpeggio = audio.ArpeggioEffect{}
		row.Continuous = false
	case ColumnEffect:
		row.Effect.Type = EffectNone
	case ColumnParam:
		row.Effect.Param = 0
	}
}

func (m *TrackerModel) advanceByEditStep() {
	step := m.EditStep
	if step <= 0 {
		step = DefaultEditStep
	}
	for range step {
		m.MoveCursorDown()
	}
}

func (m *TrackerModel) moveSubcolumn(delta int) {
	idx := m.CursorTrack*int(trackerColumnCount) + int(m.CursorCol)
	idx += delta
	if idx < 0 {
		idx = 0
	}
	maxIdx := m.NumTracks*int(trackerColumnCount) - 1
	if idx > maxIdx {
		idx = maxIdx
	}
	m.CursorTrack = idx / int(trackerColumnCount)
	m.CursorCol = TrackerColumn(idx % int(trackerColumnCount))
}

func (m *TrackerModel) clearSelection() {
	m.selection.Active = false
}

func (m *TrackerModel) startSelection() {
	if m.selection.Active {
		return
	}
	m.selection = trackerSelection{
		Active:      true,
		AnchorRow:   m.CursorRow,
		AnchorTrack: m.CursorTrack,
		EndRow:      m.CursorRow,
		EndTrack:    m.CursorTrack,
	}
}

func (m *TrackerModel) extendSelection() {
	if !m.selection.Active {
		return
	}
	m.selection.EndRow = m.CursorRow
	m.selection.EndTrack = m.CursorTrack
}

func (m *TrackerModel) selectionBounds() (int, int, int, int, bool) {
	if !m.selection.Active {
		return m.CursorRow, m.CursorRow, m.CursorTrack, m.CursorTrack, false
	}
	r0 := min(m.selection.AnchorRow, m.selection.EndRow)
	r1 := max(m.selection.AnchorRow, m.selection.EndRow)
	t0 := min(m.selection.AnchorTrack, m.selection.EndTrack)
	t1 := max(m.selection.AnchorTrack, m.selection.EndTrack)
	return r0, r1, t0, t1, true
}

func (m *TrackerModel) isSelected(row, track int) bool {
	r0, r1, t0, t1, ok := m.selectionBounds()
	if !ok {
		return false
	}
	return row >= r0 && row <= r1 && track >= t0 && track <= t1
}

func (m *TrackerModel) copySelectionToClipboard() {
	r0, r1, t0, t1, ok := m.selectionBounds()
	if !ok {
		r0, r1, t0, t1 = m.CursorRow, m.CursorRow, m.CursorTrack, m.CursorTrack
	}

	height := r1 - r0 + 1
	width := t1 - t0 + 1
	buf := make([][]TrackRow, height)
	for r := range height {
		buf[r] = make([]TrackRow, width)
		for t := range width {
			buf[r][t] = m.Tracks[t0+t].Rows[r0+r]
		}
	}
	m.clipboard = trackerClipboard{HasData: true, Cells: buf}
}

func (m *TrackerModel) clearSelectedCells() {
	r0, r1, t0, t1, ok := m.selectionBounds()
	if !ok {
		r0, r1, t0, t1 = m.CursorRow, m.CursorRow, m.CursorTrack, m.CursorTrack
	}
	for r := r0; r <= r1; r++ {
		for t := t0; t <= t1; t++ {
			m.Tracks[t].Rows[r] = TrackRow{Note: audio.Off()}
		}
	}
}

func (m *TrackerModel) pasteClipboard() bool {
	if !m.clipboard.HasData || len(m.clipboard.Cells) == 0 {
		return false
	}
	for r := range len(m.clipboard.Cells) {
		dstRow := m.CursorRow + r
		if dstRow >= m.NumRows {
			break
		}
		for t := range len(m.clipboard.Cells[r]) {
			dstTrack := m.CursorTrack + t
			if dstTrack >= m.NumTracks {
				break
			}
			m.Tracks[dstTrack].Rows[dstRow] = m.clipboard.Cells[r][t]
		}
	}
	return true
}

func (m *TrackerModel) transposeSelection(semitones int) bool {
	r0, r1, t0, t1, hasSelection := m.selectionBounds()
	if !hasSelection {
		r0, r1, t0, t1 = m.CursorRow, m.CursorRow, m.CursorTrack, m.CursorTrack
	}

	edited := false
	for r := r0; r <= r1; r++ {
		for t := t0; t <= t1; t++ {
			row := &m.Tracks[t].Rows[r]
			if row.Note.Base == audio.BaseOff {
				continue
			}
			if n, ok := row.Note.Transpose(semitones); ok {
				row.Note = n
				edited = true
			}
		}
	}

	return edited
}

func (m *TrackerModel) insertTrackSpace() {
	t := m.CursorTrack
	for row := m.NumRows - 1; row > m.CursorRow; row-- {
		m.Tracks[t].Rows[row] = m.Tracks[t].Rows[row-1]
	}
	m.Tracks[t].Rows[m.CursorRow] = TrackRow{Note: audio.Off()}
}

func (m *TrackerModel) insertGlobalRowSpace() {
	for t := 0; t < m.NumTracks; t++ {
		for row := m.NumRows - 1; row > m.CursorRow; row-- {
			m.Tracks[t].Rows[row] = m.Tracks[t].Rows[row-1]
		}
		m.Tracks[t].Rows[m.CursorRow] = TrackRow{Note: audio.Off()}
	}
}

func (m *TrackerModel) clearNibbleBuffer() {
	m.nibbleHi = nil
}

func (m *TrackerModel) pushNibble(v int) *int {
	if m.nibbleHi == nil {
		hi := v
		m.nibbleHi = &hi
		return nil
	}
	val := (*m.nibbleHi << 4) | v
	m.nibbleHi = nil
	return &val
}

func parseHexNibble(key string) (int, bool) {
	if len(key) != 1 {
		return 0, false
	}
	v, err := strconv.ParseInt(key, 16, 32)
	if err != nil {
		return 0, false
	}
	return int(v), true
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

func formatEffectType(t EffectType) string {
	switch t {
	case EffectVibrato:
		return "V"
	case EffectVolumeSlide:
		return "S"
	case EffectNoteCut:
		return "C"
	case EffectNoteDelay:
		return "D"
	default:
		return "."
	}
}

func formatEffectParam(param int) string {
	if param < 0 {
		param = 0
	}
	if param > 255 {
		param = 255
	}
	return fmt.Sprintf("%02X", param)
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
