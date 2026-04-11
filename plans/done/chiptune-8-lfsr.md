# Chiptune-8: Periodic / LFSR Noise

**Status:** Done

**Priority:** Low

## Problem

Noise generator produces white noise (`rand.Float64()`). Classic chips use a Linear Feedback Shift Register (LFSR) which produces pitched, periodic noise.

## Why It Matters

NES, Game Boy and SID all use LFSR noise. The characteristic buzzy snare and lo-fi percussion come from this, not white noise.

## Required

- `OscillatorType = NoisePeriodic` backed by a 15-bit or 7-bit LFSR
- `Oscillator.NoisePeriod int` to control pitch of the periodic noise

## Current Implementation Context

- **`Oscillator` struct** (in `audio/synth.go`):
  ```go
  type Oscillator struct {
      Type       OscillatorType
      Phase      float64
      PulseWidth float64
      Detune     float64 // fine tuning in cents
  }
  ```
  No `NoisePeriod` field exists yet.

- **`OscillatorType` constants** (in `audio/oscilator.go`):
  `Sine`, `Square`, `Triangle`, `Sawtooth`, `SawtoothReverse`, `Noise`, `Silent`.
  No `NoisePeriodic` constant yet.

- **`oscillatorGenerator` struct**: holds `frequency float64` (raw Hz), `detuneMultiplier float64` (applied at stream time in `phaseIncrement = frequency * detuneMultiplier / sampleRate`). The existing `Noise` case does `rand.Float64()*2 - 1` — no LFSR state.

- **`Synth.NewPatch(sampleRate, frequency, noteSamples) *Patch`** builds the pipeline; oscillators are constructed via `NewOscillator(oscType, frequency, sampleRate, phase, pulseWidth, detuneCents)` and passed to `newModulatedOscillatorStreamer`.

## Implementation Plan

1. **Add `NoisePeriodic` constant** in `audio/oscilator.go` alongside the existing `OscillatorType` constants.

2. **Extend `Oscillator` struct** in `audio/synth.go` with `NoisePeriod int` (samples per LFSR clock; 0 = clock every sample = near-white).

3. **Pass `NoisePeriod` to `NewOscillator`** — add a parameter and store on `oscillatorGenerator`:
   ```go
   type oscillatorGenerator struct {
       // ... existing fields ...
       noisePeriod  int    // samples per LFSR clock
       lfsrState    uint16 // 15-bit shift register; init to 0x7FFF
       lfsrCounter  int    // counts down to next clock
   }
   ```

4. **LFSR logic** (NES-style 15-bit Galois LFSR) — new `case NoisePeriodic` in `oscillatorGenerator.Stream()`:
   ```go
   case NoisePeriodic:
       if g.lfsrCounter <= 0 {
           feedback := (g.lfsrState & 1) ^ ((g.lfsrState >> 1) & 1)
           g.lfsrState = (g.lfsrState >> 1) | (feedback << 14)
           g.lfsrCounter = g.noisePeriod
       }
       g.lfsrCounter--
       if (g.lfsrState & 1) == 0 {
           sample = 1.0
       } else {
           sample = -1.0
       }
   ```
   Output is held constant between clock events. Phase field is not used for LFSR noise.

5. **Seed** `lfsrState = 0x7FFF` on generator construction to guarantee a non-zero initial state and a deterministic sequence.

6. **Frequency-to-period mapping**: when `NoisePeriod == 0`, map oscillator `frequency` to clock rate: `noisePeriod = int(sampleRate / frequency)`. This makes `Oscillator.Detune` and `Patch.SetFrequency()` work naturally for pitched noise.

## Impact

- **Files touched:** `audio/oscilator.go` (new constant, new `oscillatorGenerator` fields, new case in switch) and `audio/synth.go` (`Oscillator` struct gets `NoisePeriod int`; `NewOscillator` call in `NewPatch` passes it).
- **Invasiveness:** Low. Changes are purely additive — new constant, two new fields on internal structs, and a new `case NoisePeriodic` block. The existing `Noise` (white noise) path is untouched.
- **Backward compatibility:** Fully backward compatible. Callers that never set `NoisePeriodic` see no behaviour change. `NoisePeriod == 0` with frequency-based clock gives a sensible pitch-controlled default.
