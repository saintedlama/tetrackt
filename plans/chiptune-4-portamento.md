# Chiptune-4: Portamento / Pitch Glide

**Status:** Planned

**Priority:** Medium

## Problem

No pitch glide between consecutive notes.

## Why It Matters

Essential for bass slides, lead smoothness and SID-style glides.

## Current Architecture (as of Patch refactor)

- `Synth` is a pure data struct (instrument definition). No `Portamento` field.
- `Patch` is the live synthesis instance created by `synth.NewPatch(sampleRate, frequency, noteSamples)`.
- `oscillatorGenerator` has `SetFrequency(hz)`, which applies the stored `detuneMultiplier`.
- `Patch.SetFrequency(hz)` delegates to both oscillators and updates `modOsc.baseFreq` for active pitch LFOs.
- `Player` owns `activePatches []audio.Streamer` — one per track, replaced each row.

## Approach: Tick-stepped glide via `Patch.StartPortamento` / `Patch.TickPortamento`

Portamento is implemented as one frequency step per player sub-tick — the classic tracker
style (e.g. ProTracker effect `3xx`). The interpolation math lives in the `audio` package;
`Player` only calls `TickPortamento()` each sub-tick without knowing anything about the
interpolation formula.

`NewPatch` signature is **unchanged**. No oscillator-internal changes needed.

## Required

- `Portamento float64` on `Synth` — glide duration in seconds; 0 = snap
- `StartPortamento` / `TickPortamento` methods on `Patch`
- Previous-note frequency tracking per track in `Player`

## Implementation Plan

### 1. Add `Portamento float64` to `Synth` (`audio/synth.go`)

```go
type Synth struct {
    Oscillator1 Oscillator
    // ...
    Portamento float64 // glide duration in seconds; 0 = snap
}
```

`Synth` fields are exported — no constructor change needed.

### 2. Add portamento state to `Patch` and expose two methods (`audio/synth.go`)

```go
type Patch struct {
    // ...existing fields...
    glideFrom  float64
    glideTo    float64
    glideStep  int // current tick
    glideSteps int // total ticks (0 = no glide)
}
```

```go
// StartPortamento begins a stepped frequency glide from `from` to `to` Hz
// over `ticks` player sub-ticks. Calling it with ticks <= 0 or from <= 0
// is a no-op and leaves the patch at its current frequency.
func (p *Patch) StartPortamento(from, to float64, ticks int) {
    if ticks <= 0 || from <= 0 {
        return
    }
    p.glideFrom  = from
    p.glideTo    = to
    p.glideStep  = 0
    p.glideSteps = ticks
    p.SetFrequency(from)
}

// TickPortamento advances the glide by one step and resets the oscillator
// frequency. Call once per player sub-tick for as long as the patch is active.
// Does nothing when no glide is in progress.
func (p *Patch) TickPortamento() {
    if p.glideSteps == 0 || p.glideStep >= p.glideSteps {
        return
    }
    p.glideStep++
    t := float64(p.glideStep) / float64(p.glideSteps)
    // Exponential interpolation = perceptually linear (equal semitones per tick)
    freq := p.glideFrom * math.Pow(p.glideTo/p.glideFrom, t)
    p.SetFrequency(freq)
}
```

`Reset()` also zeroes `glideStep` and `glideSteps` so a gate restart cancels any in-progress glide.

### 3. Previous-note tracking in `Player` (`player/player.go`)

`Player` already owns `activePatches []audio.Streamer` per track. Add a parallel slice:

```go
type Player struct {
    // ...
    prevFreqs []float64 // last triggered frequency per track; 0 if no prior note
}
```

In `playRowNotes`, after creating the patch:

```go
targetFreq := trackRow.Note.Frequency()
patch := track.Synth.NewPatch(sampleRate, targetFreq, noteSamples)
if track.Synth.Portamento > 0 && p.prevFreqs[trackIdx] > 0 {
    ticks := int(math.Round(track.Synth.Portamento * float64(sampleRate) / float64(noteSamples)))
    patch.StartPortamento(p.prevFreqs[trackIdx], targetFreq, ticks)
}
p.prevFreqs[trackIdx] = targetFreq
```

In `Tick`, after `playRowNotes`, call `TickPortamento` on every active patch each sub-tick:

```go
for _, patch := range p.activePatches {
    if pp, ok := patch.(*audio.Patch); ok {
        pp.TickPortamento()
    }
}
```

In `Player.Reset()`, clear `prevFreqs` alongside `activePatches`.

`StartPreview` does not call `StartPortamento` — no glide for one-shot previews.

### 4. Expose `*Patch` from `activePatches`

`activePatches` is currently `[]audio.Streamer`. Switch it to `[]*audio.Patch` so the type
assertion in `Tick` is unnecessary:

```go
type Player struct {
    activePatches []*audio.Patch
    prevFreqs     []float64
    previewPatch  audio.Streamer
}
```

### 5. Persistence (`persistence/song.go`)

Add `Portamento float64 \`yaml:"portamento,omitempty"\``back to`SavedTrack`, and wire it in
`TracksToSong`/`SongToTracks`. Old saves without the field default to 0 (snap), which is correct.

---

## Impact

| Dimension                  | Assessment                                                                                                                                                                            |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Invasiveness**           | Low. `oscillatorGenerator` is untouched. Glide state lives entirely in `Patch`; it is a no-op when `glideSteps == 0`.                                                                 |
| **Files touched**          | `audio/synth.go` (`Synth.Portamento` field + `Patch` glide fields + two methods), `persistence/song.go` (new field), `player/player.go` (prev-freq tracking + `TickPortamento` call). |
| **Backward compatibility** | Fully additive. `NewPatch` signature is unchanged; `Portamento` defaults to zero. No existing call sites break.                                                                       |
| **Risk**                   | Exponential interpolation requires non-zero `glideFrom`; `StartPortamento` guards against this and is a no-op when `from <= 0`.                                                       |
