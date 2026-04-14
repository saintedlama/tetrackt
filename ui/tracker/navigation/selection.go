package navigation

// Cell holds a row/track position in the grid.
type Cell struct {
	Row   int
	Track int
}

// selection represents a rectangular selection in a 2D grid.
type selection struct {
	active      bool
	anchorRow   int
	anchorTrack int
	endRow      int
	endTrack    int
}

// newSelection creates an inactive selection.
func newSelection() *selection {
	return &selection{
		active: false,
	}
}

// isActive returns true if the selection is active.
func (s *selection) isActive() bool {
	return s.active
}

// start begins a new selection anchored at the given position.
func (s *selection) start(row, track int) {
	s.active = true
	s.anchorRow = row
	s.anchorTrack = track
	s.endRow = row
	s.endTrack = track
}

// extend updates the selection end point to the given position.
func (s *selection) extend(row, track int) {
	if !s.active {
		return
	}
	s.endRow = row
	s.endTrack = track
}

// clear deactivates the selection.
func (s *selection) clear() {
	s.active = false
}

// bounds returns the normalized selection bounds (minRow, maxRow, minTrack, maxTrack).
// If selection is not active, returns the anchor position as both min and max.
func (s *selection) bounds() (minRow, maxRow, minTrack, maxTrack int) {
	if !s.active {
		return s.anchorRow, s.anchorRow, s.anchorTrack, s.anchorTrack
	}

	minRow = min(s.anchorRow, s.endRow)
	maxRow = max(s.anchorRow, s.endRow)
	minTrack = min(s.anchorTrack, s.endTrack)
	maxTrack = max(s.anchorTrack, s.endTrack)
	return
}

// contains returns true if the given position is within the selection bounds.
func (s *selection) contains(row, track int) bool {
	if !s.active {
		return false
	}

	minRow, maxRow, minTrack, maxTrack := s.bounds()
	return row >= minRow && row <= maxRow && track >= minTrack && track <= maxTrack
}

// cells returns all (row, track) positions in the selection.
// Returns nil if the selection is not active.
// Order: rows from min to max, tracks from min to max (outer loop: rows, inner loop: tracks).
func (s *selection) cells() []Cell {
	if !s.active {
		return nil
	}

	minRow, maxRow, minTrack, maxTrack := s.bounds()
	result := make([]Cell, 0, (maxRow-minRow+1)*(maxTrack-minTrack+1))
	for row := minRow; row <= maxRow; row++ {
		for track := minTrack; track <= maxTrack; track++ {
			result = append(result, Cell{Row: row, Track: track})
		}
	}
	return result
}
