# Chiptune-10: Tracker Effect Commands

**Status:** Done

**Priority:** Medium

## Problem

Only portamento is fully wired into tick-rate playback. Arpeggio data is stored and editable but ignored during playback. Vibrato, VolumeSlide, NoteCut, and NoteDelay do not exist yet.

## Why It Matters

Classic trackers express vibrato, portamento, volume slide, note cut, note delay and arpeggio via effect codes evaluated per tick. These bridge the gap between the static synth and live, expressive playback.

## Current Implementation Context

### Already Implemented

**Portamento** (`Synth.Portamento float64`, `Patch.StartPortamento`, `Patch.TickPortamento`):
- `Synth.Portamento float64` — glide duration in seconds; 0 = snap
- `Patch.StartPortamento(fromFrequency, toFrequency float64, ticks int)` — begins stepped glide
- `Patch.TickPortamento()` — advances glide one step; called every sub-tick
- `Player.Tick()` calls `patch.TickPortamento()` for all active patches each sub-tick
- `Player.prevFrequencies []float64` tracks last triggered Hz per track

**Tick-rate infrastructure** (`player/player.go`):
- `Player.Tick()` subdivides rows into `speed` sub-ticks (`TrackerModel.Speed`, default `tracker.DefaultSpeed`)
- On sub-tick 0 of each row, `playRowNotes()` creates new `Patch` instances and submits them to the speaker
- `activePatches []*audio.Patch` is kept across sub-ticks so per-tick hooks can be called

**Arpeggio data structure** (`audio/effects.go`):
```go
type ArpeggioEffect struct {
    Offsets []int // semitone offsets per arp step, e.g. [0, 4, 7]
}
func (a ArpeggioEffect) IsActive() bool { return len(a.Offsets) > 0 }
```
- `TrackRow.Arpeggio ArpeggioEffect` holds per-row arpeggio data
- Editable via `RowEffectsDialog`; persisted to YAML
- **Not applied during playback** — `playRowNotes()` ignores `trackRow.Arpeggio`

### Not Yet Implemented

- Arpeggio tick-rate cycling (frequency step per sub-tick)
- Vibrato, VolumeSlide, NoteCut, NoteDelay
- General effect column on `TrackRow`

## Implementation Plan

### Phase 1: Wire Arpeggio Playback

**1a. Add arpeggio tick index to `Player`** (`player/player.go`):

```go
type Player struct {
    subTickCount    int
    activePatches   []*audio.Patch
    previewPatch    audio.Streamer
    prevFrequencies []float64
    arpTickIdx      []int // current arpeggio step per track; -1 when inactive
}
```

**1b. Initialize in `playRowNotes()`**: when `trackRow.Arpeggio.IsActive()`, set `p.arpTickIdx[trackIdx] = 0` and apply the first offset immediately via `patch.SetFrequency(targetFrequency * math.Pow(2, float64(offsets[0])/12))`. Otherwise set to `-1`.

**1c. Advance in `Tick()`** — before `patch.TickPortamento()`:

```go
for trackIdx, patch := range p.activePatches {
    if patch == nil || p.arpTickIdx[trackIdx] < 0 {
        continue
    }
    arp := trackerModel.Tracks[trackIdx].Rows[trackerModel.PlaybackRow].Arpeggio
    if arp.IsActive() {
        p.arpTickIdx[trackIdx]++
        idx := p.arpTickIdx[trackIdx] % len(arp.Offsets)
        mult := math.Pow(2, float64(arp.Offsets[idx])/12)
        patch.SetFrequency(p.prevFrequencies[trackIdx] * mult)
    }
}
```

### Phase 2: General Effects Column

**2a. Extend `TrackRow`** in `ui/tracker/tracker.go`:

```go
type TrackRow struct {
    Note       audio.Note
    Volume     int
    Ticks      int
    Continuous bool
    Arpeggio   audio.ArpeggioEffect
    Effect     TrackerEffect // NEW
}

type TrackerEffect struct {
    Type   EffectType
    Param  int // effect-specific value
}

type EffectType int

const (
    EffectNone      EffectType = iota
    EffectVibrato           // Param: packed (speed<<4 | depth)
    EffectVolumeSlide       // Param: positive = slide up, negative = down (per tick)
    EffectNoteCut           // Param: sub-tick number to cut at
    EffectNoteDelay         // Param: sub-tick number to delay note-on until
)
```

**2b. Add per-track effect state to `Player`**:

```go
type channelEffectState struct {
    vibratoPhase float64
    volume       float64 // [0,1]; 0 = use patch default
}
```

**2c. Per-effect tick logic** in `Player.Tick()` for each track:

| Effect | Tick action |
|---|---|
| `EffectVibrato` | Advance `vibratoPhase`; call `patch.SetFrequency(baseFreq * math.Pow(2, depth*math.Sin(phase)/12))` |
| `EffectVolumeSlide` | `volume += delta`; clamp to `[0,1]`; call `patch.SetVolume(volume)` (new `Patch` method) |
| `EffectNoteCut` | When `subTickCount == param`, call `patch.SetVolume(0)` |
| `EffectNoteDelay` | Suppress note-on at sub-tick 0; trigger `patch.NoteOn()` at sub-tick `param` |

**2d. Add `Patch.SetVolume(v float64)`** in `audio/synth.go` — stores volume scalar applied to output samples in `Stream()`.

### Phase 3: Effect Column UI

Extend `RowEffectsDialog` in `ui/tracker/roweffectsdialog.go` to expose `TrackerEffect` type and parameter editing alongside the existing arpeggio controls.

## Impact

### Phase 1 (Arpeggio playback)
- **Files touched:** `player/player.go` only — add `arpTickIdx` field, update `playRowNotes()` and `Tick()`.
- **Invasiveness:** Low. Purely additive; existing patterns without arpeggio are unaffected (`IsActive() == false`).

### Phase 2 (General effects)
- **Files touched:** `ui/tracker/tracker.go` (add `Effect` field to `TrackRow`), `player/player.go` (effect state + tick logic), `audio/synth.go` (`Patch.SetVolume`).
- **Invasiveness:** Moderate. `Effect` field defaults to `EffectNone` — fully backward compatible. Pattern files with no effect round-trip unchanged.

### Phase 3 (UI)
- **Files touched:** `ui/tracker/roweffectsdialog.go` only.
- **Invasiveness:** Low.
