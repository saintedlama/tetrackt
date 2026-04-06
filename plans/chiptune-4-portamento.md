# Chiptune-4: Portamento / Pitch Glide

> **Priority:** Medium

## Problem

No pitch glide between consecutive notes.

## Why It Matters

Essential for bass slides, lead smoothness and SID-style glides.

## Required

- `Portamento float64` – time in seconds (or ticks) to slide from previous note frequency to current
- Implemented as a per-sample frequency interpolation in the oscillator
