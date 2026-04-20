package ui

import (
	"strings"

	"github.com/tetrackt/tetrackt/notes"
)

type InputProfile string

const (
	InputProfileQWERTY InputProfile = "qwerty"
	InputProfileQWERTZ InputProfile = "qwertz"
)

var currentInputProfile = InputProfileQWERTY

// NoteKeys maps keyboard-piano characters to note base names for note input.
// Shared between tracker/synth/patch-bank note preview paths.
var NoteKeys = noteKeysForProfile(currentInputProfile)

func noteKeysForProfile(profile InputProfile) map[string]notes.Base {
	naturalLower := "zxcvbnm"
	naturalUpper := "qwertyu"
	if profile == InputProfileQWERTZ {
		naturalLower = "yxcvbnm"
		naturalUpper = "qwertzu"
	}

	m := map[string]notes.Base{
		"s": "C#",
		"d": "D#",
		"g": "F#",
		"h": "G#",
		"j": "A#",

		"2": "C#",
		"3": "D#",
		"5": "F#",
		"6": "G#",
		"7": "A#",
	}

	assignNaturals := func(keys string) {
		m[string(keys[0])] = "C"
		m[string(keys[1])] = "D"
		m[string(keys[2])] = "E"
		m[string(keys[3])] = "F"
		m[string(keys[4])] = "G"
		m[string(keys[5])] = "A"
		m[string(keys[6])] = "B"
	}

	assignNaturals(naturalLower)
	assignNaturals(naturalUpper)
	return m
}

func InputProfileFromString(raw string) InputProfile {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == string(InputProfileQWERTZ) {
		return InputProfileQWERTZ
	}
	return InputProfileQWERTY
}

func SetInputProfile(profile InputProfile) {
	if profile != InputProfileQWERTZ {
		profile = InputProfileQWERTY
	}
	currentInputProfile = profile
	NoteKeys = noteKeysForProfile(profile)
}

func SetInputProfileFromString(raw string) InputProfile {
	profile := InputProfileFromString(raw)
	SetInputProfile(profile)
	return profile
}

func CurrentInputProfile() InputProfile {
	return currentInputProfile
}

func NoteMappingRows(profile InputProfile) (lower, upper string) {
	if profile == InputProfileQWERTZ {
		return "Y S X D C V G B H N J M", "Q 2 W 3 E R 5 T 6 Z 7 U"
	}

	return "Z S X D C V G B H N J M", "Q 2 W 3 E R 5 T 6 Y 7 U"
}
