# Speed Row in the Settings Panel

## Feature Description

The Settings panel exposes **Volume** and **BPM** but the global `Speed` (sub-ticks per row) is
neither visible nor editable there. It pairs naturally with BPM and should be added as a third
settings row.

`Speed` controls how many sub-ticks fire per row. `main.go` divides `BPMDuration()` by `Speed` to
get the per-tick interval, so it directly governs arpeggio granularity and the temporal feel of a
row without changing the BPM label. Exposing it in the Settings panel makes live tempo shaping
possible without editing YAML by hand.

---

## Step-by-Step Implementation

### Step 1 — Add `MinSpeed` / `MaxSpeed` constants
**File:** `ui/tracker/tracker.go`

Add two new constants directly below the existing `DefaultSpeed`:

```go
const DefaultSpeed = 6 // sub-ticks per row
const MinSpeed     = 1
const MaxSpeed     = 16
```

`MinSpeed = 1` means the row fires once (no arpeggio sub-steps).
`MaxSpeed = 16` gives sixteen sub-ticks, which is the practical ceiling for chiptune-style arpeggios.

---

### Step 2 — Add `SpeedChanged` message type
**File:** `ui/tracker/screen.go`

Add alongside the existing `VolumeChanged` and `BPMChanged` structs:

```go
// SpeedChanged is emitted when the user adjusts Speed via the settings panel.
type SpeedChanged struct {
    Speed int
}
```

---

### Step 3 — Expand `settingsFocus` cycling from 2 to 3 items
**File:** `ui/tracker/screen.go`, inside `Update()` — the `activePanel == 1` branch

Two arithmetic expressions must change:

| Before | After |
|--------|-------|
| `(t.settingsFocus - 1 + 2) % 2` | `(t.settingsFocus - 1 + 3) % 3` |
| `(t.settingsFocus + 1) % 2`     | `(t.settingsFocus + 1) % 3`     |

Both must be updated together; missing either causes broken wrap-around or a stuck focus.

---

### Step 4 — Add Speed key handler
**File:** `ui/tracker/screen.go`, inside `Update()` — after the existing BPM block

The BPM block currently ends after clamping and returning `BPMChanged`. Immediately after it, add:

```go
// Speed row focused
if t.settingsFocus == 2 {
    // Normalize zero-value (uninitialized) before editing
    if t.Tracker.Speed <= 0 {
        t.Tracker.Speed = DefaultSpeed
    }
    switch msg.String() {
    case "left":
        t.Tracker.Speed--
    case "shift+left":
        t.Tracker.Speed -= 2
    case "right":
        t.Tracker.Speed++
    case "shift+right":
        t.Tracker.Speed += 2
    }
    if t.Tracker.Speed < MinSpeed {
        t.Tracker.Speed = MinSpeed
    } else if t.Tracker.Speed > MaxSpeed {
        t.Tracker.Speed = MaxSpeed
    }
    return t, func() tea.Msg { return SpeedChanged{Speed: t.Tracker.Speed} }
}
```

The zero-value guard mirrors the pattern already used in `main.go`'s `tick()` and `previewTick()`.

---

### Step 5 — Add Speed row to `View()`
**File:** `ui/tracker/screen.go`, inside `View()`

Add a `speedRow` variable and append it to `settingsContent`:

```go
speed := t.Tracker.Speed
if speed <= 0 {
    speed = DefaultSpeed
}
speedRow := render(settingsPanelActive && t.settingsFocus == 2, "Speed") +
    fmt.Sprintf("    %3d", speed)
settingsContent := volumeRow + "\n" + bpmRow + "\n" + speedRow
```

The `fmt.Sprintf` padding (`%3d`) right-justifies the integer to match the BPM row width. Verify the
visual alignment in the running TUI — the "Speed" label is one character shorter than "Volume" so
the padding constant may need a small nudge.

---

### Step 6 — Handle `SpeedChanged` in `main.go`
**File:** `main.go`, inside `Update()` — alongside `tracker.BPMChanged`

```go
case tracker.SpeedChanged:
    // Speed is already updated on TrackerModel; tick() and previewTick() read it directly.
```

No further action is needed. Both tick functions read `trackerModel.Speed` on every invocation, so
the new value takes effect on the very next scheduled tick.

---

## Affected Files

| File | What changes |
|------|-------------|
| `ui/tracker/tracker.go` | Add `MinSpeed = 1` and `MaxSpeed = 16` constants next to `DefaultSpeed` |
| `ui/tracker/screen.go` | Add `SpeedChanged` message type; change focus-cycling modulus from 2 → 3; add speed key-handler block; add `speedRow` to `View()` and append to `settingsContent` |
| `main.go` | Add `case tracker.SpeedChanged:` no-op handler in `Update()` |

**No persistence changes needed.** `persistence/song.go` already declares `SavedSong.Speed int`
with `yaml:"speed,omitempty"`, and `SongToTracks` already falls back to `DefaultSpeed` when the
field is absent in older saves.

---

## Min/Max Range and Display Format

| Property | Value |
|----------|-------|
| `MinSpeed` | `1` — one sub-tick per row; arpeggio fires only once |
| `MaxSpeed` | `16` — sixteen sub-ticks; maximum arpeggio resolution |
| `DefaultSpeed` | `6` (already defined in `tracker.go`) |
| Display | Right-justified integer, width 3: `fmt.Sprintf("    %3d", speed)` |
| Step (left/right) | ±1 |
| Large step (shift+left/right) | ±2 |

---

## Risks and Considerations

1. **Two places use the hardcoded `2`** for focus cycling arithmetic. Both
   `(settingsFocus - 1 + 2) % 2` and `(settingsFocus + 1) % 2` must be updated to use `3`. A
   partial update compiles silently but breaks up/down navigation.

2. **Zero-value `Speed`**: `TrackerModel.Speed` is an `int` and initialises to `0` in Go. The
   display and key handler must both normalise `0 → DefaultSpeed`. Forgetting this in the display
   shows `  0` until the user edits it; forgetting it in the handler lets the first `left` press
   decrement to `-1` and immediately clamp back to `MinSpeed`, which is confusing.

3. **Settings panel width**: The panel is sized by its content. Adding a third row does not widen
   it, but it will grow one line taller. Confirm the combined panels still fit within the terminal
   width at common sizes (80 columns).

4. **No new tick plumbing needed**: Because `tick()` and `previewTick()` in `main.go` read `Speed`
   directly from the model on each call, a change to `Speed` takes effect the moment the next tick
   fires — there is no stale cached value to invalidate.

5. **Interaction with row-level `Ticks` override**: Individual rows can carry a `Ticks` field
   (set via the Row Effects dialog) that overrides the global `Speed` for that row. This feature is
   not affected by the Settings panel change, but the footer/help text could eventually mention
   both controls to avoid user confusion.
