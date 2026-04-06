# Chiptune-8: Periodic / LFSR Noise

**Status:** Planned

**Priority:** Low

## Problem

Noise generator produces white noise (`rand.Float64()`). Classic chips use a Linear Feedback Shift Register (LFSR) which produces pitched, periodic noise.

## Why It Matters

NES, Game Boy and SID all use LFSR noise. The characteristic buzzy snare and lo-fi percussion come from this, not white noise.

## Required

- `OscillatorType = NoisePeriodic` backed by a 15-bit or 7-bit LFSR
- `Oscillator.NoisePeriod int` to control pitch of the periodic noise

## Implementation Plan

1. **Add `NoisePeriodic` constant** in `oscillator.go` alongside the existing `OscillatorType` constants.

2. **Extend `Oscillator` struct** with `NoisePeriod int` (samples per LFSR clock; higher value = lower pitch; 0 clocks every sample).

3. **Extend `oscillatorGenerator` struct** with:
   - `lfsrState uint16` — current 15-bit shift register value; initialise to `0x7FFF`.
   - `lfsrCounter int` — counts down samples until the next LFSR clock.

4. **LFSR logic** (NES-style 15-bit Galois LFSR):
   ```
   feedback = (lfsrState & 1) ^ ((lfsrState >> 1) & 1)
   lfsrState = (lfsrState >> 1) | (feedback << 14)
   ```
   Output mapped from bit 0: `+1.0` if `lfsrState & 1 == 0`, `-1.0` otherwise.
   Clock the LFSR every `NoisePeriod` samples (decrement `lfsrCounter`; reload from `NoisePeriod` when it reaches 0). The output level is held constant between clocks.

5. **Seed** `lfsrState = 0x7FFF` on generator construction/reset to guarantee a non-zero initial state and a deterministic sequence.

## Impact

- **Files touched:** `audio/oscillator.go` only (struct fields, constant, and sample-generation switch case).
- **Invasiveness:** Low. Changes are purely additive — new constant, two new fields on internal structs, and a new `case NoisePeriodic` block. The existing `Noise` (white noise) path is untouched.
- **Backward compatibility:** Fully backward compatible. Callers that never set `NoisePeriodic` see no behaviour change. `NoisePeriod == 0` can be treated as "clock every sample" (maximum rate / near-white), providing a safe default.
