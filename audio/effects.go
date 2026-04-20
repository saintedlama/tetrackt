package audio

import (
	"math"
	"time"
)

// ArpeggioEffect cycles through frequency offsets, one per tick.
// Offsets are semitone values relative to the base frequency and are converted
// to multipliers via 2^(offset/12), making the effect purely frequency-based.
// An empty Offsets slice means the effect is inactive.
type ArpeggioEffect struct {
	Offsets []int // one semitone offset per tick, e.g. [0, 4, 7, 4] for a 4-tick major arp
}

// IsActive reports whether the arpeggio effect will produce any retuning.
func (a ArpeggioEffect) IsActive() bool {
	return len(a.Offsets) > 0
}

// VibratoEffect applies pitch modulation at subtick granularity.
// Depth is the maximum pitch deviation in semitones; Rate counts how many full
// sine oscillations occur over the total note length (e.g. 2 = two full waves).
type VibratoEffect struct {
	Depth float64 // semitones (>= 0); 0 = disabled
	Rate  float64 // oscillations per note length (> 0 to be active)
}

// IsActive reports whether vibrato will produce any pitch modulation.
func (v VibratoEffect) IsActive() bool { return v.Depth > 0 && v.Rate > 0 }

// PortamentoEffect controls pitch glide from a previous note.
// When Ticks > 0 and a non-zero prevFreq is passed to Streamer, the pitch
// slides exponentially from prevFreq to the target note's frequency.
// The glide spans exactly min(Ticks, available sub-ticks) steps and always
// arrives at the target frequency on the last glide step. Sub-ticks after the
// glide stay at the target frequency.
// Ticks = 0 means snap (no glide). Row granularity (smoothness) is controlled
// by the row's sub-tick count: more sub-ticks → smoother glide.
// Not designed to be combined with RetriggerEnvelope (Reset clears the glide).
type PortamentoEffect struct {
	Ticks int // number of sub-ticks for the glide; 0 = snap (no glide)
}

// VolumeSlideEffect adjusts the output volume by a fixed delta at every subtick.
// Delta is the per-subtick change in [−1, 1]; positive increases volume,
// negative decreases it. Volume is clamped to [0, 1] after each step.
// A delta of 0 disables the effect.
type VolumeSlideEffect struct {
	Delta float64
}

// IsActive reports whether the slide will change volume.
func (v VolumeSlideEffect) IsActive() bool { return v.Delta != 0 }

// NoteCutEffect silences the note at a specific subtick by setting volume to
// zero. Once triggered, the silence persists for the remainder of the note;
// subsequent VolumeSlide steps are ignored after the cut.
// Tick <= 0 disables the effect.
type NoteCutEffect struct {
	Tick int // 0-based subtick index at which to cut; <= 0 = inactive
}

// IsActive reports whether the cut will fire.
func (n NoteCutEffect) IsActive() bool { return n.Tick > 0 }

// NoteDelayEffect defers note playback to a specific subtick. Before Tick the
// streamer outputs silence; from Tick onward the patch plays with its ADSR
// envelope starting at that moment, so the full envelope applies only to the
// remaining duration.
// Tick <= 0 disables the effect (immediate playback).
type NoteDelayEffect struct {
	Tick int // 0-based subtick index at which playback begins; <= 0 = immediate
}

// IsActive reports whether playback will be deferred.
func (n NoteDelayEffect) IsActive() bool { return n.Tick > 0 }

// EffectDefs bundles all per-note effect definitions for an EffectsPatch.
type EffectDefs struct {
	Arpeggio          ArpeggioEffect
	Portamento        PortamentoEffect
	Vibrato           VibratoEffect
	RetriggerEnvelope bool // restart ADSR at every subtick boundary (useful with Arpeggio)
	VolumeSlide       VolumeSlideEffect
	NoteCut           NoteCutEffect
	NoteDelay         NoteDelayEffect
}

// EffectsPatch couples a Synth with time-aware effect definitions.
// It knows the note duration in milliseconds and the number of subticks, and
// applies effects at each subtick boundary when streaming.
//
// Typical usage:
//
//	ep := audio.NewEffectsPatch(synth, audio.EffectDefs{Arpeggio: arp}, 125.0, 4)
//	streamer := ep.Streamer(sr, note, prevFreq)
//	speaker.Play(streamer)
type EffectsPatch struct {
	synth      *Synth
	effects    EffectDefs
	durationMs float64
	subticks   int
}

// NewEffectsPatch creates an EffectsPatch from the given Synth definition, effect
// definitions, note duration in milliseconds, and subtick count.
// subticks < 1 is clamped to 1.
func NewEffectsPatch(synth *Synth, fx EffectDefs, durationMs float64, subticks int) *EffectsPatch {
	if subticks < 1 {
		subticks = 1
	}
	return &EffectsPatch{
		synth:      synth,
		effects:    fx,
		durationMs: durationMs,
		subticks:   subticks,
	}
}

// Streamer returns an audio.Streamer that renders the given frequency for
// durationMs milliseconds, applying effects at each subtick boundary.
//
// prevFreq is the frequency of the last note played on the same track; pass 0
// when there is no previous note. It is only used when Portamento.Ticks > 0.
func (ep *EffectsPatch) Streamer(sr SampleRate, freq float64, prevFreq float64) Streamer {
	totalSamples := sr.N(time.Duration(ep.durationMs * float64(time.Millisecond)))

	startFreq := freq
	gliding := ep.effects.Portamento.Ticks > 0 && prevFreq > 0 && freq > 0
	if gliding {
		startFreq = prevFreq
	}

	subtickSamples := totalSamples / ep.subticks
	remainder := totalSamples - subtickSamples*ep.subticks

	s := &effectsStreamer{
		noteFreq:       freq,
		arp:            ep.effects.Arpeggio,
		portamento:     ep.effects.Portamento,
		vibrato:        ep.effects.Vibrato,
		retrigger:      ep.effects.RetriggerEnvelope,
		volumeSlide:    ep.effects.VolumeSlide,
		noteCut:        ep.effects.NoteCut,
		noteDelay:      ep.effects.NoteDelay,
		gliding:        gliding,
		subtickSamples: subtickSamples,
		remainder:      remainder,
		totalSubticks:  ep.subticks,
		currentVolume:  1.0,
		pendingSubtick: true,
		// Fields for deferred NoteDelay patch creation.
		synth:     ep.synth,
		sr:        sr,
		startFreq: startFreq,
		prevFreq:  prevFreq,
	}

	// Create the patch immediately unless NoteDelay defers it.
	if !ep.effects.NoteDelay.IsActive() {
		s.patch = ep.synth.NewPatch(sr, startFreq, totalSamples)
	}

	return s
}

type effectsStreamer struct {
	patch    *Patch // nil before NoteDelay fires
	noteFreq float64

	arp        ArpeggioEffect
	portamento PortamentoEffect
	vibrato    VibratoEffect
	retrigger  bool
	gliding    bool // portamento glide in progress

	volumeSlide   VolumeSlideEffect
	noteCut       NoteCutEffect
	noteDelay     NoteDelayEffect
	currentVolume float64 // running volume for VolumeSlide; starts at 1.0
	cut           bool    // true after NoteCut fires

	subtickSamples int // base samples per subtick (last subtick may be longer)
	remainder      int // extra samples added to the very last subtick
	totalSubticks  int
	currentSubtick int
	samplesInTick  int
	pendingSubtick bool

	// Retained for deferred patch creation when NoteDelay is active.
	synth     *Synth
	sr        SampleRate
	startFreq float64
	prevFreq  float64
}

// tickSize returns the number of samples in the current subtick.
// The last subtick absorbs any rounding remainder so the total sample count
// exactly matches the originally requested duration.
func (e *effectsStreamer) tickSize() int {
	if e.currentSubtick == e.totalSubticks-1 {
		return e.subtickSamples + e.remainder
	}
	return e.subtickSamples
}

// Stream implements audio.Streamer.
// Effects are applied at the start of each subtick before pulling samples
// from the underlying Patch.
func (e *effectsStreamer) Stream(samples [][2]float64) (int, bool) {
	total := 0
	for len(samples) > 0 {
		if e.currentSubtick >= e.totalSubticks {
			return total, false
		}

		if e.pendingSubtick {
			e.applySubtickEffects()
			e.pendingSubtick = false
		}

		tickSize := e.tickSize()
		available := tickSize - e.samplesInTick
		chunk := samples
		if len(chunk) > available {
			chunk = samples[:available]
		}

		var n int
		var ok bool
		if e.patch != nil {
			n, ok = e.patch.Stream(chunk)
		} else {
			// NoteDelay: output silence until the patch is created.
			for i := range chunk {
				chunk[i] = [2]float64{}
			}
			n, ok = len(chunk), true
		}

		total += n
		e.samplesInTick += n
		samples = samples[n:]

		if e.samplesInTick >= tickSize {
			e.currentSubtick++
			e.samplesInTick = 0
			if e.currentSubtick < e.totalSubticks {
				if e.retrigger && e.patch != nil {
					e.patch.Reset()
				}
				e.pendingSubtick = true
			}
		}

		if !ok || n == 0 {
			return total, false
		}
	}
	return total, e.currentSubtick < e.totalSubticks
}

// applySubtickEffects fires all time-based effects for the current subtick.
//
// Pitch priority (highest wins): Arpeggio → Vibrato → Portamento.
// Volume priority: NoteCut silences permanently; VolumeSlide is suppressed
// after a cut. Both are independent of the pitch effects.
func (e *effectsStreamer) applySubtickEffects() {
	tick := e.currentSubtick

	// NoteDelay: create the patch when the delay tick arrives; return early
	// for all preceding ticks so no effects fire during the silence period.
	if e.noteDelay.IsActive() {
		if e.patch == nil {
			if tick < e.noteDelay.Tick {
				return
			}
			// Delay fires: create a patch sized for the remaining duration so
			// the full ADSR envelope applies to exactly the ticks that will play.
			ticksLeft := e.totalSubticks - tick
			remainingSamples := ticksLeft*e.subtickSamples + e.remainder
			e.patch = e.synth.NewPatch(e.sr, e.startFreq, remainingSamples)
		}
	}

	if e.patch == nil {
		return
	}

	// Effective tick index relative to when the patch started playing
	// (always 0 on the first active subtick, regardless of NoteDelay).
	effectiveTick := tick
	effectiveTotal := e.totalSubticks
	if e.noteDelay.IsActive() {
		effectiveTick = tick - e.noteDelay.Tick
		effectiveTotal = e.totalSubticks - e.noteDelay.Tick
	}

	// Pitch effects.
	if e.arp.IsActive() {
		offset := e.arp.Offsets[effectiveTick%len(e.arp.Offsets)]
		arpFreq := e.noteFreq * math.Pow(2, float64(offset)/12.0)
		e.patch.SetFrequency(arpFreq)
	} else if e.vibrato.IsActive() {
		phase := 2 * math.Pi * e.vibrato.Rate * float64(effectiveTick) / float64(effectiveTotal)
		semitoneShift := e.vibrato.Depth * math.Sin(phase)
		e.patch.SetFrequency(e.noteFreq * math.Pow(2, semitoneShift/12.0))
	} else if e.portamento.Ticks > 0 && e.gliding {
		// Exponential glide from prevFreq to noteFreq over min(Ticks, effectiveTotal)
		// sub-ticks. Clamping ensures the glide always arrives at noteFreq on the
		// last glide step, even when Ticks exceeds the available sub-tick count.
		// Sub-ticks after the glide snap to noteFreq.
		glideTotal := e.portamento.Ticks
		if glideTotal > effectiveTotal {
			glideTotal = effectiveTotal
		}
		if effectiveTick < glideTotal {
			t := float64(effectiveTick+1) / float64(glideTotal)
			glideFreq := e.prevFreq * math.Pow(e.noteFreq/e.prevFreq, t)
			e.patch.SetFrequency(glideFreq)
		} else {
			e.patch.SetFrequency(e.noteFreq)
		}
	}

	// Volume effects. NoteCut fires once and suppresses all subsequent
	// VolumeSlide steps so the silence cannot be un-done by a slide.
	if e.noteCut.IsActive() && tick == e.noteCut.Tick {
		e.cut = true
		e.patch.SetVolume(0)
	}
	if !e.cut && e.volumeSlide.IsActive() {
		e.currentVolume = math.Max(0, math.Min(1, e.currentVolume+e.volumeSlide.Delta))
		e.patch.SetVolume(e.currentVolume)
	}
}

// Err implements audio.Streamer.
func (e *effectsStreamer) Err() error {
	if e.patch != nil {
		return e.patch.Err()
	}
	return nil
}
