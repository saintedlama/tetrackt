# Chiptune-10: Tracker Effect Commands

**Status:** Planned

**Priority:** Low

## Problem

No tracker-side effect column processing beyond what the synth natively produces.

## Why It Matters

Classic trackers express vibrato, portamento, volume slide, note cut, note delay and arpeggio via effect codes evaluated per tick. These bridge the gap between the static synth and live, expressive playback.

## Required

- Effect type enum + value: `Vibrato(speed, depth)`, `VolumeSlide`, `NoteCut(tick)`, `NoteDelay(tick)`, `Arpeggio(semi1, semi2)`
- Tick-rate processor in the playback engine that applies effects each tick

## Implementation Plan

### 1. Define effect types (new `tracker` or `playback` package)

```go
type EffectType int

const (
    EffectNone EffectType = iota
    EffectArpeggio
    EffectVibrato
    EffectVolumeSlide
    EffectNoteCut
    EffectNoteDelay
)

type Effect struct {
    Type           EffectType
    Param1, Param2 int
}
```

Location: `tracker/effect.go` or `playback/effect.go`. The `audio` package is not touched.

### 2. Extend the pattern cell

Add an `Effect` field to the existing cell/note struct in the tracker layer:

```go
type Cell struct {
    Note       int
    Instrument int
    Effect     Effect
}
```

This is a pure additive change; zero-valued `Effect{Type: EffectNone}` is a no-op.

### 3. `TickProcessor`

A `TickProcessor` is created once per playback session and holds per-channel state:

```go
type ChannelState struct {
    BaseFreq   float64
    Volume     float64
    LFOPhase   float64
    ArpStep    int
    TicksInRow int  // counts up to speed
}

type TickProcessor struct {
    BPM    int
    Speed  int  // ticks per row
    Channels []ChannelState
}
```

`TickProcessor.Tick(ch int, cell Cell)` is called by the playback loop at every tick boundary before audio is rendered.

### 4. Per-effect logic

| Effect | Tick action |
|---|---|
| `Arpeggio(semi1, semi2)` | Each tick cycles `BaseFreq` → `BaseFreq*semi(0)` → `BaseFreq*semi(semi1)` → `BaseFreq*semi(semi2)` → repeat; calls `SetFrequency` on the active oscillator (depends on chiptune-3). |
| `Vibrato(speed, depth)` | Advance `LFOPhase += speed`; modulate frequency by `depth * sin(LFOPhase)`; calls `SetFrequency`. Can be done entirely in the tick processor without touching the `audio` LFO (avoids dependency on chiptune-2). |
| `VolumeSlide(delta)` | `ChannelState.Volume += Param1 - Param2` per tick; clamp to `[0, 1]`; calls `SetVolume` on the mixer/channel. |
| `NoteCut(tick)` | When `TicksInRow == Param1`, calls `Silence()` on the channel. |
| `NoteDelay(tick)` | Suppresses note-on until `TicksInRow == Param1`, then triggers the note normally. |

### 5. Interface boundary

The tick processor communicates with the audio engine through a narrow interface, keeping `audio` tracker-agnostic:

```go
type ChannelControl interface {
    SetFrequency(hz float64)
    SetVolume(v float64)
    Silence()
}
```

The concrete implementation lives in the audio/playback bridge. `TickProcessor` depends only on this interface.

## Impact

**Invasiveness:** Moderate. No existing `audio` package code changes. The playback loop gains a tick subdivision (currently row-rate → tick-rate), which is the most structural change.

**Files/packages touched:**
- `tracker/effect.go` (new) — `EffectType`, `Effect`
- `tracker/cell.go` (or equivalent) — add `Effect` field to `Cell`
- `playback/tick_processor.go` (new) — `TickProcessor`, `ChannelState`, `ChannelControl` interface
- `playback/engine.go` (existing) — replace per-row loop with per-tick loop; instantiate `TickProcessor`
- `audio` package — **not touched** (only accessed via `ChannelControl`)

**Compatibility:** Additive for `audio` and `tracker` data structures. The playback engine loop change is internal and not part of any public API, so no callers break. Pattern files with no effect column round-trip identically.
