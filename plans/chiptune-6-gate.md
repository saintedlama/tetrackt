# Chiptune-6: Note-On / Note-Off Gate Model

> **Priority:** Medium

## Problem

`Synth.Streamer` takes a fixed `time.Duration`. ADSR sustain duration is derived by subtracting attack+decay+release from the total note length — there is no real "hold until note-off" concept.

## Why It Matters

In a real tracker, the note length is decoupled from note-off: a pattern cell sets when note-on fires; the envelope sustains indefinitely until the next note-on or an explicit cut command. The current model bakes duration in at note creation time, which prevents proper tracker integration and live play.

## Required

- Gate-based envelope: `NoteOn()` / `NoteOff()` transitions
- Streamer that sustains in the sustain stage until `NoteOff()` is called
