package tracker

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCtrlArrowSwitchesPanels(t *testing.T) {
	s := NewTrackerScreen(NewTracker(2, 8, 80, 24))

	_, _ = s.Update(tea.KeyPressMsg{Text: "ctrl+right"})
	if s.activePanel != 1 {
		t.Fatalf("expected settings panel active after ctrl+right, got %d", s.activePanel)
	}

	_, _ = s.Update(tea.KeyPressMsg{Text: "ctrl+left"})
	if s.activePanel != 0 {
		t.Fatalf("expected tracker panel active after ctrl+left, got %d", s.activePanel)
	}
}
