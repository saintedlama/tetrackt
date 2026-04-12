package ui

import "github.com/tetrackt/tetrackt/audio"

// NoteKeys maps keyboard-piano characters to note base names for note input.
// Shared between main.go and any dialog that needs note playback.
// Layout (QWERTY):
//
//	Lower row: Z S X D C V G B H N J M
//	Upper row: Q 2 W 3 E R 5 T 6 Y 7 U
var NoteKeys = map[string]audio.Base{
	"z": "C",
	"s": "C#",
	"x": "D",
	"d": "D#",
	"c": "E",
	"v": "F",
	"g": "F#",
	"b": "G",
	"h": "G#",
	"n": "A",
	"j": "A#",
	"m": "B",

	"q": "C",
	"2": "C#",
	"w": "D",
	"3": "D#",
	"e": "E",
	"r": "F",
	"5": "F#",
	"t": "G",
	"6": "G#",
	"y": "A",
	"7": "A#",
	"u": "B",
}
