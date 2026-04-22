package audio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper functions for safely testing streamers without infinite loops

// streamWithLimit streams at most maxSamples from a streamer to prevent infinite loops.
// Returns the samples and whether the streamer completed (ok=false).
func streamWithLimit(s Streamer, maxSamples int) ([][2]float64, bool) {
	var result [][2]float64
	buf := make([][2]float64, 512)
	totalSamples := 0
	
	for totalSamples < maxSamples {
		remaining := maxSamples - totalSamples
		if len(buf) > remaining {
			buf = buf[:remaining]
		}
		
		n, ok := s.Stream(buf)
		result = append(result, buf[:n]...)
		totalSamples += n
		
		if !ok {
			return result, true // Streamer completed
		}
		
		if n == 0 {
			// Streamer returned no samples but ok=true - this could indicate an issue
			break
		}
	}
	
	return result, false // Hit limit, streamer didn't complete
}

// finiteTestStreamer creates a finite streamer that produces exactly n samples
type finiteTestStreamer struct {
	remaining int
	value     float64
}

func newFiniteTestStreamer(samples int, value float64) *finiteTestStreamer {
	return &finiteTestStreamer{remaining: samples, value: value}
}

func (f *finiteTestStreamer) Stream(samples [][2]float64) (int, bool) {
	if f.remaining <= 0 {
		return 0, false
	}
	
	n := len(samples)
	if n > f.remaining {
		n = f.remaining
	}
	
	for i := 0; i < n; i++ {
		samples[i] = [2]float64{f.value, f.value}
	}
	
	f.remaining -= n
	return n, f.remaining > 0
}

func (f *finiteTestStreamer) Err() error { return nil }

// Tests for silenceStreamer - documents infinite behavior
func TestSilenceStreamerIsInfinite(t *testing.T) {
	s := &silenceStreamer{}
	
	// Stream should always return ok=true and fill buffer with zeros
	buf := make([][2]float64, 100)
	n, ok := s.Stream(buf)
	
	assert.Equal(t, 100, n, "should fill entire buffer")
	assert.True(t, ok, "should always return ok=true (infinite)")
	
	for i, sample := range buf {
		assert.Equal(t, [2]float64{}, sample, "sample %d should be zero", i)
	}
	
	// Should still be infinite after many calls
	for i := 0; i < 10; i++ {
		n, ok := s.Stream(buf)
		assert.Equal(t, 100, n, "call %d: should still fill buffer", i)
		assert.True(t, ok, "call %d: should still be infinite", i)
	}
}

// Tests for scaledStreamer completion behavior
func TestScaledStreamerCompletesWithFiniteSource(t *testing.T) {
	source := newFiniteTestStreamer(50, 0.5)
	scaled := newScaledStreamer(source, 2.0)
	
	samples, completed := streamWithLimit(scaled, 1000)
	
	assert.True(t, completed, "scaled streamer should complete when source completes")
	assert.Equal(t, 50, len(samples), "should produce exactly 50 samples")
	
	for i, sample := range samples {
		assert.Equal(t, [2]float64{1.0, 1.0}, sample, "sample %d should be scaled 0.5 * 2.0 = 1.0", i)
	}
}

func TestScaledStreamerSilentModeCompletesWithFiniteSource(t *testing.T) {
	source := newFiniteTestStreamer(30, 0.8)
	scaled := newScaledStreamer(source, 0.0) // Silent mode
	
	samples, completed := streamWithLimit(scaled, 1000)
	
	assert.True(t, completed, "silent scaled streamer should complete when source completes")
	assert.Equal(t, 30, len(samples), "should produce exactly 30 samples")
	
	for i, sample := range samples {
		assert.Equal(t, [2]float64{}, sample, "sample %d should be zero in silent mode", i)
	}
}

func TestScaledStreamerPassthroughWithUnityGain(t *testing.T) {
	source := newFiniteTestStreamer(20, 0.3)
	scaled := newScaledStreamer(source, 1.0) // Should return source directly
	
	assert.Equal(t, source, scaled, "unity gain should return source directly")
}

func TestScaledStreamerInfiniteWithInfiniteSource(t *testing.T) {
	silence := &silenceStreamer{}
	scaled := newScaledStreamer(silence, 0.5)
	
	// Should be infinite since source is infinite
	samples, completed := streamWithLimit(scaled, 1000)
	
	assert.False(t, completed, "scaled infinite source should remain infinite")
	assert.Equal(t, 1000, len(samples), "should produce samples up to limit")
	
	for i, sample := range samples {
		assert.Equal(t, [2]float64{}, sample, "sample %d should be zero", i)
	}
}

// Tests for mixStreamer completion behavior
func TestMixStreamerCompletesWhenAllSourcesComplete(t *testing.T) {
	source1 := newFiniteTestStreamer(40, 0.2)
	source2 := newFiniteTestStreamer(40, 0.3)
	mixed := mixAll(source1, source2)
	
	samples, completed := streamWithLimit(mixed, 1000)
	
	assert.True(t, completed, "mix should complete when all sources complete")
	assert.Equal(t, 40, len(samples), "should produce 40 samples")
	
	for i, sample := range samples {
		assert.InDelta(t, 0.5, sample[0], 1e-9, "sample %d L should be 0.2 + 0.3 = 0.5", i)
		assert.InDelta(t, 0.5, sample[1], 1e-9, "sample %d R should be 0.2 + 0.3 = 0.5", i)
	}
}

func TestMixStreamerCompletesWhenLongestSourceCompletes(t *testing.T) {
	source1 := newFiniteTestStreamer(20, 0.4) // Shorter
	source2 := newFiniteTestStreamer(50, 0.1) // Longer
	mixed := mixAll(source1, source2)
	
	samples, completed := streamWithLimit(mixed, 1000)
	
	assert.True(t, completed, "mix should complete when longest source completes")
	assert.Equal(t, 50, len(samples), "should produce 50 samples (length of longest)")
	
	// First 20 samples should have both sources mixed
	for i := 0; i < 20; i++ {
		assert.InDelta(t, 0.5, samples[i][0], 1e-9, "sample %d should mix both sources", i)
	}
	
	// Remaining samples should only have source2
	for i := 20; i < 50; i++ {
		assert.InDelta(t, 0.1, samples[i][0], 1e-9, "sample %d should only have source2", i)
	}
}

func TestMixStreamerEmptySourcesReturnsSilence(t *testing.T) {
	mixed := mixAll() // No sources
	
	// Should return silenceStreamer (infinite)
	_, isSilence := mixed.(*silenceStreamer)
	assert.True(t, isSilence, "empty mix should return silenceStreamer")
	
	samples, completed := streamWithLimit(mixed, 100)
	
	assert.False(t, completed, "empty mix should be infinite")
	assert.Equal(t, 100, len(samples), "should produce samples up to limit")
	
	for i, sample := range samples {
		assert.Equal(t, [2]float64{}, sample, "sample %d should be zero", i)
	}
}

func TestMixStreamerSingleSourceReturnsSourceDirectly(t *testing.T) {
	source := newFiniteTestStreamer(30, 0.7)
	mixed := mixAll(source)
	
	assert.Equal(t, source, mixed, "single source mix should return source directly")
}

func TestMixStreamerInfiniteWithInfiniteSource(t *testing.T) {
	finite := newFiniteTestStreamer(25, 0.2)
	infinite := &silenceStreamer{}
	mixed := mixAll(finite, infinite)
	
	samples, completed := streamWithLimit(mixed, 100)
	
	assert.False(t, completed, "mix with infinite source should be infinite")
	assert.Equal(t, 100, len(samples), "should produce samples up to limit")
	
	// First 25 samples should have finite source mixed in
	for i := 0; i < 25; i++ {
		assert.InDelta(t, 0.2, samples[i][0], 1e-9, "sample %d should have finite source", i)
	}
	
	// Remaining samples should be just silence from infinite source
	for i := 25; i < 100; i++ {
		assert.Equal(t, [2]float64{}, samples[i], "sample %d should be silence", i)
	}
}

// Tests for effectsStreamer completion behavior
func TestEffectsStreamerCompletesAfterDuration(t *testing.T) {
	synth := &Synth{
		Oscillator1: Oscillator{Type: Silent},
		Envelope1:   Envelope{Sustain: 1.0},
		Mixer:       Mixer{Volume1: 1.0},
	}
	
	const sr = SampleRate(44100)
	const durationMs = 10.0
	expectedSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))
	
	ep := NewEffectsPatch(synth, EffectDefinitions{Ticks: 1}, durationMs)
	streamer := ep.Streamer(sr, 440.0, 0)
	
	samples, completed := streamWithLimit(streamer, expectedSamples*2) // Generous limit
	
	assert.True(t, completed, "effects streamer should complete after duration")
	assert.Equal(t, expectedSamples, len(samples), "should produce exact number of samples for duration")
}

func TestEffectsStreamerWithVolumeZeroCompletes(t *testing.T) {
	synth := &Synth{
		Oscillator1: Oscillator{Type: Sine},
		Envelope1:   Envelope{Sustain: 1.0},
		Mixer:       Mixer{Volume1: 1.0},
	}
	
	const sr = SampleRate(44100)
	const durationMs = 5.0
	expectedSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))
	
	fx := EffectDefinitions{
		Ticks:  1,
		Volume: VolumeEffect{Level: 0.0, Active: true}, // This was the bug source
	}
	ep := NewEffectsPatch(synth, fx, durationMs)
	streamer := ep.Streamer(sr, 440.0, 0)
	
	samples, completed := streamWithLimit(streamer, expectedSamples*2)
	
	assert.True(t, completed, "effects streamer with zero volume should complete")
	assert.Equal(t, expectedSamples, len(samples), "should produce exact number of samples")
	
	// All samples should be silent
	for i, sample := range samples {
		assert.Equal(t, [2]float64{}, sample, "sample %d should be silent with zero volume", i)
	}
}

func TestEffectsStreamerWithMultipleTicksCompletes(t *testing.T) {
	synth := &Synth{
		Oscillator1: Oscillator{Type: Silent},
		Envelope1:   Envelope{Sustain: 1.0},
		Mixer:       Mixer{Volume1: 1.0},
	}
	
	const sr = SampleRate(44100)
	const durationMs = 20.0
	const ticks = 4
	expectedSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))
	
	ep := NewEffectsPatch(synth, EffectDefinitions{Ticks: ticks}, durationMs)
	streamer := ep.Streamer(sr, 440.0, 0)
	
	samples, completed := streamWithLimit(streamer, expectedSamples*2)
	
	assert.True(t, completed, "effects streamer with multiple ticks should complete")
	assert.Equal(t, expectedSamples, len(samples), "should produce exact number of samples")
}

// Tests for panStreamer completion behavior
func TestPanStreamerCompletesWithFiniteSource(t *testing.T) {
	source := newFiniteTestStreamer(35, 0.6)
	panned := newPanStreamer(source, 0.5) // Center pan
	
	samples, completed := streamWithLimit(panned, 1000)
	
	assert.True(t, completed, "pan streamer should complete when source completes")
	assert.Equal(t, 35, len(samples), "should produce same number of samples as source")
	
	for i, sample := range samples {
		assert.InDelta(t, 0.6, sample[0], 1e-9, "sample %d L should be unmodified at center pan", i)
		assert.InDelta(t, 0.6, sample[1], 1e-9, "sample %d R should be unmodified at center pan", i)
	}
}

func TestPanStreamerLeftPan(t *testing.T) {
	source := newFiniteTestStreamer(10, 0.8)
	panned := newPanStreamer(source, -1.0) // Full left
	
	samples, completed := streamWithLimit(panned, 1000)
	
	assert.True(t, completed, "pan streamer should complete")
	assert.Equal(t, 10, len(samples), "should produce correct number of samples")
	
	for i, sample := range samples {
		assert.InDelta(t, 0.8, sample[0], 1e-9, "sample %d L should be full volume", i)
		assert.InDelta(t, 0.0, sample[1], 1e-9, "sample %d R should be silent", i)
	}
}

func TestPanStreamerRightPan(t *testing.T) {
	source := newFiniteTestStreamer(15, 0.4)
	panned := newPanStreamer(source, 1.0) // Full right
	
	samples, completed := streamWithLimit(panned, 1000)
	
	assert.True(t, completed, "pan streamer should complete")
	assert.Equal(t, 15, len(samples), "should produce correct number of samples")
	
	for i, sample := range samples {
		assert.InDelta(t, 0.0, sample[0], 1e-9, "sample %d L should be silent", i)
		assert.InDelta(t, 0.4, sample[1], 1e-9, "sample %d R should be full volume", i)
	}
}

// Tests for modulated streamers completion behavior
func TestModulatedOscillatorCompletesWithFiniteSource(t *testing.T) {
	source := newFiniteTestStreamer(25, 0.5)
	
	// Test that finite sources properly signal completion through modulation
	buf := make([][2]float64, 50)
	n, ok := source.Stream(buf)
	
	assert.Equal(t, 25, n, "should return correct sample count")
	assert.False(t, ok, "should complete when source completes")
}

// Tests for Volume streamer behavior
func TestVolumeStreamerWithEmptySourcesReturnsSilence(t *testing.T) {
	vol := NewVolume(0.5)
	streamer := vol.Streamer() // No sources
	
	// Should return silenceStreamer wrapped in scaledStreamer
	scaled, isScaled := streamer.(*scaledStreamer)
	assert.True(t, isScaled, "should return scaledStreamer")
	
	_, isSilence := scaled.s.(*silenceStreamer)
	assert.True(t, isSilence, "should wrap silenceStreamer")
	
	// Test that it behaves as infinite silence
	samples, completed := streamWithLimit(streamer, 100)
	
	assert.False(t, completed, "empty volume streamer should be infinite")
	assert.Equal(t, 100, len(samples), "should produce samples up to limit")
	
	for i, sample := range samples {
		assert.Equal(t, [2]float64{}, sample, "sample %d should be zero", i)
	}
}

func TestVolumeStreamerWithFiniteSourceCompletes(t *testing.T) {
	source := newFiniteTestStreamer(30, 0.8)
	vol := NewVolume(0.25)
	streamer := vol.Streamer(source)
	
	samples, completed := streamWithLimit(streamer, 1000)
	
	assert.True(t, completed, "volume streamer should complete when source completes")
	assert.Equal(t, 30, len(samples), "should produce same number of samples as source")
	
	for i, sample := range samples {
		expected := 0.8 * 0.25 // source value * volume
		assert.InDelta(t, expected, sample[0], 1e-9, "sample %d L should be scaled", i)
		assert.InDelta(t, expected, sample[1], 1e-9, "sample %d R should be scaled", i)
	}
}

// Tests for Patch completion behavior
func TestPatchCompletesAfterSampleCount(t *testing.T) {
	synth := &Synth{
		Oscillator1: Oscillator{Type: Silent},
		Envelope1:   Envelope{Sustain: 1.0},
		Mixer:       Mixer{Volume1: 1.0},
	}
	
	const sr = SampleRate(44100)
	const sampleCount = 100
	
	patch := synth.NewPatch(sr, 440.0, sampleCount)
	
	samples, completed := streamWithLimit(patch, sampleCount*2)
	
	assert.True(t, completed, "patch should complete after specified sample count")
	assert.Equal(t, sampleCount, len(samples), "should produce exactly the specified number of samples")
}

func TestPatchWithEnvelopeCompletesCorrectly(t *testing.T) {
	synth := &Synth{
		Oscillator1: Oscillator{Type: Silent},
		Envelope1:   Envelope{Attack: 10 * time.Millisecond, Sustain: 1.0, Release: 10 * time.Millisecond},
		Mixer:       Mixer{Volume1: 1.0},
	}
	
	const sr = SampleRate(1000) // 1 sample = 1ms for easy calculation
	const sampleCount = 50      // 50ms total
	
	patch := synth.NewPatch(sr, 440.0, sampleCount)
	
	samples, completed := streamWithLimit(patch, sampleCount*2)
	
	assert.True(t, completed, "patch with envelope should complete")
	assert.Equal(t, sampleCount, len(samples), "should produce exactly the specified number of samples")
}

// Safety tests to ensure we don't have obvious infinite loops
func TestStreamersSafetyLimits(t *testing.T) {
	const maxTestSamples = 10000 // Safety limit for all tests
	
	t.Run("scaled silence safety", func(t *testing.T) {
		silence := &silenceStreamer{}
		scaled := newScaledStreamer(silence, 0.5)
		
		samples, completed := streamWithLimit(scaled, maxTestSamples)
		assert.False(t, completed, "should not complete (infinite)")
		assert.Equal(t, maxTestSamples, len(samples), "should hit safety limit")
	})
	
	t.Run("mixed silence safety", func(t *testing.T) {
		silence := &silenceStreamer{}
		mixed := mixAll(silence)
		
		samples, completed := streamWithLimit(mixed, maxTestSamples)
		assert.False(t, completed, "should not complete (infinite)")
		assert.Equal(t, maxTestSamples, len(samples), "should hit safety limit")
	})
	
	t.Run("volume silence safety", func(t *testing.T) {
		vol := NewVolume(0.3)
		streamer := vol.Streamer() // Empty sources -> silence
		
		samples, completed := streamWithLimit(streamer, maxTestSamples)
		assert.False(t, completed, "should not complete (infinite)")
		assert.Equal(t, maxTestSamples, len(samples), "should hit safety limit")
	})
}

// Test the StreamN function behavior
func TestStreamNWithFiniteSource(t *testing.T) {
	source := newFiniteTestStreamer(200, 0.7)
	
	// StreamN should return exactly n samples, regardless of source length
	samples := StreamN(source, 100)
	
	assert.Equal(t, 100, len(samples), "StreamN should return exactly n samples")
	
	for i, sample := range samples {
		assert.Equal(t, [2]float64{0.7, 0.7}, sample, "sample %d should have correct value", i)
	}
}

func TestStreamNWithInsufficientSource(t *testing.T) {
	source := newFiniteTestStreamer(50, 0.9)
	
	// Request more samples than source has
	samples := StreamN(source, 100)
	
	assert.Equal(t, 100, len(samples), "StreamN should still allocate full buffer")
	
	// First 50 samples should have the source value
	for i := 0; i < 50; i++ {
		assert.Equal(t, [2]float64{0.9, 0.9}, samples[i], "sample %d should have source value", i)
	}
	
	// Remaining samples should be zero (uninitialized)
	for i := 50; i < 100; i++ {
		assert.Equal(t, [2]float64{}, samples[i], "sample %d should be zero", i)
	}
}