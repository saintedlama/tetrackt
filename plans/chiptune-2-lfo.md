# Chiptune-2: LFO (Low Frequency Oscillator)

> **Priority:** High

## Problem

No modulation sources exist. Every parameter is static for the lifetime of a note.

## Why It Matters

Vibrato, tremolo, auto-wah, PWM sweep — all classic chiptune articulations — require a periodic modulation signal routed to a destination.

## Required

- `LFO` struct: waveform, rate (Hz), depth, delay (onset time)
- Modulation destinations: pitch, volume, filter cutoff, pulse width
- At minimum one LFO per voice; ideally one LFO per destination
