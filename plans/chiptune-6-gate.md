# Chiptune-6: Note-On / Note-Off Gate Model

**Status:** Planned

**Priority:** Medium

## Problem

`Synth.Streamer` takes a fixed `time.Duration`. ADSR sustain duration is derived by subtracting attack+decay+release from the total note length — there is no real "hold until note-off" concept.

## Why It Matters

In a real tracker, the note length is decoupled from note-off: a pattern cell sets when note-on fires; the envelope sustains indefinitely until the next note-on or an explicit cut command. The current model bakes duration in at note creation time, which prevents proper tracker integration and live play.

## Required

- Gate-based envelope: `NoteOn()` / `NoteOff()` transitions
- Streamer that sustains in the sustain stage until `NoteOff()` is called

## Implementation Plan

### 1. `GatedEnvelope` type (`effects.go`)

Add a new type alongside the existing `envelopeGenerator`. It holds ADSR *absolute* sample counts (attack, decay, release) and a level for sustain, but no `sustainSamples` — sustain duration is open-ended.

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

    mu       sync.Mutex
    state    gateState
    pos      int // samples elapsed in current stage
    releaseLevel float64 // amplitude at the moment NoteOff fired
}
```

`NewGatedEnvelope(sampleRate beep.SampleRate, a, d, r time.Duration, sustainLevel float64) *GatedEnvelope`
converts durations to sample counts; starts in `gateIdle`.

### 2. State machine transitions

| Event / condition | From | To |
|---|---|---|
| `NoteOn()` called | Idle | Attack |
| Attack samples exhausted | Attack | Decay |
| Decay samples exhausted | Decay | Sustain |
| `NoteOff()` called | Attack / Decay / Sustain | Release (snapshot `releaseLevel`) |
| Release samples exhausted | Release | Done |

`Amplitude(pos int) float64` is replaced by a stateful `Next() (float64, bool)` that advances `pos` and applies the appropriate linear ramp for the current stage. Returns `(0, false)` when `gateDone`.

### 3. Thread safety

`NoteOff()` acquires `mu` before transitioning state and snapshotting `releaseLevel`. `Next()` also acquires `mu` for the state read/write, keeping the critical section small (a single sample tick). A plain `sync.Mutex` is sufficient; contention is negligible at audio-thread cadence.

### 4. `GatedStreamer` (`synth.go`)

```go
// GatedStreamer returns a Streamer that sustains until env.NoteOff() is called.
// The caller is responsible for calling env.NoteOn() before streaming begins.
func (s *Synth) GatedStreamer(note Note, env *GatedEnvelope) beep.Streamer
```

Internally it generates oscillator samples and multiplies each by `env.Next()`. It returns `(n, false)` once `env` reaches `gateDone` (no `beep.Take` wrapper needed — the envelope itself signals end-of-stream).

`Synth.Streamer(note Note, d time.Duration)` is **unchanged**.

### 5. Caller workflow

```go
env := audio.NewGatedEnvelope(sr, 10*time.Millisecond, 20*time.Millisecond, 30*time.Millisecond, 0.7)
env.NoteOn()
streamer := synth.GatedStreamer(note, env)
speaker.Play(streamer)
// ... later, on note-off event:
env.NoteOff()
```

## Impact

- **Files touched:** `audio/effects.go` (new `GatedEnvelope` type + methods), `audio/synth.go` (new `GatedStreamer` method).
- **Additive / backward compatible:** fully additive — existing `envelopeGenerator` and `Synth.Streamer` are untouched, no callers break.
- **Invasiveness:** low. No changes to `Instrument`, `Note`, persistence, or UI layers. The new type lives entirely within the `audio` package.
- **Risk area:** goroutine safety between the audio callback (`Stream`) and the sequencer/UI goroutine calling `NoteOff`. The `sync.Mutex` in `GatedEnvelope` addresses this; no lock-free tricks required given the low contention profile.
