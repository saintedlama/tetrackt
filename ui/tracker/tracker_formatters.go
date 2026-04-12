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

// formatArpeggio formats an arpeggio effect for display (3 chars).
// Active arp with offsets [0,4,7] shows "A47"; inactive shows "---".
func formatArpeggio(arp audio.ArpeggioEffect) string {
	if !arp.IsActive() {
		return "---"
	}
	o1, o2 := 0, 0
	if len(arp.Offsets) > 1 {
		o1 = arp.Offsets[1] % 10
	}
	if len(arp.Offsets) > 2 {
		o2 = arp.Offsets[2] % 10
	}
	return fmt.Sprintf("A%d%d", o1, o2)
}

func formatEffectType(t EffectType) string {
	switch t {
	case EffectVibrato:
		return "V"
	case EffectVolumeSlide:
		return "S"
	case EffectNoteCut:
		return "C"
	case EffectNoteDelay:
		return "D"
	case EffectRowTicks:
		return "T"
	case EffectContinuous:
		return "O"
	case EffectArpPreset:
		return "A"
	default:
		return "."
	}
}

func formatEffectParam(effectType EffectType, param int) string {
	if effectType == EffectVolumeSlide && param < 0 {
		return fmt.Sprintf("%02X", uint8(int8(param)))
	}
	if param < 0 {
		param = 0
	}
	if param > 255 {
		param = 255
	}
	return fmt.Sprintf("%02X", param)
}
