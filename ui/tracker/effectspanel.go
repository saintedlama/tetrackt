package tracker

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

// EffectsPanelApplied is emitted when the user confirms the effects panel.
// FX contains the full effect definitions to apply to the row; FX.Ticks
// carries the row tick count so no separate field is needed.
type EffectsPanelApplied struct {
	TrackIdx int
	RowIdx   int
	FX       audio.EffectDefinitions
}

// pitchMode enumerates the mutually exclusive pitch effects.
// Arpeggio, Portamento and Vibrato cannot be active at the same time because
// applytickEffects selects exactly one of them (if arp → else if vibrato →
// else if portamento).
type pitchMode int

const (
	pitchModeNone       pitchMode = iota // no pitch effect
	pitchModeArp                         // cycle semitone offsets per tick
	pitchModePortamento                  // exponential pitch glide from previous note
	pitchModeVibrato                     // sinusoidal pitch modulation
	pitchModeCount
)

var epPitchModeNames = []string{"None", "Arpeggio", "Portamento", "Vibrato"}

// epFieldKind identifies the logical parameter a cursor position targets.
type epFieldKind int

const (
	epfPitchMode    epFieldKind = iota // pitch mode selector (always visible)
	epfTicks                           // row tick count (always visible, shared by all pitch effects)
	epfArpTicks                        // arp: number of offsets in the sequence (may loop within ticks)
	epfArpFillPreset                   // arp: pattern generator to use when filling offsets
	epfArpFillStep                     // arp: semitone step between pattern degrees
	epfArpOffset                       // arp: one semitone offset; offsetIdx selects which
	epfPortStart                       // portamento: sub-tick where glide begins
	epfPortTicks                       // portamento: sub-ticks for the glide
	epfVibratoDepth                    // vibrato: peak deviation in semitones
	epfVibratoRate                     // vibrato: sine oscillations per note
	epfRetrigger                       // retrigger ADSR at every sub-tick boundary
	epfVolume                          // volume: initial playback volume level
	epfVolSlide                        // volume slide: per-tick delta
	epfNoteCut                         // note cut: sub-tick to silence (0 = off)
	epfNoteDelay                       // note delay: sub-tick to start (0 = immediate)
)

// epField is a single focusable entry in the panel's field list.
type epField struct {
	kind      epFieldKind
	offsetIdx int // only meaningful when kind == epfArpOffset
}

const (
	epMinArpTicks    = 1
	epMaxArpTicks    = 8
	epMinArpOffset   = -24
	epMaxArpOffset   = 24
	epMinTicks       = 1
	epMaxTicks       = 32
	epMinFillStep    = 1
	epMaxFillStep    = 24
	epMaxPortStart   = 32
	epMaxPortTicks   = 64
	epMaxVibDepth    = 4.0
	epMaxVibRate     = 16.0
	epVolSlideStep   = 0.01
	epVolStep        = 0.05
	epMaxNoteCut     = 32
	epMaxNoteDelay   = 32
	epBarWidth       = 10
)

// epArpFillPresets lists the non-trivial arp fill patterns in display order.
// ArpPresetNone is excluded — manual editing covers that case.
var epArpFillPresets = []ArpPreset{
	ArpPresetUp,
	ArpPresetDown,
	ArpPresetConverge,
	ArpPresetDiverge,
	ArpPresetRandom,
}

var epArpFillPresetNames = []string{"Up", "Down", "Converge", "Diverge", "Random"}

// EffectsPanelDialog is a prototype effects configuration panel opened with ctrl+r.
// It surfaces the complete EffectDefinitions API with a linear, navigable field list.
// Pressing enter applies the changes to the track row; esc discards.
type EffectsPanelDialog struct {
	trackIdx int
	rowIdx   int
	focusIdx int

	// Pitch section — ticks is shared; mode is mutually exclusive
	ticks int
	mode  pitchMode

	// Arpeggio
	arpTicks      int
	arpOffsets    []int
	arpFillPreset int // index into epArpFillPresets
	arpFillStep   int // semitone step between pattern degrees
	arpFillSeed   int // incremented on each Random fill for variety

	// Portamento
	portStart int
	portTicks int

	// Vibrato
	vibDepth float64
	vibRate  float64

	// Volume & Timing — independent, combinable
	retrigger     bool
	volActive     bool
	volLevel      float64
	volSlideDelta float64
	noteCutTick   int
	noteDelayTick int
}

// NewEffectsPanelDialog returns a dialog pre-populated from the given track row.
func NewEffectsPanelDialog(trackIdx, rowIdx int, row TrackRow) *EffectsPanelDialog {
	d := &EffectsPanelDialog{
		trackIdx:    trackIdx,
		rowIdx:      rowIdx,
		ticks:       4,
		arpTicks:    4,
		arpOffsets:  []int{0, 4, 7, 0},
		arpFillStep: 4,
		portTicks:   4,
		vibDepth:    0.5,
		vibRate:     2.0,
	}
	d.loadFromEffectDefs(row.FX)
	return d
}

// loadFromEffectDefs populates the dialog from a full EffectDefinitions value.
// The row tick count is read from fx.Ticks (0 means unset; default to 4).
func (d *EffectsPanelDialog) loadFromEffectDefs(fx audio.EffectDefinitions) {
	d.ticks = fx.Ticks
	if d.ticks <= 0 {
		d.ticks = 4
	}

	if fx.Pitch.Arpeggio != nil && fx.Pitch.Arpeggio.IsActive() {
		d.mode = pitchModeArp
		d.arpTicks = len(fx.Pitch.Arpeggio.Offsets)
		d.arpOffsets = make([]int, d.arpTicks)
		copy(d.arpOffsets, fx.Pitch.Arpeggio.Offsets)
	} else if fx.Pitch.Portamento != nil && (fx.Pitch.Portamento.Ticks > 0 || fx.Pitch.Portamento.StartTick > 0) {
		d.mode = pitchModePortamento
		d.portStart = fx.Pitch.Portamento.StartTick
		d.portTicks = fx.Pitch.Portamento.Ticks
	} else if fx.Pitch.Vibrato != nil && fx.Pitch.Vibrato.IsActive() {
		d.mode = pitchModeVibrato
		d.vibDepth = fx.Pitch.Vibrato.Depth
		d.vibRate = fx.Pitch.Vibrato.Rate
	}

	d.retrigger = fx.RetriggerEnvelope
	d.volActive = fx.Volume.Active
	d.volLevel = fx.Volume.Level
	if !d.volActive {
		d.volLevel = 1.0
	}
	d.volSlideDelta = fx.VolumeSlide.Delta
	d.noteCutTick = fx.NoteCut.Tick
	d.noteDelayTick = fx.NoteDelay.Tick
}

// toEffectDefs builds an audio.EffectDefinitions from the current dialog state.
func (d *EffectsPanelDialog) toEffectDefs() audio.EffectDefinitions {
	var fx audio.EffectDefinitions

	fx.Ticks = d.ticks

	switch d.mode {
	case pitchModeArp:
		offsets := make([]int, d.arpTicks)
		copy(offsets, d.arpOffsets[:d.arpTicks])
		a := audio.ArpeggioEffect{Offsets: offsets}
		fx.Pitch.Arpeggio = &a
	case pitchModePortamento:
		p := audio.PortamentoEffect{StartTick: d.portStart, Ticks: d.portTicks}
		fx.Pitch.Portamento = &p
	case pitchModeVibrato:
		v := audio.VibratoEffect{Depth: d.vibDepth, Rate: d.vibRate}
		fx.Pitch.Vibrato = &v
	}

	fx.RetriggerEnvelope = d.retrigger
	fx.Volume = audio.VolumeEffect{Level: d.volLevel, Active: d.volActive}
	fx.VolumeSlide = audio.VolumeSlideEffect{Delta: d.volSlideDelta}
	fx.NoteCut = audio.NoteCutEffect{Tick: d.noteCutTick}
	fx.NoteDelay = audio.NoteDelayEffect{Tick: d.noteDelayTick}
	return fx
}

// fields returns the ordered list of focusable fields for the current state.
// Ticks appears directly after the mode selector — it is the shared
// tick count that gives meaning to every per-tick value below it.
// Mode-specific parameters follow immediately after.
func (d *EffectsPanelDialog) fields() []epField {
	fs := []epField{
		{epfPitchMode, 0},
		{epfTicks, 0},
	}
	switch d.mode {
	case pitchModeArp:
		fs = append(fs, epField{epfArpTicks, 0})
		fs = append(fs, epField{epfArpFillPreset, 0})
		fs = append(fs, epField{epfArpFillStep, 0})
		for i := range d.arpTicks {
			fs = append(fs, epField{epfArpOffset, i})
		}
	case pitchModePortamento:
		fs = append(fs, epField{epfPortStart, 0})
		fs = append(fs, epField{epfPortTicks, 0})
	case pitchModeVibrato:
		fs = append(fs, epField{epfVibratoDepth, 0})
		fs = append(fs, epField{epfVibratoRate, 0})
	}
	fs = append(fs,
		epField{epfRetrigger, 0},
		epField{epfVolume, 0},
		epField{epfVolSlide, 0},
		epField{epfNoteCut, 0},
		epField{epfNoteDelay, 0},
	)
	return fs
}

func (d *EffectsPanelDialog) Init() tea.Cmd { return nil }

func (d *EffectsPanelDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return d, func() tea.Msg { return ui.CloseDialogMsg{} }
		case "enter":
			trackIdx, rowIdx := d.trackIdx, d.rowIdx
			fx := d.toEffectDefs()
			return d, func() tea.Msg {
				return ui.CloseDialogMsg{Payload: EffectsPanelApplied{
					TrackIdx: trackIdx,
					RowIdx:   rowIdx,
					FX:       fx,
				}}
			}
		case "up":
			if d.focusIdx > 0 {
				d.focusIdx--
			}
		case "down":
			if d.focusIdx < len(d.fields())-1 {
				d.focusIdx++
			}
		case "left":
			d.adjustFocused(-1)
		case "right":
			d.adjustFocused(1)
		case "shift+left":
			d.adjustFocused(-5)
		case "shift+right":
			d.adjustFocused(5)
		case "f":
			if d.mode == pitchModeArp {
				d.applyArpFill()
			}
		}
	}
	return d, nil
}

// adjustFocused changes the focused field's value by dir steps.
func (d *EffectsPanelDialog) adjustFocused(dir int) {
	fields := d.fields()
	if d.focusIdx >= len(fields) {
		return
	}
	f := fields[d.focusIdx]
	switch f.kind {

	case epfPitchMode:
		d.mode = pitchMode((int(d.mode) + dir + int(pitchModeCount)) % int(pitchModeCount))
		// The field list length may have changed; keep cursor in bounds.
		if d.focusIdx >= len(d.fields()) {
			d.focusIdx = len(d.fields()) - 1
		}

	case epfTicks:
		d.ticks = max(epMinTicks, min(epMaxTicks, d.ticks+dir))
		// Clamp portamento ticks so they stay within the new tick budget.
		if d.portStart >= d.ticks {
			d.portStart = d.ticks - 1
		}
		if d.portTicks > d.ticks {
			d.portTicks = d.ticks
		}
		// Clamp note-cut / note-delay the same way.
		if d.noteCutTick >= d.ticks {
			d.noteCutTick = d.ticks - 1
		}
		if d.noteDelayTick >= d.ticks {
			d.noteDelayTick = d.ticks - 1
		}

	case epfArpTicks:
		d.arpTicks = max(epMinArpTicks, min(epMaxArpTicks, d.arpTicks+dir))
		// Grow or trim the offset slice to match the new tick count.
		for len(d.arpOffsets) < d.arpTicks {
			d.arpOffsets = append(d.arpOffsets, 0)
		}
		if len(d.arpOffsets) > d.arpTicks {
			d.arpOffsets = d.arpOffsets[:d.arpTicks]
		}
		if d.focusIdx >= len(d.fields()) {
			d.focusIdx = len(d.fields()) - 1
		}

	case epfArpFillPreset:
		d.arpFillPreset = (d.arpFillPreset + dir + len(epArpFillPresets)) % len(epArpFillPresets)

	case epfArpFillStep:
		d.arpFillStep = max(epMinFillStep, min(epMaxFillStep, d.arpFillStep+dir))

	case epfArpOffset:
		d.arpOffsets[f.offsetIdx] = max(epMinArpOffset, min(epMaxArpOffset, d.arpOffsets[f.offsetIdx]+dir))

	case epfPortStart:
		d.portStart = max(0, min(d.ticks-1, d.portStart+dir))

	case epfPortTicks:
		d.portTicks = max(0, min(d.ticks, d.portTicks+dir))

	case epfVibratoDepth:
		d.vibDepth = max(0, min(epMaxVibDepth, d.vibDepth+float64(dir)*0.1))
		d.vibDepth = epRound1(d.vibDepth)

	case epfVibratoRate:
		d.vibRate = max(0, min(epMaxVibRate, d.vibRate+float64(dir)*0.5))
		d.vibRate = epRound1(d.vibRate)

	case epfRetrigger:
		if dir != 0 {
			d.retrigger = !d.retrigger
		}

	case epfVolume:
		if !d.volActive {
			if dir > 0 {
				d.volActive = true
				d.volLevel = 1.0
			}
		} else {
			d.volLevel = epRound2(d.volLevel + float64(dir)*epVolStep)
			if d.volLevel < 0 {
				d.volActive = false
				d.volLevel = 1.0
			} else {
				d.volLevel = min(1.0, d.volLevel)
			}
		}

	case epfVolSlide:
		d.volSlideDelta = max(-1.0, min(1.0, d.volSlideDelta+float64(dir)*epVolSlideStep))
		d.volSlideDelta = epRound2(d.volSlideDelta)

	case epfNoteCut:
		d.noteCutTick = max(0, min(epMaxNoteCut, d.noteCutTick+dir))

	case epfNoteDelay:
		d.noteDelayTick = max(0, min(epMaxNoteDelay, d.noteDelayTick+dir))
	}
}

// applyArpFill overwrites the offset slice with the current fill pattern and
// step. For the Random preset the seed is incremented so repeated presses
// produce different orderings.
func (d *EffectsPanelDialog) applyArpFill() {
	preset := epArpFillPresets[d.arpFillPreset]
	if preset == ArpPresetRandom {
		d.arpFillSeed++
	}
	d.arpOffsets = generateArpOffsets(preset, d.arpTicks, d.arpFillStep, d.arpFillSeed)
}

func (d *EffectsPanelDialog) View() tea.View {
	return tea.NewView(d.render())
}

func (d *EffectsPanelDialog) render() string {
	fields := d.fields()

	titleStyle := lipgloss.NewStyle().Foreground(common.ColorAccentPrimary).Bold(true)
	sectionStyle := lipgloss.NewStyle().Foreground(common.ColorTextMuted)
	hintStyle := lipgloss.NewStyle().Foreground(common.ColorTextDisabled)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Effects") + "\n\n")

	fieldIdx := 0

	// Pitch section
	b.WriteString(sectionStyle.Render("Pitch") + "\n")
	for fieldIdx < len(fields) {
		f := fields[fieldIdx]
		if f.kind == epfRetrigger {
			break // volume & timing section starts here
		}
		b.WriteString(d.renderField(f, fieldIdx == d.focusIdx) + "\n")
		fieldIdx++
	}

	// Volume & Timing section
	b.WriteString("\n" + sectionStyle.Render("Volume & Timing") + "\n")
	for fieldIdx < len(fields) {
		f := fields[fieldIdx]
		b.WriteString(d.renderField(f, fieldIdx == d.focusIdx) + "\n")
		fieldIdx++
	}

	b.WriteString("\n" + hintStyle.Render("↑↓ navigate  ◀▶ adjust  shift+◀▶ ±5  enter apply  esc cancel"))
	if d.mode == pitchModeArp {
		b.WriteString("  " + hintStyle.Render("f fill offsets"))
	}
	return b.String()
}

func (d *EffectsPanelDialog) renderField(f epField, focused bool) string {
	label, value, hint := d.fieldParts(f)
	var line string
	if hint != "" {
		line = fmt.Sprintf("  %-16s %-20s %s", label, value, hint)
	} else {
		line = fmt.Sprintf("  %-16s %s", label, value)
	}
	if focused {
		return common.StyleSelected.Render(line)
	}
	return line
}

// fieldParts returns the display label, value string, and optional hint for f.
func (d *EffectsPanelDialog) fieldParts(f epField) (label, value, hint string) {
	bar := func(lo, hi, v float64) string {
		return common.NewBar(lo, hi, v, epBarWidth).View()
	}

	switch f.kind {

	case epfPitchMode:
		label = "Pitch Mode"
		value = fmt.Sprintf("◀ %-11s ▶", epPitchModeNames[d.mode])

	case epfTicks:
		label = "  Ticks"
		value = fmt.Sprintf("%s  %d", bar(float64(epMinTicks), float64(epMaxTicks), float64(d.ticks)), d.ticks)
		hint = "row tick count"

	case epfArpTicks:
		label = "  Seq Length"
		value = fmt.Sprintf("%s  %d", bar(float64(epMinArpTicks), float64(epMaxArpTicks), float64(d.arpTicks)), d.arpTicks)
		if d.arpTicks < d.ticks {
			hint = fmt.Sprintf("loops every %d of %d ticks", d.arpTicks, d.ticks)
		} else if d.arpTicks == d.ticks {
			hint = "one offset per tick"
		}

	case epfArpFillPreset:
		label = "  Fill Pattern"
		value = fmt.Sprintf("◀ %-9s ▶", epArpFillPresetNames[d.arpFillPreset])
		hint = "f to fill offsets"

	case epfArpFillStep:
		label = "  Fill Step"
		value = fmt.Sprintf("%s  %d st", bar(float64(epMinFillStep), float64(epMaxFillStep), float64(d.arpFillStep)), d.arpFillStep)
		hint = "semitone step between degrees"

	case epfArpOffset:
		offset := d.arpOffsets[f.offsetIdx]
		label = fmt.Sprintf("  Offset [%d]", f.offsetIdx)
		value = fmt.Sprintf("%s %+3d st", bar(float64(epMinArpOffset), float64(epMaxArpOffset), float64(offset)), offset)
		if offset == 0 {
			hint = "root"
		}

	case epfPortStart:
		label = "  Start Tick"
		value = fmt.Sprintf("%s  %d", bar(0, float64(d.ticks), float64(d.portStart)), d.portStart)
		if d.portStart == 0 {
			hint = fmt.Sprintf("glide from tick 0 of %d", d.ticks)
		} else {
			hint = fmt.Sprintf("hold pitch for %d ticks first", d.portStart)
		}

	case epfPortTicks:
		label = "  Glide Ticks"
		value = fmt.Sprintf("%s  %d", bar(0, float64(d.ticks), float64(d.portTicks)), d.portTicks)
		if d.portTicks == 0 {
			hint = "snap (no glide)"
		} else {
			hint = fmt.Sprintf("of %d total ticks", d.ticks)
		}

	case epfVibratoDepth:
		label = "  Depth"
		value = fmt.Sprintf("%s  %.1f st", bar(0, epMaxVibDepth, d.vibDepth), d.vibDepth)

	case epfVibratoRate:
		label = "  Rate"
		value = fmt.Sprintf("%s  %.1f/note", bar(0, epMaxVibRate, d.vibRate), d.vibRate)

	case epfRetrigger:
		label = "Retrigger Env"
		if d.retrigger {
			value = "◀ On  ▶"
			hint = "ADSR restarts each tick"
		} else {
			value = "◀ Off ▶"
		}

	case epfVolume:
		label = "Volume"
		if !d.volActive {
			value = "◀ Off ▶"
			hint = "initial playback volume"
		} else {
			value = fmt.Sprintf("%s  %.0f%%", bar(0, 1, d.volLevel), d.volLevel*100)
			if d.volLevel == 0 {
				hint = "mute"
			} else {
				hint = "initial playback volume"
			}
		}

	case epfVolSlide:
		label = "Volume Slide"
		value = fmt.Sprintf("%s %+.2f/tick", bar(-1, 1, d.volSlideDelta), d.volSlideDelta)
		switch {
		case d.volSlideDelta == 0:
			hint = "off"
		case d.volSlideDelta > 0:
			hint = "fade in"
		default:
			hint = "fade out"
		}

	case epfNoteCut:
		label = "Note Cut"
		value = fmt.Sprintf("%s  tick %d", bar(0, float64(epMaxNoteCut), float64(d.noteCutTick)), d.noteCutTick)
		if d.noteCutTick == 0 {
			hint = "off"
		}

	case epfNoteDelay:
		label = "Note Delay"
		value = fmt.Sprintf("%s  tick %d", bar(0, float64(epMaxNoteDelay), float64(d.noteDelayTick)), d.noteDelayTick)
		if d.noteDelayTick == 0 {
			hint = "off (immediate)"
		}
	}
	return
}

// epRound1 rounds f to one decimal place.
func epRound1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}

// epRound2 rounds f to two decimal places.
func epRound2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
