package effects

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tetrackt/tetrackt/audio"
)

// Type identifies the per-row playback effect evaluated each sub-tick.
type Type int

const (
	None        Type = iota
	Vibrato          // Param hi-nibble: speed (1-15 ticks/cycle); lo-nibble: depth (semitones*4, 0-15)
	VolumeSlide      // Param: volume delta per tick in 1/64 units; positive = louder
	NoteCut          // Param: sub-tick at which to silence the note
	NoteDelay        // Param: sub-tick at which to trigger NoteOn
	RowTicks         // Param: 00 clears row tick override; 01..20 sets per-row sub-ticks
	Continuous       // Param: 00 = off, 01 = on
	ArpPreset        // Param: high nibble preset, low nibble step bucket
)

// Effect is a per-row effect command evaluated every sub-tick during playback.
type Effect struct {
	Type  Type
	Param int
}

// Format returns the display string for the effect type (1 character).
func (t Type) Format() string {
	switch t {
	case Vibrato:
		return "V"
	case VolumeSlide:
		return "S"
	case NoteCut:
		return "C"
	case NoteDelay:
		return "D"
	case RowTicks:
		return "T"
	case Continuous:
		return "O"
	case ArpPreset:
		return "A"
	default:
		return "."
	}
}

// FormatParam returns the 2-character hex display string for the effect parameter.
func (t Type) FormatParam(param int) string {
	if t == VolumeSlide && param < 0 {
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

// FormatArpeggio formats an arpeggio effect for display (3 chars).
// Active arp with offsets [0,4,7] shows "A47"; inactive shows "---".
func FormatArpeggio(arp audio.ArpeggioEffect) string {
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

// ParseKey parses a single key press into an effect type.
// Supports both letter aliases (v, s, c, d, t, o, a) and hex nibbles (0-7).
func ParseKey(key string) (Type, bool) {
	switch strings.ToLower(key) {
	case "v":
		return Vibrato, true
	case "s":
		return VolumeSlide, true
	case "c":
		return NoteCut, true
	case "d":
		return NoteDelay, true
	case "t":
		return RowTicks, true
	case "o":
		return Continuous, true
	case "a":
		return ArpPreset, true
	}

	if v, ok := parseHexNibble(key); ok {
		return ParseNibble(v)
	}

	return None, false
}

// ParseNibble parses a hex nibble (0-7) into an effect type.
func ParseNibble(v int) (Type, bool) {
	if v < int(None) || v > int(ArpPreset) {
		return None, false
	}
	return Type(v), true
}

// DecodeParam decodes and validates an effect parameter based on the effect type.
// Returns the decoded value and whether it was valid.
func (t Type) DecodeParam(param int) (int, bool) {
	if param < 0 || param > 255 {
		return 0, false
	}

	switch t {
	case None:
		return 0, true
	case Vibrato, NoteCut, NoteDelay, ArpPreset:
		return param, true
	case VolumeSlide:
		return int(int8(uint8(param))), true
	case RowTicks:
		if param == 0 || (param >= 1 && param <= 32) {
			return param, true
		}
		return 0, false
	case Continuous:
		if param == 0 || param == 1 {
			return param, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// ApplyResult describes what an effect application changed.
type ApplyResult struct {
	Effect     Effect
	Speed      int  // Row speed override (0 = no override)
	Continuous bool // Whether the note should continue
	Arpeggio   audio.ArpeggioEffect
}

// Apply applies an effect with the given parameter, returning the changes to make.
// The defaultSpeed is used for arpeggio preset calculation when the row speed is 0.
func Apply(effectType Type, param int, currentSpeed int, defaultSpeed int) (ApplyResult, bool) {
	decoded, ok := effectType.DecodeParam(param)
	if !ok {
		return ApplyResult{}, false
	}

	result := ApplyResult{
		Effect: Effect{Type: effectType, Param: param},
	}

	switch effectType {
	case None:
		result.Effect.Param = 0
		return result, true

	case VolumeSlide:
		result.Effect.Param = decoded // signed delta for playback
		return result, true

	case Vibrato, NoteCut, NoteDelay:
		result.Effect.Param = decoded
		return result, true

	case RowTicks:
		if decoded == 0 {
			result.Speed = 0
		} else {
			result.Speed = decoded
		}
		return result, true

	case Continuous:
		result.Continuous = decoded == 1
		return result, true

	case ArpPreset:
		preset := (param >> 4) & 0xF
		step := param & 0xF
		if step == 0 {
			step = 4
		}
		if preset == 0 {
			result.Arpeggio = audio.ArpeggioEffect{}
			return result, true
		}
		if preset > 5 {
			return ApplyResult{}, false
		}

		ticks := currentSpeed
		if ticks <= 0 {
			ticks = defaultSpeed
		}
		result.Arpeggio = audio.ArpeggioEffect{Offsets: generateArpOffsets(preset, ticks, step)}
		result.Continuous = true
		return result, true

	default:
		return ApplyResult{}, false
	}
}

// parseHexNibble parses a single hex character into its numeric value.
func parseHexNibble(key string) (int, bool) {
	if len(key) != 1 {
		return 0, false
	}
	v, err := strconv.ParseInt(key, 16, 32)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

// generateArpOffsets generates arpeggio offsets for the given preset pattern.
func generateArpOffsets(preset, ticks, step int) []int {
	if ticks <= 0 {
		return nil
	}

	degrees := make([]int, ticks)
	for i := range ticks {
		degrees[i] = i * step
	}

	switch preset {
	case 1: // Up
		return degrees
	case 2: // Down
		out := make([]int, ticks)
		for i, v := range degrees {
			out[ticks-1-i] = v
		}
		return out
	case 3: // Converge
		out := make([]int, 0, ticks)
		lo, hi := 0, ticks-1
		for lo <= hi {
			out = append(out, degrees[lo])
			lo++
			if lo <= hi {
				out = append(out, degrees[hi])
				hi--
			}
		}
		return out
	case 4: // Diverge
		out := make([]int, 0, ticks)
		lo := (ticks - 1) / 2
		hi := ticks / 2
		if ticks%2 == 1 {
			out = append(out, degrees[lo])
			lo--
			hi++
		}
		for hi < ticks && len(out) < ticks {
			if lo >= 0 {
				out = append(out, degrees[lo])
			}
			out = append(out, degrees[hi])
			lo--
			hi++
		}
		if len(out) > ticks {
			return out[:ticks]
		}
		return out
	case 5: // Random (stable LCG)
		out := make([]int, ticks)
		copy(out, degrees)
		s := uint64(ticks*131 + step*17 + preset)
		for i := ticks - 1; i > 0; i-- {
			s = s*6364136223846793005 + 1442695040888963407
			j := int(s>>33) % (i + 1)
			out[i], out[j] = out[j], out[i]
		}
		return out
	default:
		return degrees
	}
}
