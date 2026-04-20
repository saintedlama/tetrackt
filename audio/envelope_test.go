package audio

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelopeAttackRises(t *testing.T) {
	const sr = SampleRate(1000)
	const n = 1000
	// At 1000 Hz, Attack=1s means all 1000 samples are in the attack stage
	env := NewEnvelope(ConstantStreamer(1.0), sr, n, Envelope{Attack: 1 * time.Second})
	samples := StreamN(env, n)

	first := samples[0][0]
	last := samples[n-1][0]

	assert.LessOrEqual(t, first, 0.01, "first attack sample should be near 0")
	assert.GreaterOrEqual(t, last, 0.9, "last attack sample should be near 1")
}

func TestEnvelopeSustainFlat(t *testing.T) {
	const sr = SampleRate(1000)
	const n = 100
	// Attack=0, Decay=0, Release=0 → all samples at sustain level
	env := NewEnvelope(ConstantStreamer(1.0), sr, n, Envelope{Sustain: 0.5})
	samples := StreamN(env, n)

	for i, s := range samples {
		assert.InDelta(t, 0.5, s[0], 1e-6, "sample %d: want 0.5", i)
	}
}

func TestEnvelopeReleaseFallsToZero(t *testing.T) {
	const sr = SampleRate(1000)
	const n = 1000
	// At 1000 Hz, Release=1s means all 1000 samples are release, level falls from 0.5 to ~0.0001
	env := NewEnvelope(ConstantStreamer(1.0), sr, n, Envelope{Sustain: 0.5, Release: 1 * time.Second})
	samples := StreamN(env, n)

	last := samples[n-1][0]
	assert.LessOrEqual(t, last, 0.001, "last release sample should be near 0")

	for i := 1; i < n; i++ {
		assert.Less(t, samples[i][0], samples[i-1][0], "release sample %d (%v) should be less than sample %d (%v)", i, samples[i][0], i-1, samples[i-1][0])
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
		assert.Equal(t, StageAttack, eg.currentStage, "sample %d: expected StageAttack", i)
	}
	for i := 25; i < 50; i++ {
		env.Stream(buf)
		assert.Equal(t, StageDecay, eg.currentStage, "sample %d: expected StageDecay", i)
	}
	for i := 50; i < 75; i++ {
		env.Stream(buf)
		assert.Equal(t, StageSustain, eg.currentStage, "sample %d: expected StageSustain", i)
	}
	for i := 75; i < 100; i++ {
		env.Stream(buf)
		assert.Equal(t, StageRelease, eg.currentStage, "sample %d: expected StageRelease", i)
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
	relativeError := math.Abs(level/end - 1.0)
	require.Less(t, relativeError, 0.005,
		"after %d steps [%v→%v]: got %v (%.3f%% error)", n, start, end, level, relativeError*100)
}
