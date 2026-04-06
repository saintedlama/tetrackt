package ui

import "github.com/tetrackt/tetrackt/audio"

// NoteKeys maps keyboard characters to note base names for note input.
// Shared between main.go and any dialog that needs note playback.
var NoteKeys = map[string]audio.Base{
	"1":  "C",
	"!":  "C#",
	"2":  "D",
	"@":  "D#",
	"\"": "D#", // german keyboard layout
	"3":  "E",
	"4":  "F",
	"$":  "F#",
	"5":  "G",
	"%":  "G#",
	"6":  "A",
	"^":  "A#",
	"&":  "A#", // german keyboard layout
	"7":  "B",
}
