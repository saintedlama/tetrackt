package tracker

import (
	"github.com/tetrackt/tetrackt/notes"
	"github.com/tetrackt/tetrackt/render"
)

// ToRenderPattern translates the tracker model into a render.Pattern.
// All note data is converted from note/octave pairs to frequencies (Hz).
// Display-only concepts (e.g. "---" rest markers) become Frequency == 0.
func (m *TrackerModel) ToRenderPattern() *render.Pattern {
	tracks := make([]render.Track, m.NumTracks)
	for i, track := range m.Tracks {
		rows := make([]render.Row, m.NumRows)
		for j, row := range track.Rows {
			rows[j] = toRenderRow(row)
		}
		tracks[i] = render.Track{
			Synth: track.Synth,
			Rows:  rows,
		}
	}
	return &render.Pattern{
		Tracks:      tracks,
		NumRows:     m.NumRows,
		NumTracks:   m.NumTracks,
		RowDuration: m.BPM.Duration(),
	}
}

// ToRenderRow translates a single TrackRow into a render.Row.
func ToRenderRow(row TrackRow) render.Row {
	return toRenderRow(row)
}

func toRenderRow(row TrackRow) render.Row {
	rr := render.Row{
		Arpeggio: row.Arpeggio,
		Effect: render.RowEffect{
			Type:  render.EffectType(row.Effect.Type),
			Param: row.Effect.Param,
		},
		Ticks: row.Ticks,
	}
	if !notes.IsOff(row.Note) {
		rr.Frequency = row.Note.Frequency()
		rr.Volume = float64(row.Volume) / 64.0
	}
	return rr
}
