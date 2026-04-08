# Chiptune-6: Note-On / Note-Off Gate Model

**Status:** Planned

**Priority:** Medium

## Problem

`Synth.Streamer` requires a fixed duration upfront — envelope sustain duration is derived by subtracting attack+decay+release from the total note length. There is no "hold until note-off" concept.

## Why It Matters

In a real tracker, note length is decoupled from note-off: a pattern cell sets when note-on fires; the envelope sustains indefinitely until the next note-on or an explicit cut command. The current model bakes duration in at note creation time, which prevents proper tracker integration and live play.

## Required

- Gate-based envelope: `NoteOn()` / `NoteOff()` transitions
- Streamer that sustains in the sustain stage until `NoteOff()` is called

## Current Implementation Context

The existing envelope system uses these types/names (must be consistent with):

- `Stages` (`StageOff`, `StageAttack`, `StageDecay`, `StageSustain`, `StageRelease`) in `effects.go`
- `envelopeGenerator` — exponential ramps via `calculateMultiplier` (log-space multiplier per sample, not linear)
- `Envelope` struct with `Attack, Decay, Sustain, Release float64` (fractional proportions of total duration)
- `Synth.Streamer(sampleRate, frequencies []float64, tickCount int, continuous bool, d time.Duration)` — current signature
- `buildChain(sampleRate, frequency float64, sampleDuration int)` — takes sample count, not duration

`GatedEnvelope` must use the same exponential ramp approach as `envelopeGenerator` (i.e. use `calculateMultiplier`) so the sound character is consistent.

## Implementation Plan

### 1. `GatedEnvelope` type (`effects.go`)

Add a new type alongside the existing `envelopeGenerator`. Uses absolute sample counts (A/D/R) and a sustain level; no `sustainSamples` — sustain is open-ended.

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

type GatedEnvelope struct {
    attackSamples  int
    decaySamples   int
    releaseSamples int
    sustainLevel   float64

    mu           sync.Mutex
    state        gateState
    pos          int     // samples elapsed in current stage
    currentLevel float64 // running amplitude (exponential)
    multiplier   float64 // per-sample multiplier for current stage
    releaseLevel float64 // amplitude snapshot at NoteOff
}
```

`NewGatedEnvelope(sampleRate beep.SampleRate, a, d, r time.Duration, sustainLevel float64) *GatedEnvelope`
converts durations to sample counts; starts in `gateIdle`. Also pre-computes the attack and decay multipliers using `calculateMultiplier`.

### 2. State machine transitions

| Event / condition         | From                     | To                                                            |
| ------------------------- | ------------------------ | ------------------------------------------------------------- |
| `NoteOn()` called         | Idle                     | Attack (set `currentLevel=0.0001`, compute attack multiplier) |
| Attack samples exhausted  | Attack                   | Decay (set `currentLevel=1.0`, compute decay multiplier)      |
| Decay samples exhausted   | Decay                    | Sustain (`currentLevel=sustainLevel`, `multiplier=1.0`)       |
| `NoteOff()` called        | Attack / Decay / Sustain | Release (snapshot `releaseLevel`, compute release multiplier) |
| `NoteOff()` called        | Idle                     | Done (no sound was playing)                                   |
| Release samples exhausted | Release                  | Done                                                          |

`Next() (float64, bool)` advances `pos` and applies the exponential multiplier for the current stage. Returns `(0, false)` when `gateDone`, `(currentLevel, true)` otherwise.

### 3. Thread safety

`NoteOff()` acquires `mu` before transitioning state and snapshotting `releaseLevel`. `Next()` also acquires `mu` for the state read/write, keeping the critical section small (one sample tick). A plain `sync.Mutex` is sufficient; contention is negligible at audio-thread cadence.

### 4. `GatedStreamer` (`synth.go`)

```go
// GatedStreamer returns a Streamer that sustains until env.NoteOff() is called.
// The caller must call env.NoteOn() before streaming begins.
// The pipeline (oscillators, LFOs, filter) is built without a fixed duration —
// oscillator sources run indefinitely; the GatedEnvelope signals end-of-stream.
func (s *Synth) GatedStreamer(sampleRate beep.SampleRate, frequency float64, env *GatedEnvelope) beep.Streamer
```

Internally calls a variant of `buildChain` that does **not** wrap sources in `NewEnvelope` or `beep.Take` — instead the gated pipeline multiplies each sample by `env.Next()` and returns `(n, false)` once `env` reaches `gateDone`.

`Synth.Streamer(...)` is **unchanged**.

### 5. Caller workflow

```go
sr := beep.SampleRate(44100)
env := audio.NewGatedEnvelope(sr, 10*time.Millisecond, 20*time.Millisecond, 30*time.Millisecond, 0.7)
env.NoteOn()
streamer := synth.GatedStreamer(sr, 440.0, env)
speaker.Play(streamer)
// ... later, on note-off event:
env.NoteOff()
```

## Impact

- **Files touched:** `audio/effects.go` (new `GatedEnvelope` type + methods), `audio/synth.go` (new `GatedStreamer` method).
- **Additive / backward compatible:** fully additive — existing `envelopeGenerator` and `Synth.Streamer` are untouched, no callers break.
- **Invasiveness:** low. No changes to `Instrument`, `Note`, persistence, or UI layers. The new type lives entirely within the `audio` package.
- **Risk area:** goroutine safety between the audio callback (`Stream`) and the sequencer/UI goroutine calling `NoteOff`. The `sync.Mutex` in `GatedEnvelope` addresses this; no lock-free tricks required given the low contention profile.
