# Chiptune-10: Tracker Effect Commands

> **Priority:** Low

## Problem

No tracker-side effect column processing beyond what the synth natively produces.

## Why It Matters

Classic trackers express vibrato, portamento, volume slide, note cut, note delay and arpeggio via effect codes evaluated per tick. These bridge the gap between the static synth and live, expressive playback.

## Required

- Effect type enum + value: `Vibrato(speed, depth)`, `VolumeSlide`, `NoteCut(tick)`, `NoteDelay(tick)`, `Arpeggio(semi1, semi2)`
- Tick-rate processor in the playback engine that applies effects each tick
