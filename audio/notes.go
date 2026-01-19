package audio

import (
	"fmt"
	"math"
)

type Note struct {
	Base   Base
	Octave Octave
}

func NewNote(base string, octave int) Note {
	return Note{Base: Base(base), Octave: Octave(octave)}
}

func Off() Note {
	return Note{Base: BaseOff, Octave: Octave0}
}

func IsOff(note Note) bool {
	return note.Base == BaseOff
}

func (note Note) String() string {
	if note.Base == BaseOff {
		return "---"
	}

	if len(string(note.Base)) < 2 {
		return fmt.Sprintf("%s-%d", note.Base, note.Octave)
	}

	return fmt.Sprintf("%s%d", note.Base, note.Octave)
}

type Base string

const (
	BaseC   Base = "C"
	BaseCs  Base = "C#"
	BaseD   Base = "D"
	BaseDs  Base = "D#"
	BaseE   Base = "E"
	BaseF   Base = "F"
	BaseFs  Base = "F#"
	BaseG   Base = "G"
	BaseGs  Base = "G#"
	BaseA   Base = "A"
	BaseAs  Base = "A#"
	BaseB   Base = "B"
	BaseOff Base = "---"
)

type Octave int

const (
	Octave0 Octave = 0
	Octave1 Octave = 1
	Octave2 Octave = 2
	Octave3 Octave = 3
	Octave4 Octave = 4
	Octave5 Octave = 5
	Octave6 Octave = 6
	Octave7 Octave = 7
	Octave8 Octave = 8
)

// Base note data (octave-agnostic)
var noteBaseFrequencies = map[Base]float64{
	"C":  261.63, // C4
	"C#": 277.18, // C#4 (Db4)
	"D":  293.66, // D4
	"D#": 311.13, // D#4 (Eb4)
	"E":  329.63, // E4
	"F":  349.23, // F4
	"F#": 369.99, // F#4 (Gb4)
	"G":  392.00, // G4
	"G#": 415.30, // G#4 (Ab4)
	"A":  440.00, // A4
	"A#": 466.16, // A#4 (Bb4)
	"B":  493.88, // B4
}

func (note Note) Transpose(delta int) (Note, bool) {
	if note.Base == BaseOff {
		return note, false
	}

	newOctave := note.Octave + Octave(delta)
	if newOctave > Octave8 {
		return note, false
	}

	return Note{Base: note.Base, Octave: newOctave}, true
}

func (note Note) Frequency() float64 {
	baseFreq, ok := noteBaseFrequencies[note.Base]
	if !ok {
		return 0
	}

	// Reference octave is 4 in noteBaseFrequencies
	offset := float64(note.Octave - 4)
	return baseFreq * math.Pow(2, offset)
}
