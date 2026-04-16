package tracker

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
)

func TestCtrlArrowSwitchesPanels(t *testing.T) {
	s := NewTrackerScreen(NewTracker(2, 8, 80, 24))

	_, _ = s.Update(tea.KeyPressMsg{Text: "ctrl+right"})
	assert.Equal(t, 1, s.activePanel, "expected settings panel active after ctrl+right")

	_, _ = s.Update(tea.KeyPressMsg{Text: "ctrl+left"})
	assert.Equal(t, 0, s.activePanel, "expected tracker panel active after ctrl+left")
}
