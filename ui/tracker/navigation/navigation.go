package navigation

// MoveResult describes what changed during a Move operation.
type MoveResult struct {
	RowChanged   bool
	TrackChanged bool
}

// Changed returns true if either row or track changed.
func (m MoveResult) Changed() bool {
	return m.RowChanged || m.TrackChanged
}

// Grid manages 2D navigation (rows × tracks), viewport, and selection
// with no knowledge of cell content or rendering.
type Grid struct {
	// Grid dimensions
	numRows   int
	numTracks int

	// Cursor position
	cursorRow   int
	cursorTrack int

	// Viewport state
	viewportRow    int
	viewportHeight int // number of visible rows

	// Selection
	selection *selection
}

// New creates a grid with the given dimensions and viewport height.
func New(numRows, numTracks, viewportHeight int) *Grid {
	g := &Grid{
		numRows:        numRows,
		numTracks:      numTracks,
		viewportHeight: viewportHeight,
		cursorRow:      0,
		cursorTrack:    0,
		viewportRow:    0,
		selection:      newSelection(),
	}
	g.clampAll()
	return g
}

// CursorPosition returns the current cursor position (row, track).
func (g *Grid) cursorPosition() (row, track int) {
	return g.cursorRow, g.cursorTrack
}

func (g *Grid) CursorRow() int {
	return g.cursorRow
}

func (g *Grid) CursorTrack() int {
	return g.cursorTrack
}

func (g *Grid) SetCursorPosition(row, track int) {
	g.cursorRow = row
	g.cursorTrack = track
	g.clampCursor()
	g.adjustViewportToCursor()
}

func (g *Grid) Move(deltaTrack, deltaRow int) MoveResult {
	oldRow := g.cursorRow
	oldTrack := g.cursorTrack

	g.cursorRow += deltaRow
	g.cursorTrack += deltaTrack
	g.clampCursor()
	g.adjustViewportToCursor()
	g.selection.clear()

	return MoveResult{
		RowChanged:   g.cursorRow != oldRow,
		TrackChanged: g.cursorTrack != oldTrack,
	}
}

func (g *Grid) MoveExtending(deltaTrack, deltaRow int) MoveResult {
	if !g.selection.isActive() {
		g.selection.start(g.cursorRow, g.cursorTrack)
	}

	oldRow := g.cursorRow
	oldTrack := g.cursorTrack

	g.cursorRow += deltaRow
	g.cursorTrack += deltaTrack
	g.clampCursor()
	g.adjustViewportToCursor()

	result := MoveResult{
		RowChanged:   g.cursorRow != oldRow,
		TrackChanged: g.cursorTrack != oldTrack,
	}

	if result.Changed() {
		g.selection.extend(g.cursorRow, g.cursorTrack)
	}
	return result
}

func (g *Grid) ViewportHeight() int {
	return g.viewportHeight
}

func (g *Grid) SetViewportHeight(h int) {
	g.viewportHeight = h
	g.clampAll()
}

func (g *Grid) ViewportRow() int {
	return g.viewportRow
}

func (g *Grid) viewportBounds() (firstRow, lastRow int) {
	firstRow = g.viewportRow
	lastRow = min(g.viewportRow+g.viewportHeight-1, g.numRows-1)
	return
}

func (g *Grid) isCursorVisible() bool {
	return g.cursorRow >= g.viewportRow && g.cursorRow < g.viewportRow+g.viewportHeight
}

func (g *Grid) HasSelection() bool {
	return g.selection.isActive()
}

func (g *Grid) ClearSelection() {
	g.selection.clear()
}

func (g *Grid) SelectAll() {
	if g.numRows > 0 && g.numTracks > 0 {
		g.selection.start(0, 0)
		g.selection.extend(g.numRows-1, g.numTracks-1)
	}
}

func (g *Grid) SelectionBounds() (minRow, maxRow, minTrack, maxTrack int, hasSelection bool) {
	if !g.selection.isActive() {
		return g.cursorRow, g.cursorRow, g.cursorTrack, g.cursorTrack, false
	}
	minRow, maxRow, minTrack, maxTrack = g.selection.bounds()
	return minRow, maxRow, minTrack, maxTrack, true
}

func (g *Grid) IsSelected(row, track int) bool {
	return g.selection.contains(row, track)
}

// SelectedCells returns all selected (row, track) positions.
// If no selection is active, returns a single-element slice with the cursor position.
func (g *Grid) SelectedCells() []Cell {
	if !g.selection.isActive() {
		return []Cell{{Row: g.cursorRow, Track: g.cursorTrack}}
	}
	return g.selection.cells()
}

func (g *Grid) clampCursor() {
	if g.cursorRow < 0 {
		g.cursorRow = 0
	}
	if g.cursorRow >= g.numRows && g.numRows > 0 {
		g.cursorRow = g.numRows - 1
	}
	if g.cursorRow >= g.numRows && g.numRows == 0 {
		g.cursorRow = 0
	}

	if g.cursorTrack < 0 {
		g.cursorTrack = 0
	}
	if g.cursorTrack >= g.numTracks && g.numTracks > 0 {
		g.cursorTrack = g.numTracks - 1
	}
	if g.cursorTrack >= g.numTracks && g.numTracks == 0 {
		g.cursorTrack = 0
	}
}

func (g *Grid) clampViewport() {
	if g.viewportRow < 0 {
		g.viewportRow = 0
	}
	if g.viewportRow >= g.numRows && g.numRows > 0 {
		g.viewportRow = max(0, g.numRows-g.viewportHeight)
	}
}

func (g *Grid) adjustViewportToCursor() {
	// Cursor above viewport - scroll up
	if g.cursorRow < g.viewportRow {
		g.viewportRow = g.cursorRow
	}

	// Cursor below viewport - scroll down
	if g.cursorRow >= g.viewportRow+g.viewportHeight {
		g.viewportRow = g.cursorRow - g.viewportHeight + 1
	}

	g.clampViewport()
}

func (g *Grid) clampAll() {
	g.clampCursor()
	g.clampViewport()
}
