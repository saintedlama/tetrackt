package tracker

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/notes"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
	"github.com/tetrackt/tetrackt/ui/tracker/effects"
	"github.com/tetrackt/tetrackt/ui/tracker/navigation"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(common.ColorAccentPrimary).
			Padding(0, 1)

	rowNumStyle = lipgloss.NewStyle().
			Foreground(common.ColorTextDisabled)

	rowNumBeatStyle = lipgloss.NewStyle().
			Foreground(common.ColorTextMuted)

	cursorRowStyle = lipgloss.NewStyle().
			Foreground(common.ColorAccentEnvelope).
			Bold(true)

	playbackRowStyle = lipgloss.NewStyle().
				Foreground(common.ColorAccentPlay).
				Bold(true)

	loopRowStyle = lipgloss.NewStyle().
			Background(common.ColorAccentModulation).
			Foreground(common.ColorWhite).
			Bold(true)

	cellStyle = lipgloss.NewStyle()

	cursorCellStyle = lipgloss.NewStyle().
			Background(common.ColorSurface).
			Foreground(common.ColorAccentPrimary)

	nibblePendingCellStyle = lipgloss.NewStyle().
				Background(common.ColorSurface).
				Foreground(common.ColorAccentWarning)
)

type Viewport struct {
	Width  int
	Height int
}

const DefaultBPM = 160
const minBPM = 40
const maxBPM = 300
const DefaultTicks = 0 // sub-ticks per row default (0 = no subdivision)
const defaultEditStep = 1

// BPM represents beats per minute with validation and duration calculation.
type BPM struct {
	value int
}

// NewBPM creates a new BPM value, clamping to valid range [minBPM, maxBPM].
func NewBPM(bpm int) BPM {
	if bpm < minBPM {
		bpm = minBPM
	}
	if bpm > maxBPM {
		bpm = maxBPM
	}
	return BPM{value: bpm}
}

// Value returns the BPM as an integer.
func (b BPM) Value() int {
	return b.value
}

// Duration returns the duration of one row at this BPM.
func (b BPM) Duration() time.Duration {
	bpm := b.value
	if bpm <= 0 {
		bpm = DefaultBPM
	}
	return time.Duration(60000/bpm) * time.Millisecond
}

// Adjust returns a new BPM with the given delta applied, clamped to valid range.
func (b BPM) Adjust(delta int) BPM {
	return NewBPM(b.value + delta)
}

// trackerColumn identifies the currently focused in-grid subcolumn.
type trackerColumn int

const (
	columnNote trackerColumn = iota
	columnVolume
	columnArpeggio
	columnPortamento
	columnEffect
	columnParam
	trackerColumnCount
)

// TrackerEdited is emitted when the user mutates tracker pattern data.
type TrackerEdited struct{}

// TrackerNoteEntered is emitted when a note is entered in the tracker grid.
type TrackerNoteEntered struct {
	Note  notes.Note
	Row   TrackRow
	Synth *audio.Synth
}

type trackerClipboard struct {
	HasData bool
	Cells   [][]TrackRow
}

// InlineEffectEdit represents an effect command + parameter pair typed inline.
type InlineEffectEdit struct {
	Type  EffectType
	Param int
}

// TrackerModel represents the state of the tracker pattern editor
type TrackerModel struct {
	Tracks    []Track
	NumRows   int
	NumTracks int

	// Navigation grid (manages cursor, viewport, selection)
	// Navigation grid (manages cursor, viewport, selection)
	nav *navigation.Grid

	IsPlaying    bool
	LoopStartSet bool
	LoopStartRow int
	LoopEndSet   bool
	LoopEndRow   int
	PlaybackRow  int
	Viewport     Viewport
	BPM          BPM
	Octave       int
	CursorCol    trackerColumn
	EditStep     int
	clipboard    trackerClipboard
	nibbleHi     *int
}

// RowTicks returns the number of sub-ticks for the given row.
// It returns the first non-zero Ticks value across all tracks at that row,
// or 0 when none is set.
func (m *TrackerModel) RowTicks(rowIdx int) int {
	if rowIdx >= 0 && rowIdx < m.NumRows {
		for _, track := range m.Tracks {
			if rowIdx < len(track.Rows) && track.Rows[rowIdx].Ticks > 0 {
				return track.Rows[rowIdx].Ticks
			}
		}
	}
	return DefaultTicks
}

// Track represents a single track in the pattern
type Track struct {
	number int
	Synth  *audio.Synth
	Rows   []TrackRow
}

// TrackRow represents a single row in a track
type TrackRow struct {
	Note       notes.Note
	Volume     int // 0-64
	Ticks      int // sub-ticks to play for this row; 0 = no subdivision
	Arpeggio   audio.ArpeggioEffect
	Portamento int // sub-tick glide count; 0 = snap
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
				audio.Oscillator{Type: audio.Silent},
				audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
				audio.Mixer{Volume1: 1.0, Volume2: 1.0},
				audio.NewFilter(),
				audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Depth: 0, Delay: 0, Dest: audio.ModPitch},
				audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Depth: 0, Delay: 0, Dest: audio.ModVolume},
				audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Depth: 0, Delay: 0, Dest: audio.ModPitch},
			),
			Rows: make([]TrackRow, numRows),
		}
		// Initialize all rows with empty data
		for j := range numRows {
			tracks[i].Rows[j] = TrackRow{
				Note:   notes.Off(),
				Volume: 0,
			}
		}
	}

	// Calculate visible rows (subtract chrome: header + separator + padding + status)
	chromeRows := 4
	visibleRows := viewportHeight - chromeRows

	return &TrackerModel{
		Tracks:      tracks,
		NumRows:     numRows,
		NumTracks:   numTracks,
		nav:         navigation.New(numRows, numTracks, visibleRows),
		IsPlaying:   false,
		PlaybackRow: 0,
		Viewport:    Viewport{Width: viewportWidth, Height: viewportHeight},
		BPM:         NewBPM(DefaultBPM),
		Octave:      4,
		CursorCol:   columnNote,
		EditStep:    defaultEditStep,
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
		if i == m.nav.CursorTrack() {
			trackHeader = headerStyle.Render(trackHeader)
		} else {
			trackHeader = headerStyle.Foreground(common.ColorGray).Render(trackHeader)
		}
		tracks.WriteString(trackHeader)
		tracks.WriteString("            ")
	}
	tracks.WriteString("\n")

	// Separator
	tracks.WriteString("    ")
	for i := 0; i < m.NumTracks; i++ {
		tracks.WriteString(strings.Repeat("─", 18))
		tracks.WriteString("   ")
	}
	tracks.WriteString("\n")

	viewportRow := m.nav.ViewportRow()
	endRow := min(viewportRow+m.visibleRows(), m.NumRows)

	// Render visible rows
	for row := viewportRow; row < endRow; row++ {
		// Row number with playback indicator
		rowNum := fmt.Sprintf("%02d", row)
		isLoopMarker := (m.LoopStartSet && row == m.LoopStartRow) || (m.LoopEndSet && row == m.LoopEndRow)
		if row == m.PlaybackRow && m.IsPlaying {
			tracks.WriteString(playbackRowStyle.Render(rowNum))
		} else if isLoopMarker {
			tracks.WriteString(loopRowStyle.Render(rowNum))
		} else if row == m.nav.CursorRow() {
			tracks.WriteString(cursorRowStyle.Render(rowNum))
		} else if row%4 == 0 {
			tracks.WriteString(rowNumBeatStyle.Render(rowNum))
		} else {
			tracks.WriteString(rowNumStyle.Render(rowNum))
		}
		tracks.WriteString("  ")

		// Track cells
		for trackIdx := 0; trackIdx < m.NumTracks; trackIdx++ {
			trackRow := m.Tracks[trackIdx].Rows[row]
			parts := []string{
				fmt.Sprintf("%-3s", formatNote(trackRow.Note)),
				formatVolume(trackRow.Volume),
				effects.FormatArpeggio(trackRow.Arpeggio),
				formatPortamento(trackRow.Portamento),
				effects.Type(trackRow.Effect.Type).Format(),
				effects.Type(trackRow.Effect.Type).FormatParam(trackRow.Effect.Param),
			}

			isCursorCell := row == m.nav.CursorRow() && trackIdx == m.nav.CursorTrack()
			selected := m.nav.IsSelected(row, trackIdx)
			for colIdx, part := range parts {
				col := trackerColumn(colIdx)
				isNibblePending := isCursorCell && m.nibbleHi != nil &&
					(col == columnVolume || col == columnArpeggio || col == columnPortamento || col == columnParam)

				if isNibblePending && col == m.CursorCol {
					switch col {
					case columnVolume:
						part = formatVolumePending(*m.nibbleHi)
					case columnArpeggio:
						part = formatArpPending(*m.nibbleHi)
					case columnPortamento:
						part = formatPortamentoPending(*m.nibbleHi)
					case columnParam:
						part = formatParamPending(*m.nibbleHi)
					}
				}

				styled := cellStyle.Render(part)
				if selected {
					styled = common.StyleSelected.Render(part)
				}
				if isCursorCell && col == m.CursorCol {
					if isNibblePending {
						styled = nibblePendingCellStyle.Render(part)
					} else {
						styled = cursorCellStyle.Render(part)
					}
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

	loopRange := ""
	if m.LoopStartSet {
		if m.LoopEndSet {
			loopRange = fmt.Sprintf("  Loop: %d-%d", m.LoopStartRow, m.LoopEndRow)
		} else {
			loopRange = fmt.Sprintf("  Loop: %d-?", m.LoopStartRow)
		}
	}
	playback := ""
	if m.IsPlaying {
		if m.LoopStartSet {
			playback = "  [LOOP]"
		} else {
			playback = "  [PLAY]"
		}
	}
	tracks.WriteString("\n")
	tracks.WriteString(fmt.Sprintf("Step: %d  Col: %s%s%s", m.EditStep, m.cursorColumnLabel(), loopRange, playback))

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
			row := m.nav.CursorRow()
			if !m.LoopStartSet {
				m.LoopStartRow = row
				m.LoopStartSet = true
			} else if !m.LoopEndSet {
				if row == m.LoopStartRow {
					m.LoopStartSet = false
				} else {
					m.LoopEndRow = row
					m.LoopEndSet = true
					if m.LoopEndRow < m.LoopStartRow {
						m.LoopStartRow, m.LoopEndRow = m.LoopEndRow, m.LoopStartRow
					}
				}
			} else {
				m.LoopStartSet = false
				m.LoopEndSet = false
			}
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
			result := m.nav.Move(-1, 0)
			m.clearNibbleBuffer()
			if result.TrackChanged {
				trackChanged = true
			}
		case "right":
			result := m.nav.Move(1, 0)
			m.clearNibbleBuffer()
			if result.TrackChanged {
				trackChanged = true
			}
		case "up":
			m.nav.Move(0, -1)
			m.clearNibbleBuffer()
		case "down":
			m.nav.Move(0, 1)
			m.clearNibbleBuffer()
		case "shift+up":
			result := m.nav.MoveExtending(0, -1)
			m.clearNibbleBuffer()
			if result.TrackChanged {
				trackChanged = true
			}
		case "shift+down":
			result := m.nav.MoveExtending(0, 1)
			m.clearNibbleBuffer()
			if result.TrackChanged {
				trackChanged = true
			}
		case "shift+left":
			result := m.nav.MoveExtending(-1, 0)
			m.clearNibbleBuffer()
			if result.TrackChanged {
				trackChanged = true
			}
		case "shift+right":
			result := m.nav.MoveExtending(1, 0)
			m.clearNibbleBuffer()
			if result.TrackChanged {
				trackChanged = true
			}
		case "home":
			m.nav.Move(0, -m.NumRows)
			m.clearNibbleBuffer()
		case "end":
			m.nav.Move(0, m.NumRows)
			m.clearNibbleBuffer()
		case "pgup":
			m.nav.Move(0, -m.nav.ViewportHeight())
			m.clearNibbleBuffer()
		case "pgdown":
			m.nav.Move(0, m.nav.ViewportHeight())
			m.clearNibbleBuffer()
		default:
			nibbleBefore := m.nibbleHi
			if msg, didEdit := m.handleEditInput(keyStr); didEdit {
				edited = true
				noteEntered = msg
			} else if nibbleBefore == nil && m.nibbleHi != nil {
				// First nibble buffered — no cell was committed but view must update
				// to show the pending digit.
				return m, nil
			}
		}

		if noteEntered != nil {
			return m, func() tea.Msg { return *noteEntered }
		}
		if trackChanged {
			currentTrack := m.Tracks[m.nav.CursorTrack()]
			cmd = func() tea.Msg {
				return ui.TrackChanged{
					Synth: currentTrack.Synth,
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

func (m *TrackerModel) handleGlobalEditingKey(key string) bool {
	switch key {
	case "ctrl+a":
		m.nav.SelectAll()
		m.clearNibbleBuffer()
		return false
	case "alt+c":
		m.copySelectionToClipboard()
		m.clearNibbleBuffer()
		return false
	case "alt+x":
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
	case "alt+shift+up", "shift+alt+up", "shift+f8":
		if m.transposeSelection(12) {
			m.clearNibbleBuffer()
			return true
		}
	case "alt+shift+down", "shift+alt+down", "shift+f7":
		if m.transposeSelection(-12) {
			m.clearNibbleBuffer()
			return true
		}
	case "alt+v":
		if m.pasteClipboard() {
			m.clearNibbleBuffer()
			return true
		}
	case "alt+shift+v", "shift+alt+v":
		if m.pasteClipboardEffectsOnly() {
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
	cursorTrack := m.nav.CursorTrack()
	cursorRow := m.nav.CursorRow()
	row := &m.Tracks[cursorTrack].Rows[cursorRow]
	switch m.CursorCol {
	case columnNote:
		if base, ok := ui.NoteKeys[key]; ok {
			note := notes.Note{Base: base, Octave: notes.Octave(m.Octave)}
			row.Note = note
			entered := &TrackerNoteEntered{Note: note, Row: *row, Synth: m.Tracks[cursorTrack].Synth}
			m.advanceByEditStep()
			m.clearNibbleBuffer()
			return entered, true
		}
	case columnVolume:
		if v, ok := parseHexNibble(key); ok {
			val := m.pushNibble(v)
			if val != nil {
				row.Volume = min(64, *val)
				m.advanceByEditStep()
				return nil, true
			}
		}
	case columnArpeggio:
		if v, ok := parseHexNibble(key); ok {
			val := m.pushNibble(v)
			if val != nil {
				o1 := (*val >> 4) & 0xF
				o2 := *val & 0xF
				row.Arpeggio = audio.ArpeggioEffect{Offsets: []int{0, o1, o2}}
				m.advanceByEditStep()
				return nil, true
			}
		}
	case columnPortamento:
		if v, ok := parseHexNibble(key); ok {
			val := m.pushNibble(v)
			if val != nil {
				row.Portamento = *val
				m.advanceByEditStep()
				return nil, true
			}
		}
	case columnEffect:
		if typ, ok := effects.ParseKey(key); ok {
			row.Effect.Type = EffectType(typ)
			m.clearNibbleBuffer()
			return nil, true
		}
	case columnParam:
		if v, ok := parseHexNibble(key); ok {
			val := m.pushNibble(v)
			if val != nil {
				if applyInlineEffect(row, row.Effect.Type, *val) {
					m.advanceByEditStep()
					return nil, true
				}
				m.clearNibbleBuffer()
				return nil, false
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
	case columnNote:
		row.Note = notes.Off()
	case columnVolume:
		row.Volume = 0
	case columnArpeggio:
		row.Arpeggio = audio.ArpeggioEffect{}
	case columnPortamento:
		row.Portamento = 0
	case columnEffect:
		switch row.Effect.Type {
		case EffectRowTicks:
			row.Ticks = 0
		case EffectArpPreset:
			row.Arpeggio = audio.ArpeggioEffect{}
		}
		row.Effect.Type = EffectNone
		row.Effect.Param = 0
	case columnParam:
		_ = applyInlineEffect(row, row.Effect.Type, 0)
	}
}

func (m *TrackerModel) advanceByEditStep() {
	step := m.EditStep
	if step <= 0 {
		step = defaultEditStep
	}
	for range step {
		m.nav.Move(0, 1)
	}
}

func (m *TrackerModel) moveSubcolumn(delta int) {
	idx := m.nav.CursorTrack()*int(trackerColumnCount) + int(m.CursorCol)
	idx += delta
	if idx < 0 {
		idx = 0
	}
	maxIdx := m.NumTracks*int(trackerColumnCount) - 1
	if idx > maxIdx {
		idx = maxIdx
	}
	newTrack := idx / int(trackerColumnCount)
	m.CursorCol = trackerColumn(idx % int(trackerColumnCount))

	// Update navigation grid if track changed
	if newTrack != m.nav.CursorTrack() {
		m.nav.SetCursorPosition(m.nav.CursorRow(), newTrack)
	}
}

func (m *TrackerModel) copySelectionToClipboard() {
	r0, r1, t0, t1, _ := m.nav.SelectionBounds()

	height := r1 - r0 + 1
	width := t1 - t0 + 1
	buf := make([][]TrackRow, height)
	for r := range height {
		buf[r] = make([]TrackRow, width)
	}

	for _, c := range m.nav.SelectedCells() {
		buf[c.Row-r0][c.Track-t0] = m.Tracks[c.Track].Rows[c.Row]
	}

	m.clipboard = trackerClipboard{HasData: true, Cells: buf}
}

func (m *TrackerModel) clearSelectedCells() {
	for _, c := range m.nav.SelectedCells() {
		m.Tracks[c.Track].Rows[c.Row] = TrackRow{Note: notes.Off()}
	}
}

func (m *TrackerModel) pasteClipboard() bool {
	if !m.clipboard.HasData || len(m.clipboard.Cells) == 0 {
		return false
	}
	cursorRow := m.nav.CursorRow()
	cursorTrack := m.nav.CursorTrack()
	for r := range len(m.clipboard.Cells) {
		dstRow := cursorRow + r
		if dstRow >= m.NumRows {
			break
		}
		for t := range len(m.clipboard.Cells[r]) {
			dstTrack := cursorTrack + t
			if dstTrack >= m.NumTracks {
				break
			}
			m.Tracks[dstTrack].Rows[dstRow] = m.clipboard.Cells[r][t]
		}
	}
	return true
}

func (m *TrackerModel) pasteClipboardEffectsOnly() bool {
	if !m.clipboard.HasData || len(m.clipboard.Cells) == 0 {
		return false
	}
	cursorRow := m.nav.CursorRow()
	cursorTrack := m.nav.CursorTrack()
	for r := range len(m.clipboard.Cells) {
		dstRow := cursorRow + r
		if dstRow >= m.NumRows {
			break
		}
		for t := range len(m.clipboard.Cells[r]) {
			dstTrack := cursorTrack + t
			if dstTrack >= m.NumTracks {
				break
			}
			src := m.clipboard.Cells[r][t]
			dst := &m.Tracks[dstTrack].Rows[dstRow]
			dst.Volume = src.Volume
			dst.Arpeggio = src.Arpeggio
			dst.Portamento = src.Portamento
			dst.Ticks = src.Ticks
			dst.Effect = src.Effect
		}
	}
	return true
}

func (m *TrackerModel) transposeSelection(semitones int) bool {
	edited := false
	for _, c := range m.nav.SelectedCells() {
		r := &m.Tracks[c.Track].Rows[c.Row]
		if r.Note.Base == notes.BaseOff {
			continue
		}
		if n, ok := r.Note.Transpose(semitones); ok {
			r.Note = n
			edited = true
		}
	}
	return edited
}

func (m *TrackerModel) insertTrackSpace() {
	t := m.nav.CursorTrack()
	cursorRow := m.nav.CursorRow()
	for row := m.NumRows - 1; row > cursorRow; row-- {
		m.Tracks[t].Rows[row] = m.Tracks[t].Rows[row-1]
	}
	m.Tracks[t].Rows[cursorRow] = TrackRow{Note: notes.Off()}
}

func (m *TrackerModel) insertGlobalRowSpace() {
	cursorRow := m.nav.CursorRow()
	for t := 0; t < m.NumTracks; t++ {
		for row := m.NumRows - 1; row > cursorRow; row-- {
			m.Tracks[t].Rows[row] = m.Tracks[t].Rows[row-1]
		}
		m.Tracks[t].Rows[cursorRow] = TrackRow{Note: notes.Off()}
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

func (m *TrackerModel) visibleRows() int {
	chromeRows := 4 // header + separator + padding
	return m.Viewport.Height - chromeRows
}

func formatNote(note notes.Note) string {
	if note.Base == notes.BaseOff {
		return "---"
	}

	if len(string(note.Base)) < 2 {
		return fmt.Sprintf("%s-%d", note.Base, note.Octave)
	}

	return fmt.Sprintf("%s%d", note.Base, note.Octave)
}

// formatVolume formats volume value for display.
func formatVolume(volume int) string {
	if volume == 0 {
		return ".."
	}
	return fmt.Sprintf("%02X", volume)
}

// formatVolumePending renders the first nibble of a volume entry in progress.
// e.g. hi=3 -> "3."
func formatVolumePending(hi int) string {
	return fmt.Sprintf("%X.", hi)
}

// formatArpPending renders the first nibble of an arpeggio entry in progress.
// e.g. hi=4 -> "A4."
func formatArpPending(hi int) string {
	return fmt.Sprintf("A%X.", hi)
}

// formatParamPending renders the first nibble of a param entry in progress.
// e.g. hi=B -> "B."
func formatParamPending(hi int) string {
	return fmt.Sprintf("%X.", hi)
}

// formatPortamento formats the portamento glide count for display.
// Returns "---" for 0, or "G" followed by two hex digits for non-zero.
func formatPortamento(p int) string {
	if p == 0 {
		return "---"
	}
	return fmt.Sprintf("G%02X", p)
}

// formatPortamentoPending renders the first nibble of a portamento entry in progress.
// e.g. hi=3 -> "G3."
func formatPortamentoPending(hi int) string {
	return fmt.Sprintf("G%X.", hi)
}

func applyInlineEffect(row *TrackRow, effectType EffectType, param int) bool {
	result, ok := effects.Apply(effects.Type(effectType), param, row.Ticks)
	if !ok {
		return false
	}

	row.Effect = result.Effect
	if result.Ticks > 0 || effectType == EffectRowTicks {
		row.Ticks = result.Ticks
	}
	if result.Arpeggio.IsActive() || effectType == EffectArpPreset {
		row.Arpeggio = result.Arpeggio
	}

	return true
}

func (m Track) CurrentRow() TrackRow {
	return m.Rows[m.number]
}

// SetCursorPosition sets the cursor to the specified row and track,
// clamping to valid ranges and adjusting the viewport if necessary.
func (m *TrackerModel) SetCursorPosition(row, track int) {
	m.nav.SetCursorPosition(row, track)
}

// SetViewport updates the viewport dimensions and keeps the navigation
// grid's viewport height in sync.
func (m *TrackerModel) SetViewport(width, height int) {
	m.Viewport = Viewport{Width: width, Height: height}
	chromeRows := 4
	visibleRows := height - chromeRows
	if visibleRows < 1 {
		visibleRows = 1
	}
	m.nav.SetViewportHeight(visibleRows)
}

// CursorRow returns the current cursor row position.
func (m *TrackerModel) CursorRow() int {
	return m.nav.CursorRow()
}

// CursorTrack returns the current cursor track position.
func (m *TrackerModel) CursorTrack() int {
	return m.nav.CursorTrack()
}
