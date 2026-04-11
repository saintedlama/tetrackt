# Chiptune-9: Filter Cutoff Envelope

**Status:** Done

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

- **`Synth`** struct holds three oscillators with envelopes (`Oscillator1`/`Envelope1`–`3`), mixer, filter, three LFOs, and portamento. **No `FilterEnvelope` field yet.**
- **`Patch`** is the live instance. Two factory methods exist:
  - `Synth.NewPatch(sampleRate, frequency, noteSamples) *Patch` — fixed-duration ADSR
  - `Synth.NewGatedPatch(sampleRate, frequency) *Patch` — sustains until `NoteOff`; uses `gatedEnvelopeGenerator` per voice
- Both pipelines end with:
  ```go
  pipeline := NewModulatedFilterStreamer(mixed, sampleRate, s.Filter, makeLFO(ModCutoff))
  ```

### Filter Implementation (`audio/filter.go` + `audio/lfo.go`)

- **`Filter`** struct:
  ```go
  type Filter struct {
      Type      FilterType  // FilterOff | FilterLowPass | FilterHighPass | FilterBandPass
      Cutoff    float64     // normalised [0, 1]; maps to ~20 Hz–18 kHz on log scale
      Resonance float64     // normalised [0, 1]; maps to Q ≈ 0.5–20
  }
  ```
- **`biquadFilter`** (in `filter.go`) — the core IIR biquad; `calcCoeffs()` recomputes coefficients from a `Filter` value.
- **`modulatedFilterStreamer`** (in `lfo.go`) — wraps `biquadFilter`; when an LFO is present, modulates cutoff per block: `newCutoff = f.Cutoff + lfo.mod * 0.5`, clamped to `[0, 1]`, then reruns `calcCoeffs`.
- **`NewModulatedFilterStreamer(src, sampleRate, f Filter, lfo *lfoGenerator)`** — constructs a `biquadFilter` with optional `modulatedFilterStreamer` wrapper; lives in `lfo.go`.

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
- **`envelopeGenerator`** applies ADSR gain with exponential ramps via `calculateMultiplier(startLevel, endLevel, lengthInSamples) float64` (unexported, but in the same `audio` package — no export needed).
- **`minEnvelopeLevel = 0.0001`** package constant guards against `math.Log(0)` in `calculateMultiplier`; reuse it in the filter envelope.
- Stages: `StageOff`, `StageAttack`, `StageDecay`, `StageSustain`, `StageRelease`.
- `Patch.Reset()` calls `env1.reset()`, `env2.reset()`, `env3.reset()`, and resets all LFOs.

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
    Oscillator3    Oscillator
    Envelope3      Envelope
    Mixer          Mixer
    Filter         Filter
    FilterEnvelope FilterEnvelope  // NEW
    LFO1           LFO
    LFO2           LFO
    LFO3           LFO
    Portamento     float64
}
```

### 3. Create `filterEnvelopeGenerator` streamer (`audio/filter.go`)

A new unexported type wrapping the filter with ADSR-driven cutoff modulation. Mirrors the `envelopeGenerator` pattern. Lives in `audio/filter.go`; `calculateMultiplier` and `minEnvelopeLevel` are in the same `audio` package so no export is needed.

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

`newFilterEnvelopeGenerator(src, sampleRate, noteSamples, f Filter, fe FilterEnvelope, lfo) *filterEnvelopeGenerator` converts durations to sample counts and uses `calculateMultiplier` and `minEnvelopeLevel` (both already in `audio`).

### 4. Integrate into `NewPatch` and `NewGatedPatch` pipelines

In `audio/synth.go`, replace the final filter line in **both** factory functions:

```go
var pipeline beep.Streamer
if s.FilterEnvelope.Depth > 0 {
    pipeline = newFilterEnvelopeGenerator(mixed, sampleRate, noteSamples, s.Filter, s.FilterEnvelope, makeLFO(ModCutoff))
} else {
    pipeline = NewModulatedFilterStreamer(mixed, sampleRate, s.Filter, makeLFO(ModCutoff))
}
```

For `NewGatedPatch`, pass `math.MaxInt` as `noteSamples` (same as `p.noteSamples` for gated patches), making the sustain phase hold indefinitely — correct behaviour for a held note.

Store a reference in `Patch` so `Reset()` can restart it:

```go
type Patch struct {
    // ... existing fields ...
    filterEnvGen *filterEnvelopeGenerator // nil for fixed-filter patches
}
```

### 5. `filterEnvelopeGenerator.Stream()` logic

For each sample:
1. Advance ADSR stage and level (same index-based transitions as `envelopeGenerator.nextSample()`). Use `minEnvelopeLevel` as the floor for `calculateMultiplier` arguments.
2. Compute effective cutoff:
   ```go
   effectiveCutoff := m.baseFilter.Cutoff + m.currentLevel*m.depth
   ```
3. Apply any LFO modulation from `lfo.nextBlock(n)` as an additive offset (identical to `modulatedFilterStreamer`).
4. Clamp to `[0, 1]`, recompute biquad coefficients, stream through filter.

### 6. Hook into `Patch.Reset()`

`Reset()` already resets `env1`–`env3` and all LFOs. Add the filter envelope:

```go
func (p *Patch) Reset() {
    p.env1.reset()
    p.env2.reset()
    p.env3.reset()
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

### 7. Persistence (`persistence/song.go`)

The existing pattern: one `Saved*` struct per audio type, with `to*` / `from*` converters. `FilterEnvelope` follows the same convention as `SavedEnvelope` (durations as float64 seconds, depth as float64).

Add `SavedFilterEnvelope`:

```go
type SavedFilterEnvelope struct {
    Attack  float64 `json:"attack,omitempty"`
    Decay   float64 `json:"decay,omitempty"`
    Sustain float64 `json:"sustain,omitempty"`
    Release float64 `json:"release,omitempty"`
    Depth   float64 `json:"depth,omitempty"`
}

func toSavedFilterEnvelope(fe audio.FilterEnvelope) SavedFilterEnvelope {
    return SavedFilterEnvelope{
        Attack:  fe.Attack.Seconds(),
        Decay:   fe.Decay.Seconds(),
        Sustain: fe.Sustain,
        Release: fe.Release.Seconds(),
        Depth:   fe.Depth,
    }
}

func fromSavedFilterEnvelope(s SavedFilterEnvelope) audio.FilterEnvelope {
    return audio.FilterEnvelope{
        Attack:  time.Duration(s.Attack * float64(time.Second)),
        Decay:   time.Duration(s.Decay * float64(time.Second)),
        Sustain: s.Sustain,
        Release: time.Duration(s.Release * float64(time.Second)),
        Depth:   s.Depth,
    }
}
```

Add `FilterEnvelope SavedFilterEnvelope \`json:"filter_envelope,omitempty"\`` to `SavedSynth`, and wire it through `toSavedSynth` / `fromSavedSynth`. All fields use `omitempty` so existing songs without the field deserialise to a zero-depth `FilterEnvelope` (disabled).

### 8. UI (`ui/synth/filter.go`)

The `FilterModel` currently edits three fields (Type, Cutoff, Resonance). Extend it to also edit the four `FilterEnvelope` ADSR fields plus Depth — but only when the filter type is not `FilterOff`.

**Add fields to `FilterModel`:**

```go
type FilterModel struct {
    Filter          audio.Filter
    FilterEnvelope  audio.FilterEnvelope  // NEW
    // ... existing unexported fields ...
    envAttackBar  common.Bar  // NEW
    envDecayBar   common.Bar  // NEW
    envSustainBar common.Bar  // NEW
    envReleaseBar common.Bar  // NEW
    envDepthBar   common.Bar  // NEW
}
```

Extend `filterField` constants:

```go
const (
    filterFieldType filterField = iota
    filterFieldCutoff
    filterFieldResonance
    filterFieldEnvDepth    // NEW
    filterFieldEnvAttack   // NEW
    filterFieldEnvDecay    // NEW
    filterFieldEnvSustain  // NEW
    filterFieldEnvRelease  // NEW
)
```

Total fields: 8. Up/down navigation wraps with `% 8`.

**`View()` additions** (only rendered when `Filter.Type != FilterOff`):

```
Env Depth   ██████░░░░  60%
Env Attack  ██░░░░░░░░  20%
Env Decay   ███░░░░░░░  30%
Env Sustain ████░░░░░░  40%
Env Release ██░░░░░░░░  20%
```

**`FilterUpdated` message** already carries `audio.Filter`; extend it:

```go
type FilterUpdated struct {
    Filter         audio.Filter
    FilterEnvelope audio.FilterEnvelope  // NEW
}
```

**`screen.go` / `ApplyTrackChange`** — the handler for `FilterUpdated` in `ui/synth/screen.go` passes the filter to `GetSynth()`; extend it to also copy `FilterEnvelope`.

## Impact

- **Files touched:** `audio/filter.go` (new `filterEnvelopeGenerator` type and `FilterEnvelope` struct), `audio/synth.go` (`FilterEnvelope` field on `Synth`, `filterEnvGen` field on `Patch`, integration in both `NewPatch` and `NewGatedPatch`, and `Reset`), `ui/synth/filter.go` (5 new ADSR/depth fields, extended navigation and view), `persistence/song.go` (`SavedFilterEnvelope` struct and wiring).
- **Invasiveness:** Moderate. Adds a new streamer type and ADSR state machine; the biquad itself is unchanged. UI and persistence additions are purely additive.
- **Compatibility:** Fully additive. `FilterEnvelope` defaults to zero `Depth`, which disables the feature — both `NewPatch` and `NewGatedPatch` fall through to the existing `NewModulatedFilterStreamer` path. All existing songs, presets, and synth configurations are unaffected.
