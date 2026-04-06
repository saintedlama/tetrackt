package synth

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

// OpenSynthPresetDialogMsg is sent to request opening the synth preset picker.
type OpenSynthPresetDialogMsg struct{}

// PlaySynthPresetNoteMsg requests that a note be played using a specific
// synth preset's parameters rather than the current track's synth settings.
type PlaySynthPresetNoteMsg struct {
	Note   audio.Note
	Preset SynthPreset
}

var synthPresetDialogHelpStyle = lipgloss.NewStyle().
	Foreground(common.ColorTextDisabled).
	Padding(1, 0)

// SynthPresetsDialog is a standalone tea.Model that wraps SynthPresetView for
// use as an overlay dialog. Navigation keys are forwarded to the view;
// enter applies the selected preset; esc cancels; note keys (1–7 etc.)
// preview the selected preset by emitting a PassThroughMsg so audio
// is played without closing the dialog.
type SynthPresetsDialog struct {
	view   *SynthPresetView
	octave int
	height int // terminal height; updated via WindowSizeMsg
}

// NewSynthPresetsDialog creates a synth preset picker dialog. The caller passes a
// persistent *SynthPresetView so selection state is preserved across opens, and
// the current octave for note previews.
func NewSynthPresetsDialog(view *SynthPresetView, octave int) *SynthPresetsDialog {
	return &SynthPresetsDialog{view: view, octave: octave}
}

func (d *SynthPresetsDialog) Init() tea.Cmd { return nil }

func (d *SynthPresetsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.height = msg.Height
		return d, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return d, func() tea.Msg { return ui.CloseDialogMsg{} }
		case "enter":
			if len(d.view.Presets) == 0 {
				return d, nil
			}
			preset := d.view.Presets[d.view.SelectedPreset]
			return d, func() tea.Msg {
				return ui.CloseDialogMsg{Payload: ui.SynthUpdated{Synth: preset.Synth}}
			}
		default:
			// Note key → preview selected preset, keep dialog open
			if base, ok := ui.NoteKeys[msg.String()]; ok {
				if len(d.view.Presets) == 0 {
					return d, nil
				}
				note := audio.Note{Base: base, Octave: audio.Octave(d.octave)}
				preset := d.view.Presets[d.view.SelectedPreset]
				return d, func() tea.Msg {
					return ui.PassThroughMsg{Payload: PlaySynthPresetNoteMsg{Note: note, Preset: preset}}
				}
			}
			// Navigation keys — forward to the view (up/down/left/right)
			_, cmd := d.view.Update(msg)
			return d, cmd
		}
	}
	return d, nil
}

func (d *SynthPresetsDialog) View() tea.View {
	// Reserve lines for: dialog border (2), category header (1), help bar (3),
	// plus a margin so the dialog never touches the screen edges.
	const overhead = 10
	visible := d.height - overhead
	if visible < 4 {
		visible = 4
	}
	d.view.MaxHeight = visible
	content := fmt.Sprintf("%s\n%s", d.view.View(), synthPresetDialogHelpStyle.Render("↑↓: Navigate | ←→: Category | Enter: Apply | Esc: Close | 1-7: Preview"))
	return tea.NewView(content)
}
