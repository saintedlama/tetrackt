# Chiptune-5: Fine Tuning / Detune

**Status:** Done

**Priority:** Medium

## Problem

`Transpose` only shifts whole octaves. There is no semitone or cent offset.

## Why It Matters

Detuning two oscillators against each other creates width and beating effects central to chiptune leads and pads.

## Required

- `Oscillator.Detune float64` in cents (±100 = ±1 semitone)
- Fix `Transpose` to accept semitone delta, not just octave

## Implementation Plan

### 1. Fix `Note.Transpose` — `audio/notes.go`

Change the semantics of `delta` from octaves to semitones. A delta of 12 equals one octave.

The current implementation adds `delta` directly to `Octave`. Replace it with chromatic index arithmetic:

```go
// Convert Note to a chromatic index, shift, then convert back.
func (n Note) Transpose(semitones int) Note {
    index := int(n.Base) + n.Octave*12 + semitones
    return Note{
        Base:   NoteBase(((index % 12) + 12) % 12), // wrap into [0,11]
        Octave: index / 12,
    }
}
```

`Frequency()` requires no change — it already combines `Base` and `Octave` correctly.

**Breaking change:** callers currently passing `delta=1` to mean "+1 octave" must be updated to pass `delta=12`.

### 2. Add `Oscillator.Detune` — `audio/oscillator.go` + `audio/synth.go`

Add the field to the struct:

```go
type Oscillator struct {
    Type  OscillatorType
    Phase float64
    Detune float64 // cents; 0 = no detune
}
```

In `synth.go`, before constructing each oscillator, apply the cent offset to the base frequency:

```go
detunedFreq := frequency * math.Pow(2, osc.Detune/1200.0)
gen := NewOscillator(osc.Type, detunedFreq, sampleRate)
```

No changes are needed inside `oscillatorGenerator` — it treats frequency as an opaque `float64`.

### 3. Persistence — `persistence/`

If `Oscillator` is serialised (JSON/binary), add `Detune` to the schema. Zero value is the neutral default, so existing saved files remain valid without migration.

## Impact

| Dimension           | Assessment                                                                                                                                                                            |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Files touched       | `audio/notes.go`, `audio/oscillator.go`, `audio/synth.go`, optionally `persistence/`                                                                                                  |
| Invasiveness        | Low — two small, localised changes                                                                                                                                                    |
| Additive / breaking | `Detune` field is purely additive (zero value = no effect). `Transpose` semitone fix **is breaking**: existing call sites that pass an octave delta must be updated to multiply by 12 |
| Risk                | Transpose change affects any code that calls `Transpose`; a project-wide grep for `\.Transpose(` is required before merging                                                           |
