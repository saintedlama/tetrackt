# Plan: Per-Row Visual Markers for Non-Default State

## Feature Description

Rows that have custom Ticks, ARP, or Continuous set look identical to plain rows in the current renderer. Apply a subtle accent to the row number (similar to the existing playback highlight) on any row that carries non-default effects, making patterns easy to scan at a glance.

---

## Current Rendering Pipeline

In `TrackerModel.View()` the row-number string is styled by a three-way branch:

```
playback (green bold) > cursor (orange bold) > plain (muted gray)
```

There is no fourth case for "row has custom effects data".

The `TrackRow` struct fields relevant to this feature are:

| Field        | Default         | Non-default condition          |
|--------------|-----------------|-------------------------------|
| `Ticks`      | `0` (use global)| `Ticks != 0`                  |
| `Continuous` | `false`         | `Continuous == true`           |
| `Arpeggio`   | empty slice     | `Arpeggio.IsActive()` is true  |

`Note` and `Volume` are intentionally excluded — the feature targets playback-behaviour modifiers, not pitch/amplitude data.

---

## Non-Default Conditions

A row index `r` is considered "marked" when **any** track satisfies at least one of the following for `track.Rows[r]`:

1. `row.Ticks != 0` — per-row tick override is active
2. `row.Continuous == true` — continuous streaming mode is set
3. `row.Arpeggio.IsActive()` — arpeggio has one or more offsets

The check spans all tracks: a row is marked if *any* track at that index has non-default effects. This matches how a musician scans the grid — the row number acts as a summary indicator for the whole horizontal slice.

---

## Implementation Steps

### Step 1 — Add the new row-number style

In the package-level `var` block in `ui/tracker/tracker.go`, add a new style entry after `playbackRowStyle`:

```go
effectRowStyle = lipgloss.NewStyle().
    Foreground(common.ColorAccentWarning)
```

`ColorAccentWarning` is `#ffb300` (yellow) — already defined in `ui/common/styles.go`. It is distinct from every existing row-number colour:

| Colour  | Token                 | Current use                 |
|---------|-----------------------|-----------------------------|
| Green   | `ColorAccentPlay`     | Playback row                |
| Orange  | `ColorAccentEnvelope` | Cursor row                  |
| **Yellow** | **`ColorAccentWarning`** | **Effects marker (new)** |
| Gray    | `ColorTextMuted`      | Plain rows                  |

The style is **not** bold, keeping it visually quieter than the cursor and playback highlights, consistent with the "subtle accent" wording in the spec.

### Step 2 — Add the helper predicate

Add a package-private function directly below `visibleRows()` in `ui/tracker/tracker.go`:

```go
// rowHasEffects reports whether any track at row index r carries non-default
// effects (custom Ticks, Continuous mode, or an active Arpeggio).
func rowHasEffects(r int, tracks []Track) bool {
    for _, t := range tracks {
        row := t.Rows[r]
        if row.Ticks != 0 || row.Continuous || row.Arpeggio.IsActive() {
            return true
        }
    }
    return false
}
```

No new types or imports are needed — `Track`, `TrackRow`, and `audio.ArpeggioEffect.IsActive()` are already in scope.

### Step 3 — Extend the row-number rendering branch

In `View()`, replace the existing three-way branch:

```go
if row == m.PlaybackRow && m.IsPlaying {
    tracks.WriteString(playbackRowStyle.Render(rowNumStr))
} else if row == m.CursorRow {
    tracks.WriteString(cursorRowStyle.Render(rowNumStr))
} else {
    tracks.WriteString(rowNumStyle.Render(rowNumStr))
}
```

with a four-way branch:

```go
if row == m.PlaybackRow && m.IsPlaying {
    tracks.WriteString(playbackRowStyle.Render(rowNumStr))
} else if row == m.CursorRow {
    tracks.WriteString(cursorRowStyle.Render(rowNumStr))
} else if rowHasEffects(row, m.Tracks) {
    tracks.WriteString(effectRowStyle.Render(rowNumStr))
} else {
    tracks.WriteString(rowNumStyle.Render(rowNumStr))
}
```

Priority is preserved: playback > cursor > effects > plain. When the cursor lands on an effect row, the orange cursor colour takes precedence; the yellow marker reappears as soon as the cursor moves away.

---

## Affected Files

| File | Change |
|------|--------|
| `ui/tracker/tracker.go` | Add `effectRowStyle` to the var block; add `rowHasEffects` helper; extend the row-number branch in `View()` |
| `ui/common/styles.go` | **No change** — `ColorAccentWarning` is already defined |
| `audio/effects.go` | **No change** — `ArpeggioEffect.IsActive()` is already defined |

No other files, message types, or persistence formats are affected.

---

## Color / Style Co-existence

```
Row 03 (plain)       → gray   (#8a8a8a), normal weight
Row 07 (has ARP)     → yellow (#ffb300), normal weight   ← new
Row 12 (cursor)      → orange (#ff7700), bold
Row 15 (playback)    → green  (#00c853), bold
Row 15+07 (both)     → green wins while playing; yellow reappears when stopped
Row 12 cursor on 07  → orange wins while cursor is on it; yellow reappears after
```

The four colours are perceptually well-separated and all tested against the `#1c1c1e` terminal background used by the app.

---

## Risks and Considerations

**Performance**: `rowHasEffects` runs once per visible row on every `View()` call. With a typical 4–8 tracks and 20–30 visible rows this is a negligible inner loop (≤ 240 iterations).

**Interactions with playback and cursor**: The priority ordering is strict and matches user expectations — the most dynamic/active state always wins, and the effects marker gracefully reappears when those conditions clear.

**`Ticks == 0` semantics**: `NewTracker` initialises rows with zero-value `TrackRow{}` (no explicit `Ticks` assignment), so `Ticks == 0` correctly represents "use global speed" and is not flagged. Only a positive per-row override is non-default.

**Arpeggio with a single offset of 0**: `ArpeggioEffect{Offsets: []int{0}}` technically `IsActive()` (len > 0) but produces no audible retuning. This is consistent with the existing `IsActive()` contract — if the user explicitly added an arpeggio record, marking the row is correct behaviour.

**Future effects fields**: When new `TrackRow` effect fields are added (portamento, gate, detune, etc.) this predicate should be extended at the same time. A comment on `rowHasEffects` noting this intent helps future contributors.

**No tests for View output currently exist**: This change does not introduce test coverage regressions and, because `View()` has no existing unit tests, no new tests are strictly required. If a snapshot test is added later, the yellow marker will be trivially observable in the rendered output.
