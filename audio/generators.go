package audio

// ConstantStreamer returns a Streamer that outputs a constant signal on both channels.
func ConstantStreamer(v float64) Streamer {
	return StreamerFunc(func(samples [][2]float64) (int, bool) {
		for i := range samples {
			samples[i][0] = v
			samples[i][1] = v
		}
		return len(samples), true
	})
}

func StreamN(s Streamer, n int) [][2]float64 {
	buf := make([][2]float64, n)
	s.Stream(buf)
	return buf
}
