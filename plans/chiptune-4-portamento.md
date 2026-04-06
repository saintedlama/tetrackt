# Chiptune-4: Portamento / Pitch Glide

**Status:** Planned

**Priority:** Medium

## Problem

No pitch glide between consecutive notes.

## Why It Matters

Essential for bass slides, lead smoothness and SID-style glides.

## Required

- `Portamento float64` – time in seconds (or ticks) to slide from previous note frequency to current
- Implemented as a per-sample frequency interpolation in the oscillator

## Implementation Plan

### 1. Extend `oscillatorGenerator` (`audio/oscillator.go`)

Add four fields:

```go
startFrequency    float64
targetFrequency   float64
portamentoSamples int   // total samples over which to slide
portamentoIdx     int   // samples elapsed so far
```

Keep `frequency float64` as the live, per-sample value. When `portamentoSamples == 0` the existing behaviour is preserved exactly.

### 2. Per-sample interpolation in `Stream()` (`audio/oscillator.go`)

Inside the sample loop, before computing `phaseIncrement`, advance the glide:

```go
if g.portamentoIdx < g.portamentoSamples {
    t := float64(g.portamentoIdx) / float64(g.portamentoSamples)
    // linear:
    g.frequency = g.startFrequency + t*(g.targetFrequency-g.startFrequency)
    // exponential alternative (perceptually linear in pitch):
    // g.frequency = g.startFrequency * math.Pow(g.targetFrequency/g.startFrequency, t)
    g.portamentoIdx++
} else {
    g.frequency = g.targetFrequency
}
phaseIncrement := g.frequency / sampleRate
```

Exponential is preferred for melodic glides (equal ratio per sample = equal semitones per sample); linear is acceptable for very short slides.

### 3. `Synth.Streamer` variant (`audio/synth.go`)

Add an optional start frequency parameter via a new method rather than changing the existing signature (backward compatible):

```go
// Streamer remains unchanged — zero startFreq means no glide.
func (s *Synth) StreamerWithGlide(note Note, d time.Duration, startFreq float64, portamento float64) beep.Streamer
```

Internally this constructs the `oscillatorGenerator` with:

```go
portamentoSamples: int(portamento * sampleRate),
startFrequency:    startFreq,          // 0 → snap to target immediately
targetFrequency:   noteFrequency(note),
```

`Streamer` can simply delegate: `s.StreamerWithGlide(note, d, 0, 0)`.

### 4. Previous-note tracking (tracker/playback layer, not `audio/`)

The audio engine is stateless by design; it must not own playback history. The playback engine (wherever pattern rows are iterated) should maintain:

```go
var prevFreq [maxChannels]float64 // last triggered frequency per channel
```

When a note-on event is emitted for channel `ch`:

1. Read `prevFreq[ch]` as `startFreq`.
2. Call `synth.StreamerWithGlide(note, dur, startFreq, instrument.Portamento)`.
3. Update `prevFreq[ch] = noteFrequency(note)`.

If `instrument.Portamento == 0` the glide is skipped transparently.

### 5. Portamento rate: seconds → samples

Convert at oscillator construction time (not at stream time):

```go
portamentoSamples = int(portamentoSeconds * float64(sampleRate))
```

This keeps `Stream()` arithmetic integer-only (cheap `portamentoIdx < portamentoSamples` compare).

---

## Impact

| Dimension                  | Assessment                                                                                                                                                  |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Invasiveness**           | Low. The oscillator change is self-contained; glide is a no-op when `portamentoSamples == 0`.                                                               |
| **Files touched**          | `audio/oscillator.go` (fields + loop body), `audio/synth.go` (new `StreamerWithGlide` method), playback engine file (prev-freq tracking).                   |
| **Backward compatibility** | Fully additive. `Synth.Streamer` signature is unchanged; `oscillatorGenerator` defaults to zero portamento. No existing call sites break.                   |
| **Risk**                   | Exponential interpolation requires a non-zero `startFrequency`; a guard (`startFreq <= 0 → linear or snap`) is needed to avoid `math.Pow` with a zero base. |
