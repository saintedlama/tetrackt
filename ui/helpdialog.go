package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/ui/common"
)

// HelpEntry is a single key binding description row.
type HelpEntry struct {
	Key  string
	Desc string
}

// HelpSection is a named group of key bindings.
type HelpSection struct {
	Title   string
	Entries []HelpEntry
}

var (
	helpTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccentPrimary)
	helpSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccentWarning)
	helpKeyStyle     = lipgloss.NewStyle().Foreground(common.ColorAccentPrimary).Bold(true)
	helpDescStyle    = lipgloss.NewStyle().Foreground(common.ColorText)
)

// GlobalHelpSections lists shortcuts that are always available regardless of
// the active screen.
var GlobalHelpSections = []HelpSection{
	{
		Title: "Global",
		Entries: []HelpEntry{
			{"?", "Open this help"},
			{"t", "Toggle Tracker / Synth screen"},
			{"p", "Play / Pause from row 0"},
			{"P", "Loop current row"},
			{"s", "Save module"},
			{"l", "Load module"},
			{"+/-", "Octave up / down"},
			{"1–7", "Enter note (C D E F G A B)"},
			{"Shift+1–6", "Enter sharp note"},
			{"q / ctrl+c", "Quit"},
		},
	},
}

// HelpDialog renders global and current-screen keyboard shortcuts.
type HelpDialog struct {
	sections []HelpSection
}

// NewHelpDialog creates a HelpDialog combining the global sections with the
// screen-specific sections provided by the caller.
func NewHelpDialog(screenSections []HelpSection) *HelpDialog {
	all := make([]HelpSection, 0, len(GlobalHelpSections)+len(screenSections))
	all = append(all, GlobalHelpSections...)
	all = append(all, screenSections...)
	return &HelpDialog{sections: all}
}

func (d *HelpDialog) Init() tea.Cmd { return nil }

func (d *HelpDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "esc", "?", "q", "enter":
			return d, func() tea.Msg { return CloseDialogMsg{} }
		}
	}
	return d, nil
}

func (d *HelpDialog) View() tea.View {
	var b strings.Builder
	fmt.Fprintln(&b, helpTitleStyle.Render("Keyboard Shortcuts"))
	fmt.Fprintln(&b)

	for _, section := range d.sections {
		fmt.Fprintln(&b, helpSectionStyle.Render(section.Title))
		for _, e := range section.Entries {
			fmt.Fprintf(&b, "  %-18s %s\n",
				helpKeyStyle.Render(e.Key),
				helpDescStyle.Render(e.Desc),
			)
		}
		fmt.Fprintln(&b)
	}

	b.WriteString(common.StyleHelp.Render("[Esc / ? / Enter: Close]"))
	return tea.NewView(b.String())
}
