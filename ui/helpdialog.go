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

// HelpDialog renders keyboard shortcuts.
type HelpDialog struct {
	sections []HelpSection
	width    int
	height   int
}

// NewHelpDialog creates a HelpDialog from pre-built help sections.
func NewHelpDialog(sections []HelpSection) *HelpDialog {
	all := make([]HelpSection, len(sections))
	copy(all, sections)
	return &HelpDialog{sections: all, width: 120, height: 32}
}

func (d *HelpDialog) Init() tea.Cmd { return nil }

func (d *HelpDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		d.width = msg.Width
		d.height = msg.Height
		return d, nil
	}

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "esc", "?", "q", "enter":
			return d, func() tea.Msg { return CloseDialogMsg{} }
		}
	}
	return d, nil
}

func (d *HelpDialog) View() tea.View {
	renderSection := func(section HelpSection) string {
		var sb strings.Builder
		fmt.Fprintln(&sb, helpSectionStyle.Render(section.Title))
		for _, e := range section.Entries {
			if e.Key == "" && e.Desc == "" {
				sb.WriteByte('\n')
				continue
			}
			fmt.Fprintf(&sb, "  %-18s %s\n",
				helpKeyStyle.Render(e.Key),
				helpDescStyle.Render(e.Desc),
			)
		}
		return strings.TrimRight(sb.String(), "\n")
	}

	leftCol := ""
	rightBlocks := make([]string, 0, len(d.sections))
	for i, section := range d.sections {
		block := renderSection(section)
		if i == 0 {
			leftCol = block
			continue
		}
		rightBlocks = append(rightBlocks, block)
	}

	body := leftCol
	if len(rightBlocks) > 0 {
		rightCol := strings.Join(rightBlocks, "\n\n")
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "    ", rightCol)
	}

	var b strings.Builder
	fmt.Fprintln(&b, helpTitleStyle.Render("Keyboard Shortcuts"))
	fmt.Fprintln(&b)
	b.WriteString(body)
	b.WriteString("\n\n")

	b.WriteString(common.StyleHelp.Render("[Esc / ? / Enter: Close]"))
	return tea.NewView(b.String())
}
