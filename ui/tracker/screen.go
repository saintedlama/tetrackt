package tracker

import (
	"fmt"
	"math"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

// VolumeChanged is emitted when the user adjusts global volume via the volume panel.
type VolumeChanged struct {
	Volume float64
}

// BPMChanged is emitted when the user adjusts BPM via the BPM panel.
type BPMChanged struct {
	BPM int
}

// TrackerScreen is the pattern editor screen.
// It wraps TrackerModel and exposes the Tracker field for main to access
// playback state and perform audio-related mutations.
type TrackerScreen struct {
	Tracker       *TrackerModel
	GlobalVolume  float64
	volumeBar     common.Bar
	activePanel   int // 0=tracker, 1=settings
	settingsFocus int // 0=volume, 1=bpm (within the settings panel)
}

const trackerPanelCount = 2

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

// Update handles tab navigation between tracker/volume panels and key events.
func (t *TrackerScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			t.activePanel = (t.activePanel + 1) % trackerPanelCount
			return t, nil
		case "shift+tab":
			t.activePanel = (t.activePanel - 1 + trackerPanelCount) % trackerPanelCount
			return t, nil
		}

		if t.activePanel == 1 {
			// Settings panel is active — up/down switches focus, left/right adjusts value
			switch msg.String() {
			case "up":
				t.settingsFocus = (t.settingsFocus - 1 + 2) % 2
				return t, nil
			case "down":
				t.settingsFocus = (t.settingsFocus + 1) % 2
				return t, nil
			}

			if t.settingsFocus == 0 {
				// Volume row focused
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

			// BPM row focused
			switch msg.String() {
			case "left":
				t.Tracker.BPM--
			case "shift+left":
				t.Tracker.BPM -= 10
			case "right":
				t.Tracker.BPM++
			case "shift+right":
				t.Tracker.BPM += 10
			}
			if t.Tracker.BPM < MinBPM {
				t.Tracker.BPM = MinBPM
			} else if t.Tracker.BPM > MaxBPM {
				t.Tracker.BPM = MaxBPM
			}
			return t, func() tea.Msg { return BPMChanged{BPM: t.Tracker.BPM} }
		}

		// Tracker panel is active
		if msg.String() == "delete" {
			t.Tracker.SetNote(audio.Off())
			return t, nil
		}
		_, cmd := t.Tracker.Update(msg)
		return t, cmd

	case ui.SynthUpdated:
		t.Tracker.Tracks[t.Tracker.CursorTrack].Synth = msg.Synth
		return t, nil
	}
	return t, nil
}

// View renders the tracker grid and a combined settings panel side by side.
func (t *TrackerScreen) View() string {
	render := func(focused bool, text string) string {
		if focused {
			return common.StyleSelected.Render(text)
		}
		return text
	}

	settingsPanelActive := t.activePanel == 1
	volumeRow := render(settingsPanelActive && t.settingsFocus == 0, "Volume") +
		fmt.Sprintf("  %s  %3d%%", t.volumeBar.View(), int(t.GlobalVolume*100))
	bpmRow := render(settingsPanelActive && t.settingsFocus == 1, "BPM") +
		fmt.Sprintf("     %3d", t.Tracker.BPM)
	settingsContent := volumeRow + "\n" + bpmRow

	currentSynth := t.Tracker.Tracks[t.Tracker.CursorTrack].Synth
	synthInfoContent := renderSynthInfo(currentSynth)

	trackerPanel := ui.RenderPanel("Tracker", common.ColorAccentPrimary, t.Tracker.View(), t.activePanel == 0)
	settingsPanel := ui.RenderPanel("Settings", common.ColorAccentModulation, settingsContent, t.activePanel == 1)
	synthInfoPanel := ui.RenderPanel("Synth", common.ColorAccentEnvelope, synthInfoContent, false)
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, settingsPanel, synthInfoPanel)
	return lipgloss.JoinHorizontal(lipgloss.Top, trackerPanel, rightColumn)
}

// Title returns the tab label for the TrackerScreen.
func (t *TrackerScreen) Title() string { return "Tracker" }

// Footer returns the help text shown in the footer bar on the Tracker screen.
func (t *TrackerScreen) Footer() string {
	return "↑↓←→: Navigate | 1-7: Notes | E: Row effects | p: Play | T: Synth | ?: Help"
}

// Help returns screen-specific keyboard shortcut sections for the help dialog.
func (t *TrackerScreen) Help() []ui.HelpSection {
	return []ui.HelpSection{
		{
			Title: "Tracker",
			Entries: []ui.HelpEntry{
				{"↑↓←→", "Navigate rows / tracks"},
				{"Home / End", "First / last row"},
				{"1–7", "Enter note (C D E F G A B)"},
				{"Shift+1–6", "Enter sharp note"},
				{"Delete", "Clear current cell"},
				{"+/-", "Octave up / down"},
				{"E", "Row effects dialog (arp, fx)"},
				{"Tab / Shift+Tab", "Switch tracker / settings panel"},
				{"p", "Play / Pause from row 0"},
				{"P", "Loop to current row"},
			},
		},
	}
}
