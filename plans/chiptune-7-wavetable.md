# Chiptune-7: Wavetable / Custom Waveform Oscillator

> **Priority:** Medium

## Problem

No support for user-defined single-cycle waveforms.

## Why It Matters

Amiga trackers play arbitrary PCM loops as oscillators. Custom wavetables unlock unique timbres beyond the standard waveforms and enable sample-based chiptune sounds.

## Required

- `OscillatorType = Wavetable`
- `Oscillator.Wavetable []float64` – one cycle of samples
- Linear interpolation on fractional phase for smooth playback
