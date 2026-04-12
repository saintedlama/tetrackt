package tracker

func generateInlineArpOffsets(preset, ticks, step int) []int {
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
