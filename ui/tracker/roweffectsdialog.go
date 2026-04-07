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
	TrackIdx   int
	RowIdx     int
	Ticks      int
	Continuous bool
	Arpeggio   audio.ArpeggioEffect
}

// ArpPreset enumerates the built-in arp order patterns.
type ArpPreset int

const (
	ArpPresetNone     ArpPreset = iota // manual per-tick entry
	ArpPresetUp                        // ascending chord degrees
	ArpPresetDown                      // descending chord degrees
	ArpPresetConverge                  // lowest → highest, working inward
	ArpPresetDiverge                   // innermost pair first, working outward
	ArpPresetRandom                    // stable pseudo-random order
)

var arpPresetNames = []string{"None", "Up", "Down", "Converge", "Diverge", "Random"}

const (
	minTicks    = 1
	maxTicks    = 32
	maxSemitone = 24
	minSemitone = -24
	minStep     = 1
	maxStep     = 24
	defaultStep = 4
)

// generateArpOffsets produces a semitone offset slice of length ticks.
// Chord degrees are [0, step, 2*step, …, (ticks-1)*step]; the preset
// determines the playback order. seed drives stable pseudo-random shuffling.
func generateArpOffsets(preset ArpPreset, ticks, step, seed int) []int {
	degrees := make([]int, ticks)
	for i := range ticks {
		degrees[i] = i * step
	}
	switch preset {
	case ArpPresetUp:
		return degrees

	case ArpPresetDown:
		result := make([]int, ticks)
		for i, v := range degrees {
			result[ticks-1-i] = v
		}
		return result

	case ArpPresetConverge:
		// lowest first, then highest, working inward
		result := make([]int, 0, ticks)
		lo, hi := 0, ticks-1
		for lo <= hi {
			result = append(result, degrees[lo])
			lo++
			if lo <= hi {
				result = append(result, degrees[hi])
				hi--
			}
		}
		return result

	case ArpPresetDiverge:
		// innermost pair (or center for odd counts) first, working outward
		result := make([]int, 0, ticks)
		lo := (ticks - 1) / 2
		hi := ticks / 2
		if ticks%2 == 1 { // odd: emit center element first
			result = append(result, degrees[lo]) // lo == hi here
			lo--
			hi++
		}
		for hi < ticks && len(result) < ticks {
			result = append(result, degrees[lo])
			result = append(result, degrees[hi])
			lo--
			hi++
		}
		return result

	case ArpPresetRandom:
		result := make([]int, ticks)
		copy(result, degrees)
		// Fisher-Yates with a fast LCG seeded from the caller-provided value
		s := uint64(seed)
		for i := ticks - 1; i > 0; i-- {
			s = s*6364136223846793005 + 1442695040888963407
			j := int(s>>33) % (i + 1)
			result[i], result[j] = result[j], result[i]
		}
		return result
	}
	return degrees
}

// RowEffectsDialog lets the user edit per-row tick count and arpeggio settings.
//
// Field layout:
//
//	0        — Ticks
//	1        — ARP on/off
//	[if ARP on]
//	  2      — Preset (None / Up / Down / Converge / Diverge / Random)
//	  [if preset == None]
//	    3…N  — per-tick semitone offsets
//	  [if preset != None]
//	    3    — Step (semitone interval between chord degrees)
type RowEffectsDialog struct {
	trackIdx   int
	rowIdx     int
	ticks      int
	arpEnabled bool
	continuous bool
	preset     ArpPreset
	step       int
	offsets    []int // used when preset == ArpPresetNone; tail preserved on shrink
	focusField int
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
		continuous: row.Continuous || row.Arpeggio.IsActive(),
		preset:     ArpPresetNone,
		step:       defaultStep,
		offsets:    offsets,
	}
}

func (d *RowEffectsDialog) Init() tea.Cmd { return nil }

func (d *RowEffectsDialog) numFields() int {
	if !d.arpEnabled {
		return 3 // ticks, arp, continuous
	}
	if d.preset == ArpPresetNone {
		return 3 + d.ticks // ticks, arp, preset, offset×ticks
	}
	return 4 // ticks, arp, preset, step
}

func (d *RowEffectsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return d, func() tea.Msg { return ui.CloseDialogMsg{} }
		case "enter":
			ticks, continuous, arp := d.build()
			trackIdx, rowIdx := d.trackIdx, d.rowIdx
			return d, func() tea.Msg {
				return ui.CloseDialogMsg{Payload: RowEffectsApplied{
					TrackIdx:   trackIdx,
					RowIdx:     rowIdx,
					Ticks:      ticks,
					Continuous: continuous,
					Arpeggio:   arp,
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
			d.preset = ArpPresetNone
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
	case 0: // Ticks
		d.ticks = max(minTicks, min(maxTicks, d.ticks+delta))
		d.growOffsets()
		if d.focusField >= d.numFields() {
			d.focusField = d.numFields() - 1
		}
	case 1: // ARP ON/OFF
		if delta < 0 {
			d.arpEnabled = false
		} else {
			d.arpEnabled = true
			d.continuous = true // ARP forces continuous
			d.growOffsets()
		}
		if !d.arpEnabled && d.focusField >= d.numFields() {
			d.focusField = 1
		}
	case 2: // Continuous (ARP off) or Preset (ARP on)
		if !d.arpEnabled {
			d.continuous = delta > 0
		} else {
			n := max(0, min(len(arpPresetNames)-1, int(d.preset)+delta))
			d.preset = ArpPreset(n)
			if d.focusField >= d.numFields() {
				d.focusField = d.numFields() - 1
			}
		}
	default: // field >= 3
		if !d.arpEnabled {
			break // no fields beyond continuous when ARP is off
		}
		if d.preset != ArpPresetNone {
			// field 3 = Step
			d.step = max(minStep, min(maxStep, d.step+delta))
		} else {
			// field 3…N = manual offset[i]
			i := d.focusField - 3
			if i < len(d.offsets) {
				d.offsets[i] = max(minSemitone, min(maxSemitone, d.offsets[i]+delta))
			}
		}
	}
}

func (d *RowEffectsDialog) build() (int, bool, audio.ArpeggioEffect) {
	if !d.arpEnabled {
		return d.ticks, d.continuous, audio.ArpeggioEffect{}
	}
	var offsets []int
	if d.preset == ArpPresetNone {
		offsets = make([]int, d.ticks)
		copy(offsets, d.offsets)
	} else {
		seed := d.trackIdx*1000 + d.rowIdx
		offsets = generateArpOffsets(d.preset, d.ticks, d.step, seed)
	}
	return d.ticks, true, audio.ArpeggioEffect{Offsets: offsets}
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

	b.WriteString(render(d.focusField == 0, fmt.Sprintf("Ticks      %3d", d.ticks)))
	b.WriteByte('\n')
	b.WriteString(render(d.focusField == 1, fmt.Sprintf("ARP mode   %s", arpStatus)))
	b.WriteByte('\n')

	if !d.arpEnabled {
		contStatus := "OFF"
		if d.continuous {
			contStatus = "ON "
		}
		b.WriteString(render(d.focusField == 2, fmt.Sprintf("Continuous %s", contStatus)))
		b.WriteByte('\n')
	}

	if d.arpEnabled {
		b.WriteByte('\n')
		b.WriteString(render(d.focusField == 2, fmt.Sprintf("Preset     %-8s", arpPresetNames[d.preset])))
		b.WriteByte('\n')

		if d.preset != ArpPresetNone {
			b.WriteString(render(d.focusField == 3, fmt.Sprintf("Step       %3d  semitones", d.step)))
			b.WriteByte('\n')
		} else {
			b.WriteByte('\n')
			for i := 0; i < d.ticks; i++ {
				offset := 0
				if i < len(d.offsets) {
					offset = d.offsets[i]
				}
				b.WriteString(render(d.focusField == i+3, fmt.Sprintf("Tick %2d    %+4d  semitones", i+1, offset)))
				b.WriteByte('\n')
			}
		}
	}

	b.WriteByte('\n')
	b.WriteString("[Enter: Apply | Esc: Cancel | ↑↓: Navigate | ←→: ±1 / ON-OFF / Preset | Shift+←→: ±6 | Del: Clear]")
	return tea.NewView(b.String())
}
