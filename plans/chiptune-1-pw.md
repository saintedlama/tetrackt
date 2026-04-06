# Chiptune-1: Pulse Width Control

> Status: **Planned**

> **Priority:** High

## Problem

The square wave oscillator is hardcoded at 50% duty cycle.

## Why It Matters

Variable pulse width is arguably _the_ defining chiptune timbre. PWM (continuous duty-cycle modulation) produces the classic wobbling/sweeping tone used across SID, NES, AY-3 and Amiga.

## Required

- `Oscillator.PulseWidth float64` parameter (0.01–0.99)
- PWM via LFO modulation of pulse width over time
