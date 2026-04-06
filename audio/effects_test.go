package audio

import (
	"math"
	"testing"

	"github.com/gopxl/beep/v2"
)

// constStreamer returns a beep.Streamer that always outputs v on both channels.
type constStreamerType struct{ v float64 }

func (c *constStreamerType) Stream(samples [][2]float64) (int, bool) {
	for i := range samples {
		samples[i][0] = c.v
		samples[i][1] = c.v
	}
	return len(samples), true
}

func (c *constStreamerType) Err() error { return nil }

func constStreamer(v float64) beep.Streamer {
	return &constStreamerType{v: v}
}

func TestEnvelopeAttackRises(t *testing.T) {
	const n = 1000
	// Attack=1.0 means all n samples are in the attack stage
	env := NewEnvelope(constStreamer(1.0), n, Envelope{Attack: 1.0})
	samples := streamN(env, n)

	first := samples[0][0]
	last := samples[n-1][0]

	if first > 0.01 {
		t.Errorf("first attack sample should be near 0, got %v", first)
	}
	if last < 0.9 {
		t.Errorf("last attack sample should be near 1, got %v", last)
	}
}

func TestEnvelopeSustainFlat(t *testing.T) {
	const n = 100
	// Attack=0, Decay=0, Release=0 → all samples at sustain level
	env := NewEnvelope(constStreamer(1.0), n, Envelope{Sustain: 0.5})
	samples := streamN(env, n)

	for i, s := range samples {
		if math.Abs(s[0]-0.5) > 1e-6 {
			t.Errorf("sample %d: want 0.5, got %v", i, s[0])
		}
	}
}

func TestEnvelopeReleaseFallsToZero(t *testing.T) {
	const n = 1000
	// Release=1.0, others=0 → all n samples are release, level falls from 0.5 to ~0.0001
	env := NewEnvelope(constStreamer(1.0), n, Envelope{Sustain: 0.5, Release: 1.0})
	samples := streamN(env, n)

	last := samples[n-1][0]
	if last > 0.001 {
		t.Errorf("last release sample should be near 0, got %v", last)
	}
}

func TestEnvelopeStagesProgression(t *testing.T) {
	const n = 100
	// Equal ADSR fractions: attack=25, decay=25, sustain=25, release=25
	env := NewEnvelope(constStreamer(1.0), n, Envelope{
		Attack: 0.25, Decay: 0.25, Sustain: 0.5, Release: 0.25,
	})
	eg := env.(*envelopeGenerator)

	buf := make([][2]float64, 1)

	for i := 0; i < 25; i++ {
		env.Stream(buf)
		if eg.currentStage != StageAttack {
			t.Errorf("sample %d: expected StageAttack, got %v", i, eg.currentStage)
		}
	}
	for i := 25; i < 50; i++ {
		env.Stream(buf)
		if eg.currentStage != StageDecay {
			t.Errorf("sample %d: expected StageDecay, got %v", i, eg.currentStage)
		}
	}
	for i := 50; i < 75; i++ {
		env.Stream(buf)
		if eg.currentStage != StageSustain {
			t.Errorf("sample %d: expected StageSustain, got %v", i, eg.currentStage)
		}
	}
	for i := 75; i < 100; i++ {
		env.Stream(buf)
		if eg.currentStage != StageRelease {
			t.Errorf("sample %d: expected StageRelease, got %v", i, eg.currentStage)
		}
	}
}

func TestCalculateMultiplier(t *testing.T) {
	start, end := 0.001, 1.0
	n := 10000
	m := calculateMultiplier(start, end, n)

	level := start
	for i := 0; i < n; i++ {
		level *= m
	}
	// The formula is a first-order approximation; accept 0.5% relative error
	if math.Abs(level/end-1.0) > 0.005 {
		t.Errorf("after %d steps [%v→%v]: got %v (%.3f%% error)",
			n, start, end, level, math.Abs(level/end-1.0)*100)
	}
}
