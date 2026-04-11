package audio

import (
	"math"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
)

// MixMode controls the post-mix summing curve applied after all channels are blended.
type MixMode int

const (
	MixLinear   MixMode = iota // additive linear sum — no extra processing
	MixNESPulse                // NES APU pulse-channel nonlinear table approximation
	MixSoftClip                // tanh-based soft saturation (SID-like warm overdrive)
)

// MixModeCount returns the total number of mix modes.
func MixModeCount() int { return 3 }

// String returns a short display name for the mode.
func (m MixMode) String() string {
	switch m {
	case MixNESPulse:
		return "NES"
	case MixSoftClip:
		return "Clip"
	default:
		return "Linear"
	}
}

// Mixer holds per-oscillator volume levels and knows how to combine two
// streamers into a single mixed output stream.
type Mixer struct {
	Volume1      float64 // 0.0–1.0, independent volume for oscillator 1
	Volume2      float64 // 0.0–1.0, independent volume for oscillator 2
	Pan1         float64 // -1.0 (full left) to +1.0 (full right); 0.0 = centre
	Pan2         float64
	MasterVolume float64 // 0.0–1.0 output gain; zero value treated as 1.0 (unity)
	Mute1        bool    // silence channel 1 while preserving its volume/pan settings
	Mute2        bool
	Mode         MixMode // post-mix summing curve
}

// panStreamer applies constant-power stereo panning to a signal.
// pan in [-1, +1]; 0 = centre (√2/2 on each side).
type panStreamer struct {
	s     beep.Streamer
	gainL float64
	gainR float64
}

func newPanStreamer(s beep.Streamer, pan float64) beep.Streamer {
	if pan == 0 {
		return s // centre: skip to preserve original amplitude
	}
	theta := (pan + 1) * math.Pi / 4
	return &panStreamer{s: s, gainL: math.Cos(theta), gainR: math.Sin(theta)}
}

func (p *panStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = p.s.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= p.gainL
		samples[i][1] *= p.gainR
	}
	return
}

func (p *panStreamer) Err() error { return p.s.Err() }

// mixModeStreamer applies a post-mix shaping curve sample-by-sample.
type mixModeStreamer struct {
	s    beep.Streamer
	mode MixMode
}

func (ms *mixModeStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = ms.s.Stream(samples)
	switch ms.mode {
	case MixNESPulse:
		for i := range samples[:n] {
			samples[i][0] = nesApprox(samples[i][0])
			samples[i][1] = nesApprox(samples[i][1])
		}
	case MixSoftClip:
		for i := range samples[:n] {
			samples[i][0] = math.Tanh(samples[i][0])
			samples[i][1] = math.Tanh(samples[i][1])
		}
	}
	return
}

func (ms *mixModeStreamer) Err() error { return ms.s.Err() }

// nesApprox models the NES APU pulse-channel lookup table for a float sample.
// Input is the summed stereo sample (~[-2,2]); output is normalised to [-1,1].
const nesMaxOut = 0.2584 // 95.88 / (8128/30 + 100) — table value at max input

func nesApprox(x float64) float64 {
	if x == 0 {
		return 0
	}
	sign := 1.0
	if x < 0 {
		sign = -1
		x = -x
	}
	// Map sample magnitude [0,2] → NES table range [0,30]
	t := (x / 2.0) * 30.0
	if t < 0.001 {
		t = 0.001
	}
	v := 95.88 / (8128.0/t + 100.0)
	return sign * v / nesMaxOut // normalise so max-input → 1.0
}

// Mix applies per-channel volume, muting and panning to s1/s2, sums them,
// applies the MixMode shaping curve, then applies master volume.
func (m Mixer) Mix(s1, s2 beep.Streamer) beep.Streamer {
	v1 := &effects.Volume{Streamer: s1, Base: 2, Volume: math.Log2(m.Volume1), Silent: m.Volume1 == 0 || m.Mute1}
	v2 := &effects.Volume{Streamer: s2, Base: 2, Volume: math.Log2(m.Volume2), Silent: m.Volume2 == 0 || m.Mute2}
	var mixed beep.Streamer = beep.Mix(newPanStreamer(v1, m.Pan1), newPanStreamer(v2, m.Pan2))

	if m.Mode != MixLinear {
		mixed = &mixModeStreamer{s: mixed, mode: m.Mode}
	}

	masterVol := m.MasterVolume
	if masterVol == 0 {
		masterVol = 1.0 // zero value = unity (backward-compatible default)
	}
	if masterVol == 1.0 {
		return mixed
	}
	return &effects.Volume{Streamer: mixed, Base: 2, Volume: math.Log2(masterVol), Silent: false}
}
