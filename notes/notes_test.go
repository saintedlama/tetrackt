package notes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoteString(t *testing.T) {
	// single-char base formats as "C-4"
	assert.Equal(t, "C-4", NewNote(BaseC, Octave4).String(), "C4")
	// sharp (two-char base) formats as "C#4"
	assert.Equal(t, "C#4", NewNote(BaseCs, Octave4).String(), "C#4")
	// off note
	assert.Equal(t, "---", Off().String(), "Off")
}

func TestIsOff(t *testing.T) {
	assert.True(t, IsOff(Off()), "Off() should be off")
	assert.False(t, IsOff(NewNote(BaseA, Octave4)), "A4 should not be off")
}

func TestNoteFrequency(t *testing.T) {
	tests := []struct {
		note Note
		want float64
	}{
		{NewNote(BaseA, Octave4), 440.0},
		{NewNote(BaseA, Octave3), 220.0}, // one octave down = half
		{NewNote(BaseA, Octave5), 880.0}, // one octave up = double
		{Off(), 0.0},
	}
	for _, tt := range tests {
		got := tt.note.Frequency()
		assert.InDelta(t, tt.want, got, 1e-6, "%s frequency", tt.note)
	}
}

func TestNoteTranspose(t *testing.T) {
	// +1 semitone: C4 → C#4
	transposed, ok := NewNote(BaseC, Octave4).Transpose(1)
	require.True(t, ok, "C4+1: expected ok=true")
	assert.Equal(t, NewNote(BaseCs, Octave4), transposed, "C4+1: expected C#4")

	// +12 semitones (1 octave): C4 → C5
	transposed, ok = NewNote(BaseC, Octave4).Transpose(12)
	require.True(t, ok, "C4+12: expected ok=true")
	assert.Equal(t, NewNote(BaseC, Octave5), transposed, "C4+12: expected C5")

	// -1 semitone: C4 → B3
	transposed, ok = NewNote(BaseC, Octave4).Transpose(-1)
	require.True(t, ok, "C4-1: expected ok=true")
	assert.Equal(t, NewNote(BaseB, Octave3), transposed, "C4-1: expected B3")

	// -12 semitones: C4 → C3
	transposed, ok = NewNote(BaseC, Octave4).Transpose(-12)
	require.True(t, ok, "C4-12: expected ok=true")
	assert.Equal(t, NewNote(BaseC, Octave3), transposed, "C4-12: expected C3")

	// beyond range: B8 + 1 semitone is out of range
	_, ok = NewNote(BaseB, Octave8).Transpose(1)
	assert.False(t, ok, "B8+1: expected ok=false (beyond octave 8)")

	// C8 + 12 -> C9 which is out of range
	_, ok = NewNote(BaseC, Octave8).Transpose(12)
	assert.False(t, ok, "C8+12: expected ok=false (beyond octave 8)")

	// C0 - 1 semitone is out of range
	_, ok = NewNote(BaseC, Octave0).Transpose(-1)
	assert.False(t, ok, "C0-1: expected ok=false (below octave 0)")

	// off note returns false
	_, ok = Off().Transpose(1)
	assert.False(t, ok, "Off+1: expected ok=false")
}
