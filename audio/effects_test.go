package audio

import (
	"math"
	"testing"
	"time"
)

func TestEnvelopeAttackRises(t *testing.T) {
	const sr = SampleRate(1000)
	const n = 1000
	// At 1000 Hz, Attack=1s means all 1000 samples are in the attack stage
	env := NewEnvelope(ConstantStreamer(1.0), sr, n, Envelope{Attack: 1 * time.Second})
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
	const sr = SampleRate(1000)
	const n = 100
	// Attack=0, Decay=0, Release=0 → all samples at sustain level
	env := NewEnvelope(ConstantStreamer(1.0), sr, n, Envelope{Sustain: 0.5})
	samples := streamN(env, n)

	for i, s := range samples {
		if math.Abs(s[0]-0.5) > 1e-6 {
			t.Errorf("sample %d: want 0.5, got %v", i, s[0])
		}
	}
}

func TestEnvelopeReleaseFallsToZero(t *testing.T) {
	const sr = SampleRate(1000)
	const n = 1000
	// At 1000 Hz, Release=1s means all 1000 samples are release, level falls from 0.5 to ~0.0001
	env := NewEnvelope(ConstantStreamer(1.0), sr, n, Envelope{Sustain: 0.5, Release: 1 * time.Second})
	samples := streamN(env, n)

	last := samples[n-1][0]
	if last > 0.001 {
		t.Errorf("last release sample should be near 0, got %v", last)
	}
}

func TestEnvelopeStagesProgression(t *testing.T) {
	const sr = SampleRate(1000)
	const n = 100
	// At 1000 Hz: 25ms = 25 samples per stage
	env := NewEnvelope(ConstantStreamer(1.0), sr, n, Envelope{
		Attack: 25 * time.Millisecond, Decay: 25 * time.Millisecond, Sustain: 0.5, Release: 25 * time.Millisecond,
	})
	eg := env.(*envelopeGenerator)

	buf := make([][2]float64, 1)

	for i := range 25 {
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
	for range n {
		level *= m
	}
	// The formula is a first-order approximation; accept 0.5% relative error
	if math.Abs(level/end-1.0) > 0.005 {
		t.Errorf("after %d steps [%v→%v]: got %v (%.3f%% error)",
			n, start, end, level, math.Abs(level/end-1.0)*100)
	}
}
