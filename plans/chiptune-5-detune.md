# Chiptune-5: Fine Tuning / Detune

> **Priority:** Medium

## Problem

`Transpose` only shifts whole octaves. There is no semitone or cent offset.

## Why It Matters

Detuning two oscillators against each other creates width and beating effects central to chiptune leads and pads.

## Required

- `Oscillator.Detune float64` in cents (±100 = ±1 semitone)
- Fix `Transpose` to accept semitone delta, not just octave
