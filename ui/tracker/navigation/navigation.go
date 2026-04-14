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
	selection *Selection
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
		selection:      NewSelection(),
	}
	g.clampAll()
	return g
}

// CursorPosition returns the current cursor position (row, track).
func (g *Grid) CursorPosition() (row, track int) {
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
	g.selection.Clear()

	return MoveResult{
		RowChanged:   g.cursorRow != oldRow,
		TrackChanged: g.cursorTrack != oldTrack,
	}
}

func (g *Grid) MoveExtending(deltaTrack, deltaRow int) MoveResult {
	if !g.selection.IsActive() {
		g.selection.Start(g.cursorRow, g.cursorTrack)
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
		g.selection.Extend(g.cursorRow, g.cursorTrack)
	}
	return result
}

func (g *Grid) ViewportHeight() int {
	return g.viewportHeight
}

func (g *Grid) ViewportRow() int {
	return g.viewportRow
}

func (g *Grid) ViewportBounds() (firstRow, lastRow int) {
	firstRow = g.viewportRow
	lastRow = min(g.viewportRow+g.viewportHeight-1, g.numRows-1)
	return
}

func (g *Grid) IsCursorVisible() bool {
	return g.cursorRow >= g.viewportRow && g.cursorRow < g.viewportRow+g.viewportHeight
}

func (g *Grid) HasSelection() bool {
	return g.selection.IsActive()
}

func (g *Grid) ClearSelection() {
	g.selection.Clear()
}

func (g *Grid) SelectAll() {
	if g.numRows > 0 && g.numTracks > 0 {
		g.selection.Start(0, 0)
		g.selection.Extend(g.numRows-1, g.numTracks-1)
	}
}

func (g *Grid) SelectionBounds() (minRow, maxRow, minTrack, maxTrack int, hasSelection bool) {
	if !g.selection.IsActive() {
		return g.cursorRow, g.cursorRow, g.cursorTrack, g.cursorTrack, false
	}
	minRow, maxRow, minTrack, maxTrack = g.selection.Bounds()
	return minRow, maxRow, minTrack, maxTrack, true
}

func (g *Grid) IsSelected(row, track int) bool {
	return g.selection.Contains(row, track)
}

func (g *Grid) NumRows() int {
	return g.numRows
}

func (g *Grid) NumTracks() int {
	return g.numTracks
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
