package render

import (
	"time"

	"github.com/tetrackt/tetrackt/audio"
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
	Ticks     int // sub-ticks override; 0 = use Pattern.DefaultTicks
}

// Track is one channel of a Pattern.
type Track struct {
	Synth *audio.Synth
	Rows  []Row
}

// Pattern is the complete frequency-domain render model.
// It is produced by the tracker UI and consumed by the RenderEngine.
type Pattern struct {
	Tracks       []Track
	NumRows      int
	NumTracks    int
	RowDuration  time.Duration // duration of one row (derived from BPM)
	DefaultTicks int           // default sub-ticks per row when Row.Ticks == 0
}

// RowTicks returns the effective sub-tick count for the given row.
func (s *Pattern) RowTicks(rowIdx int) int {
	if rowIdx >= 0 && rowIdx < s.NumRows {
		for _, track := range s.Tracks {
			if rowIdx < len(track.Rows) && track.Rows[rowIdx].Ticks > 0 {
				return track.Rows[rowIdx].Ticks
			}
		}
	}
	if s.DefaultTicks > 0 {
		return s.DefaultTicks
	}
	return 6
}
