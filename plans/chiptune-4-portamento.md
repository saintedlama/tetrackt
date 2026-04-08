# Chiptune-4: Portamento / Pitch Glide

**Status:** Done

**Priority:** Medium

## Problem

No pitch glide between consecutive notes.

## Why It Matters

Essential for bass slides, lead smoothness and SID-style glides.

## Required

- `Portamento float64` on `Synth` – seconds to slide from previous note to current
- Per-sample frequency interpolation in the oscillator
- No new `Synth` methods; portamento is driven through the existing `Streamer` interface

## Implementation Plan

### 1. Add `Portamento float64` to `Synth` (`audio/synth.go`)

Add the field alongside `Filter`, `LFO1`, `LFO2`:

```go
type Synth struct {
    Oscillator1 Oscillator
    // ...
    Portamento float64 // glide duration in seconds; 0 = snap
}
```

`NewSynth` gains a `portamento float64` parameter (or the caller sets it directly — `Synth` fields are exported). Existing callers that leave it zero are unaffected.

### 2. Extend `oscillatorGenerator` (`audio/oscillator.go`)

Add four fields:

```go
startFrequency    float64
targetFrequency   float64
portamentoSamples int   // total samples over which to slide
portamentoIdx     int   // samples elapsed so far
```

Keep `frequency float64` as the live, per-sample value. When `portamentoSamples == 0` the existing behaviour is preserved exactly.

### 3. Per-sample interpolation in `Stream()` (`audio/oscillator.go`)

Inside the sample loop, before computing `phaseIncrement`, advance the glide:

```go
if g.portamentoIdx < g.portamentoSamples {
    t := float64(g.portamentoIdx) / float64(g.portamentoSamples)
    // exponential (perceptually linear in pitch — equal semitones per sample):
    g.frequency = g.startFrequency * math.Pow(g.targetFrequency/g.startFrequency, t)
    g.portamentoIdx++
} else {
    g.frequency = g.targetFrequency
}
phaseIncrement := g.frequency / sampleRate
```

Guard: if `startFrequency <= 0`, skip the exponential and snap to `targetFrequency` immediately.

### 4. `buildChain` reads glide params from `Synth` (`audio/synth.go`)

`buildChain` already receives `frequency float64` (the target). When `s.Portamento > 0` and two frequencies are present (i.e. caller passed `[prevFreq, targetFreq]`), the oscillators are constructed with a glide:

```go
// In buildChain, after computing portamentoSamples from s.Portamento:
osc1 := NewOscillator(...)
osc1.startFrequency    = startFreq   // 0 if no glide
osc1.targetFrequency   = frequency
osc1.portamentoSamples = int(s.Portamento * sr)
```

`NewOscillator` sets `frequency = startFreq` (or `targetFreq` when no glide) so the first sample is already correct.

**Caller convention:** pass `frequencies = [prevFreq, targetFreq]` when portamento is desired. `buildChain` uses `frequencies[0]` as `startFreq` and `frequencies[1]` (or `[0]` if only one) as `targetFreq`. No new method needed — `Streamer` already accepts `[]float64`.

`tickCount=1, continuous=true` with two frequencies is the glide case. The `tickingStreamer` path is only entered when `tickCount > 1 && continuous`, so a two-element slice with `tickCount=1` routes through the normal single-chain path.

### 5. Previous-note tracking (tracker/playback layer, not `audio/`)

The audio engine is stateless by design; it must not own playback history. The playback engine should maintain:

```go
var prevFreq [maxChannels]float64 // last triggered frequency per channel
```

When a note-on event is emitted for channel `ch`:

1. Read `prevFreq[ch]` as `startFreq`.
2. Call `synth.Streamer(sr, []float64{startFreq, targetFreq}, 1, true, dur)`.
3. Update `prevFreq[ch] = targetFreq`.

If `synth.Portamento == 0`, the glide code is a no-op regardless of `startFreq`.

### 6. Persistence (`persistence/song.go`)

Add `Portamento float64 \`yaml:"portamento,omitempty"\`` to the saved track struct, alongside the existing synth fields.

---

## Impact

| Dimension                  | Assessment                                                                                                                                                                                   |
| -------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Invasiveness**           | Low. Oscillator change is self-contained; glide is a no-op when `portamentoSamples == 0`.                                                                                                    |
| **Files touched**          | `audio/oscillator.go` (fields + loop body), `audio/synth.go` (`Synth.Portamento` field + `buildChain` glide wiring), `persistence/song.go` (new field), playback layer (prev-freq tracking). |
| **Backward compatibility** | Fully additive. `Synth.Streamer` signature is unchanged; `Portamento` defaults to zero. No existing call sites break.                                                                        |
| **No new methods**         | `StreamerWithGlide` is dropped — glide is driven via `frequencies[0]` + `Synth.Portamento`, keeping the API surface minimal.                                                                 |
| **Risk**                   | Exponential interpolation requires non-zero `startFrequency`; guard (`startFreq <= 0 → snap`) prevents `math.Pow` with zero base.                                                            |
