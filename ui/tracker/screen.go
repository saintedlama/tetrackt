package tracker

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

// OpenRowEffectsMsg asks main to open the row effects dialog for a cell.
type OpenRowEffectsMsg struct {
	TrackIdx int
	RowIdx   int
	Row      TrackRow
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

// Update routes key events between tracker grid and settings panel.
func (t *TrackerScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		handled, cmd := ui.DispatchShortcutSections(msg, t.shortcuts())
		if handled {
			return t, cmd
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
		_, cmd = t.Tracker.Update(msg)
		return t, cmd

	case ui.SynthUpdated:
		track := &t.Tracker.Tracks[t.Tracker.CursorTrack]
		track.Synth = msg.Synth
		if msg.HasPatchMeta {
			track.PatchName = msg.PatchName
			track.PatchCategory = msg.PatchCategory
			track.PatchTags = append([]string(nil), msg.PatchTags...)
		}
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

	currentTrack := t.Tracker.Tracks[t.Tracker.CursorTrack]
	synthInfoContent := renderSynthInfo(currentTrack.Synth, currentTrack.PatchName, currentTrack.PatchCategory, currentTrack.PatchTags)

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
	mode := "NAV"
	if t.Tracker.Mode == EditMode {
		mode = "EDIT"
	}
	return fmt.Sprintf("%s | Space: Mode | Tab: Column | Ctrl+←→: Panel | Ctrl+E: Effects | p: Play | Ctrl+T: Synth | ?: Help", mode)
}

// Help returns screen-specific keyboard shortcut sections for the help dialog.
func (t *TrackerScreen) Help() []ui.HelpSection {
	sections := append([]ui.ShortcutSection(nil), t.shortcuts()...)
	profile := ui.CurrentInputProfile()
	lower, upper := ui.NoteMappingRows(profile)

	sections = append(sections, ui.ShortcutSection{
		Title: "Tracker",
		Shortcuts: []ui.Shortcut{
			{KeyLabel: "Space", Description: "Toggle Navigate / Edit mode"},
			{KeyLabel: "↑↓←→", Description: "Navigate rows / tracks"},
			{KeyLabel: "Tab / Shift+Tab", Description: "Move subcolumn focus"},
			{},
			{KeyLabel: "PgUp / PgDn", Description: "Jump by viewport height"},
			{KeyLabel: "Home / End", Description: "First / last row"},
			{},
			{KeyLabel: "0-9 / A-F", Description: "Hex entry for volume/fx columns"},
			{KeyLabel: "FX cmd (0..7)", Description: "0 none, 1 vib, 2 volslide, 3 cut, 4 delay, 5 ticks, 6 cont, 7 arp"},
			{KeyLabel: "FX aliases", Description: "V S C D T O A in Effect column"},
			{KeyLabel: "FX param", Description: "2 hex nibbles in Param column"},
			{KeyLabel: "Delete", Description: "Clear focused subcolumn"},
			{},
			{KeyLabel: "Shift+Arrows", Description: "Rectangular selection"},
			{KeyLabel: "Ctrl+A", Description: "Select full pattern"},
			{KeyLabel: "Alt+C / Alt+X / Alt+V", Description: "Copy / cut / paste block"},
			{KeyLabel: "Alt+Shift+V", Description: "Paste effects only (keep note)"},
			{},
			{KeyLabel: "Alt+↑/F8 / Alt+↓/F7", Description: "Transpose notes +/- 1 semitone"},
			{KeyLabel: "Alt+Shift+↑/Shift+F8 / Alt+Shift+↓/Shift+F7", Description: "Transpose notes +/- 1 octave"},
			{},
			{KeyLabel: "Insert / Shift+Insert", Description: "Insert track / global row space"},
		},
	})

	help := ui.HelpSectionsFromShortcutSections(sections)
	help = append(help, ui.HelpSection{
		Title: fmt.Sprintf("Note Mapping (%s)", strings.ToUpper(string(profile))),
		Entries: []ui.HelpEntry{
			{Key: "Lower row", Desc: lower},
			{Key: "Upper row", Desc: upper},
		},
	},
	)

	return help
}

func (t *TrackerScreen) shortcuts() []ui.ShortcutSection {
	return []ui.ShortcutSection{
		{
			Title: "Tracker Panels",
			Shortcuts: []ui.Shortcut{
				{
					Keys:        []string{"ctrl+right"},
					Description: "Focus next tracker panel",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						t.activePanel = (t.activePanel + 1) % trackerPanelCount
						return true, nil
					},
				},
				{
					Keys:        []string{"ctrl+e"},
					Description: "Open advanced row effects editor",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						row := t.Tracker.Tracks[t.Tracker.CursorTrack].Rows[t.Tracker.CursorRow]
						msg := OpenRowEffectsMsg{TrackIdx: t.Tracker.CursorTrack, RowIdx: t.Tracker.CursorRow, Row: row}
						return true, func() tea.Msg { return msg }
					},
				},
				{
					Keys:        []string{"ctrl+left"},
					Description: "Focus previous tracker panel",
					Action: func(_ tea.KeyPressMsg) (bool, tea.Cmd) {
						t.activePanel = (t.activePanel - 1 + trackerPanelCount) % trackerPanelCount
						return true, nil
					},
				},
			},
		},
	}
}
