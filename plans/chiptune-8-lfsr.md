# Chiptune-8: Periodic / LFSR Noise

> **Priority:** Low

## Problem

Noise generator produces white noise (`rand.Float64()`). Classic chips use a Linear Feedback Shift Register (LFSR) which produces pitched, periodic noise.

## Why It Matters

NES, Game Boy and SID all use LFSR noise. The characteristic buzzy snare and lo-fi percussion come from this, not white noise.

## Required

- `OscillatorType = NoisePeriodic` backed by a 15-bit or 7-bit LFSR
- `Oscillator.NoisePeriod int` to control pitch of the periodic noise
