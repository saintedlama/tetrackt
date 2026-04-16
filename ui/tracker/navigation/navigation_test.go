package navigation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_InitializesCursorAtOrigin(t *testing.T) {
	g := New(10, 4, 5)
	row, track := g.cursorPosition()
	assert.Equal(t, 0, row, "expected cursor row 0")
	assert.Equal(t, 0, track, "expected cursor track 0")
}

func TestMove_Down_FromOrigin(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(0, 1)
	assert.True(t, result.Changed(), "expected move to succeed")
	assert.True(t, result.RowChanged, "expected row to change")
	assert.False(t, result.TrackChanged, "expected track to stay the same")
	assert.Equal(t, 1, g.CursorRow(), "expected row 1")
}

func TestMove_Down_AtBottom_Clamps(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(9, 0)
	result := g.Move(0, 1)
	assert.False(t, result.Changed(), "expected move to be clamped")
	assert.Equal(t, 9, g.CursorRow(), "expected row 9")
}

func TestMove_Up_AtTop_Clamps(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(0, -1)
	assert.False(t, result.Changed(), "expected move to be clamped")
	assert.Equal(t, 0, g.CursorRow(), "expected row 0")
}

func TestMove_Right_FromOrigin(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(1, 0)
	assert.True(t, result.Changed(), "expected move to succeed")
	assert.False(t, result.RowChanged, "expected row to stay the same")
	assert.True(t, result.TrackChanged, "expected track to change")
	assert.Equal(t, 1, g.CursorTrack(), "expected track 1")
}

func TestMove_Right_AtRightmost_Clamps(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(0, 3)
	result := g.Move(1, 0)
	assert.False(t, result.Changed(), "expected move to be clamped")
	assert.Equal(t, 3, g.CursorTrack(), "expected track 3")
}

func TestMove_Left_AtLeftmost_Clamps(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(-1, 0)
	assert.False(t, result.Changed(), "expected move to be clamped")
	assert.Equal(t, 0, g.CursorTrack(), "expected track 0")
}

func TestMove_Diagonal(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(1, 2)
	assert.True(t, result.Changed(), "expected move to succeed")
	assert.True(t, result.RowChanged, "expected row to change")
	assert.True(t, result.TrackChanged, "expected track to change")
	assert.Equal(t, 2, g.CursorRow(), "expected row 2")
	assert.Equal(t, 1, g.CursorTrack(), "expected track 1")
}

func TestMove_ByViewportHeight(t *testing.T) {
	g := New(20, 4, 5)
	vh := g.ViewportHeight()
	g.Move(0, vh)
	assert.Equal(t, 5, g.CursorRow(), "expected row 5")
}

func TestViewportScrolls_WhenCursorMovesDown(t *testing.T) {
	g := New(20, 4, 5)
	// Move cursor to bottom of initial viewport (row 4)
	g.SetCursorPosition(4, 0)
	assert.Equal(t, 0, g.ViewportRow(), "expected viewport at 0")

	// Move one more down - viewport should scroll
	g.Move(0, 1)
	assert.Equal(t, 5, g.CursorRow(), "expected cursor at 5")
	assert.Equal(t, 1, g.ViewportRow(), "expected viewport at 1")
}

func TestViewportScrolls_WhenCursorMovesUp(t *testing.T) {
	g := New(20, 4, 5)
	g.SetCursorPosition(10, 0)
	viewportStart := g.ViewportRow()

	// Move up to viewport boundary
	g.SetCursorPosition(viewportStart, 0)
	assert.Equal(t, viewportStart, g.ViewportRow(), "viewport should not have moved yet")

	// Move one more up - viewport should scroll
	g.Move(0, -1)
	assert.Equal(t, viewportStart-1, g.ViewportRow(), "expected viewport to scroll up")
}

func TestSetCursorPosition_ClampsToValid(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(100, 100)
	assert.Equal(t, 9, g.CursorRow(), "expected row 9")
	assert.Equal(t, 3, g.CursorTrack(), "expected track 3")
}

func TestSetCursorPosition_ClampsNegative(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(-5, -5)
	assert.Equal(t, 0, g.CursorRow(), "expected row 0")
	assert.Equal(t, 0, g.CursorTrack(), "expected track 0")
}

func TestClearSelection_DeactivatesSelection(t *testing.T) {
	g := New(10, 4, 5)
	g.SelectAll()
	require.True(t, g.HasSelection(), "expected selection to be active")
	g.ClearSelection()
	assert.False(t, g.HasSelection(), "expected selection to be cleared")
}

func TestSelectAll_SelectsEntireGrid(t *testing.T) {
	g := New(10, 4, 5)
	g.SelectAll()
	require.True(t, g.HasSelection(), "expected selection to be active")
	minRow, maxRow, minTrack, maxTrack, _ := g.SelectionBounds()
	assert.Equal(t, 0, minRow)
	assert.Equal(t, 9, maxRow)
	assert.Equal(t, 0, minTrack)
	assert.Equal(t, 3, maxTrack)
}

func TestMoveExtending_StartsSelection(t *testing.T) {
	g := New(10, 4, 5)
	require.False(t, g.HasSelection(), "expected no initial selection")
	g.MoveExtending(0, 1)
	assert.True(t, g.HasSelection(), "expected selection to be started")
}

func TestMoveExtending_ExtendsSelection(t *testing.T) {
	g := New(10, 4, 5)
	g.MoveExtending(0, 1)
	g.MoveExtending(1, 1)

	minRow, maxRow, minTrack, maxTrack, hasSelection := g.SelectionBounds()
	require.True(t, hasSelection, "expected selection to be active")
	assert.Equal(t, 0, minRow)
	assert.Equal(t, 2, maxRow)
	assert.Equal(t, 0, minTrack)
	assert.Equal(t, 1, maxTrack)
}

func TestMove_ClearsSelection(t *testing.T) {
	g := New(10, 4, 5)
	g.SelectAll()
	g.Move(0, 1)
	assert.False(t, g.HasSelection(), "expected selection to be cleared after normal move")
}

func TestIsSelected_ReturnsTrueForSelectedCells(t *testing.T) {
	g := New(10, 4, 5)
	g.SelectAll()
	assert.True(t, g.IsSelected(5, 2), "expected (5,2) to be selected")
}

func TestIsSelected_ReturnsFalseWithoutSelection(t *testing.T) {
	g := New(10, 4, 5)
	assert.False(t, g.IsSelected(5, 2), "expected (5,2) to not be selected")
}

func TestSelectionBounds_NoSelection_ReturnsCursor(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(3, 2)
	minRow, maxRow, minTrack, maxTrack, hasSelection := g.SelectionBounds()
	assert.False(t, hasSelection, "expected hasSelection to be false")
	assert.Equal(t, 3, minRow)
	assert.Equal(t, 3, maxRow)
	assert.Equal(t, 2, minTrack)
	assert.Equal(t, 2, maxTrack)
}

func TestViewportBounds_ReturnsCorrectRange(t *testing.T) {
	g := New(20, 4, 5)
	first, last := g.viewportBounds()
	assert.Equal(t, 0, first)
	assert.Equal(t, 4, last)
}

func TestIsCursorVisible_TrueWhenInViewport(t *testing.T) {
	g := New(20, 4, 5)
	g.SetCursorPosition(2, 0)
	assert.True(t, g.isCursorVisible(), "expected cursor to be visible")
}

func TestIsCursorVisible_FalseWhenOutsideViewport(t *testing.T) {
	g := New(20, 4, 5)
	// Move cursor far down so viewport scrolls
	g.SetCursorPosition(15, 0)
	// Viewport should now be at row 11 (15 - viewportHeight + 1)
	// Now test if row 0 would be visible (it shouldn't be)
	g.cursorRow = 0 // directly set without adjusting viewport
	assert.False(t, g.isCursorVisible(), "expected cursor to not be visible")
}

func TestNumRows_ReturnsCorrectValue(t *testing.T) {
	g := New(10, 4, 5)
	assert.Equal(t, 10, g.numRows, "expected 10 rows")
}

func TestNumTracks_ReturnsCorrectValue(t *testing.T) {
	g := New(10, 4, 5)
	assert.Equal(t, 4, g.numTracks, "expected 4 tracks")
}

func TestViewportHeight_ReturnsCorrectValue(t *testing.T) {
	g := New(10, 4, 5)
	assert.Equal(t, 5, g.ViewportHeight(), "expected viewport height 5")
}

func TestEdgeCase_ZeroRows(t *testing.T) {
	g := New(0, 4, 5)
	assert.Equal(t, 0, g.CursorRow(), "expected cursor at 0")
	g.Move(0, 1)
	assert.Equal(t, 0, g.CursorRow(), "expected cursor to stay at 0")
}

func TestEdgeCase_ZeroTracks(t *testing.T) {
	g := New(10, 0, 5)
	assert.Equal(t, 0, g.CursorTrack(), "expected cursor at 0")
	g.Move(1, 0)
	assert.Equal(t, 0, g.CursorTrack(), "expected cursor to stay at 0")
}

func TestEdgeCase_ViewportLargerThanGrid(t *testing.T) {
	g := New(3, 4, 10)
	g.SetCursorPosition(2, 0)
	assert.Equal(t, 0, g.ViewportRow(), "expected viewport at 0")
}

func TestEdgeCase_SingleRowGrid(t *testing.T) {
	g := New(1, 4, 5)
	g.Move(0, 1)
	assert.Equal(t, 0, g.CursorRow(), "expected cursor to stay at 0")
}

func TestEdgeCase_SingleTrackGrid(t *testing.T) {
	g := New(10, 1, 5)
	g.Move(1, 0)
	assert.Equal(t, 0, g.CursorTrack(), "expected cursor to stay at 0")
}

func TestSelectedCells_WithSelection_ReturnsAllCells(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(2, 1)
	g.MoveExtending(1, 2) // move by deltaTrack=1, deltaRow=2 -> cursor moves to (4,2)

	cells := g.SelectedCells()

	expected := 6 // 3 rows * 2 tracks
	require.Len(t, cells, expected, "expected %d cells", expected)

	// Verify iteration order: rows outer, tracks inner
	expectedCells := []Cell{
		{2, 1}, {2, 2},
		{3, 1}, {3, 2},
		{4, 1}, {4, 2},
	}
	for i, c := range expectedCells {
		assert.Equal(t, c.Row, cells[i].Row, "cell %d: expected row %d", i, c.Row)
		assert.Equal(t, c.Track, cells[i].Track, "cell %d: expected track %d", i, c.Track)
	}
}

func TestSelectedCells_NoSelection_ReturnsCursorOnly(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(3, 2)

	cells := g.SelectedCells()

	require.Len(t, cells, 1, "expected 1 cell")
	assert.Equal(t, 3, cells[0].Row)
	assert.Equal(t, 2, cells[0].Track)
}

func TestSelectedCells_SingleCellSelection_ReturnsOne(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(5, 1)
	g.MoveExtending(0, 0) // start selection at current position

	cells := g.SelectedCells()

	require.Len(t, cells, 1, "expected 1 cell")
	assert.Equal(t, 5, cells[0].Row)
	assert.Equal(t, 1, cells[0].Track)
}
