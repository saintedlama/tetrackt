# Chiptune-6: Note-On / Note-Off Gate Model

**Status:** Planned

**Priority:** Medium

## Problem

`Synth.NewPatch` requires a fixed `noteSamples` duration upfront — sustain duration is derived by subtracting attack+decay+release from the total note length. There is no "hold until note-off" concept.

## Why It Matters

In a real tracker, note length is decoupled from note-off: a pattern cell sets when note-on fires; the envelope sustains indefinitely until the next note-on or an explicit cut command. The current model bakes duration in at `Patch` creation time, which prevents proper tracker integration and live play.

## Required

- Gate-based envelope: `NoteOn()` / `NoteOff()` transitions
- `Patch` that sustains in the sustain stage until `NoteOff()` is called

## Current Implementation Context

The existing envelope system uses these types/names (must be consistent with):

- `Stages` (`StageOff`, `StageAttack`, `StageDecay`, `StageSustain`, `StageRelease`) in `effects.go`
- `envelopeGenerator` — exponential ramps via `calculateMultiplier` (log-space multiplier per sample, not linear)
- `Envelope` struct with `Attack, Decay, Sustain, Release time.Duration` fields
- `Patch` — live synthesis instance created by `Synth.NewPatch(sampleRate, frequency, noteSamples)`; `remaining` counter limits total samples streamed
- `Patch.Reset()` — restarts envelopes and LFOs; sets `remaining = noteSamples`

The gated envelope generator must use the same exponential ramp approach (i.e. `calculateMultiplier`) so the sound character is consistent.

## Implementation Plan

### 1. `gatedEnvelopeGenerator` type (`effects.go`)

Add a new unexported type alongside the existing `envelopeGenerator`. Uses absolute sample counts (A/D/R) and a sustain level; no `sustainSamples` — sustain is open-ended.

```go
type gateState int

const (
    gateIdle gateState = iota
    gateAttack
    gateDecay
    gateSustain
    gateRelease
    gateDone
)

type gatedEnvelopeGenerator struct {
    Streamer beep.Streamer

    attackSamples  int
    decaySamples   int
    releaseSamples int
    sustainLevel   float64

    mu           sync.Mutex
    state        gateState
    pos          int     // samples elapsed in current stage
    currentLevel float64 // running amplitude (exponential)
    multiplier   float64 // per-sample multiplier for current stage
}
```

`newGatedEnvelopeGenerator(streamer beep.Streamer, sampleRate beep.SampleRate, env Envelope) *gatedEnvelopeGenerator`
converts `env.Attack`, `env.Decay`, `env.Release` durations to sample counts and `env.Sustain` to a level; starts in `gateIdle`.

### 2. State machine transitions

| Event / condition         | From                     | To                                                             |
| ------------------------- | ------------------------ | -------------------------------------------------------------- |
| `NoteOn()` called         | Idle                     | Attack (set `currentLevel=0.0001`, compute attack multiplier)  |
| Attack samples exhausted  | Attack                   | Decay (set `currentLevel=1.0`, compute decay multiplier)       |
| Decay samples exhausted   | Decay                    | Sustain (`currentLevel=sustainLevel`, `multiplier=1.0`)        |
| `NoteOff()` called        | Attack / Decay / Sustain | Release (snapshot current level, compute release multiplier)   |
| `NoteOff()` called        | Idle                     | Done (no sound was playing)                                    |
| Release samples exhausted | Release                  | Done                                                           |

`Stream(samples [][2]float64) (int, bool)` delegates to `Streamer`, then multiplies each sample by the current envelope level. Returns `(n, false)` when `gateDone`.

### 3. Thread safety

`NoteOff()` acquires `mu` before transitioning state. `Stream()` also acquires `mu` per-sample tick for the state read/write. A plain `sync.Mutex` is sufficient; contention is negligible at audio-thread cadence.

### 4. `NewGatedPatch` on `Synth` (`synth.go`)

```go
// NewGatedPatch builds a synthesis pipeline that sustains until NoteOff is called.
// Call patch.NoteOn() once to start the envelope; call patch.NoteOff() to
// begin the release phase. The patch streams until the release completes.
func (s *Synth) NewGatedPatch(sampleRate beep.SampleRate, frequency float64) *Patch
```

Internally builds the same oscillator/LFO pipeline as `NewPatch` but replaces both `envelopeGenerator` wrappers with `gatedEnvelopeGenerator`. Sets `Patch.remaining = math.MaxInt` so the `remaining` counter never limits the stream; the gated envelopes signal end-of-stream via `(n, false)` instead.

`Patch` gains two additional fields and two new methods:

```go
type Patch struct {
    // ... existing fields ...
    gatedEnv1 *gatedEnvelopeGenerator // nil for fixed-duration patches
    gatedEnv2 *gatedEnvelopeGenerator
}

// NoteOn starts the envelope attack. No-op for fixed-duration patches.
func (p *Patch) NoteOn()

// NoteOff triggers the release phase. No-op for fixed-duration patches.
func (p *Patch) NoteOff()
```

`Synth.NewPatch(...)` is **unchanged** — `gatedEnv1`/`gatedEnv2` remain nil.

### 5. Player integration (`player/player.go`)

`playRowNotes` currently replaces `activePatches` every row. For gate support it must also fire `NoteOff()` on the previous patch for each track before starting the new one:

```go
// Before creating the new patch for trackIdx:
if p.activePatches[trackIdx] != nil {
    p.activePatches[trackIdx].NoteOff()
    // old patch continues streaming its release through the speaker's mixer
}
patch := track.Synth.NewGatedPatch(sampleRate, targetFrequency)
patch.NoteOn()
```

`activePatches` is kept from row to row (already the case) so the reference is available for the next row's `NoteOff()` call. The speaker's built-in mixing drains the releasing patch naturally — no extra bookkeeping needed.

### 6. Caller workflow (unit test / preview)

```go
patch := synth.NewGatedPatch(sampleRate, 440.0)
patch.NoteOn()
speaker.Play(patch)
// ... later, on note-off event:
patch.NoteOff()
// patch streams release samples and then returns (n, false) — speaker drains it
```

## Impact

- **Files touched:** `audio/effects.go` (new `gatedEnvelopeGenerator` type + methods), `audio/synth.go` (`NewGatedPatch`, `Patch.NoteOn`/`NoteOff`, two new `Patch` fields), `player/player.go` (call `NoteOff` on previous patch before creating new one).
- **Additive / backward compatible:** fully additive — `envelopeGenerator`, `NewPatch`, and all existing callers are untouched.
- **Invasiveness:** low. No changes to persistence or UI layers. The new type lives entirely within the `audio` package; the player change is a handful of lines.
- **Risk area:** goroutine safety between the audio callback (`Stream`) and the sequencer/UI goroutine calling `NoteOff`. The `sync.Mutex` in `gatedEnvelopeGenerator` addresses this; no lock-free tricks required given the low contention profile.
