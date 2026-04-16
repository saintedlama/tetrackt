package audio

// LinearStreamer returns a Streamer that outputs a constant 1.0 signal on both channels.
func LinearStreamer() Streamer {
	return StreamerFunc(func(samples [][2]float64) (int, bool) {
		for i := range samples {
			samples[i][0] = 1.0
			samples[i][1] = 1.0
		}
		return len(samples), true
	})
}
