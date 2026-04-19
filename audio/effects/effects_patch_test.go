package effects

import (
	"math"
	"testing"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tetrackt/tetrackt/audio"
)

func sineSynth() *audio.Synth {
	return &audio.Synth{
		Oscillator1: audio.Oscillator{Type: audio.Sine},
		Envelope1:   audio.Envelope{Sustain: 1.0},
		Mixer:       audio.Mixer{Volume1: 1.0},
	}
}

func silentSynth() *audio.Synth {
	return &audio.Synth{
		Oscillator1: audio.Oscillator{Type: audio.Silent},
		Envelope1:   audio.Envelope{Sustain: 1.0},
		Mixer:       audio.Mixer{Volume1: 1.0},
	}
}

func streamAll(s audio.Streamer) [][2]float64 {
	var result [][2]float64
	buf := make([][2]float64, 512)
	for {
		n, ok := s.Stream(buf)
		result = append(result, buf[:n]...)
		if !ok {
			break
		}
	}
	return result
}

func rmsSlice(s [][2]float64) float64 {
	var sum float64
	for _, v := range s {
		sum += v[0] * v[0]
	}
	if len(s) == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(len(s)))
}

func countZeroCrossings(samples [][2]float64) int {
	count := 0
	for i := 1; i < len(samples); i++ {
		if samples[i-1][0] <= 0 && samples[i][0] > 0 {
			count++
		}
	}
	return count
}

func TestEffectsPatchSampleCount(t *testing.T) {
	const sr = beep.SampleRate(44100)
	const durationMs = 100.0
	totalSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))

	ep := NewEffectsPatch(silentSynth(), EffectDefs{}, durationMs, 4)
	samples := streamAll(ep.Streamer(sr, 440.0, 0))

	assert.Equal(t, totalSamples, len(samples))
}

func TestEffectsPatchSampleCountOddDivision(t *testing.T) {
	// 7 subticks into 100ms: remainder must be absorbed so total is exact.
	const sr = beep.SampleRate(44100)
	const durationMs = 100.0
	totalSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))

	ep := NewEffectsPatch(silentSynth(), EffectDefs{}, durationMs, 7)
	samples := streamAll(ep.Streamer(sr, 440.0, 0))

	assert.Equal(t, totalSamples, len(samples))
}

func TestEffectsPatchSampleCountSingleSubtick(t *testing.T) {
	const sr = beep.SampleRate(44100)
	const durationMs = 50.0
	totalSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))

	ep := NewEffectsPatch(silentSynth(), EffectDefs{}, durationMs, 1)
	samples := streamAll(ep.Streamer(sr, 440.0, 0))

	assert.Equal(t, totalSamples, len(samples))
}

func TestEffectsPatchSubticksClampedToOne(t *testing.T) {
	const sr = beep.SampleRate(44100)
	const durationMs = 50.0
	totalSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))

	ep := NewEffectsPatch(silentSynth(), EffectDefs{}, durationMs, 0)
	samples := streamAll(ep.Streamer(sr, 440.0, 0))

	assert.Equal(t, totalSamples, len(samples))
}

func TestEffectsPatchArpeggio(t *testing.T) {
	// ARP [0, 12]: subtick 0 = A4 (440 Hz), subtick 1 = A5 (880 Hz).
	// Expect ~2x zero crossings in subtick 1.
	const sr = beep.SampleRate(44100)
	const durationMs = 100.0
	subtickSamples := sr.N(time.Duration(durationMs / 2 * float64(time.Millisecond)))

	arp := audio.ArpeggioEffect{Offsets: []int{0, 12}}
	ep := NewEffectsPatch(sineSynth(), EffectDefs{Arpeggio: arp}, durationMs, 2)
	streamer := ep.Streamer(sr, 440.0, 0)

	buf0 := make([][2]float64, subtickSamples)
	n0, _ := streamer.Stream(buf0)

	buf1 := make([][2]float64, subtickSamples)
	n1, _ := streamer.Stream(buf1)

	c0 := countZeroCrossings(buf0[:n0])
	c1 := countZeroCrossings(buf1[:n1])

	ratio := float64(c1) / float64(c0)
	assert.InDelta(t, 2.0, ratio, 0.3,
		"expected ~2x crossings for +12 semitone ARP (subtick0=%d subtick1=%d)", c0, c1)
}

func TestEffectsPatchArpeggioWraps(t *testing.T) {
	// 3-entry ARP over 6 subticks: pattern repeats twice.
	const sr = beep.SampleRate(44100)
	const durationMs = 120.0
	subtickSamples := sr.N(time.Duration(durationMs / 6 * float64(time.Millisecond)))

	arp := audio.ArpeggioEffect{Offsets: []int{0, 7, 12}}
	ep := NewEffectsPatch(sineSynth(), EffectDefs{Arpeggio: arp}, durationMs, 6)
	streamer := ep.Streamer(sr, 440.0, 0)

	buf := make([][2]float64, subtickSamples)

	// Collect zero crossings for all 6 subticks.
	crossings := make([]int, 6)
	for i := range crossings {
		n, _ := streamer.Stream(buf)
		crossings[i] = countZeroCrossings(buf[:n])
	}

	// Subtick 0 and 3 both use offset 0 → same base frequency → similar count.
	ratio03 := float64(crossings[3]) / float64(crossings[0])
	assert.InDelta(t, 1.0, ratio03, 0.2,
		"subtick 0 and 3 share offset 0, expected similar crossings (%d vs %d)",
		crossings[0], crossings[3])
}

func TestEffectsPatchPortamento(t *testing.T) {
	// Glide from 220 Hz to 440 Hz over 2 subticks.
	// Subtick 1 should have more zero crossings than subtick 0.
	const sr = beep.SampleRate(44100)
	const durationMs = 100.0
	subtickSamples := sr.N(time.Duration(durationMs / 2 * float64(time.Millisecond)))

	ep := NewEffectsPatch(sineSynth(), EffectDefs{Portamento: PortamentoEffect{Active: true}}, durationMs, 2)
	streamer := ep.Streamer(sr, 440.0, 220.0)

	buf0 := make([][2]float64, subtickSamples)
	n0, _ := streamer.Stream(buf0)

	buf1 := make([][2]float64, subtickSamples)
	n1, _ := streamer.Stream(buf1)

	c0 := countZeroCrossings(buf0[:n0])
	c1 := countZeroCrossings(buf1[:n1])

	assert.Greater(t, c1, c0,
		"expected more crossings in subtick 1 (higher freq after glide): subtick0=%d subtick1=%d", c0, c1)
}

func TestEffectsPatchPortamentoNoPrevFreq(t *testing.T) {
	// With portamento active but prevFreq=0, it should behave like no portamento.
	const sr = beep.SampleRate(44100)
	const durationMs = 50.0
	totalSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))

	ep := NewEffectsPatch(sineSynth(), EffectDefs{Portamento: PortamentoEffect{Active: true}}, durationMs, 4)

	samples := streamAll(ep.Streamer(sr, 440.0, 0))
	assert.Equal(t, totalSamples, len(samples))
}

func TestEffectsPatchVolumeSlideDecreases(t *testing.T) {
	// A negative delta should fade the note out over subticks.
	// RMS of the second half of the note must be lower than the first.
	const sr = beep.SampleRate(44100)
	const durationMs = 100.0
	const subticks = 8
	halfSamples := sr.N(time.Duration(durationMs / 2 * float64(time.Millisecond)))

	// delta = -0.15/tick: after 4 ticks volume = 1 - 4*0.15 = 0.4
	slide := VolumeSlideEffect{Delta: -0.15}
	ep := NewEffectsPatch(sineSynth(), EffectDefs{VolumeSlide: slide}, durationMs, subticks)
	samples := streamAll(ep.Streamer(sr, 440.0, 0))

	firstHalf := rmsSlice(samples[:halfSamples])
	secondHalf := rmsSlice(samples[halfSamples:])

	assert.Greater(t, firstHalf, secondHalf,
		"expected RMS to decrease with negative VolumeSlide (first=%v, second=%v)", firstHalf, secondHalf)
}

func TestEffectsPatchVolumeSlideClampsAtZero(t *testing.T) {
	// A very large negative delta should clamp at 0, not go negative.
	const sr = beep.SampleRate(44100)
	const durationMs = 100.0
	const subticks = 4

	slide := VolumeSlideEffect{Delta: -1.0} // max drop per tick; hits 0 on tick 1
	ep := NewEffectsPatch(sineSynth(), EffectDefs{VolumeSlide: slide}, durationMs, subticks)

	subtickSamples := sr.N(time.Duration(durationMs / subticks * float64(time.Millisecond)))
	streamer := ep.Streamer(sr, 440.0, 0)

	// Skip subtick 0 (volume drops from 1 to 0 on first applySubtickEffects).
	buf := make([][2]float64, subtickSamples)
	streamer.Stream(buf)

	// Remaining subticks 1-3 should be silent (volume clamped at 0).
	rest := streamAll(streamer)
	assert.InDelta(t, 0.0, rmsSlice(rest), 1e-9,
		"expected silence after volume clamped to zero")
}

func TestEffectsPatchNoteCut(t *testing.T) {
	// NoteCut at subtick 1 (out of 3): subtick 0 is audible, subtick 1+ is silent.
	const sr = beep.SampleRate(44100)
	const durationMs = 60.0
	const subticks = 3
	subtickSamples := sr.N(time.Duration(durationMs / subticks * float64(time.Millisecond)))

	ep := NewEffectsPatch(sineSynth(), EffectDefs{NoteCut: NoteCutEffect{Tick: 1}}, durationMs, subticks)
	streamer := ep.Streamer(sr, 440.0, 0)

	buf0 := make([][2]float64, subtickSamples)
	streamer.Stream(buf0)

	// Subticks 1 and 2 should both be silent.
	rest := streamAll(streamer)

	assert.Greater(t, rmsSlice(buf0), 0.1,
		"expected audible output in subtick 0 before the cut")
	assert.InDelta(t, 0.0, rmsSlice(rest), 1e-9,
		"expected silence after NoteCut at subtick 1")
}

func TestEffectsPatchNoteCutDoesNotInteractWithVolumeSlide(t *testing.T) {
	// When NoteCut fires, subsequent VolumeSlide steps must not re-enable volume.
	const sr = beep.SampleRate(44100)
	const durationMs = 80.0
	const subticks = 4
	subtickSamples := sr.N(time.Duration(durationMs / subticks * float64(time.Millisecond)))

	fx := EffectDefs{
		NoteCut:     NoteCutEffect{Tick: 1},
		VolumeSlide: VolumeSlideEffect{Delta: 0.5}, // positive slide would raise volume if unchecked
	}
	ep := NewEffectsPatch(sineSynth(), EffectDefs(fx), durationMs, subticks)
	streamer := ep.Streamer(sr, 440.0, 0)

	buf := make([][2]float64, subtickSamples)
	streamer.Stream(buf) // subtick 0

	rest := streamAll(streamer) // subticks 1-3: cut fires on 1, slide on 1-3

	assert.InDelta(t, 0.0, rmsSlice(rest), 1e-9,
		"VolumeSlide must not un-silence a note after NoteCut")
}

func TestEffectsPatchNoteDelayOutputsSilenceBeforeTick(t *testing.T) {
	// NoteDelay at tick 2 out of 4: first two subticks must be all zeros.
	const sr = beep.SampleRate(44100)
	const durationMs = 80.0
	const subticks = 4
	subtickSamples := sr.N(time.Duration(durationMs / subticks * float64(time.Millisecond)))

	ep := NewEffectsPatch(sineSynth(), EffectDefs{NoteDelay: NoteDelayEffect{Tick: 2}}, durationMs, subticks)
	streamer := ep.Streamer(sr, 440.0, 0)

	pre := make([][2]float64, subtickSamples*2)
	streamer.Stream(pre)

	for i, s := range pre {
		assert.Equal(t, [2]float64{}, s, "sample %d should be silence before NoteDelay fires", i)
	}
}

func TestEffectsPatchNoteDelayPlaysAfterTick(t *testing.T) {
	// NoteDelay at tick 2: subticks 2-3 must contain actual audio.
	const sr = beep.SampleRate(44100)
	const durationMs = 80.0
	const subticks = 4
	subtickSamples := sr.N(time.Duration(durationMs / subticks * float64(time.Millisecond)))

	ep := NewEffectsPatch(sineSynth(), EffectDefs{NoteDelay: NoteDelayEffect{Tick: 2}}, durationMs, subticks)
	streamer := ep.Streamer(sr, 440.0, 0)

	pre := make([][2]float64, subtickSamples*2)
	streamer.Stream(pre) // consume the silent pre-delay ticks

	post := streamAll(streamer) // subticks 2-3: should be audible
	assert.Greater(t, rmsSlice(post), 0.1,
		"expected audible output after NoteDelay fires")
}

func TestEffectsPatchNoteDelaySampleCount(t *testing.T) {
	// Total sample count must still equal durationMs regardless of NoteDelay.
	const sr = beep.SampleRate(44100)
	const durationMs = 100.0
	totalSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))

	ep := NewEffectsPatch(sineSynth(), EffectDefs{NoteDelay: NoteDelayEffect{Tick: 3}}, durationMs, 8)
	samples := streamAll(ep.Streamer(sr, 440.0, 0))

	assert.Equal(t, totalSamples, len(samples))
}

func TestEffectsPatchNoteDelayEnvelopeStartsAtDelay(t *testing.T) {
	// With a 40ms attack at sr=1000 and NoteDelay at subtick 1, the attack
	// begins at the delay tick. The first samples after the delay should
	// have low amplitude (start of attack), not full sustain.
	const sr = beep.SampleRate(1000) // 1 sample = 1 ms
	const durationMs = 100.0
	const subticks = 2
	subtickSamples := sr.N(time.Duration(durationMs / subticks * float64(time.Millisecond)))

	synth := &audio.Synth{
		Oscillator1: audio.Oscillator{Type: audio.Sine},
		Envelope1:   audio.Envelope{Attack: 40 * time.Millisecond, Sustain: 1.0},
		Mixer:       audio.Mixer{Volume1: 1.0},
	}
	ep := NewEffectsPatch(synth, EffectDefs{NoteDelay: NoteDelayEffect{Tick: 1}}, durationMs, subticks)
	streamer := ep.Streamer(sr, 440.0, 0)

	pre := make([][2]float64, subtickSamples)
	streamer.Stream(pre) // subtick 0: silence

	post := streamAll(streamer) // subtick 1: attack starts here

	// First 5 samples after delay should have lower amplitude than last 5
	// (envelope is in early attack vs sustain).
	startRMS := rmsSlice(post[:5])
	endRMS := rmsSlice(post[len(post)-5:])

	assert.Greater(t, endRMS, startRMS*3,
		"expected envelope to start fresh after NoteDelay (startRMS=%v endRMS=%v)", startRMS, endRMS)
}

func TestEffectsPatchRetriggerRestartsEnvelope(t *testing.T) {
	// sr=1000 means 1 sample = 1 ms: easy to reason about timing.
	// 100ms note, 2 subticks → 50 samples each.
	// 30ms attack: by sample 30 the envelope is at full sustain.
	// With retrigger, subtick 1 restarts the attack so its first samples
	// have near-zero amplitude while the last samples of subtick 0 are near full.
	const sr = beep.SampleRate(1000)
	const durationMs = 100.0
	const subticks = 2
	subtickSamples := sr.N(time.Duration(durationMs / subticks * float64(time.Millisecond)))

	synth := &audio.Synth{
		Oscillator1: audio.Oscillator{Type: audio.Sine},
		Envelope1:   audio.Envelope{Attack: 30 * time.Millisecond, Sustain: 1.0},
		Mixer:       audio.Mixer{Volume1: 1.0},
	}

	ep := NewEffectsPatch(synth, EffectDefs{RetriggerEnvelope: true}, durationMs, subticks)
	streamer := ep.Streamer(sr, 440.0, 0)

	buf0 := make([][2]float64, subtickSamples)
	streamer.Stream(buf0)

	buf1 := make([][2]float64, subtickSamples)
	streamer.Stream(buf1)

	// Last 10 samples of subtick 0 should be at sustain (high amplitude).
	// First 10 samples of subtick 1 should be near the attack floor (low amplitude).
	endRMS := rmsSlice(buf0[len(buf0)-10:])
	startRMS := rmsSlice(buf1[:10])

	assert.Greater(t, endRMS, startRMS*5,
		"retrigger should restart envelope: end of tick0 RMS=%v, start of tick1 RMS=%v",
		endRMS, startRMS)
}

func TestEffectsPatchNoEffectsMatchesDirectPatch(t *testing.T) {
	// With no effects and a single subtick, EffectsPatch must produce the same
	// zero-crossing count as calling Synth.NewPatch directly.
	const sr = beep.SampleRate(44100)
	const durationMs = 20.0
	const freq = 440.0
	totalSamples := sr.N(time.Duration(durationMs * float64(time.Millisecond)))

	synth := sineSynth()

	epSamples := streamAll(NewEffectsPatch(synth, EffectDefs{}, durationMs, 1).Streamer(sr, freq, 0))

	directSamples := make([][2]float64, totalSamples)
	synth.NewPatch(sr, freq, totalSamples).Stream(directSamples)

	assert.Equal(t, len(directSamples), len(epSamples))
	assert.InDelta(t,
		float64(countZeroCrossings(directSamples)),
		float64(countZeroCrossings(epSamples)),
		5, // allow tiny rounding difference
	)
}

func TestEffectsPatchVibratoModulatesFrequency(t *testing.T) {
	// Vibrato with depth=3 semitones and 1 full cycle over the note.
	// The frequency should deviate from the base (440 Hz) across subticks.
	// We verify by checking that the crossing counts are not all identical.
	const sr = beep.SampleRate(44100)
	const durationMs = 80.0
	const subticks = 8
	subtickSamples := sr.N(time.Duration(durationMs / subticks * float64(time.Millisecond)))

	vib := VibratoEffect{Depth: 3.0, Rate: 1.0}
	ep := NewEffectsPatch(sineSynth(), EffectDefs{Vibrato: vib}, durationMs, subticks)
	streamer := ep.Streamer(sr, 440.0, 0)

	crossings := make([]int, subticks)
	buf := make([][2]float64, subtickSamples)
	for i := range crossings {
		n, _ := streamer.Stream(buf)
		crossings[i] = countZeroCrossings(buf[:n])
	}

	// At least some subticks should differ (vibrato changes pitch).
	allEqual := true
	for i := 1; i < len(crossings); i++ {
		if crossings[i] != crossings[0] {
			allEqual = false
			break
		}
	}
	assert.False(t, allEqual, "vibrato should produce varying zero-crossing counts: %v", crossings)
}
