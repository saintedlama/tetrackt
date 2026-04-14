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
	"github.com/tetrackt/tetrackt/ui/tracker/navigation"
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
const minBPM = 40
const maxBPM = 300
const DefaultSpeed = 6 // sub-ticks per row
const defaultEditStep = 1

// trackerEditMode controls whether typing edits cells or only navigates.
type trackerEditMode int

const (
	navigateMode trackerEditMode = iota
	editMode
)

// trackerColumn identifies the currently focused in-grid subcolumn.
type trackerColumn int

const (
	columnNote trackerColumn = iota
	columnVolume
	columnArpeggio
	columnEffect
	columnParam
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
	EffectRowTicks               // Param: 00 clears row tick override; 01..20 sets per-row sub-ticks
	EffectContinuous             // Param: 00 = off, 01 = on
	EffectArpPreset              // Param: high nibble preset, low nibble step bucket
)

// TrackerEffect is a per-row effect command evaluated every sub-tick during playback.
type TrackerEffect struct {
	Type  EffectType
	Param int
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

	IsPlaying   bool
	LoopToRow   bool
	LoopEndRow  int
	PlaybackRow int
	Viewport    Viewport
	BPM         int
	Speed       int // sub-ticks per row; 0 treated as DefaultSpeed
	Octave      int
	Mode        trackerEditMode
	CursorCol   trackerColumn
	EditStep    int
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

	// Calculate visible rows (subtract chrome: header + separator + padding + status)
	chromeRows := 4
	visibleRows := viewportHeight - chromeRows

	return &TrackerModel{
		Tracks:      tracks,
		NumRows:     numRows,
		NumTracks:   numTracks,
		nav:         navigation.New(numRows, numTracks, visibleRows),
		IsPlaying:   false,
		LoopToRow:   false,
		PlaybackRow: 0,
		Viewport:    Viewport{Width: viewportWidth, Height: viewportHeight},
		BPM:         DefaultBPM,
		Speed:       DefaultSpeed,
		Octave:      4,
		Mode:        navigateMode,
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

	viewportRow := m.nav.ViewportRow()
	endRow := min(viewportRow+m.visibleRows(), m.NumRows)

	// Render visible rows
	for row := viewportRow; row < endRow; row++ {
		// Row number with playback indicator
		rowNumStr := fmt.Sprintf("%02d  ", row)
		if row == m.PlaybackRow && m.IsPlaying {
			tracks.WriteString(playbackRowStyle.Render(rowNumStr))
		} else if row == m.nav.CursorRow() {
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
				formatEffectParam(trackRow.Effect.Type, trackRow.Effect.Param),
			}

			selected := m.isSelected(row, trackIdx)
			for colIdx, part := range parts {
				styled := cellStyle.Render(part)
				if selected {
					styled = common.StyleSelected.Render(part)
				}
				if row == m.nav.CursorRow() && trackIdx == m.nav.CursorTrack() && trackerColumn(colIdx) == m.CursorCol {
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
	if m.Mode == editMode {
		mode = "EDIT"
	}
	tracks.WriteString("\n")
	tracks.WriteString(fmt.Sprintf("Mode: %s  Step: %d  Col: %s", mode, m.EditStep, m.cursorColumnLabel()))

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
			if m.Mode == editMode {
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
			currentTrack := m.Tracks[m.nav.CursorTrack()]
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
	if m.Mode == editMode {
		m.Mode = navigateMode
		return
	}
	m.Mode = editMode
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
			note := audio.Note{Base: base, Octave: audio.Octave(m.Octave)}
			row.Note = note
			entered := &TrackerNoteEntered{Note: note, Row: *row, Synth: m.Tracks[cursorTrack].Synth, Speed: m.Speed}
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
				row.Continuous = true
				m.advanceByEditStep()
				return nil, true
			}
		}
	case columnEffect:
		if effectType, ok := parseEffectCommandKey(key); ok {
			row.Effect.Type = effectType
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
		row.Note = audio.Off()
	case columnVolume:
		row.Volume = 0
	case columnArpeggio:
		row.Arpeggio = audio.ArpeggioEffect{}
		row.Continuous = false
	case columnEffect:
		switch row.Effect.Type {
		case EffectRowTicks:
			row.Ticks = 0
		case EffectContinuous:
			row.Continuous = false
		case EffectArpPreset:
			row.Arpeggio = audio.ArpeggioEffect{}
			row.Continuous = false
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
		m.moveCursorDown()
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

func (m *TrackerModel) clearSelection() {
	m.nav.ClearSelection()
}

func (m *TrackerModel) startSelection() {
	// Navigation grid handles this automatically in MoveExtending
}

func (m *TrackerModel) extendSelection() {
	// Navigation grid handles this automatically in MoveExtending
}

func (m *TrackerModel) selectionBounds() (int, int, int, int, bool) {
	return m.nav.SelectionBounds()
}

func (m *TrackerModel) isSelected(row, track int) bool {
	return m.nav.IsSelected(row, track)
}

func (m *TrackerModel) copySelectionToClipboard() {
	r0, r1, t0, t1, _ := m.selectionBounds()

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
	r0, r1, t0, t1, _ := m.selectionBounds()
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
			dst.Ticks = src.Ticks
			dst.Continuous = src.Continuous
			dst.Arpeggio = src.Arpeggio
			dst.Effect = src.Effect
		}
	}
	return true
}

func (m *TrackerModel) transposeSelection(semitones int) bool {
	r0, r1, t0, t1, hasSelection := m.selectionBounds()
	if !hasSelection {
		r0, r1, t0, t1 = m.nav.CursorRow(), m.nav.CursorRow(), m.nav.CursorTrack(), m.nav.CursorTrack()
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
	t := m.nav.CursorTrack()
	cursorRow := m.nav.CursorRow()
	for row := m.NumRows - 1; row > cursorRow; row-- {
		m.Tracks[t].Rows[row] = m.Tracks[t].Rows[row-1]
	}
	m.Tracks[t].Rows[cursorRow] = TrackRow{Note: audio.Off()}
}

func (m *TrackerModel) insertGlobalRowSpace() {
	cursorRow := m.nav.CursorRow()
	for t := 0; t < m.NumTracks; t++ {
		for row := m.NumRows - 1; row > cursorRow; row-- {
			m.Tracks[t].Rows[row] = m.Tracks[t].Rows[row-1]
		}
		m.Tracks[t].Rows[cursorRow] = TrackRow{Note: audio.Off()}
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

func applyInlineEffect(row *TrackRow, effectType EffectType, param int) bool {
	decoded, ok := decodeEffectParam(effectType, param)
	if !ok {
		return false
	}

	switch effectType {
	case EffectNone:
		row.Effect.Type = effectType
		row.Effect.Param = 0
		return true
	case EffectVolumeSlide:
		row.Effect.Type = effectType
		row.Effect.Param = decoded // signed delta for playback
		return true
	case EffectVibrato, EffectNoteCut, EffectNoteDelay:
		row.Effect.Type = effectType
		row.Effect.Param = decoded
		return true
	case EffectRowTicks:
		row.Effect.Type = effectType
		row.Effect.Param = param
		if decoded == 0 {
			row.Ticks = 0
		} else {
			row.Ticks = decoded
		}
		return true
	case EffectContinuous:
		row.Effect.Type = effectType
		row.Effect.Param = param
		row.Continuous = decoded == 1
		return true
	case EffectArpPreset:
		preset := (param >> 4) & 0xF
		step := param & 0xF
		if step == 0 {
			step = 4
		}
		if preset == 0 {
			row.Effect.Type = effectType
			row.Effect.Param = param
			row.Arpeggio = audio.ArpeggioEffect{}
			return true
		}
		if preset > 5 {
			return false
		}

		ticks := row.Ticks
		if ticks <= 0 {
			ticks = DefaultSpeed
		}
		row.Effect.Type = effectType
		row.Effect.Param = param
		row.Arpeggio = audio.ArpeggioEffect{Offsets: generateInlineArpOffsets(preset, ticks, step)}
		row.Continuous = true
		return true
	default:
		return false
	}
}

func (m Track) CurrentRow() TrackRow {
	return m.Rows[m.number]
}

// moveCursorDown advances the cursor one row, clamping at the last row
// and scrolling the viewport if necessary.
func (m *TrackerModel) moveCursorDown() {
	m.nav.Move(0, 1)
}

// SetCursorPosition sets the cursor to the specified row and track,
// clamping to valid ranges and adjusting the viewport if necessary.
func (m *TrackerModel) SetCursorPosition(row, track int) {
	m.nav.SetCursorPosition(row, track)
}

// CursorRow returns the current cursor row position.
func (m *TrackerModel) CursorRow() int {
	return m.nav.CursorRow()
}

// CursorTrack returns the current cursor track position.
func (m *TrackerModel) CursorTrack() int {
	return m.nav.CursorTrack()
}
