package audio

import (
	"math"
	"time"

	"github.com/gopxl/beep/v2/effects"
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

// VibratoEffect applies pitch modulation at tick granularity.
// Depth is the maximum pitch deviation in semitones; Rate counts how many full
// sine oscillations occur over the total note length (e.g. 2 = two full waves).
type VibratoEffect struct {
	Depth float64 // semitones (>= 0); 0 = disabled
	Rate  float64 // oscillations per note length (> 0 to be active)
}

// IsActive reports whether vibrato will produce any pitch modulation.
func (v VibratoEffect) IsActive() bool { return v.Depth > 0 && v.Rate > 0 }

// PortamentoEffect controls pitch glide from a previous note.
//
// StartTick is the first tick at which the glide begins. Before StartTick
// the oscillator holds at prevFreq. Ticks is the number of ticks over
// which the pitch slides exponentially from prevFreq to the target frequency.
// After the glide completes the pitch snaps to and holds at the target.
//
// Common patterns:
//   - StartTick=0, Ticks=0  → snap immediately (no portamento)
//   - StartTick=0, Ticks=N  → glide immediately over N ticks
//   - StartTick=S, Ticks=0  → hold prevFreq, snap to target at tick S
//   - StartTick=S, Ticks=N  → hold prevFreq, glide over N ticks from tick S
//
// Glide duration is clamped to the available ticks so the pitch always
// arrives at the target by the last tick.
// Row smoothness is controlled by the row's tick count: more ticks
// produce smoother glides.
// Not designed to be combined with RetriggerEnvelope (Reset clears the glide).
type PortamentoEffect struct {
	StartTick int // tick index at which the glide begins; 0 = immediate
	Ticks     int // number of ticks for the glide; 0 = snap (no glide)
}

// VolumeEffect sets the playback volume of the rendered note.
// Level is a linear gain in [0, 1]: 0 = silence, 1 = full amplitude.
// When Active is false the effect is disabled and the patch plays at its
// default volume (1.0). VolumeSlide, when also active, starts its fade
// from Level rather than from 1.0.
type VolumeEffect struct {
	Level  float64 // initial volume [0, 1]; only meaningful when Active is true
	Active bool
}

// IsActive reports whether the volume override is enabled.
func (v VolumeEffect) IsActive() bool { return v.Active }

// VolumeSlideEffect adjusts the output volume by a fixed delta at every tick.
// Delta is the per-tick change in [−1, 1]; positive increases volume,
// negative decreases it. Volume is clamped to [0, 1] after each step.
// A delta of 0 disables the effect.
type VolumeSlideEffect struct {
	Delta float64
}

// IsActive reports whether the slide will change volume.
func (v VolumeSlideEffect) IsActive() bool { return v.Delta != 0 }

// NoteCutEffect silences the note at a specific tick by setting volume to
// zero. Once triggered, the silence persists for the remainder of the note;
// subsequent VolumeSlide steps are ignored after the cut.
// Tick <= 0 disables the effect.
type NoteCutEffect struct {
	Tick int // 0-based tick index at which to cut; <= 0 = inactive
}

// IsActive reports whether the cut will fire.
func (n NoteCutEffect) IsActive() bool { return n.Tick > 0 }

// NoteDelayEffect defers note playback to a specific tick. Before Tick the
// streamer outputs silence; from Tick onward the patch plays with its ADSR
// envelope starting at that moment, so the full envelope applies only to the
// remaining duration.
// Tick <= 0 disables the effect (immediate playback).
type NoteDelayEffect struct {
	Tick int // 0-based tick index at which playback begins; <= 0 = immediate
}

// IsActive reports whether playback will be deferred.
func (n NoteDelayEffect) IsActive() bool { return n.Tick > 0 }

// PitchEffect holds at most one active pitch modulation.
// Exactly one of the pointer fields may be non-nil; all three being nil
// means no pitch effect is applied.
type PitchEffect struct {
	Arpeggio   *ArpeggioEffect
	Portamento *PortamentoEffect
	Vibrato    *VibratoEffect
}

// EffectDefinitions bundles all per-note effect definitions for an EffectsPatch.
type EffectDefinitions struct {
	Ticks             int // number of ticks per row; 0 means no subdivision (treated as 1)
	Pitch             PitchEffect
	Volume            VolumeEffect
	RetriggerEnvelope bool // restart ADSR at every tick boundary (useful with Arpeggio)
	VolumeSlide       VolumeSlideEffect
	NoteCut           NoteCutEffect
	NoteDelay         NoteDelayEffect
}

// NewEffectDefinitions returns a validated and clamped copy of fx.
//
// Clamping rules:
//   - Ticks < 1 is set to 1.
//   - Portamento.StartTick and Portamento.Ticks are clamped to >= 0.
//   - Vibrato.Depth and Vibrato.Rate are clamped to >= 0.
//   - Volume.Level is clamped to [0, 1] when Volume is active.
//   - VolumeSlide.Delta is clamped to [−1, 1].
//   - NoteCut.Tick and NoteDelay.Tick are clamped to >= 0
//     (negative values are normalised to 0, which means inactive).
func NewEffectDefinitions(fx EffectDefinitions) EffectDefinitions {
	if fx.Ticks < 1 {
		fx.Ticks = 1
	}

	if fx.Pitch.Portamento != nil {
		p := *fx.Pitch.Portamento
		if p.StartTick < 0 {
			p.StartTick = 0
		}
		if p.Ticks < 0 {
			p.Ticks = 0
		}
		fx.Pitch.Portamento = &p
	}

	if fx.Pitch.Vibrato != nil {
		v := *fx.Pitch.Vibrato
		if v.Depth < 0 {
			v.Depth = 0
		}
		if v.Rate < 0 {
			v.Rate = 0
		}
		fx.Pitch.Vibrato = &v
	}

	if fx.Volume.Active {
		fx.Volume.Level = max(0, min(1, fx.Volume.Level))
	}

	fx.VolumeSlide.Delta = max(-1, min(1, fx.VolumeSlide.Delta))

	if fx.NoteCut.Tick < 0 {
		fx.NoteCut.Tick = 0
	}
	if fx.NoteDelay.Tick < 0 {
		fx.NoteDelay.Tick = 0
	}

	return fx
}

// EffectsPatch couples a Synth with time-aware effect definitions.
// It knows the note duration in milliseconds and derives the tick count from
// EffectDefinitions.Ticks (0 is treated as 1), applying effects at each tick
// boundary when streaming.
//
// Typical usage:
//
//	ep := audio.NewEffectsPatch(synth, audio.EffectDefinitions{Ticks: 4, Pitch: audio.PitchEffect{Arpeggio: &arp}}, 125.0)
//	streamer := ep.Streamer(sr, note, prevFreq)
//	speaker.Play(streamer)
type EffectsPatch struct {
	synth      *Synth
	effects    EffectDefinitions
	durationMs float64
}

// NewEffectsPatch creates an EffectsPatch from the given Synth definition, effect
// definitions, and note duration in milliseconds. The tick count is taken from
// fx.Ticks; a value less than 1 is treated as 1. fx is validated and clamped
// via NewEffectDefinitions before being stored.
func NewEffectsPatch(synth *Synth, fx EffectDefinitions, durationMs float64) *EffectsPatch {
	return &EffectsPatch{
		synth:      synth,
		effects:    NewEffectDefinitions(fx),
		durationMs: durationMs,
	}
}

// Streamer returns an audio.Streamer that renders the given frequency for
// durationMs milliseconds, applying effects at each tick boundary.
//
// prevFreq is the frequency of the last note played on the same track; pass 0
// when there is no previous note. It is only used when Portamento.Ticks > 0.
func (ep *EffectsPatch) Streamer(sr SampleRate, freq float64, prevFreq float64) Streamer {
	totalSamples := sr.N(time.Duration(ep.durationMs * float64(time.Millisecond)))

	startFreq := freq
	p := ep.effects.Pitch.Portamento
	gliding := p != nil && (p.Ticks > 0 || p.StartTick > 0) && prevFreq > 0 && freq > 0
	if gliding {
		startFreq = prevFreq
	}

	ticks := ep.effects.Ticks
	if ticks < 1 {
		ticks = 1
	}
	tickSamples := totalSamples / ticks
	remainder := totalSamples - tickSamples*ticks

	initialVolume := 1.0
	if ep.effects.Volume.IsActive() {
		initialVolume = ep.effects.Volume.Level
	}

	s := &effectsStreamer{
		noteFreq:       freq,
		pitch:          ep.effects.Pitch,
		retrigger:      ep.effects.RetriggerEnvelope,
		volumeSlide:    ep.effects.VolumeSlide,
		noteCut:        ep.effects.NoteCut,
		noteDelay:      ep.effects.NoteDelay,
		gliding:        gliding,
		tickSamples: tickSamples,
		remainder:      remainder,
		totalTicks:  ticks,
		currentVolume:  initialVolume,
		pendingTick: true,
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

	// Wrap with a gain stage when VolumeEffect is active so the level is
	// applied uniformly to every sample regardless of tick boundaries.
	if ep.effects.Volume.IsActive() {
		return &effects.Gain{Streamer: s, Gain: initialVolume - 1}
	}
	return s
}

type effectsStreamer struct {
	patch    *Patch // nil before NoteDelay fires
	noteFreq float64

	pitch     PitchEffect
	retrigger bool
	gliding   bool // portamento glide in progress

	volumeSlide   VolumeSlideEffect
	noteCut       NoteCutEffect
	noteDelay     NoteDelayEffect
	currentVolume float64 // running volume for VolumeSlide; starts at 1.0
	cut           bool    // true after NoteCut fires

	tickSamples    int // base samples per tick (last tick may be longer)
	remainder      int // extra samples added to the very last tick
	totalTicks  int
	currentTick int
	samplesInTick  int
	pendingTick bool

	// Retained for deferred patch creation when NoteDelay is active.
	synth     *Synth
	sr        SampleRate
	startFreq float64
	prevFreq  float64
}

// tickSize returns the number of samples in the current tick.
// The last tick absorbs any rounding remainder so the total sample count
// exactly matches the originally requested duration.
func (e *effectsStreamer) tickSize() int {
	if e.currentTick == e.totalTicks-1 {
		return e.tickSamples + e.remainder
	}
	return e.tickSamples
}

// Stream implements audio.Streamer.
// Effects are applied at the start of each tick before pulling samples
// from the underlying Patch.
func (e *effectsStreamer) Stream(samples [][2]float64) (int, bool) {
	total := 0
	for len(samples) > 0 {
		if e.currentTick >= e.totalTicks {
			return total, false
		}

		if e.pendingTick {
			e.applyTickEffects()
			e.pendingTick = false
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
			e.currentTick++
			e.samplesInTick = 0
			if e.currentTick < e.totalTicks {
				if e.retrigger && e.patch != nil {
					e.patch.Reset()
				}
				e.pendingTick = true
			}
		}

		if !ok || n == 0 {
			return total, false
		}
	}
	return total, e.currentTick < e.totalTicks
}

// applyTickEffects fires all time-based effects for the current tick.
//
// Pitch priority (highest wins): Arpeggio → Vibrato → Portamento.
// Volume priority: NoteCut silences permanently; VolumeSlide is suppressed
// after a cut. Both are independent of the pitch effects.
func (e *effectsStreamer) applyTickEffects() {
	tick := e.currentTick

	// NoteDelay: create the patch when the delay tick arrives; return early
	// for all preceding ticks so no effects fire during the silence period.
	if e.noteDelay.IsActive() {
		if e.patch == nil {
			if tick < e.noteDelay.Tick {
				return
			}
			// Delay fires: create a patch sized for the remaining duration so
			// the full ADSR envelope applies to exactly the ticks that will play.
			ticksLeft := e.totalTicks - tick
			remainingSamples := ticksLeft*e.tickSamples + e.remainder
			e.patch = e.synth.NewPatch(e.sr, e.startFreq, remainingSamples)
		}
	}

	if e.patch == nil {
		return
	}

	// Effective tick index relative to when the patch started playing
	// (always 0 on the first active tick, regardless of NoteDelay).
	effectiveTick := tick
	effectiveTotal := e.totalTicks
	if e.noteDelay.IsActive() {
		effectiveTick = tick - e.noteDelay.Tick
		effectiveTotal = e.totalTicks - e.noteDelay.Tick
	}

	// Pitch effects.
	if e.pitch.Arpeggio != nil && e.pitch.Arpeggio.IsActive() {
		offset := e.pitch.Arpeggio.Offsets[effectiveTick%len(e.pitch.Arpeggio.Offsets)]
		arpFreq := e.noteFreq * math.Pow(2, float64(offset)/12.0)
		e.patch.SetFrequency(arpFreq)
	} else if e.pitch.Vibrato != nil && e.pitch.Vibrato.IsActive() {
		phase := 2 * math.Pi * e.pitch.Vibrato.Rate * float64(effectiveTick) / float64(effectiveTotal)
		semitoneShift := e.pitch.Vibrato.Depth * math.Sin(phase)
		e.patch.SetFrequency(e.noteFreq * math.Pow(2, semitoneShift/12.0))
	} else if e.gliding {
		// Three phases: pre-glide (hold prevFreq), glide, post-glide (hold noteFreq).
		start := e.pitch.Portamento.StartTick
		glideEnd := start + e.pitch.Portamento.Ticks
		if glideEnd > effectiveTotal {
			glideEnd = effectiveTotal
		}
		switch {
		case effectiveTick < start:
			e.patch.SetFrequency(e.prevFreq)
		case effectiveTick < glideEnd:
			progress := effectiveTick - start
			total := glideEnd - start
			t := float64(progress+1) / float64(total)
			e.patch.SetFrequency(e.prevFreq * math.Pow(e.noteFreq/e.prevFreq, t))
		default:
			e.patch.SetFrequency(e.noteFreq)
		}
	}

	// Volume effects. NoteCut fires once and suppresses all subsequent
	// VolumeSlide steps so the silence cannot be un-done by a slide.
	if e.noteCut.IsActive() && tick == e.noteCut.Tick {
		e.cut = true
		e.patch.SetVolume(0)
	}
	// VolumeSlide steps starting from tick 1, so tick 0 plays at the
	// initial (or VolumeEffect-seeded) volume before any delta is applied.
	if !e.cut && e.volumeSlide.IsActive() && effectiveTick > 0 {
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
