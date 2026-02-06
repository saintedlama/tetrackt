package audio

import (
	"fmt"
	"strconv"
)

// Effect represents a tracker effect command
type Effect struct {
	Code   int // Effect code (0 for arpeggio)
	Param1 int // First parameter (x in 0xy)
	Param2 int // Second parameter (y in 0xy)
}

// NoEffect returns an empty effect
func NoEffect() Effect {
	return Effect{Code: -1, Param1: 0, Param2: 0}
}

// IsNoEffect checks if an effect is empty
func IsNoEffect(e Effect) bool {
	return e.Code == -1
}

// ParseEffect parses an effect string (e.g., "0C7") into an Effect struct
func ParseEffect(effectStr string) (Effect, error) {
	// Handle empty or default effect
	if effectStr == "" || effectStr == "---" || effectStr == "..." {
		return NoEffect(), nil
	}

	// Ensure we have at least 3 characters
	if len(effectStr) < 3 {
		return NoEffect(), fmt.Errorf("invalid effect format: %s", effectStr)
	}

	// Parse effect code (first character, hex)
	code, err := strconv.ParseInt(string(effectStr[0]), 16, 32)
	if err != nil {
		return NoEffect(), fmt.Errorf("invalid effect code: %s", effectStr)
	}

	// Parse first parameter (second character, hex)
	param1, err := strconv.ParseInt(string(effectStr[1]), 16, 32)
	if err != nil {
		return NoEffect(), fmt.Errorf("invalid effect param1: %s", effectStr)
	}

	// Parse second parameter (third character, hex)
	param2, err := strconv.ParseInt(string(effectStr[2]), 16, 32)
	if err != nil {
		return NoEffect(), fmt.Errorf("invalid effect param2: %s", effectStr)
	}

	return Effect{
		Code:   int(code),
		Param1: int(param1),
		Param2: int(param2),
	}, nil
}

// String formats an Effect as a string (e.g., "0C7")
func (e Effect) String() string {
	if IsNoEffect(e) {
		return "---"
	}
	return fmt.Sprintf("%X%X%X", e.Code, e.Param1, e.Param2)
}

// IsArpeggio checks if this is an arpeggio effect (code 0)
func (e Effect) IsArpeggio() bool {
	return e.Code == 0 && (e.Param1 != 0 || e.Param2 != 0)
}

// GetArpeggioOffset returns the semitone offset for the given tick in an arpeggio cycle
// tick 0: base note (offset 0)
// tick 1: base + param1 semitones
// tick 2: base + param2 semitones
// then repeats
func (e Effect) GetArpeggioOffset(tick int) int {
	if !e.IsArpeggio() {
		return 0
	}

	switch tick % 3 {
	case 0:
		return 0
	case 1:
		return e.Param1
	case 2:
		return e.Param2
	default:
		return 0
	}
}
