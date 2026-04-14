package tracker

import (
	"fmt"

	"github.com/tetrackt/tetrackt/audio"
)

func formatNote(note audio.Note) string {
	if note.Base == audio.BaseOff {
		return "---"
	}

	if len(string(note.Base)) < 2 {
		return fmt.Sprintf("%s-%d", note.Base, note.Octave)
	}

	return fmt.Sprintf("%s%d", note.Base, note.Octave)
}

// formatVolume formats volume value for display.
func formatVolume(volume int) string {
	if volume == 0 {
		return ".."
	}
	return fmt.Sprintf("%02d", volume)
}
