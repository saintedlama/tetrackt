package tracker

import (
	"strconv"
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
