package synth

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

// OpenWavetableDialogMsg is emitted by an oscillator panel to request opening
// the wavetable picker dialog.
type OpenWavetableDialogMsg struct {
	BankIdx  int
	EntryIdx int
}

// WavetablePickedMsg is the payload carried by CloseDialogMsg when the user
// confirms a wavetable selection in the picker.
type WavetablePickedMsg struct {
	Entry    WavetableEntry
	BankIdx  int
	EntryIdx int
}

type wavetableFocus int

const (
	wtFocusBanks wavetableFocus = iota
	wtFocusEntries
)

const wtColumnWidth = 24

var (
	wtDialogHeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccentPrimary)
	wtDialogHelpStyle    = lipgloss.NewStyle().Foreground(common.ColorTextDisabled).Padding(1, 0)
	wtDialogDimStyle     = lipgloss.NewStyle().Foreground(common.ColorTextDisabled)
	wtDialogColumnStyle  = lipgloss.NewStyle().Width(wtColumnWidth + 2)
	wtDialogSubheadStyle = lipgloss.NewStyle().Foreground(common.ColorTextMuted)
)

// WavetableDialog is a two-column modal picker for wavetable banks and entries.
type WavetableDialog struct {
	banks    []string
	bankIdx  int
	entries  []WavetableEntry
	entryIdx int
	focus    wavetableFocus
	height   int
}

// NewWavetableDialog creates a WavetableDialog pre-selected at bankIdx/entryIdx.
func NewWavetableDialog(bankIdx, entryIdx int) *WavetableDialog {
	banks := WavetableBanksInOrder()
	if bankIdx >= len(banks) {
		bankIdx = 0
	}
	var entries []WavetableEntry
	if len(banks) > 0 {
		entries = WavetableEntriesForBank(banks[bankIdx])
	}
	if entryIdx >= len(entries) {
		entryIdx = 0
	}
	return &WavetableDialog{
		banks:    banks,
		bankIdx:  bankIdx,
		entries:  entries,
		entryIdx: entryIdx,
		focus:    wtFocusBanks,
	}
}

func (d *WavetableDialog) Init() tea.Cmd { return nil }

func (d *WavetableDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.height = msg.Height
		return d, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return d, func() tea.Msg { return ui.CloseDialogMsg{} }
		case "up":
			if d.focus == wtFocusBanks {
				if d.bankIdx > 0 {
					d.bankIdx--
					d.entries = WavetableEntriesForBank(d.banks[d.bankIdx])
					d.entryIdx = 0
				}
			} else {
				if d.entryIdx > 0 {
					d.entryIdx--
				}
			}
		case "down":
			if d.focus == wtFocusBanks {
				if d.bankIdx < len(d.banks)-1 {
					d.bankIdx++
					d.entries = WavetableEntriesForBank(d.banks[d.bankIdx])
					d.entryIdx = 0
				}
			} else {
				if d.entryIdx < len(d.entries)-1 {
					d.entryIdx++
				}
			}
		case "right":
			if d.focus == wtFocusBanks {
				d.focus = wtFocusEntries
			}
		case "left":
			if d.focus == wtFocusEntries {
				d.focus = wtFocusBanks
			}
		case "enter":
			if len(d.entries) == 0 {
				return d, nil
			}
			entry := d.entries[d.entryIdx]
			bankIdx, entryIdx := d.bankIdx, d.entryIdx
			return d, func() tea.Msg {
				return ui.CloseDialogMsg{Payload: WavetablePickedMsg{Entry: entry, BankIdx: bankIdx, EntryIdx: entryIdx}}
			}
		}
	}
	return d, nil
}

func (d *WavetableDialog) View() tea.View {
	// Reserve lines for: border (2), header+blank (2), column headers (1), help bar with padding (3).
	const overhead = 8
	visible := d.height - overhead
	if visible < 8 {
		visible = 8
	}

	var sb strings.Builder
	sb.WriteString(wtDialogHeaderStyle.Render("Wavetable"))
	sb.WriteString("\n\n")
	sb.WriteString(wtDialogSubheadStyle.Render(fmt.Sprintf("%-26s%-26s", "Bank", "Entry")))
	sb.WriteString("\n")

	bankNames := make([]string, len(d.banks))
	for i, b := range d.banks {
		bankNames[i] = b
	}
	bankLines := wtScrollList(bankNames, d.bankIdx, visible, d.focus == wtFocusBanks)

	entryNames := make([]string, len(d.entries))
	for i, e := range d.entries {
		name := strings.TrimPrefix(e.Name, "AKWF_")
		entryNames[i] = name
	}
	entryLines := wtScrollList(entryNames, d.entryIdx, visible, d.focus == wtFocusEntries)

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		wtDialogColumnStyle.Render(strings.Join(bankLines, "\n")),
		"  ",
		wtDialogColumnStyle.Render(strings.Join(entryLines, "\n")),
	))

	help := wtDialogHelpStyle.Render("↑↓: Navigate | ←→: Switch column | Enter: Apply | Esc: Close")
	return tea.NewView(fmt.Sprintf("%s\n%s", sb.String(), help))
}

// wtScrollList renders a scrolling list of items centred on the selected index.
// The selected item is highlighted based on whether the column is focused.
func wtScrollList(items []string, selected, maxVisible int, focused bool) []string {
	if len(items) == 0 {
		return []string{"(empty)"}
	}

	start := selected - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(items) {
		end = len(items)
		start = max(0, end-maxVisible)
	}

	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		label := items[i]
		if len(label) > wtColumnWidth {
			label = label[:wtColumnWidth]
		}
		label = fmt.Sprintf("%-*s", wtColumnWidth, label)
		if i == selected && focused {
			label = common.StyleSelected.Render(label)
		} else if i == selected {
			label = wtDialogDimStyle.Render(label)
		}
		lines = append(lines, label)
	}
	return lines
}
