package render

import (
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/audio/effects"
)

// EffectType identifies per-row playback effects evaluated each sub-tick.
type EffectType int

const (
	EffectNone        EffectType = iota
	EffectVibrato                // Param hi-nibble: speed (1-15 ticks/cycle); lo-nibble: depth (semitones*4, 0-15)
	EffectVolumeSlide            // Param: signed volume delta per tick in 1/64 units; positive = louder
	EffectNoteCut                // Param: sub-tick at which to silence the note
	EffectNoteDelay              // Param: sub-tick at which to trigger NoteOn
)

// RowEffect is a per-row effect command evaluated every sub-tick during render.
type RowEffect struct {
	Type  EffectType
	Param int
}

// Row is the frequency-domain representation of a single pattern cell.
// It contains no display strings — all note data has been converted to Hz.
type Row struct {
	Frequency float64 // Hz; 0 = rest (no note)
	Volume    float64 // 0 = use synth default; range 0..1
	Arpeggio  audio.ArpeggioEffect
	Effect    RowEffect
	Ticks     int // sub-ticks per row; 0 = no subdivision
}

// Track is one channel of a Pattern.
type Track struct {
	Synth *audio.Synth
	Rows  []Row
}

// Pattern is the complete frequency-domain render model.
// It is produced by the tracker UI and consumed by the RenderEngine.
type Pattern struct {
	Tracks      []Track
	NumRows     int
	NumTracks   int
	RowDuration time.Duration // duration of one row (derived from BPM)
}

// RowTicks returns the effective sub-tick count for the given row.
// Returns 0 when no track at that row has a non-zero tick count set.
func (s *Pattern) RowTicks(rowIdx int) int {
	if rowIdx >= 0 && rowIdx < s.NumRows {
		for _, track := range s.Tracks {
			if rowIdx < len(track.Rows) && track.Rows[rowIdx].Ticks > 0 {
				return track.Rows[rowIdx].Ticks
			}
		}
	}
	return 0
}

// rowToEffectDefs converts a Row's effect into the effects.EffectDefs used by
// EffectsPatch. subticks is the number of sub-ticks in the row; portamento
// indicates whether the synth has portamento enabled.
func rowToEffectDefs(row Row, subticks int, portamento bool) effects.EffectDefs {
	fx := effects.EffectDefs{
		Arpeggio:   row.Arpeggio,
		Portamento: effects.PortamentoEffect{Active: portamento},
	}

	switch row.Effect.Type {
	case EffectVibrato:
		speed := (row.Effect.Param >> 4) & 0xF
		depth := row.Effect.Param & 0xF
		if speed > 0 {
			fx.Vibrato = effects.VibratoEffect{
				Depth: float64(depth) / 4.0,
				Rate:  float64(subticks) / float64(speed),
			}
		}
	case EffectVolumeSlide:
		fx.VolumeSlide = effects.VolumeSlideEffect{
			Delta: float64(int8(uint8(row.Effect.Param))) / 64.0,
		}
	case EffectNoteCut:
		fx.NoteCut = effects.NoteCutEffect{Tick: row.Effect.Param}
	case EffectNoteDelay:
		fx.NoteDelay = effects.NoteDelayEffect{Tick: row.Effect.Param}
	}

	return fx
}
