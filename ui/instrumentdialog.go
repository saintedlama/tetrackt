package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/ui/common"
	"github.com/tetrackt/tetrackt/audio"
)

// OpenInstrumentDialogMsg is sent to request opening the instrument picker.
type OpenInstrumentDialogMsg struct{}

var instrumentDialogHelpStyle = lipgloss.NewStyle().
	Foreground(common.ColorTextDisabled).
	Padding(1, 0)

// InstrumentDialog is a standalone tea.Model that wraps InstrumentView for
// use as an overlay dialog. Navigation keys are forwarded to the view;
// enter applies the selected instrument; esc cancels; note keys (1–7 etc.)
// preview the selected instrument by emitting a PassThroughMsg so audio
// is played without closing the dialog.
type InstrumentDialog struct {
	view   *InstrumentView
	octave int
	height int // terminal height; updated via WindowSizeMsg
}

// NewInstrumentDialog creates an instrument picker dialog. The caller passes a
// persistent *InstrumentView so selection state is preserved across opens, and
// the current octave for note previews.
func NewInstrumentDialog(view *InstrumentView, octave int) *InstrumentDialog {
	return &InstrumentDialog{view: view, octave: octave}
}

func (d *InstrumentDialog) Init() tea.Cmd { return nil }

func (d *InstrumentDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.height = msg.Height
		return d, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return d, func() tea.Msg { return CloseDialogMsg{} }
		case "enter":
			if len(d.view.Presets) == 0 {
				return d, nil
			}
			preset := d.view.Presets[d.view.SelectedPreset]
			return d, func() tea.Msg {
				return CloseDialogMsg{Payload: InstrumentApplied{Instrument: preset}}
			}
		default:
			// Note key → preview selected instrument, keep dialog open
			if base, ok := NoteKeys[msg.String()]; ok {
				if len(d.view.Presets) == 0 {
					return d, nil
				}
				note := audio.Note{Base: base, Octave: audio.Octave(d.octave)}
				preset := d.view.Presets[d.view.SelectedPreset]
				return d, func() tea.Msg {
					return PassThroughMsg{Payload: PlayInstrumentNoteMsg{Note: note, Instrument: preset}}
				}
			}
			// Navigation keys — forward to the view (up/down/left/right)
			_, cmd := d.view.Update(msg)
			return d, cmd
		}
	}
	return d, nil
}

func (d *InstrumentDialog) View() tea.View {
	// Reserve lines for: dialog border (2), category header (1), help bar (3),
	// plus a margin so the dialog never touches the screen edges.
	const overhead = 10
	visible := d.height - overhead
	if visible < 4 {
		visible = 4
	}
	d.view.MaxHeight = visible
	content := fmt.Sprintf("%s\n%s", d.view.View(), instrumentDialogHelpStyle.Render("↑↓: Navigate | ←→: Category | Enter: Apply | Esc: Close | 1-7: Preview"))
	return tea.NewView(content)
}
