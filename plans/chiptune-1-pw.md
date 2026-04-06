# Chiptune-1: Pulse Width Control

**Status:** Done

**Priority:** High

## Problem

The square wave oscillator is hardcoded at 50% duty cycle.

## Why It Matters

Variable pulse width is arguably _the_ defining chiptune timbre. PWM (continuous duty-cycle modulation) produces the classic wobbling/sweeping tone used across SID, NES, AY-3 and Amiga.

## Required

- `Oscillator.PulseWidth float64` parameter (0.01–0.99)
- PWM via LFO modulation of pulse width over time

## Implementation Plan

### 1. Add `PulseWidth` to `Oscillator` struct (`audio/oscilator.go`)

```go
type Oscillator struct {
    Type       OscillatorType
    Phase      float64  // normalized initial phase [0..1)
    PulseWidth float64  // duty cycle [0.01..0.99]; only used by Square; 0 means default (0.5)
}
```

A zero value defaults to 0.5 so all existing callers remain valid without change.

### 2. Thread `PulseWidth` into `oscillatorGenerator` (`audio/oscilator.go`)

Add the field to the internal generator and pass it through `NewOscillator`:

```go
type oscillatorGenerator struct {
    oscillatorType OscillatorType
    frequency      float64
    sampleRate     beep.SampleRate
    phase          float64
    pulseWidth     float64  // resolved to 0.5 when zero
}

func NewOscillator(oscillatorType OscillatorType, frequency float64, sampleRate beep.SampleRate, initialPhase float64, pulseWidth float64) beep.Streamer {
    pw := pulseWidth
    if pw == 0 {
        pw = 0.5
    }
    return &oscillatorGenerator{..., pulseWidth: pw}
}
```

### 3. Replace the hardcoded threshold in `Stream()` (`audio/oscilator.go`)

Change the `Square` case from:

```go
case Square:
    if g.phase < 0.5 {
```

to:

```go
case Square:
    if g.phase < g.pulseWidth {
```

This is the only sample-generation change required.

### 4. Update the two `NewOscillator` call sites (`audio/synth.go`)

`Synth.Streamer()` constructs both oscillators. Pass `s.oscillator1.PulseWidth` and `s.oscillator2.PulseWidth` as the new argument:

```go
oscillator1 := NewOscillator(s.oscillator1.Type, frequency, s.sampleRate, s.oscillator1.Phase, s.oscillator1.PulseWidth)
oscillator2 := NewOscillator(s.oscillator2.Type, frequency, s.sampleRate, s.oscillator2.Phase, s.oscillator2.PulseWidth)
```

### 5. PWM hook-in point (future LFO plan)

When a subsequent plan introduces an LFO, the `oscillatorGenerator.pulseWidth` field becomes the modulation target. The LFO streamer will write into it (or a shared `*float64` pointer) once per sample before the `Square` threshold comparison — no structural changes to `Stream()` are needed beyond step 3 above.

## Impact

| Dimension | Assessment |
|---|---|
| **Files touched** | `audio/oscilator.go` (struct + generator + `NewOscillator` signature + `Stream` body), `audio/synth.go` (two call sites) |
| **Invasiveness** | Minimal. One new field on two structs, one new parameter on one function, one comparison changed. |
| **Backward compatibility** | `NewOscillator` gains a parameter — a **breaking signature change** for any direct caller outside `synth.go`. Within the package the only caller is `Synth.Streamer()`, which is updated in the same PR. UI or persistence layers that construct `Oscillator` literals need to add `PulseWidth: 0` (or omit it for zero-value default). |
| **Risk** | Near-zero. Non-square waveforms are untouched. The zero-value default preserves all existing timbres exactly. |
