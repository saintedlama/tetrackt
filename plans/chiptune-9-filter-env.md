# Chiptune-9: Filter Cutoff Envelope

**Status:** Planned

**Priority:** Low

## Problem

Filter cutoff is static per note.

## Why It Matters

Filter sweeps (e.g. opening LP filter on a bass hit) are a staple effect.

## Required

- `FilterEnvelope` with its own ADSR and depth
- Applied as an additive offset to `Filter.Cutoff` per sample

## Implementation Plan

1. **Define `FilterEnvelope`** in `audio/filter.go` (or a new `audio/filter_envelope.go`):
   ```go
   type FilterEnvelope struct {
       Attack, Decay, Sustain, Release float64 // seconds / level, matching ADSR in effects.go
       Depth                           float64 // normalised 0–1; scales additive offset on Filter.Cutoff
   }
   ```
   `Depth * maxCutoffHz` gives the maximum cutoff offset in Hz.

2. **Wire into `Synth`**: add a `FilterEnvelope FilterEnvelope` field alongside the existing `Filter Filter` field in `synth.go`. Pass both to `NewFilterStreamer` (or a new wrapper streamer).

3. **Envelope state**: add an `envelopeState` (phase, level, sample counter) to `biquadFilter` — mirroring the exponential-multiplier ADSR logic already used in `effects.go`. Advance state per sample inside `biquadFilter.Stream()`.

4. **Coefficient recomputation strategy** (preferred — avoids per-sample cost):
   - Track `lastCutoff float64` on `biquadFilter`.
   - After advancing the envelope, compute `effectiveCutoff = baseCutoff + envelope.Depth * envelopeLevel * maxCutoffHz`.
   - Call `calcCoeffs` only when `|effectiveCutoff - lastCutoff| > threshold` (e.g. 1 Hz). This amortises the trig cost across many samples at chiptune tick rates.

5. **Envelope triggering**: expose `Trigger(note)` / `Release()` on `biquadFilter` (or on a thin `FilterEnvelopeStreamer` wrapper) so `Synth.Streamer()` can signal note-on/off, consistent with how amplitude envelopes are triggered today.

## Impact

- **Invasiveness**: moderate. The hot path (`biquadFilter.Stream`) gains a branch and occasional `calcCoeffs` call; the struct gains a few fields.
- **Files touched**: `audio/filter.go` (primary), `audio/synth.go` (add field + pass-through), `audio/effects.go` (reference only — no changes, envelope logic is copied/shared).
- **Compatibility**: fully additive. `FilterEnvelope` defaults to zero `Depth`, which produces no offset and leaves `calcCoeffs` called once as today — existing callers are unaffected.
