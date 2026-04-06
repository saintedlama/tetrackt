package ui

import (
	"fmt"
	"math"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/common"
)

// VolumeChanged is emitted when the user adjusts global volume via the volume panel.
type VolumeChanged struct {
	Volume float64
}

// TrackerScreen is the pattern editor screen.
// It wraps TrackerModel and exposes the Tracker field for main to access
// playback state and perform audio-related mutations.
type TrackerScreen struct {
	Tracker      *TrackerModel
	GlobalVolume float64
	volumeBar    common.Bar
	activePanel  int // 0=tracker, 1=volume
}

// NewTrackerScreen creates a new TrackerScreen wrapping the given tracker.
func NewTrackerScreen(tracker *TrackerModel) *TrackerScreen {
	return &TrackerScreen{
		Tracker:      tracker,
		GlobalVolume: 1.0,
		volumeBar:    common.NewBar(0, 1, 1.0, 10),
	}
}

// SetGlobalVolume updates the global volume display and bar.
func (t *TrackerScreen) SetGlobalVolume(v float64) {
	t.GlobalVolume = v
	t.volumeBar.Value = v
}

func (t *TrackerScreen) Init() tea.Cmd { return nil }

// Update handles tab navigation between tracker/volume panels and key events.
func (t *TrackerScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			t.activePanel = (t.activePanel + 1) % 2
			return t, nil
		case "shift+tab":
			t.activePanel = (t.activePanel - 1 + 2) % 2
			return t, nil
		}

		if t.activePanel == 1 {
			// Volume panel is active — adjust global volume
			switch msg.String() {
			case "left":
				t.GlobalVolume -= 0.05
			case "shift+left":
				t.GlobalVolume -= 0.1
			case "right":
				t.GlobalVolume += 0.05
			case "shift+right":
				t.GlobalVolume += 0.1
			}
			t.GlobalVolume = math.Round(t.GlobalVolume*100) / 100
			if t.GlobalVolume < 0 {
				t.GlobalVolume = 0
			} else if t.GlobalVolume > 1 {
				t.GlobalVolume = 1
			}
			t.volumeBar.Value = t.GlobalVolume
			return t, func() tea.Msg { return VolumeChanged{Volume: t.GlobalVolume} }
		}

		// Tracker panel is active
		if msg.String() == "delete" {
			t.Tracker.SetNote(audio.Off())
			return t, nil
		}
		_, cmd := t.Tracker.Update(msg)
		return t, cmd
	}
	return t, nil
}

// View renders the tracker grid and a global volume panel side by side.
func (t *TrackerScreen) View() string {
	volumeContent := fmt.Sprintf("%s  %3d%%", t.volumeBar.View(), int(t.GlobalVolume*100))
	trackerPanel := RenderPanel("Tracker", common.ColorAccentPrimary, t.Tracker.View(), t.activePanel == 0)
	volumePanel := RenderPanel("Volume", common.ColorAccentModulation, volumeContent, t.activePanel == 1)
	return lipgloss.JoinHorizontal(lipgloss.Top, trackerPanel, volumePanel)
}

// Title returns the tab label for the TrackerScreen.
func (t *TrackerScreen) Title() string { return "Tracker" }

// ModeLabel returns the mode name shown in the persistent header.
func (t *TrackerScreen) ModeLabel() string { return "TRACK" }

// Footer returns the help text shown in the footer bar on the Tracker screen.
func (t *TrackerScreen) Footer() string {
	return "Tab/Shift+Tab: Switch panel | ↑↓←→: Navigate | 1-7: Notes | Shift+1-6: Sharp Notes | Delete: Clear | +/-: Octave | p: Play/Pause | P: Loop | S: Save | L: Load | T: Synth | Q: Quit"
}
