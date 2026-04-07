package tracker

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

// RowEffectsApplied is the result payload when the user confirms the row effects dialog.
type RowEffectsApplied struct {
	TrackIdx int
	RowIdx   int
	Ticks    int
	Arpeggio audio.ArpeggioEffect
}

const (
	minTicks    = 1
	maxTicks    = 32
	maxSemitone = 24
	minSemitone = -24
)

// RowEffectsDialog lets the user edit per-row tick count and arpeggio settings.
//
// Navigation:
//
//	Field 0    — Ticks (←/→ to adjust)
//	Field 1    — ARP on/off (Space to toggle)
//	Fields 2…N — one semitone offset per tick (visible only when ARP is on)
type RowEffectsDialog struct {
	trackIdx   int
	rowIdx     int
	ticks      int
	arpEnabled bool
	offsets    []int // len >= ticks when arp is on; extra tail entries preserved on shrink
	focusField int   // 0=ticks, 1=arp toggle, 2..ticks+1=offset[i]
}

// NewRowEffectsDialog creates a dialog pre-populated with the current row's settings.
func NewRowEffectsDialog(row TrackRow, trackIdx, rowIdx int) *RowEffectsDialog {
	ticks := row.Ticks
	if ticks <= 0 {
		ticks = DefaultSpeed / 2
		if ticks < 1 {
			ticks = 1
		}
	}
	offsets := make([]int, ticks)
	if row.Arpeggio.IsActive() {
		copy(offsets, row.Arpeggio.Offsets)
	}
	return &RowEffectsDialog{
		trackIdx:   trackIdx,
		rowIdx:     rowIdx,
		ticks:      ticks,
		arpEnabled: row.Arpeggio.IsActive(),
		offsets:    offsets,
	}
}

func (d *RowEffectsDialog) Init() tea.Cmd { return nil }

func (d *RowEffectsDialog) numFields() int {
	if d.arpEnabled {
		return 2 + d.ticks
	}
	return 2
}

func (d *RowEffectsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return d, func() tea.Msg { return ui.CloseDialogMsg{} }
		case "enter":
			ticks, arp := d.build()
			trackIdx, rowIdx := d.trackIdx, d.rowIdx
			return d, func() tea.Msg {
				return ui.CloseDialogMsg{Payload: RowEffectsApplied{
					TrackIdx: trackIdx,
					RowIdx:   rowIdx,
					Ticks:    ticks,
					Arpeggio: arp,
				}}
			}
		case "up":
			d.focusField = (d.focusField - 1 + d.numFields()) % d.numFields()
		case "down":
			d.focusField = (d.focusField + 1) % d.numFields()
		case "left":
			d.adjustFocused(-1)
		case "shift+left":
			d.adjustFocused(-6)
		case "right":
			d.adjustFocused(1)
		case "shift+right":
			d.adjustFocused(6)
		case "delete":
			d.arpEnabled = false
			d.offsets = make([]int, d.ticks)
			if d.focusField >= d.numFields() {
				d.focusField = 1
			}
		}
	}
	return d, nil
}

func (d *RowEffectsDialog) growOffsets() {
	for len(d.offsets) < d.ticks {
		d.offsets = append(d.offsets, 0)
	}
}

func (d *RowEffectsDialog) adjustFocused(delta int) {
	switch d.focusField {
	case 0:
		d.ticks = max(minTicks, min(maxTicks, d.ticks+delta))
		d.growOffsets()
		// clamp focus in case ticks shrank below current offset field
		if d.focusField >= d.numFields() {
			d.focusField = d.numFields() - 1
		}
	case 1:
		if delta < 0 {
			d.arpEnabled = false
		} else {
			d.arpEnabled = true
			d.growOffsets()
		}
		if !d.arpEnabled && d.focusField >= d.numFields() {
			d.focusField = 1
		}
	default:
		i := d.focusField - 2
		if i < len(d.offsets) {
			d.offsets[i] = max(minSemitone, min(maxSemitone, d.offsets[i]+delta))
		}
	}
}

func (d *RowEffectsDialog) build() (int, audio.ArpeggioEffect) {
	if !d.arpEnabled {
		return d.ticks, audio.ArpeggioEffect{}
	}
	offsets := make([]int, d.ticks)
	copy(offsets, d.offsets)
	return d.ticks, audio.ArpeggioEffect{Offsets: offsets}
}

func (d *RowEffectsDialog) View() tea.View {
	render := func(focused bool, text string) string {
		if focused {
			return common.StyleSelected.Render(text)
		}
		return text
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Row Effects — Track %d, Row %02d\n\n", d.trackIdx+1, d.rowIdx)

	arpStatus := "OFF"
	if d.arpEnabled {
		arpStatus = "ON "
	}

	b.WriteString(render(d.focusField == 0, fmt.Sprintf("Ticks      %3d  sub-ticks per step", d.ticks)))
	b.WriteByte('\n')
	b.WriteString(render(d.focusField == 1, fmt.Sprintf("ARP mode   %s", arpStatus)))
	b.WriteByte('\n')

	if d.arpEnabled {
		b.WriteString("\n")
		for i := 0; i < d.ticks; i++ {
			offset := 0
			if i < len(d.offsets) {
				offset = d.offsets[i]
			}
			b.WriteString(render(d.focusField == i+2, fmt.Sprintf("Tick %2d    %+4d  semitones", i+1, offset)))
			b.WriteByte('\n')
		}
	}

	b.WriteString("\n")
	b.WriteString("[Enter: Apply | Esc: Cancel | ↑↓: Navigate | ←→: ±1 / ON-OFF | Shift+←→: ±6 | Del: Clear]")
	return tea.NewView(b.String())
}
