package tracker

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
)

// RowEffectsApplied is the result payload when the user confirms the row effects dialog.
type RowEffectsApplied struct {
	TrackIdx int
	RowIdx   int
	Arpeggio audio.ArpeggioEffect
}

// RowEffectsDialog lets the user edit per-row arpeggio settings.
// Three editable fields:
//   - Note 2 offset  — semitone shift for the 2nd arp step (0 = arp off)
//   - Note 3 offset  — semitone shift for the 3rd arp step (0 = 2-note arp)
//   - Ticks/step     — sub-ticks before advancing to the next arp step
type RowEffectsDialog struct {
	trackIdx     int
	rowIdx       int
	offset1      int // semitones for 2nd arp note (0 = arp inactive)
	offset2      int // semitones for 3rd arp note (0 = 2-note arp only)
	ticksPerStep int
	focusField   int // 0 = offset1, 1 = offset2, 2 = ticksPerStep
}

// NewRowEffectsDialog creates a dialog pre-populated with the arpeggio from the given row.
func NewRowEffectsDialog(arp audio.ArpeggioEffect, trackIdx, rowIdx int) *RowEffectsDialog {
	d := &RowEffectsDialog{
		trackIdx:     trackIdx,
		rowIdx:       rowIdx,
		ticksPerStep: arp.TicksPerStep,
	}
	if len(arp.Offsets) > 1 {
		d.offset1 = arp.Offsets[1]
	}
	if len(arp.Offsets) > 2 {
		d.offset2 = arp.Offsets[2]
	}
	if d.ticksPerStep <= 0 {
		d.ticksPerStep = DefaultSpeed / 2
		if d.ticksPerStep < 1 {
			d.ticksPerStep = 1
		}
	}
	return d
}

func (d *RowEffectsDialog) Init() tea.Cmd { return nil }

func (d *RowEffectsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return d, func() tea.Msg { return ui.CloseDialogMsg{} }
		case "enter":
			arp := d.buildArpeggio()
			trackIdx, rowIdx := d.trackIdx, d.rowIdx
			return d, func() tea.Msg {
				return ui.CloseDialogMsg{Payload: RowEffectsApplied{
					TrackIdx: trackIdx,
					RowIdx:   rowIdx,
					Arpeggio: arp,
				}}
			}
		case "up":
			d.focusField = (d.focusField - 1 + 3) % 3
		case "down":
			d.focusField = (d.focusField + 1) % 3
		case "left":
			d.adjustFocused(-1)
		case "shift+left":
			d.adjustFocused(-6)
		case "right":
			d.adjustFocused(1)
		case "shift+right":
			d.adjustFocused(6)
		case "delete":
			// Clear arpeggio
			d.offset1 = 0
			d.offset2 = 0
		}
	}
	return d, nil
}

func (d *RowEffectsDialog) adjustFocused(delta int) {
	const maxSemitones = 24
	switch d.focusField {
	case 0:
		d.offset1 = max(0, min(maxSemitones, d.offset1+delta))
	case 1:
		d.offset2 = max(0, min(maxSemitones, d.offset2+delta))
	case 2:
		d.ticksPerStep = max(1, min(DefaultSpeed, d.ticksPerStep+delta))
	}
}

func (d *RowEffectsDialog) buildArpeggio() audio.ArpeggioEffect {
	if d.offset1 == 0 {
		return audio.ArpeggioEffect{} // inactive
	}
	offsets := []int{0, d.offset1}
	if d.offset2 > 0 {
		offsets = append(offsets, d.offset2)
	}
	return audio.ArpeggioEffect{
		Offsets:      offsets,
		TicksPerStep: d.ticksPerStep,
	}
}

func (d *RowEffectsDialog) View() tea.View {
	type field struct {
		label string
		value int
		unit  string
	}
	fields := []field{
		{"Note 2 offset", d.offset1, "semitones (0 = off)"},
		{"Note 3 offset", d.offset2, "semitones (0 = 2-note)"},
		{"Ticks/step   ", d.ticksPerStep, "sub-ticks per step"},
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Row Effects — Track %d, Row %02d\n\n", d.trackIdx+1, d.rowIdx)

	for i, f := range fields {
		cursor := "  "
		if d.focusField == i {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%s  %3d  %s\n", cursor, f.label, f.value, f.unit)
	}

	b.WriteString("\n")
	b.WriteString("[Enter: Apply | Esc: Cancel | ↑↓: Navigate | ←→: ±1 | Shift+←→: ±6 | Del: Clear arp]")
	return tea.NewView(b.String())
}
