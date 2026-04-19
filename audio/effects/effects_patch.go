package effects

import (
	"math"
	"time"

	"github.com/tetrackt/tetrackt/audio"
)

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
// When Active and a non-zero prevFreq is passed to Streamer, the pitch slides
// over every subtick from prevFreq to the target note's frequency.
// Not designed to be combined with RetriggerEnvelope (Reset clears the glide).
type PortamentoEffect struct {
	Active bool
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
	Arpeggio          audio.ArpeggioEffect
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
//	ep := effects.New(synth, effects.EffectDefs{Arpeggio: arp}, 125.0, 4)
//	streamer := ep.Streamer(sr, note, prevFreq)
//	speaker.Play(streamer)
type EffectsPatch struct {
	synth      *audio.Synth
	effects    EffectDefs
	durationMs float64
	subticks   int
}

// NewEffectsPatch creates an EffectsPatch from the given Synth definition, effect
// definitions, note duration in milliseconds, and subtick count.
// subticks < 1 is clamped to 1.
func NewEffectsPatch(synth *audio.Synth, fx EffectDefs, durationMs float64, subticks int) *EffectsPatch {
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
// when there is no previous note. It is only used when Portamento.Active is
// true.
func (ep *EffectsPatch) Streamer(sr audio.SampleRate, freq float64, prevFreq float64) audio.Streamer {
	totalSamples := sr.N(time.Duration(ep.durationMs * float64(time.Millisecond)))

	startFreq := freq
	gliding := ep.effects.Portamento.Active && prevFreq > 0 && freq > 0
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
	patch    *audio.Patch // nil before NoteDelay fires
	noteFreq float64

	arp        audio.ArpeggioEffect
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
	synth     *audio.Synth
	sr        audio.SampleRate
	startFreq float64
	prevFreq  float64
}

// tickSize returns the number of samples in the current subtick.
// The last subtick absorbs any rounding remainder so the total sample count
// exactly matches the originally requested duration.
func (s *effectsStreamer) tickSize() int {
	if s.currentSubtick == s.totalSubticks-1 {
		return s.subtickSamples + s.remainder
	}
	return s.subtickSamples
}

// Stream implements audio.Streamer.
// Effects are applied at the start of each subtick before pulling samples
// from the underlying Patch.
func (s *effectsStreamer) Stream(samples [][2]float64) (int, bool) {
	total := 0
	for len(samples) > 0 {
		if s.currentSubtick >= s.totalSubticks {
			return total, false
		}

		if s.pendingSubtick {
			s.applySubtickEffects()
			s.pendingSubtick = false
		}

		tickSize := s.tickSize()
		available := tickSize - s.samplesInTick
		chunk := samples
		if len(chunk) > available {
			chunk = samples[:available]
		}

		var n int
		var ok bool
		if s.patch != nil {
			n, ok = s.patch.Stream(chunk)
		} else {
			// NoteDelay: output silence until the patch is created.
			for i := range chunk {
				chunk[i] = [2]float64{}
			}
			n, ok = len(chunk), true
		}

		total += n
		s.samplesInTick += n
		samples = samples[n:]

		if s.samplesInTick >= tickSize {
			s.currentSubtick++
			s.samplesInTick = 0
			if s.currentSubtick < s.totalSubticks {
				if s.retrigger && s.patch != nil {
					s.patch.Reset()
				}
				s.pendingSubtick = true
			}
		}

		if !ok || n == 0 {
			return total, false
		}
	}
	return total, s.currentSubtick < s.totalSubticks
}

// applySubtickEffects fires all time-based effects for the current subtick.
//
// Pitch priority (highest wins): Arpeggio → Vibrato → Portamento.
// Volume priority: NoteCut silences permanently; VolumeSlide is suppressed
// after a cut. Both are independent of the pitch effects.
func (s *effectsStreamer) applySubtickEffects() {
	tick := s.currentSubtick

	// NoteDelay: create the patch when the delay tick arrives; return early
	// for all preceding ticks so no effects fire during the silence period.
	if s.noteDelay.IsActive() {
		if s.patch == nil {
			if tick < s.noteDelay.Tick {
				return
			}
			// Delay fires: create a patch sized for the remaining duration so
			// the full ADSR envelope applies to exactly the ticks that will play.
			ticksLeft := s.totalSubticks - tick
			remainingSamples := ticksLeft*s.subtickSamples + s.remainder
			s.patch = s.synth.NewPatch(s.sr, s.startFreq, remainingSamples)
			if s.gliding {
				s.patch.StartPortamento(s.prevFreq, s.noteFreq, ticksLeft)
			}
		}
	}

	if s.patch == nil {
		return
	}

	// Effective tick index relative to when the patch started playing
	// (always 0 on the first active subtick, regardless of NoteDelay).
	effectiveTick := tick
	effectiveTotal := s.totalSubticks
	if s.noteDelay.IsActive() {
		effectiveTick = tick - s.noteDelay.Tick
		effectiveTotal = s.totalSubticks - s.noteDelay.Tick
	}

	// Pitch effects.
	if s.arp.IsActive() {
		offset := s.arp.Offsets[effectiveTick%len(s.arp.Offsets)]
		arpFreq := s.noteFreq * math.Pow(2, float64(offset)/12.0)
		s.patch.SetFrequency(arpFreq)
	} else if s.vibrato.IsActive() {
		phase := 2 * math.Pi * s.vibrato.Rate * float64(effectiveTick) / float64(effectiveTotal)
		semitoneShift := s.vibrato.Depth * math.Sin(phase)
		s.patch.SetFrequency(s.noteFreq * math.Pow(2, semitoneShift/12.0))
	} else if s.portamento.Active && s.gliding {
		// Exponential glide: same formula as Patch.TickPortamento.
		// effectiveTick goes 0..N-1; step 0 advances to t=1/N so the glide
		// reaches the target frequency exactly on the last subtick.
		t := float64(effectiveTick+1) / float64(effectiveTotal)
		glideFreq := s.prevFreq * math.Pow(s.noteFreq/s.prevFreq, t)
		s.patch.SetFrequency(glideFreq)
	}

	// Volume effects. NoteCut fires once and suppresses all subsequent
	// VolumeSlide steps so the silence cannot be un-done by a slide.
	if s.noteCut.IsActive() && tick == s.noteCut.Tick {
		s.cut = true
		s.patch.SetVolume(0)
	}
	if !s.cut && s.volumeSlide.IsActive() {
		s.currentVolume = math.Max(0, math.Min(1, s.currentVolume+s.volumeSlide.Delta))
		s.patch.SetVolume(s.currentVolume)
	}
}

// Err implements audio.Streamer.
func (s *effectsStreamer) Err() error {
	if s.patch != nil {
		return s.patch.Err()
	}
	return nil
}
