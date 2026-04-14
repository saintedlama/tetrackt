package navigation

// Selection represents a rectangular selection in a 2D grid.
type Selection struct {
	active      bool
	anchorRow   int
	anchorTrack int
	endRow      int
	endTrack    int
}

// NewSelection creates an inactive selection.
func NewSelection() *Selection {
	return &Selection{
		active: false,
	}
}

// IsActive returns true if the selection is active.
func (s *Selection) IsActive() bool {
	return s.active
}

// Start begins a new selection anchored at the given position.
func (s *Selection) Start(row, track int) {
	s.active = true
	s.anchorRow = row
	s.anchorTrack = track
	s.endRow = row
	s.endTrack = track
}

// Extend updates the selection end point to the given position.
func (s *Selection) Extend(row, track int) {
	if !s.active {
		return
	}
	s.endRow = row
	s.endTrack = track
}

// Clear deactivates the selection.
func (s *Selection) Clear() {
	s.active = false
}

// Bounds returns the normalized selection bounds (minRow, maxRow, minTrack, maxTrack).
// If selection is not active, returns the anchor position as both min and max.
func (s *Selection) Bounds() (minRow, maxRow, minTrack, maxTrack int) {
	if !s.active {
		return s.anchorRow, s.anchorRow, s.anchorTrack, s.anchorTrack
	}

	minRow = min(s.anchorRow, s.endRow)
	maxRow = max(s.anchorRow, s.endRow)
	minTrack = min(s.anchorTrack, s.endTrack)
	maxTrack = max(s.anchorTrack, s.endTrack)
	return
}

// Contains returns true if the given position is within the selection bounds.
func (s *Selection) Contains(row, track int) bool {
	if !s.active {
		return false
	}

	minRow, maxRow, minTrack, maxTrack := s.Bounds()
	return row >= minRow && row <= maxRow && track >= minTrack && track <= maxTrack
}
