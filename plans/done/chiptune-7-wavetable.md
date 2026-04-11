# Chiptune-7: Wavetable / Custom Waveform Oscillator

**Status:** Done

**Priority:** Medium

## Problem

No support for user-defined single-cycle waveforms.

## Why It Matters

Custom single-cycle wavetables unlock unique timbres beyond the built-in waveforms. A wavetable is a short `[]float64` slice (64–2048 samples) representing one complete waveform period; the oscillator loops through it at whatever pitch is needed by adjusting `phaseIncrement`. A 2048-sample table is only 16 KB — defined once on `Oscillator`, not per note or per track.

This is distinct from Amiga-style full-length sample playback (which plays entire recorded PCM files with loop points). That is a separate, heavier feature not covered here.

## Wavetable Data Sources

Three options, in order of complexity:

### Option A: Built-in computed tables (recommended starting point)

Generate mathematically exact waveforms at package init time — no files, no I/O:

```go
// 256-sample band-limited additive sawtooth
table := make([]float64, 256)
for i := range table {
    t := float64(i) / 256
    for h := 1; h <= 16; h++ {
        table[i] += math.Sin(2*math.Pi*float64(h)*t) / float64(h)
    }
}
```

Ship a small set of named presets: `WavetableSoftSaw`, `WavetableSoftSquare`, `WavetableOrgan`, etc. as package-level `var` values in `audio/wavetables.go`.

### Option B: AKWF free wavetable packs

[Adventure Kid Waveforms (AKWF)](https://www.adventurekid.se/akrt/waveforms/) provides 4,000+ free single-cycle WAV files (public domain), each 2048 samples. Load one with a small helper that reads the PCM into `[]float64`:

```go
func LoadWavetableWAV(path string) ([]float64, error)
```

This adds file I/O and a WAV decoder dependency but gives users access to a huge library of timbres.

### Option C: User-drawn / exported from a DAW

Record exactly one period of any soft synth at a known pitch, crop to one cycle, export as WAV. Same loader as Option B.

### Recommended approach for Tetrackt

Start with **Option A only**: ship 4–8 computed built-in tables as named presets. Add Option B loader later if needed. No new dependencies required initially.

## Required

- `OscillatorType = Wavetable`
- `Oscillator.Wavetable []float64` – one cycle of samples
- Linear interpolation on fractional phase for smooth playback

## Current Implementation Context

The synthesis pipeline is constructed in `Synth.NewPatch(sampleRate, frequency, noteSamples) *Patch`:

1. **Oscillator layer**: Each oscillator is created by:
   ```go
   NewOscillator(oscType OscillatorType, frequency float64, sampleRate beep.SampleRate, phase, pulseWidth, detuneCents float64) *oscillatorGenerator
   ```
   Then wrapped by `newModulatedOscillatorStreamer(osc, baseFreq, baseDuty, pitchLFO, pwmLFO, detuneLFO)` for LFO modulation.

2. **`oscillatorGenerator` fields**:
   - `oscillatorType OscillatorType`
   - `frequency float64` — raw Hz value (transparent; never has detune baked in)
   - `detuneMultiplier float64` — applied only at stream time
   - `phase float64` — current phase `[0, 1)`
   - `pulseWidth float64`

3. **Waveform generation** — `oscillatorGenerator.Stream()` contains a switch over `oscillatorType` covering : `Sine`, `Square`, `Triangle`, `Sawtooth`, `SawtoothReverse`, `Noise`, `Silent`. Phase is advanced as:
   ```go
   phaseIncrement := g.frequency * g.detuneMultiplier / float64(g.sampleRate)
   g.phase += phaseIncrement
   if g.phase >= 1.0 { g.phase -= 1.0 }
   ```
   The `Wavetable` type does **not** exist yet.

4. **`Oscillator` struct** (parameter definition in `synth.go`):
   ```go
   type Oscillator struct {
       Type       OscillatorType
       Phase      float64
       PulseWidth float64
       Detune     float64 // fine tuning in cents
   }
   ```
   No `Wavetable` field yet.

## Implementation Plan

1. **Add `Wavetable` constant** in `audio/oscilator.go`:

   ```go
   const (
       // ... existing constants ...
       Wavetable OscillatorType = "wavetable"
   )
   ```

2. **Extend `Oscillator` struct** in `audio/synth.go` with an optional slice field:

   ```go
   type Oscillator struct {
       Type       OscillatorType
       Phase      float64
       PulseWidth float64
       Detune     float64
       Wavetable  []float64 // nil for all non-wavetable types
   }
   ```

3. **Propagate wavetable into generator** — add `wavetable []float64` to `oscillatorGenerator`; pass `Oscillator.Wavetable` through `NewOscillator` and set it on the struct at construction.

4. **Handle `Wavetable` in `Stream()` switch** in `audio/oscilator.go`:

   ```go
   case Wavetable:
       if len(g.wavetable) > 0 {
           n := float64(len(g.wavetable))
           pos := g.phase * n
           i := int(pos) % len(g.wavetable)
           j := (i + 1) % len(g.wavetable)
           frac := pos - math.Floor(pos)
           sample = g.wavetable[i]*(1-frac) + g.wavetable[j]*frac
       } // empty wavetable → sample stays 0 (silent)
   ```

   Phase advances as normal (existing `phaseIncrement` calculation is unchanged).

5. **Validate in `NewOscillator`**: when `oscType == Wavetable` and `len(wavetable) == 0`, fall through silently or return a `Silent` generator — do not panic.

## Impact

- **Files touched:** `audio/oscilator.go` (new constant, `oscillatorGenerator` field, new case in switch) and `audio/synth.go` (`Oscillator` struct gets `Wavetable []float64`).
- **Invasiveness:** Minimal. One new constant, one new struct field (zero-value safe), one new `oscillatorGenerator` field, one additional `case`. No existing code paths are altered.
- **Backward compatibility:** Fully additive. All current `OscillatorType` values (`Sine`, `Square`, `Triangle`, `Sawtooth`, `SawtoothReverse`, `Noise`, `Silent`) are unaffected; the new `Wavetable` field on `Oscillator` is `nil` by default and ignored by every existing case.
