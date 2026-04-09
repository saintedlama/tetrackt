# Chiptune-9: Filter Cutoff Envelope

**Status:** Planned

**Priority:** Low

## Problem

Filter cutoff is static per note, or modulated only by a global LFO.

## Why It Matters

Filter envelope sweeps (e.g. opening an LP filter on a bass hit, closing it on release) are a staple of subtractive synthesis and chiptune sound design.

## Required

- `FilterEnvelope` with its own ADSR and depth
- Applied as an additive offset to `Filter.Cutoff` per sample

## Current Implementation Context

### Synth & Patch Pipeline

- **`Synth`** struct holds parameter definitions (two oscillators with envelopes, mixer, filter, two LFOs, portamento). **No `FilterEnvelope` field yet.**
- **`Patch`** is the live instance created via `Synth.NewPatch(sampleRate, frequency, noteSamples) *Patch`.
- The pipeline in `NewPatch()` ends with:
  ```go
  pipeline := NewModulatedFilterStreamer(mixed, sampleRate, s.Filter, makeLFO(ModCutoff))
  ```

### Filter Implementation (`audio/filter.go`)

- **`Filter`** struct:
  ```go
  type Filter struct {
      Type      FilterType  // FilterOff | FilterLowPass | FilterHighPass | FilterBandPass
      Cutoff    float64     // normalised [0, 1]; maps to ~20 Hz–18 kHz on log scale
      Resonance float64     // normalised [0, 1]; maps to Q ≈ 0.5–20
  }
  ```
- **`biquadFilter`** — the core IIR biquad; `calcCoeffs()` recomputes coefficients from a `Filter` value.
- **`NewModulatedFilterStreamer(src, sampleRate, f Filter, lfo *lfoGenerator)`** — when an LFO is present, cutoff is modulated per block: `newCutoff = f.Cutoff + lfo.mod * 0.5`, clamped to `[0, 1]`, then `calcCoeffs` is rerun.

### Envelope Implementation (`audio/effects.go`)

- **`Envelope`** struct:
  ```go
  type Envelope struct {
      Attack  time.Duration
      Decay   time.Duration
      Sustain float64 // [0, 1] sustain level
      Release time.Duration
  }
  ```
- **`envelopeGenerator`** applies ADSR gain with exponential ramps via `calculateMultiplier(startLevel, endLevel, lengthInSamples) float64`.
- Stages: `StageOff`, `StageAttack`, `StageDecay`, `StageSustain`, `StageRelease`.
- `Patch.Reset()` calls `env1.reset()` and `env2.reset()` to restart envelopes.

## Implementation Plan

### 1. Define `FilterEnvelope` struct

In `audio/synth.go` (alongside `Envelope`):

```go
type FilterEnvelope struct {
    Attack  time.Duration
    Decay   time.Duration
    Sustain float64 // [0, 1]
    Release time.Duration
    Depth   float64 // [0, 1]; scales maximum additive offset on Filter.Cutoff
}
```

### 2. Add `FilterEnvelope` to `Synth`

```go
type Synth struct {
    Oscillator1    Oscillator
    Envelope1      Envelope
    Oscillator2    Oscillator
    Envelope2      Envelope
    Mixer          Mixer
    Filter         Filter
    FilterEnvelope FilterEnvelope  // NEW
    LFO1           LFO
    LFO2           LFO
    Portamento     float64
}
```

### 3. Create `filterEnvelopeGenerator` streamer (`audio/filter.go`)

A new unexported type wrapping the filter with ADSR-driven cutoff modulation. Mirrors the `envelopeGenerator` pattern:

```go
type filterEnvelopeGenerator struct {
    src        beep.Streamer
    filter     *biquadFilter
    lfo        *lfoGenerator
    baseFilter Filter
    sampleRate beep.SampleRate
    depth      float64

    // ADSR state (same pattern as envelopeGenerator)
    idx              int
    currentStage     Stages
    currentLevel     float64
    currentMultiplier float64
    sustain          float64
    attackSamples    int
    decaySamples     int
    sustainSamples   int
    releaseSamples   int
}
```

`newFilterEnvelopeGenerator(src, sampleRate, noteSamples, f Filter, fe FilterEnvelope, lfo) *filterEnvelopeGenerator` converts durations to sample counts and uses `calculateMultiplier` for exponential ramps. (Export `calculateMultiplier` from `effects.go` if needed, or duplicate it.)

### 4. Integrate into `NewPatch` pipeline

In `audio/synth.go`, replace the final filter line:

```go
var pipeline beep.Streamer
if s.FilterEnvelope.Depth > 0 {
    pipeline = newFilterEnvelopeGenerator(mixed, sampleRate, noteSamples, s.Filter, s.FilterEnvelope, makeLFO(ModCutoff))
} else {
    pipeline = NewModulatedFilterStreamer(mixed, sampleRate, s.Filter, makeLFO(ModCutoff))
}
```

Store a reference in `Patch` so `Reset()` can restart it:

```go
type Patch struct {
    // ... existing fields ...
    filterEnvGen *filterEnvelopeGenerator // nil for fixed-filter patches
}
```

### 5. `filterEnvelopeGenerator.Stream()` logic

For each sample:
1. Advance ADSR stage and level (same index-based transitions as `envelopeGenerator.nextSample()`).
2. Compute effective cutoff:
   ```go
   effectiveCutoff := m.baseFilter.Cutoff + m.currentLevel*m.depth
   ```
3. Apply any LFO modulation from `lfo.nextBlock(n)` as an additive offset.
4. Clamp to `[0, 1]`, recompute biquad coefficients, stream through filter.

### 6. Hook into `Patch.Reset()`

```go
func (p *Patch) Reset() {
    p.env1.reset()
    p.env2.reset()
    if p.filterEnvGen != nil {
        p.filterEnvGen.reset()
    }
    for _, lfo := range p.lfos {
        lfo.reset()
    }
    p.remaining = p.noteSamples
    p.portamento = portamento{}
}
```

## Impact

- **Files touched:** `audio/filter.go` (new `filterEnvelopeGenerator` type), `audio/synth.go` (`FilterEnvelope` field on `Synth`, `filterEnvGen` field on `Patch`, integration in `NewPatch` and `Reset`).
- **Invasiveness:** Moderate. Adds a new streamer type and ADSR state machine; the biquad itself is unchanged.
- **Compatibility:** Fully additive. `FilterEnvelope` defaults to zero `Depth`, which disables the feature — `NewPatch` falls through to the existing `NewModulatedFilterStreamer` path. All existing songs, presets, and synth configurations are unaffected.
