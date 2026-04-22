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
	Volume   int
	Ticks    int
	Arpeggio audio.ArpeggioEffect
	Effect   TrackerEffect
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
	maxVolume   = 64
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

// RowEffectsDialog lets the user edit per-row volume, tick count, arpeggio, and effect settings.
//
// Field layout:
//
//	0        — Volume
//	1        — Ticks
//	2        — ARP on/off
//	[if ARP on]
//	  3      — Preset (None / Up / Down / Converge / Diverge / Random)
//	  [if preset == None]
//	    4…N  — per-tick semitone offsets
//	  [if preset != None]
//	    4    — Step (semitone interval between chord degrees)
//	[if ARP off]
//	N        — Effect type
//	N+1      — Effect param (type-specific)
//	N+2      — Vibrato depth (vibrato only)
type RowEffectsDialog struct {
	trackIdx     int
	rowIdx       int
	volume       int
	ticks        int
	arpEnabled   bool
	preset       ArpPreset
	step         int
	offsets      []int // used when preset == ArpPresetNone; tail preserved on shrink
	effectType   EffectType
	vibratoSpeed int // hi-nibble for vibrato (1-15); 0 disables vibrato cycling
	vibratoDepth int // lo-nibble for vibrato (0-15, semitones * 4)
	effectParam  int // param for VolumeSlide / NoteCut / NoteDelay / RowTicks / ArpPreset
	focusField   int
}

// NewRowEffectsDialog creates a dialog pre-populated with the current row's settings.
func NewRowEffectsDialog(row TrackRow, trackIdx, rowIdx int) *RowEffectsDialog {
	ticks := row.FX.Ticks
	if ticks <= 0 {
		ticks = 1
	}
	offsets := make([]int, ticks)
	if row.FX.Pitch.Arpeggio != nil && row.FX.Pitch.Arpeggio.IsActive() {
		copy(offsets, row.FX.Pitch.Arpeggio.Offsets)
	}

	var effectType EffectType
	var effectParam int
	var vibratoSpeed, vibratoDepth int
	vibratoSpeed = 4 // sensible default

	if row.FX.VolumeSlide.Delta != 0 {
		effectType = EffectVolumeSlide
		effectParam = int(row.FX.VolumeSlide.Delta * 64)
	} else if row.FX.NoteCut.Tick > 0 {
		effectType = EffectNoteCut
		effectParam = row.FX.NoteCut.Tick
	} else if row.FX.NoteDelay.Tick > 0 {
		effectType = EffectNoteDelay
		effectParam = row.FX.NoteDelay.Tick
	} else if row.FX.Pitch.Vibrato != nil && row.FX.Pitch.Vibrato.IsActive() {
		effectType = EffectVibrato
		effectParam = 0 // vibrato uses separate speed/depth fields
		if row.FX.Pitch.Vibrato.Rate > 0 && ticks > 0 {
			vibratoSpeed = int(float64(ticks) / row.FX.Pitch.Vibrato.Rate)
			if vibratoSpeed < 1 {
				vibratoSpeed = 1
			} else if vibratoSpeed > 15 {
				vibratoSpeed = 15
			}
		}
		vibratoDepth = int(row.FX.Pitch.Vibrato.Depth * 4)
		if vibratoDepth > 15 {
			vibratoDepth = 15
		}
	}

	var portamento int
	if row.FX.Pitch.Portamento != nil {
		portamento = (row.FX.Pitch.Portamento.StartTick << 4) | (row.FX.Pitch.Portamento.Ticks & 0xF)
	}

	arpActive := row.FX.Pitch.Arpeggio != nil && row.FX.Pitch.Arpeggio.IsActive()
	preset := ArpPresetNone
	if !arpActive {
		preset = ArpPresetUp
	}

	// Derive volume from FX.Volume (0.0–1.0 → 0–maxVolume).
	volume := 0
	if row.FX.Volume.Active {
		volume = int(row.FX.Volume.Level * float64(maxVolume))
	}

	_ = portamento // portamento is informational; dialog manages it separately

	return &RowEffectsDialog{
		trackIdx:     trackIdx,
		rowIdx:       rowIdx,
		volume:       volume,
		ticks:        ticks,
		arpEnabled:   arpActive,
		preset:       preset,
		step:         defaultStep,
		offsets:      offsets,
		effectType:   effectType,
		vibratoSpeed: vibratoSpeed,
		vibratoDepth: vibratoDepth,
		effectParam:  effectParam,
	}
}

// FocusForEffect moves initial focus to the field most relevant for the current
// inline effect command, keeping Ctrl+E as an advanced fallback editor.
func (d *RowEffectsDialog) FocusForEffect(effect TrackerEffect) {
	arpBase := d.numArpFields()
	switch effect.Type {
	case EffectRowTicks:
		d.focusField = 1 // Ticks field
	case EffectArpPreset:
		if !d.arpEnabled {
			d.focusField = 2
			return
		}
		d.focusField = 3
	case EffectVibrato, EffectVolumeSlide, EffectNoteCut, EffectNoteDelay:
		d.focusField = arpBase
	default:
		d.focusField = 0
	}
}

func (d *RowEffectsDialog) Init() tea.Cmd { return nil }

// numArpFields returns the number of fields before the effect section.
// Layout: Volume(1) + Ticks(1) + ARP(1) + conditional fields
func (d *RowEffectsDialog) numArpFields() int {
	if !d.arpEnabled {
		return 3 // volume, ticks, arp
	}
	if d.preset == ArpPresetNone {
		return 4 + d.ticks // volume, ticks, arp, preset, offset×ticks
	}
	return 5 // volume, ticks, arp, preset, step
}

func (d *RowEffectsDialog) numEffectFields() int {
	switch d.effectType {
	case EffectVibrato:
		return 3 // type, speed, depth
	case EffectNone:
		return 1
	default:
		return 2 // type, param
	}
}

func (d *RowEffectsDialog) numFields() int {
	return d.numArpFields() + d.numEffectFields()
}

func (d *RowEffectsDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return d, func() tea.Msg { return ui.CloseDialogMsg{} }
		case "enter":
			ticks, arp, effect := d.build()
			trackIdx, rowIdx, volume := d.trackIdx, d.rowIdx, d.volume
			return d, func() tea.Msg {
				return ui.CloseDialogMsg{Payload: RowEffectsApplied{
					TrackIdx: trackIdx,
					RowIdx:   rowIdx,
					Volume:   volume,
					Ticks:    ticks,
					Arpeggio: arp,
					Effect:   effect,
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
			d.volume = 0
			d.arpEnabled = false
			d.preset = ArpPresetNone
			d.offsets = make([]int, d.ticks)
			d.effectType = EffectNone
			d.effectParam = 0
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
	arpBase := d.numArpFields()
	switch {
	case d.focusField == 0: // Volume
		d.volume = max(0, min(maxVolume, d.volume+delta))
	case d.focusField == 1: // Ticks
		d.ticks = max(minTicks, min(maxTicks, d.ticks+delta))
		d.growOffsets()
		if d.focusField >= d.numFields() {
			d.focusField = d.numFields() - 1
		}
	case d.focusField == 2: // ARP ON/OFF
		if delta < 0 {
			d.arpEnabled = false
		} else {
			d.arpEnabled = true
			d.growOffsets()
		}
		if !d.arpEnabled && d.focusField >= d.numFields() {
			d.focusField = 2
		}
	case d.focusField == arpBase: // Effect type
		n := int(d.effectType) + delta
		if n < 0 {
			n = 0
		}
		if n > int(EffectArpPreset) {
			n = int(EffectArpPreset)
		}
		d.effectType = EffectType(n)
		if d.focusField >= d.numFields() {
			d.focusField = d.numFields() - 1
		}
	case d.focusField == arpBase+1: // Effect param or vibrato speed
		switch d.effectType {
		case EffectVibrato:
			d.vibratoSpeed = max(1, min(15, d.vibratoSpeed+delta))
		case EffectVolumeSlide:
			d.effectParam = max(-16, min(16, d.effectParam+delta))
		case EffectRowTicks:
			d.effectParam = max(0, min(maxTicks, d.effectParam+delta))
		case EffectArpPreset:
			// hi nibble = preset (0–5), lo nibble = step bucket (0–15)
			hi := (d.effectParam >> 4) & 0xF
			lo := d.effectParam & 0xF
			hi = max(0, min(5, hi+delta))
			d.effectParam = (hi << 4) | lo
		default:
			d.effectParam = max(0, min(31, d.effectParam+delta))
		}
	case d.focusField == arpBase+2: // Vibrato depth or ArpPreset lo-nibble
		switch d.effectType {
		case EffectVibrato:
			d.vibratoDepth = max(0, min(15, d.vibratoDepth+delta))
		case EffectArpPreset:
			hi := (d.effectParam >> 4) & 0xF
			lo := d.effectParam & 0xF
			lo = max(0, min(15, lo+delta))
			d.effectParam = (hi << 4) | lo
		}
	case d.focusField == 3 && d.arpEnabled: // Preset (ARP on only)
		n := max(0, min(len(arpPresetNames)-1, int(d.preset)+delta))
		d.preset = ArpPreset(n)
		if d.focusField >= d.numFields() {
			d.focusField = d.numFields() - 1
		}
	case d.focusField >= 4 && d.focusField < arpBase: // arp step or manual offsets
		if d.preset != ArpPresetNone {
			// field 4 = Step
			d.step = max(minStep, min(maxStep, d.step+delta))
		} else {
			// field 4…N = manual offset[i]
			i := d.focusField - 4
			if i < len(d.offsets) {
				d.offsets[i] = max(minSemitone, min(maxSemitone, d.offsets[i]+delta))
			}
		}
	}
}

func (d *RowEffectsDialog) build() (int, audio.ArpeggioEffect, TrackerEffect) {
	var arp audio.ArpeggioEffect
	if d.arpEnabled {
		if d.preset == ArpPresetNone {
			offsets := make([]int, d.ticks)
			copy(offsets, d.offsets)
			arp = audio.ArpeggioEffect{Offsets: offsets}
		} else {
			seed := d.trackIdx*1000 + d.rowIdx
			arp = audio.ArpeggioEffect{Offsets: generateArpOffsets(d.preset, d.ticks, d.step, seed)}
		}
	}
	var effect TrackerEffect
	effect.Type = d.effectType
	switch d.effectType {
	case EffectVibrato:
		effect.Param = (d.vibratoSpeed << 4) | (d.vibratoDepth & 0xF)
	default:
		effect.Param = d.effectParam
	}
	return d.ticks, arp, effect
}

var effectNames = []string{
	"None", "Vibrato", "VolumeSlide", "NoteCut", "NoteDelay",
	"RowTicks", "ArpPreset",
}

const dialogBarWidth = 16

// barField renders "Label  [████▫▫▫▫]  value unit" as a single line.
// minVal/maxVal define the bar range; value is the current value.
func barField(label string, value int, minVal, maxVal int, unit string) string {
	bar := common.NewBar(float64(minVal), float64(maxVal), float64(value), dialogBarWidth)
	if unit != "" {
		return fmt.Sprintf("%-11s %s  %3d %s", label, bar.View(), value, unit)
	}
	return fmt.Sprintf("%-11s %s  %3d", label, bar.View(), value)
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

	// Volume
	b.WriteString(render(d.focusField == 0, barField("Volume", d.volume, 0, maxVolume, "")))
	b.WriteByte('\n')

	// Ticks
	b.WriteString(render(d.focusField == 1, barField("Ticks", d.ticks, minTicks, maxTicks, "")))
	b.WriteByte('\n')

	// ARP on/off
	arpStatus := "OFF"
	if d.arpEnabled {
		arpStatus = "ON "
	}
	b.WriteString(render(d.focusField == 2, fmt.Sprintf("%-11s %s", "ARP mode", arpStatus)))
	b.WriteByte('\n')

	if d.arpEnabled {
		b.WriteByte('\n')
		b.WriteString(render(d.focusField == 3, fmt.Sprintf("%-11s %-8s", "Preset", arpPresetNames[d.preset])))
		b.WriteByte('\n')

		if d.preset != ArpPresetNone {
			b.WriteString(render(d.focusField == 4, barField("Step", d.step, minStep, maxStep, "semitones")))
			b.WriteByte('\n')
		} else {
			b.WriteByte('\n')
			for i := 0; i < d.ticks; i++ {
				offset := 0
				if i < len(d.offsets) {
					offset = d.offsets[i]
				}
				label := fmt.Sprintf("Tick %2d", i+1)
				b.WriteString(render(d.focusField == i+4, barField(label, offset, minSemitone, maxSemitone, "semitones")))
				b.WriteByte('\n')
			}
		}
	}

	// Effect section
	arpBase := d.numArpFields()
	b.WriteByte('\n')
	b.WriteString(render(d.focusField == arpBase, fmt.Sprintf("%-11s %-11s", "Effect", effectNames[d.effectType])))
	b.WriteByte('\n')
	switch d.effectType {
	case EffectVibrato:
		b.WriteString(render(d.focusField == arpBase+1, barField("  Speed", d.vibratoSpeed, 1, 15, "ticks/cycle")))
		b.WriteByte('\n')
		b.WriteString(render(d.focusField == arpBase+2, barField("  Depth", d.vibratoDepth, 0, 15, "×0.25 semitones")))
		b.WriteByte('\n')
	case EffectVolumeSlide:
		b.WriteString(render(d.focusField == arpBase+1, barField("  Delta", d.effectParam, -16, 16, "/64 per tick")))
		b.WriteByte('\n')
	case EffectNoteCut:
		b.WriteString(render(d.focusField == arpBase+1, barField("  At tick", d.effectParam, 0, 31, "")))
		b.WriteByte('\n')
	case EffectNoteDelay:
		b.WriteString(render(d.focusField == arpBase+1, barField("  At tick", d.effectParam, 0, 31, "")))
		b.WriteByte('\n')
	case EffectRowTicks:
		b.WriteString(render(d.focusField == arpBase+1, barField("  Ticks", d.effectParam, 0, maxTicks, "(0 = clear)")))
		b.WriteByte('\n')
	case EffectArpPreset:
		presetIdx := (d.effectParam >> 4) & 0xF
		stepBucket := d.effectParam & 0xF
		presetName := "None"
		if presetIdx > 0 && presetIdx < len(arpPresetNames) {
			presetName = arpPresetNames[presetIdx]
		}
		b.WriteString(render(d.focusField == arpBase+1, fmt.Sprintf("%-11s %s  %d %-8s", "  Preset", common.NewBar(0, 5, float64(presetIdx), dialogBarWidth).View(), presetIdx, presetName)))
		b.WriteByte('\n')
		b.WriteString(render(d.focusField == arpBase+2, barField("  Step", stepBucket, 0, 15, "(bucket)")))
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString("[Enter: Apply | Esc: Cancel | ↑↓: Navigate | ←→: ±1 | Shift+←→: ±6 | Del: Clear]")
	return tea.NewView(b.String())
}
