package synth

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/notes"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

// OpenPatchBankMsg is sent to request opening the patch bank dialog.
type OpenPatchBankMsg struct{}

// PlayPatchNoteMsg requests that a note be played using a specific patch's
// parameters rather than the current track's synth settings.
type PlayPatchNoteMsg struct {
	Note  notes.Note
	Patch SynthPatch
}

// PatchSaveRequestedMsg is emitted when the user confirms saving a new patch.
type PatchSaveRequestedMsg struct {
	Name  string
	Bank  string
	Tags  []string
	Synth *audio.Synth
}

// PatchDeleteRequestedMsg is emitted when the user confirms deleting a custom patch.
type PatchDeleteRequestedMsg struct {
	PatchName string
}

// PatchRenameRequestedMsg is emitted when the user confirms renaming a custom patch.
type PatchRenameRequestedMsg struct {
	OldName string
	NewName string
}

type patchBankMode int

const (
	modeBrowse   patchBankMode = iota
	modeSaveName               // entering patch name
	modeSaveBank               // entering patch bank
	modeSaveTags               // entering comma-separated tags
	modeRename                 // renaming a custom patch
)

var patchBankDialogHelpStyle = lipgloss.NewStyle().
	Foreground(common.ColorTextDisabled).
	Padding(1, 0)

// SynthPatchBankDialog is a standalone tea.Model for browsing and managing patches.
// It operates in one of four modes: browse, save-name, save-bank, and rename.
type SynthPatchBankDialog struct {
	view         *SynthPatchBankView
	octave       int
	height       int
	mode         patchBankMode
	input        string       // current text input
	saveName     string       // patch name captured during modeSaveName
	saveBank     string       // bank captured during modeSaveBank
	currentSynth *audio.Synth // synth snapshot used when saving
}

// NewSynthPatchBankDialog creates a patch bank dialog. The caller passes a
// persistent *SynthPatchBankView so selection state is preserved across opens,
// the current octave for note previews, and the current synth for saving.
func NewSynthPatchBankDialog(view *SynthPatchBankView, octave int, currentSynth *audio.Synth) *SynthPatchBankDialog {
	return &SynthPatchBankDialog{view: view, octave: octave, currentSynth: currentSynth}
}

func (d *SynthPatchBankDialog) Init() tea.Cmd { return nil }

func (d *SynthPatchBankDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.height = msg.Height
		return d, nil
	case tea.KeyPressMsg:
		switch d.mode {
		case modeSaveName, modeSaveBank, modeSaveTags, modeRename:
			return d.updateTextInput(msg)
		default:
			return d.updateBrowse(msg)
		}
	}
	return d, nil
}

func (d *SynthPatchBankDialog) updateBrowse(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return d, func() tea.Msg { return ui.CloseDialogMsg{} }
	case "enter":
		// Only apply when focus is on a patch in the list.
		if d.view.focus != focusList || len(d.view.Patches) == 0 {
			_, cmd := d.view.Update(msg)
			return d, cmd
		}
		patch := d.view.Patches[d.view.SelectedPatch]
		return d, func() tea.Msg {
			return ui.CloseDialogMsg{Payload: ui.SynthUpdated{
				Synth: patch.Synth,
			}}
		}
	case "up", "down", "left", "right":
		// All navigation is handled by the view.
		_, cmd := d.view.Update(msg)
		return d, cmd
	case "s":
		d.mode = modeSaveName
		d.input = ""
		return d, nil
	case "r":
		if len(d.view.Patches) == 0 {
			return d, nil
		}
		patch := d.view.Patches[d.view.SelectedPatch]
		if !patch.IsCustom() {
			return d, nil // built-in patches cannot be renamed
		}
		d.mode = modeRename
		d.input = patch.Name
		return d, nil
	case "d":
		if len(d.view.Patches) == 0 {
			return d, nil
		}
		patch := d.view.Patches[d.view.SelectedPatch]
		if !patch.IsCustom() {
			return d, nil // built-in patches cannot be deleted
		}
		name := patch.Name
		return d, func() tea.Msg {
			return ui.PassThroughMsg{Payload: PatchDeleteRequestedMsg{PatchName: name}}
		}
	default:
		// Note key → preview selected patch, keep dialog open
		if base, ok := ui.NoteKeys[msg.String()]; ok {
			if d.view.focus != focusList || len(d.view.Patches) == 0 {
				return d, nil
			}
			note := notes.Note{Base: base, Octave: notes.Octave(d.octave)}
			patch := d.view.Patches[d.view.SelectedPatch]
			return d, func() tea.Msg {
				return ui.PassThroughMsg{Payload: PlayPatchNoteMsg{Note: note, Patch: patch}}
			}
		}
		return d, nil
	}
}

func (d *SynthPatchBankDialog) updateTextInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.mode = modeBrowse
		d.input = ""
		return d, nil
	case "enter":
		switch d.mode {
		case modeSaveName:
			if strings.TrimSpace(d.input) == "" {
				return d, nil
			}
			d.saveName = strings.TrimSpace(d.input)
			d.mode = modeSaveBank
			d.input = ""
		case modeSaveBank:
			d.saveBank = strings.TrimSpace(d.input)
			d.mode = modeSaveTags
			d.input = ""
		case modeSaveTags:
			name := d.saveName
			bank := d.saveBank
			var tags []string
			for _, t := range strings.Split(d.input, ",") {
				if s := strings.TrimSpace(t); s != "" {
					tags = append(tags, s)
				}
			}
			currentSynth := d.currentSynth
			d.mode = modeBrowse
			d.input = ""
			d.saveName = ""
			d.saveBank = ""
			return d, func() tea.Msg {
				return ui.PassThroughMsg{Payload: PatchSaveRequestedMsg{
					Name:  name,
					Bank:  bank,
					Tags:  tags,
					Synth: currentSynth,
				}}
			}
		case modeRename:
			if strings.TrimSpace(d.input) == "" {
				return d, nil
			}
			if len(d.view.Patches) == 0 {
				d.mode = modeBrowse
				return d, nil
			}
			oldName := d.view.Patches[d.view.SelectedPatch].Name
			newName := strings.TrimSpace(d.input)
			d.mode = modeBrowse
			d.input = ""
			return d, func() tea.Msg {
				return ui.PassThroughMsg{Payload: PatchRenameRequestedMsg{OldName: oldName, NewName: newName}}
			}
		}
		return d, nil
	case "backspace":
		if len(d.input) > 0 {
			d.input = d.input[:len(d.input)-1]
		}
		return d, nil
	default:
		if len(msg.String()) == 1 {
			d.input += msg.String()
		}
		return d, nil
	}
}

func (d *SynthPatchBankDialog) View() tea.View {
	// Reserve lines for: rounded border (2), help bar with padding (3), filter rows (3), blank line (1).
	const overhead = 9
	visible := d.height - overhead
	if visible < 10 {
		visible = 10
	}
	d.view.MaxHeight = visible

	bodyStyle := lipgloss.NewStyle().Width(50).Height(visible)

	var body, help string
	switch d.mode {
	case modeSaveName:
		body = fmt.Sprintf("Patch Bank\n\nSave patch — Name: %s_", d.input)
		help = patchBankDialogHelpStyle.Render("Type name | Enter: Next | Esc: Cancel")
	case modeSaveBank:
		body = fmt.Sprintf("Patch Bank\n\nSave patch — Bank (optional): %s_", d.input)
		help = patchBankDialogHelpStyle.Render("Type bank | Enter: Next | Esc: Cancel")
	case modeSaveTags:
		body = fmt.Sprintf("Patch Bank\n\nSave patch — Tags, comma-separated (optional): %s_", d.input)
		help = patchBankDialogHelpStyle.Render("e.g. NES, C64 | Enter: Save | Esc: Cancel")
	case modeRename:
		body = fmt.Sprintf("Patch Bank\n\nRename: %s_", d.input)
		help = patchBankDialogHelpStyle.Render("Type new name | Enter: Confirm | Esc: Cancel")
	default:
		body = d.view.View()
		help = patchBankDialogHelpStyle.Render("↑↓: Navigate | ←→: Category | Enter: Apply | S: Save | R: Rename | D: Delete | Z-M/Q-U: Preview | Esc: Close")
	}

	return tea.NewView(fmt.Sprintf("%s\n%s", bodyStyle.Render(body), help))
}
