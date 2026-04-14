package navigation

import "testing"

func TestNew_InitializesCursorAtOrigin(t *testing.T) {
	g := New(10, 4, 5)
	if row, track := g.cursorPosition(); row != 0 || track != 0 {
		t.Fatalf("expected cursor at (0,0), got (%d,%d)", row, track)
	}
}

func TestMove_Down_FromOrigin(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(0, 1)
	if !result.Changed() {
		t.Fatal("expected move to succeed")
	}
	if !result.RowChanged {
		t.Fatal("expected row to change")
	}
	if result.TrackChanged {
		t.Fatal("expected track to stay the same")
	}
	if g.CursorRow() != 1 {
		t.Fatalf("expected row 1, got %d", g.CursorRow())
	}
}

func TestMove_Down_AtBottom_Clamps(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(9, 0)
	result := g.Move(0, 1)
	if result.Changed() {
		t.Fatal("expected move to be clamped")
	}
	if g.CursorRow() != 9 {
		t.Fatalf("expected row 9, got %d", g.CursorRow())
	}
}

func TestMove_Up_AtTop_Clamps(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(0, -1)
	if result.Changed() {
		t.Fatal("expected move to be clamped")
	}
	if g.CursorRow() != 0 {
		t.Fatalf("expected row 0, got %d", g.CursorRow())
	}
}

func TestMove_Right_FromOrigin(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(1, 0)
	if !result.Changed() {
		t.Fatal("expected move to succeed")
	}
	if result.RowChanged {
		t.Fatal("expected row to stay the same")
	}
	if !result.TrackChanged {
		t.Fatal("expected track to change")
	}
	if g.CursorTrack() != 1 {
		t.Fatalf("expected track 1, got %d", g.CursorTrack())
	}
}

func TestMove_Right_AtRightmost_Clamps(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(0, 3)
	result := g.Move(1, 0)
	if result.Changed() {
		t.Fatal("expected move to be clamped")
	}
	if g.CursorTrack() != 3 {
		t.Fatalf("expected track 3, got %d", g.CursorTrack())
	}
}

func TestMove_Left_AtLeftmost_Clamps(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(-1, 0)
	if result.Changed() {
		t.Fatal("expected move to be clamped")
	}
	if g.CursorTrack() != 0 {
		t.Fatalf("expected track 0, got %d", g.CursorTrack())
	}
}

func TestMove_Diagonal(t *testing.T) {
	g := New(10, 4, 5)
	result := g.Move(1, 2)
	if !result.Changed() {
		t.Fatal("expected move to succeed")
	}
	if !result.RowChanged {
		t.Fatal("expected row to change")
	}
	if !result.TrackChanged {
		t.Fatal("expected track to change")
	}
	if g.CursorRow() != 2 || g.CursorTrack() != 1 {
		t.Fatalf("expected (2,1), got (%d,%d)", g.CursorRow(), g.CursorTrack())
	}
}

func TestMove_ByViewportHeight(t *testing.T) {
	g := New(20, 4, 5)
	vh := g.ViewportHeight()
	g.Move(0, vh)
	if g.CursorRow() != 5 {
		t.Fatalf("expected row 5, got %d", g.CursorRow())
	}
}

func TestViewportScrolls_WhenCursorMovesDown(t *testing.T) {
	g := New(20, 4, 5)
	// Move cursor to bottom of initial viewport (row 4)
	g.SetCursorPosition(4, 0)
	if g.ViewportRow() != 0 {
		t.Fatalf("expected viewport at 0, got %d", g.ViewportRow())
	}

	// Move one more down - viewport should scroll
	g.Move(0, 1)
	if g.CursorRow() != 5 {
		t.Fatalf("expected cursor at 5, got %d", g.CursorRow())
	}
	if g.ViewportRow() != 1 {
		t.Fatalf("expected viewport at 1, got %d", g.ViewportRow())
	}
}

func TestViewportScrolls_WhenCursorMovesUp(t *testing.T) {
	g := New(20, 4, 5)
	g.SetCursorPosition(10, 0)
	viewportStart := g.ViewportRow()

	// Move up to viewport boundary
	g.SetCursorPosition(viewportStart, 0)
	if g.ViewportRow() != viewportStart {
		t.Fatal("viewport should not have moved yet")
	}

	// Move one more up - viewport should scroll
	g.Move(0, -1)
	if g.ViewportRow() != viewportStart-1 {
		t.Fatalf("expected viewport to scroll up, got %d", g.ViewportRow())
	}
}

func TestSetCursorPosition_ClampsToValid(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(100, 100)
	if g.CursorRow() != 9 || g.CursorTrack() != 3 {
		t.Fatalf("expected (9,3), got (%d,%d)", g.CursorRow(), g.CursorTrack())
	}
}

func TestSetCursorPosition_ClampsNegative(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(-5, -5)
	if g.CursorRow() != 0 || g.CursorTrack() != 0 {
		t.Fatalf("expected (0,0), got (%d,%d)", g.CursorRow(), g.CursorTrack())
	}
}

func TestClearSelection_DeactivatesSelection(t *testing.T) {
	g := New(10, 4, 5)
	g.SelectAll()
	if !g.HasSelection() {
		t.Fatal("expected selection to be active")
	}
	g.ClearSelection()
	if g.HasSelection() {
		t.Fatal("expected selection to be cleared")
	}
}

func TestSelectAll_SelectsEntireGrid(t *testing.T) {
	g := New(10, 4, 5)
	g.SelectAll()
	if !g.HasSelection() {
		t.Fatal("expected selection to be active")
	}
	minRow, maxRow, minTrack, maxTrack, _ := g.SelectionBounds()
	if minRow != 0 || maxRow != 9 || minTrack != 0 || maxTrack != 3 {
		t.Fatalf("expected (0,9,0,3), got (%d,%d,%d,%d)", minRow, maxRow, minTrack, maxTrack)
	}
}

func TestMoveExtending_StartsSelection(t *testing.T) {
	g := New(10, 4, 5)
	if g.HasSelection() {
		t.Fatal("expected no initial selection")
	}
	g.MoveExtending(0, 1)
	if !g.HasSelection() {
		t.Fatal("expected selection to be started")
	}
}

func TestMoveExtending_ExtendsSelection(t *testing.T) {
	g := New(10, 4, 5)
	g.MoveExtending(0, 1)
	g.MoveExtending(1, 1)

	minRow, maxRow, minTrack, maxTrack, hasSelection := g.SelectionBounds()
	if !hasSelection {
		t.Fatal("expected selection to be active")
	}
	if minRow != 0 || maxRow != 2 || minTrack != 0 || maxTrack != 1 {
		t.Fatalf("expected (0,2,0,1), got (%d,%d,%d,%d)", minRow, maxRow, minTrack, maxTrack)
	}
}

func TestMove_ClearsSelection(t *testing.T) {
	g := New(10, 4, 5)
	g.SelectAll()
	g.Move(0, 1)
	if g.HasSelection() {
		t.Fatal("expected selection to be cleared after normal move")
	}
}

func TestIsSelected_ReturnsTrueForSelectedCells(t *testing.T) {
	g := New(10, 4, 5)
	g.SelectAll()
	if !g.IsSelected(5, 2) {
		t.Fatal("expected (5,2) to be selected")
	}
}

func TestIsSelected_ReturnsFalseWithoutSelection(t *testing.T) {
	g := New(10, 4, 5)
	if g.IsSelected(5, 2) {
		t.Fatal("expected (5,2) to not be selected")
	}
}

func TestSelectionBounds_NoSelection_ReturnsCursor(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(3, 2)
	minRow, maxRow, minTrack, maxTrack, hasSelection := g.SelectionBounds()
	if hasSelection {
		t.Fatal("expected hasSelection to be false")
	}
	if minRow != 3 || maxRow != 3 || minTrack != 2 || maxTrack != 2 {
		t.Fatalf("expected (3,3,2,2), got (%d,%d,%d,%d)", minRow, maxRow, minTrack, maxTrack)
	}
}

func TestViewportBounds_ReturnsCorrectRange(t *testing.T) {
	g := New(20, 4, 5)
	first, last := g.viewportBounds()
	if first != 0 || last != 4 {
		t.Fatalf("expected (0,4), got (%d,%d)", first, last)
	}
}

func TestIsCursorVisible_TrueWhenInViewport(t *testing.T) {
	g := New(20, 4, 5)
	g.SetCursorPosition(2, 0)
	if !g.isCursorVisible() {
		t.Fatal("expected cursor to be visible")
	}
}

func TestIsCursorVisible_FalseWhenOutsideViewport(t *testing.T) {
	g := New(20, 4, 5)
	// Move cursor far down so viewport scrolls
	g.SetCursorPosition(15, 0)
	// Viewport should now be at row 11 (15 - viewportHeight + 1)
	// Now test if row 0 would be visible (it shouldn't be)
	g.cursorRow = 0 // directly set without adjusting viewport
	if g.isCursorVisible() {
		t.Fatal("expected cursor to not be visible")
	}
}

func TestNumRows_ReturnsCorrectValue(t *testing.T) {
	g := New(10, 4, 5)
	if g.numRows != 10 {
		t.Fatalf("expected 10 rows, got %d", g.numRows)
	}
}

func TestNumTracks_ReturnsCorrectValue(t *testing.T) {
	g := New(10, 4, 5)
	if g.numTracks != 4 {
		t.Fatalf("expected 4 tracks, got %d", g.numTracks)
	}
}

func TestViewportHeight_ReturnsCorrectValue(t *testing.T) {
	g := New(10, 4, 5)
	if g.ViewportHeight() != 5 {
		t.Fatalf("expected viewport height 5, got %d", g.ViewportHeight())
	}
}

func TestEdgeCase_ZeroRows(t *testing.T) {
	g := New(0, 4, 5)
	if g.CursorRow() != 0 {
		t.Fatalf("expected cursor at 0, got %d", g.CursorRow())
	}
	g.Move(0, 1)
	if g.CursorRow() != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", g.CursorRow())
	}
}

func TestEdgeCase_ZeroTracks(t *testing.T) {
	g := New(10, 0, 5)
	if g.CursorTrack() != 0 {
		t.Fatalf("expected cursor at 0, got %d", g.CursorTrack())
	}
	g.Move(1, 0)
	if g.CursorTrack() != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", g.CursorTrack())
	}
}

func TestEdgeCase_ViewportLargerThanGrid(t *testing.T) {
	g := New(3, 4, 10)
	g.SetCursorPosition(2, 0)
	if g.ViewportRow() != 0 {
		t.Fatalf("expected viewport at 0, got %d", g.ViewportRow())
	}
}

func TestEdgeCase_SingleRowGrid(t *testing.T) {
	g := New(1, 4, 5)
	g.Move(0, 1)
	if g.CursorRow() != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", g.CursorRow())
	}
}

func TestEdgeCase_SingleTrackGrid(t *testing.T) {
	g := New(10, 1, 5)
	g.Move(1, 0)
	if g.CursorTrack() != 0 {
		t.Fatalf("expected cursor to stay at 0, got %d", g.CursorTrack())
	}
}

func TestSelectedCells_WithSelection_ReturnsAllCells(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(2, 1)
	g.MoveExtending(1, 2) // move by deltaTrack=1, deltaRow=2 -> cursor moves to (4,2)

	cells := g.SelectedCells()

	expected := 6 // 3 rows * 2 tracks
	if len(cells) != expected {
		t.Fatalf("expected %d cells, got %d", expected, len(cells))
	}

	// Verify iteration order: rows outer, tracks inner
	expectedCells := []Cell{
		{2, 1}, {2, 2},
		{3, 1}, {3, 2},
		{4, 1}, {4, 2},
	}
	for i, c := range expectedCells {
		if cells[i].Row != c.Row || cells[i].Track != c.Track {
			t.Fatalf("cell %d: expected (%d,%d), got (%d,%d)",
				i, c.Row, c.Track, cells[i].Row, cells[i].Track)
		}
	}
}

func TestSelectedCells_NoSelection_ReturnsCursorOnly(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(3, 2)

	cells := g.SelectedCells()

	if len(cells) != 1 {
		t.Fatalf("expected 1 cell, got %d", len(cells))
	}
	if cells[0].Row != 3 || cells[0].Track != 2 {
		t.Fatalf("expected (3,2), got (%d,%d)", cells[0].Row, cells[0].Track)
	}
}

func TestSelectedCells_SingleCellSelection_ReturnsOne(t *testing.T) {
	g := New(10, 4, 5)
	g.SetCursorPosition(5, 1)
	g.MoveExtending(0, 0) // start selection at current position

	cells := g.SelectedCells()

	if len(cells) != 1 {
		t.Fatalf("expected 1 cell, got %d", len(cells))
	}
	if cells[0].Row != 5 || cells[0].Track != 1 {
		t.Fatalf("expected (5,1), got (%d,%d)", cells[0].Row, cells[0].Track)
	}
}
