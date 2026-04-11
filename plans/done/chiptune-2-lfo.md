# Chiptune-2: LFO (Low Frequency Oscillator)

**Status:** Done

**Priority:** High

## Problem

No modulation sources exist. Every parameter is static for the lifetime of a note.

## Why It Matters

Vibrato, tremolo, auto-wah, PWM sweep — all classic chiptune articulations — require a periodic modulation signal routed to a destination.

## Required

- `LFO` struct: waveform, rate (Hz), depth, delay (onset time)
- Modulation destinations: pitch, volume, filter cutoff, pulse width
- At minimum one LFO per voice; ideally one LFO per destination

## Implementation Plan

### 1. `LFO` struct (`audio/lfo.go`)

```go
type LFOWaveform int

const (
    LFOSine LFOWaveform = iota
    LFOTriangle
    LFOSquare
    LFOSawtooth
)

type LFO struct {
    Waveform LFOWaveform
    Rate     float64 // Hz
    Depth    float64 // [0, 1] — scales the [-1,+1] output
    Delay    float64 // seconds before onset; output is 0 before this
}
```

### 2. `lfoGenerator` (`audio/lfo.go`)

Implement as a `beep.Streamer` (or bare sample-producer) with its own phase accumulator, mirroring the pattern in `oscillatorGenerator`:

```go
type lfoGenerator struct {
    lfo      LFO
    phase    float64 // [0, 1)
    elapsed  float64 // seconds since note start
    sampleRate float64
}

func (g *lfoGenerator) sample() float64 {
    g.elapsed += 1.0 / g.sampleRate
    if g.elapsed < g.lfo.Delay {
        return 0
    }
    g.phase += g.lfo.Rate / g.sampleRate
    if g.phase >= 1 {
        g.phase -= 1
    }
    raw := lfoWaveformSample(g.lfo.Waveform, g.phase) // returns [-1, +1]
    return raw * g.lfo.Depth
}
```

`lfoWaveformSample` is a pure function selecting the waveform formula (sine via `math.Sin`, triangle via piecewise linear, square via sign, sawtooth via `2*phase - 1`).

### 3. Modulation per destination

| Destination   | Formula (per sample)                                            |
| ------------- | --------------------------------------------------------------- |
| Pitch         | `effectiveFreq = baseFreq * (1 + mod)`                          |
| Volume        | `effectiveLevel = envelopeLevel * (1 + mod)`                    |
| Filter cutoff | `effectiveCutoff = clamp(baseCutoff + mod * cutoffRange, 0, 1)` |
| Pulse width   | `effectiveDuty = clamp(baseDuty + mod * 0.5, 0.05, 0.95)`       |

`mod` is the scalar returned by `lfoGenerator.sample()` for that tick.

### 4. Pipeline integration

**Pitch (oscillator frequency)** — wrap `oscillatorGenerator` in a new `modulatedOscillatorGenerator`:

```go
type modulatedOscillatorGenerator struct {
    inner   *oscillatorGenerator
    lfo     *lfoGenerator
    baseFreq float64
}
// Stream(): compute mod, update inner.freq = baseFreq * (1 + mod), then delegate to inner.
```

`oscillatorGenerator` needs one exported/unexported `freq` field (or a `setFreq` method) to allow per-sample updates. This is a minimal, localised change to `oscillator.go`.

**Volume (envelope)** — wrap the ADSR streamer from `effects.go` in a `modulatedEnvelopeStreamer` that multiplies each output sample by `(1 + lfo.sample())` after the envelope stage.

**Filter cutoff** — recompute biquad coefficients per-sample (or per-block at a lower rate) inside a `modulatedFilterStreamer` that wraps the filter from `filter.go` and calls a `setNormalisedCutoff(c float64)` helper each tick.

**Pulse width** — pass the per-sample duty value into `oscillatorGenerator` alongside frequency; the square-wave branch reads it instead of a fixed constant.

### 5. Exposing LFO configuration on `Synth` (`audio/synth.go`)

Add an `LFOs` map (or fixed array of four) on `Synth`:

```go
type ModDest int

const (
    ModPitch  ModDest = iota
    ModVolume
    ModCutoff
    ModPulseWidth
)

// On Synth:
LFOs map[ModDest]*LFO
```

Inside `Streamer(note, duration)`, after constructing the base pipeline, iterate `Synth.LFOs`; for each entry, instantiate the corresponding `lfoGenerator` and wrap the relevant pipeline stage with the appropriate modulated wrapper. If `LFOs` is nil or empty, the pipeline is constructed exactly as today — no overhead.

## Impact

**Invasiveness:** Low to moderate. The core pipeline topology in `synth.go` is unchanged; LFO wrappers slot in as optional decorators.

**Files touched:**

- `audio/lfo.go` — new file (struct, generator, waveform math)
- `audio/oscillator.go` — add mutable `freq`/`duty` fields and per-sample setter; minimal change (~5 lines)
- `audio/filter.go` — add `setNormalisedCutoff` helper to recompute coefficients on demand
- `audio/synth.go` — add `LFOs` field; extend `Streamer` to wire modulated wrappers when LFOs are configured

**Backward compatibility:** Fully additive. `LFOs` defaults to nil; all existing `Synth` instances behave identically. No public signatures are removed or reordered. The only structural mutation is the addition of settable fields on `oscillatorGenerator`, which is an unexported type.
