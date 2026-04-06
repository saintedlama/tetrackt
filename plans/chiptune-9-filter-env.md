# Chiptune-9: Filter Cutoff Envelope

> **Priority:** Low

## Problem

Filter cutoff is static per note.

## Why It Matters

Filter sweeps (e.g. opening LP filter on a bass hit) are a staple effect.

## Required

- `FilterEnvelope` with its own ADSR and depth
- Applied as an additive offset to `Filter.Cutoff` per sample
