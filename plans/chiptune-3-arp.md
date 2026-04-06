# Chiptune-3: Arpeggio Effect

**Status:** Planned

**Priority:** High

## Problem

No arpeggio. In trackers, arpeggio cycles through 2–3 semitone offsets per tick to simulate chords with a single oscillator.

## Why It Matters

Arpeggio is ubiquitous in chiptune — used for chords, bass riffs and leads alike.

## Required

- `ArpeggioEffect` with semitone offsets (e.g. `[0, 4, 7]`) and speed (ticks per step)
- Integrated into the tracker tick pipeline, not the audio engine directly

## Implementation Plan

### 1. Fix `Note.Transpose` — semitone-accurate transposition (prerequisite)

`audio/notes.go` — current `Transpose(delta int)` shifts by whole octaves (`Octave + delta`). Replace with chromatic semitone arithmetic:

```go
// Ordered chromatic scale for index lookup
var chromaticScale = []Base{BaseC, BaseCs, BaseD, BaseDs, BaseE, BaseF, BaseFs, BaseG, BaseGs, BaseA, BaseAs, BaseB}

func (note Note) Transpose(semitones int) (Note, bool) {
    idx := slices.Index(chromaticScale, note.Base) // -1 if BaseOff
    if idx < 0 {
        return note, false
    }
    total := int(note.Octave)*12 + idx + semitones
    if total < 0 || total > int(Octave8)*12+11 {
        return note, false
    }
    return Note{Base: chromaticScale[total%12], Octave: Octave(total / 12)}, true
}
```

This is a **breaking change** to the existing signature only in semantics — callers that pass `1` expecting "+1 octave" must be updated to pass `12`.

---

### 2. Define `ArpeggioEffect`

New file `audio/effects.go` (or a tracker-layer file, e.g. `tracker/arpeggio.go`):

```go
type ArpeggioEffect struct {
    Offsets      []int // semitone offsets cycled each step, e.g. [0, 4, 7]
    TicksPerStep int   // ticks before advancing to the next offset
}
```

`Offsets[0]` should be `0` (the root note) by convention. An empty `Offsets` or `TicksPerStep == 0` means the effect is inactive.

---

### 3. Oscillator frequency mutation — `SetFrequency`

`audio/oscilator.go` — add a method to `oscillatorGenerator` so the playback engine can retune the oscillator between ticks without reconstructing the entire streamer graph:

```go
func (g *oscillatorGenerator) SetFrequency(hz float64) {
    g.frequency = hz
}
```

Expose it via an interface or by returning `*oscillatorGenerator` from `NewOscillator` instead of `beep.Streamer`. The latter is the minimal change: change the return type of `NewOscillator` to `*oscillatorGenerator` (it already satisfies `beep.Streamer`) and keep all existing call sites working since `*oscillatorGenerator` is assignable to `beep.Streamer`.

`synth.go` — `Synth.Streamer` currently returns an opaque `beep.Streamer`. To expose `SetFrequency`, introduce a narrow interface or return a concrete handle:

```go
type FrequencyStreamer interface {
    beep.Streamer
    SetFrequency(hz float64)
}
```

`Synth.Streamer` can return two `FrequencyStreamer` handles (one per oscillator) alongside the composed `beep.Streamer`, or `Synth` can expose a `SetFrequency(hz float64)` that updates both oscillators internally.

The second option is simpler and keeps the change contained to `synth.go`:

```go
func (s *Synth) SetFrequency(hz float64) {
    s.osc1Handle.SetFrequency(hz)
    s.osc2Handle.SetFrequency(hz)
}
```

Store `osc1Handle` and `osc2Handle` as `*oscillatorGenerator` fields on an active-voice struct returned by `Synth.Streamer`.

---

### 4. Arpeggio evaluation in the playback loop

Arpeggio is **not** computed in the audio engine. It is driven by the tracker's tick clock (e.g. every N audio frames at the configured BPM/speed).

Integration point — wherever the playback engine processes a tick (likely `main.go` or a future `tracker/player.go`):

```
on each tick:
    step = (tickCount / arpeggio.TicksPerStep) % len(arpeggio.Offsets)
    transposedNote, ok = currentNote.Transpose(arpeggio.Offsets[step])
    if ok:
        activeSynth.SetFrequency(transposedNote.Frequency())
```

The tick counter resets when a new note is triggered. Arpeggio is evaluated per-channel independently.

---

### 5. Integration checklist

| Step | File(s) | Action |
|------|---------|--------|
| 1 | `audio/notes.go` | Rewrite `Transpose` for semitone arithmetic |
| 2 | `audio/notes.go` or `tracker/` | Add `chromaticScale` index helper |
| 3 | `audio/oscilator.go` | Add `SetFrequency` method; change `NewOscillator` return to `*oscillatorGenerator` |
| 4 | `audio/synth.go` | Store oscillator handles; expose `SetFrequency` on active voice |
| 5 | `tracker/` (new or existing) | Define `ArpeggioEffect`; wire into tick loop |
| 6 | Existing callers of `Transpose` | Update `delta` from octave units to semitones (`×12`) |

## Impact

**Invasiveness:** Moderate. The oscillator and synth changes are additive (new method, widened return type); the `Transpose` fix is a semantic breaking change limited to a single function signature.

**Files touched:**
- `audio/notes.go` — `Transpose` rewrite + chromatic scale table
- `audio/oscilator.go` — `SetFrequency` method, return type adjustment
- `audio/synth.go` — active-voice struct holding oscillator handles, `SetFrequency` delegation
- `tracker/` (new file) — `ArpeggioEffect` definition and tick-loop integration
- Any existing callers of `Note.Transpose` that pass octave-unit deltas

**Backward compatibility:**
- `NewOscillator` return-type change (`*oscillatorGenerator` → still satisfies `beep.Streamer`) is **source-compatible** at all current call sites.
- `Synth.Streamer` signature is **unchanged**; `SetFrequency` is added on a new active-voice handle.
- `Note.Transpose` is a **breaking semantic change** — all callers must be audited. Currently only used in a narrow surface area, so the blast radius is small.
