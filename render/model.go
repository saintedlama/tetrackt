package render

import (
	"time"

	"github.com/tetrackt/tetrackt/audio"
)

// Row is the frequency-domain representation of a single pattern cell.
// It contains no display strings — all note data has been converted to Hz.
// FX carries the complete effect definitions for this row; Ticks within
// FX determines how many sub-ticks the row is divided into (0 = 1 sub-tick).
type Row struct {
	Frequency float64              // Hz; 0 = rest (no note)
	FX        audio.EffectDefinitions
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
