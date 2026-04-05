package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
)

// TrackerScreen is the pattern editor screen.
// It wraps TrackerModel and exposes the Tracker field for main to access
// playback state and perform audio-related mutations.
type TrackerScreen struct {
	Tracker *TrackerModel
}

// NewTrackerScreen creates a new TrackerScreen wrapping the given tracker.
func NewTrackerScreen(tracker *TrackerModel) *TrackerScreen {
	return &TrackerScreen{Tracker: tracker}
}

func (t *TrackerScreen) Init() tea.Cmd { return nil }

// Update forwards keyboard events to the tracker. The delete key is handled
// here before forwarding so the tracker grid does not need to know about it.
func (t *TrackerScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "delete" {
			t.Tracker.SetNote(audio.Off())
			return t, nil
		}
		_, cmd := t.Tracker.Update(msg)
		return t, cmd
	}
	return t, nil
}

// View renders the tracker grid at full height inside a labelled panel.
func (t *TrackerScreen) View() string {
	return RenderPanel("Tracker", ColorAccentPrimary, t.Tracker.View(), true)
}

// Title returns the tab label for the TrackerScreen.
func (t *TrackerScreen) Title() string { return "Tracker" }

// ModeLabel returns the mode name shown in the persistent header.
func (t *TrackerScreen) ModeLabel() string { return "TRACK" }

// Footer returns the help text shown in the footer bar on the Tracker screen.
func (t *TrackerScreen) Footer() string {
	return "↑↓←→: Navigate | 1-7: Notes | Shift+1-6: Sharp Notes | Delete: Clear | +/-: Octave | [/]: Volume | p: Play/Pause | P: Loop | S: Save | L: Load | T: Synth | Q: Quit"
}
