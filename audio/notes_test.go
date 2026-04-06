package audio

import (
	"math"
	"testing"
)

func TestNoteString(t *testing.T) {
	// single-char base formats as "C-4"
	if got := NewNote(BaseC, Octave4).String(); got != "C-4" {
		t.Errorf("C4: want C-4, got %q", got)
	}
	// sharp (two-char base) formats as "C#4"
	if got := NewNote(BaseCs, Octave4).String(); got != "C#4" {
		t.Errorf("C#4: want C#4, got %q", got)
	}
	// off note
	if got := Off().String(); got != "---" {
		t.Errorf("Off: want ---, got %q", got)
	}
}

func TestIsOff(t *testing.T) {
	if !IsOff(Off()) {
		t.Error("Off() should be off")
	}
	if IsOff(NewNote(BaseA, Octave4)) {
		t.Error("A4 should not be off")
	}
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
		if math.Abs(got-tt.want) > 1e-6 {
			t.Errorf("%s frequency: want %v, got %v", tt.note, tt.want, got)
		}
	}
}

func TestNoteTranspose(t *testing.T) {
	note := NewNote(BaseC, Octave4)

	transposed, ok := note.Transpose(1)
	if !ok {
		t.Error("C4+1: expected ok=true")
	}
	if transposed.Octave != Octave5 {
		t.Errorf("C4+1: expected octave 5, got %v", transposed.Octave)
	}

	// beyond octave 8 returns false
	_, ok = NewNote(BaseC, Octave8).Transpose(1)
	if ok {
		t.Error("C8+1: expected ok=false (beyond octave 8)")
	}

	// off note returns false
	_, ok = Off().Transpose(1)
	if ok {
		t.Error("Off+1: expected ok=false")
	}
}
