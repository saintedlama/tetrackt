package tracker

import "github.com/tetrackt/tetrackt/ui/tracker/effects"

// Type aliases for backward compatibility with existing code.
// New code should use the effects package directly.

type EffectType = effects.Type
type TrackerEffect = effects.Effect

const (
	EffectNone        = effects.None
	EffectVibrato     = effects.Vibrato
	EffectVolumeSlide = effects.VolumeSlide
	EffectNoteCut     = effects.NoteCut
	EffectNoteDelay   = effects.NoteDelay
	EffectRowTicks    = effects.RowTicks
	EffectContinuous  = effects.Continuous
	EffectArpPreset   = effects.ArpPreset
)
