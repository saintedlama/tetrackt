# Chiptune-3: Arpeggio Effect

> **Priority:** High

## Problem

No arpeggio. In trackers, arpeggio cycles through 2–3 semitone offsets per tick to simulate chords with a single oscillator.

## Why It Matters

Arpeggio is ubiquitous in chiptune — used for chords, bass riffs and leads alike.

## Required

- `ArpeggioEffect` with semitone offsets (e.g. `[0, 4, 7]`) and speed (ticks per step)
- Integrated into the tracker tick pipeline, not the audio engine directly
