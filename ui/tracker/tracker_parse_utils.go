package tracker

import (
	"strconv"
	"strings"
)

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

func parseEffectCommandNibble(v int) (EffectType, bool) {
	if v < int(EffectNone) || v > int(EffectArpPreset) {
		return EffectNone, false
	}
	return EffectType(v), true
}

func parseEffectCommandKey(key string) (EffectType, bool) {
	switch strings.ToLower(key) {
	case "v":
		return EffectVibrato, true
	case "s":
		return EffectVolumeSlide, true
	case "c":
		return EffectNoteCut, true
	case "d":
		return EffectNoteDelay, true
	case "t":
		return EffectRowTicks, true
	case "o":
		return EffectContinuous, true
	case "a":
		return EffectArpPreset, true
	}

	if v, ok := parseHexNibble(key); ok {
		return parseEffectCommandNibble(v)
	}

	return EffectNone, false
}

func decodeEffectParam(effectType EffectType, param int) (int, bool) {
	if param < 0 || param > 255 {
		return 0, false
	}

	switch effectType {
	case EffectNone:
		return 0, true
	case EffectVibrato, EffectNoteCut, EffectNoteDelay, EffectArpPreset:
		return param, true
	case EffectVolumeSlide:
		return int(int8(uint8(param))), true
	case EffectRowTicks:
		if param == 0 || (param >= 1 && param <= 32) {
			return param, true
		}
		return 0, false
	case EffectContinuous:
		if param == 0 || param == 1 {
			return param, true
		}
		return 0, false
	default:
		return 0, false
	}
}
