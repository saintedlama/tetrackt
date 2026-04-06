# Chiptune-7: Wavetable / Custom Waveform Oscillator

**Status:** Planned

**Priority:** Medium

## Problem

No support for user-defined single-cycle waveforms.

## Why It Matters

Amiga trackers play arbitrary PCM loops as oscillators. Custom wavetables unlock unique timbres beyond the standard waveforms and enable sample-based chiptune sounds.

## Required

- `OscillatorType = Wavetable`
- `Oscillator.Wavetable []float64` – one cycle of samples
- Linear interpolation on fractional phase for smooth playback

## Implementation Plan

1. **Add `Wavetable` constant** in `oscillator.go`:
   ```go
   Wavetable OscillatorType = "wavetable"
   ```

2. **Extend `Oscillator` struct** with an optional slice field:
   ```go
   Wavetable []float64 // nil for all non-wavetable types
   ```

3. **Propagate wavetable into generator** — add `wavetable []float64` to `oscillatorGenerator`, populated from `Oscillator.Wavetable` when constructing the streamer in `NewOscillator`.

4. **Handle `Wavetable` in `Stream()` switch**:
   ```go
   case Wavetable:
       n := float64(len(g.wavetable))
       pos := g.phase * n
       i := int(pos) % len(g.wavetable)
       j := (i + 1) % len(g.wavetable)
       frac := pos - math.Floor(pos)
       sample = g.wavetable[i]*(1-frac) + g.wavetable[j]*frac
   ```
   Phase advances as normal (`g.phase += frequency / sampleRate`, wrapped to `[0, 1)`).

5. **Validate in `NewOscillator`**: when `oscType == Wavetable`, return an error (or panic) if `Oscillator.Wavetable` is nil or empty before constructing the generator.

## Impact

- **Files touched:** `audio/oscillator.go` only.
- **Invasiveness:** Minimal. One new constant, one new struct field (zero-value safe for existing types), one new `oscillatorGenerator` field, and one additional `case` in the existing switch. No existing code paths are altered.
- **Backward compatibility:** Fully additive. All current `OscillatorType` values (`square`, `triangle`, `sawtooth`, `noise`) are unaffected; the new `Wavetable` field on `Oscillator` is nil by default and ignored by every existing case.
