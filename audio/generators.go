package audio

// StreamerFunc adapts a plain function to the Streamer interface.
type StreamerFunc func(samples [][2]float64) (n int, ok bool)

func (f StreamerFunc) Stream(samples [][2]float64) (n int, ok bool) { return f(samples) }
func (f StreamerFunc) Err() error                                    { return nil }

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

// silenceStreamer is an infinite-silence stream (ok=true always).
type silenceStreamer struct{}

func (s *silenceStreamer) Stream(buf [][2]float64) (int, bool) {
	for i := range buf {
		buf[i] = [2]float64{}
	}
	return len(buf), true
}

func (s *silenceStreamer) Err() error { return nil }

// scaledStreamer multiplies every sample by a constant linear gain.
// When silent is true, it outputs zeros without pulling from the source.
// This replaces beep/v2/effects.Volume{Base:2, Volume:math.Log2(scale)},
// which reduces to plain multiplication: sample * 2^log2(scale) = sample * scale.
type scaledStreamer struct {
	s      Streamer
	scale  float64
	silent bool
}

func newScaledStreamer(s Streamer, scale float64) Streamer {
	if scale == 0 {
		return &scaledStreamer{s: s, silent: true}
	}
	if scale == 1.0 {
		return s
	}
	return &scaledStreamer{s: s, scale: scale}
}

func (ss *scaledStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if ss.silent {
		n, ok = ss.s.Stream(samples)
		for i := range samples[:n] {
			samples[i] = [2]float64{}
		}
		return n, ok
	}
	n, ok = ss.s.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= ss.scale
		samples[i][1] *= ss.scale
	}
	return
}

func (ss *scaledStreamer) Err() error { return ss.s.Err() }

// mixStreamer additively combines N source Streamers into a single output.
// This replaces beep.Mix().
type mixStreamer struct{ sources []Streamer }

// mixAll returns an additive mix of the given sources.
// Returns silence when sources is empty; returns the single source when len==1.
func mixAll(sources ...Streamer) Streamer {
	switch len(sources) {
	case 0:
		return &silenceStreamer{}
	case 1:
		return sources[0]
	}
	return &mixStreamer{sources: sources}
}

func (m *mixStreamer) Stream(samples [][2]float64) (int, bool) {
	for i := range samples {
		samples[i] = [2]float64{}
	}
	tmp := make([][2]float64, len(samples))
	maxN, anyOk := 0, false
	for _, src := range m.sources {
		n, ok := src.Stream(tmp[:len(samples)])
		for i := 0; i < n; i++ {
			samples[i][0] += tmp[i][0]
			samples[i][1] += tmp[i][1]
		}
		if n > maxN {
			maxN = n
		}
		if ok {
			anyOk = true
		}
	}
	return maxN, anyOk
}

func (m *mixStreamer) Err() error { return nil }

