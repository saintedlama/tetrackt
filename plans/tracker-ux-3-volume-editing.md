# Plan: Volume Editing Directly in the Grid

## Feature Description

Volume can currently only be set via the row effects dialog (`E` key), which requires opening a separate overlay. In classic trackers the volume column is edited in-place by typing a two-digit decimal value. Adding a volume sub-cursor — cycled to with `V` — lets the composer set per-row volumes without leaving the grid, which is significantly faster during composition.

---

## Current State

- `TrackRow.Volume int` (range 0–64) already exists in the data model (`tracker.go`).
- `formatVolume` already renders it as `..` (zero) or `%02d` in the cell.
- The grid cell is rendered as one styled string:
  `fmt.Sprintf("%-3s %2s %3s", formatNote, formatVolume, formatArpeggio)`
- `RowEffectsApplied` (the dialog result message) does **not** carry a volume field, so the dialog does not currently write to `TrackRow.Volume` either. The field is present but there is no write path at all.
- Note entry (`1`–`7`, shifted variants) is intercepted in `main.go` **before** being forwarded to screens, because playing the audio preview requires `main.go`-owned resources (`sampleRate`, `globalVolume`, `player`).
- `V` is unused; nothing in any footer, key map, or handler references it.

---

## Implementation Approach

### Step 1 — Add sub-cursor state to `TrackerModel`

**File: `ui/tracker/tracker.go`**

Add two exported constants:

```go
const (
    ColNote   = 0
    ColVolume = 1
)
```

Add three fields to `TrackerModel`:

```go
CursorCol        int  // ColNote or ColVolume
volDigitPending  bool // true after first digit typed, awaiting second
volFirstDigit    int  // 0–6, the tens digit typed so far
```

`CursorCol` is exported (like `CursorRow` / `CursorTrack`) so `main.go` can read it for note-key guarding.
`volDigitPending` and `volFirstDigit` are unexported — they are pure transient edit state inside the tracker.

---

### Step 2 — Handle `V` and volume digit keys in `TrackerModel.Update`

**File: `ui/tracker/tracker.go`**

Extend the `tea.KeyPressMsg` switch in `Update()`:

1. **`"v"` key** — toggle `CursorCol`.  Cancel any pending digit (`volDigitPending = false`).

2. **`"0"`–`"9"` keys, only when `CursorCol == ColVolume`** — implement two-digit entry:
   - If `!volDigitPending`: store the digit as `volFirstDigit`, set `volDigitPending = true`.
     Optionally do a "live preview" update: set `Volume = digit * 10` (tentative).
     Do **not** advance the row yet.
   - If `volDigitPending`: compute `value = volFirstDigit*10 + digit`.
     Clamp: `if value > 64 { value = 64 }`.
     Call `SetVolume(value)`.
     Reset `volDigitPending = false`.
     Advance `CursorRow` by one (same auto-advance behaviour classical trackers use).

3. **`"delete"` key, when `CursorCol == ColVolume`** — call `SetVolume(0)`, cancel pending digit.
   (The existing `"delete"` handler in `screen.go` calls `Tracker.SetNote(audio.Off())` — guard it so it only fires when `CursorCol == ColNote`.)

4. **Navigation keys (`"up"`, `"down"`, `"left"`, `"right"`, `"home"`, `"end"`)** — always cancel any pending digit (`volDigitPending = false, volFirstDigit = 0`) before performing the existing navigation logic.  `CursorCol` is **preserved** across navigation so the user stays in volume-entry mode while stepping through rows.

Add a helper method:

```go
func (m *TrackerModel) SetVolume(vol int) {
    m.Tracks[m.CursorTrack].Rows[m.CursorRow].Volume = vol
}
```

---

### Step 3 — Guard note-key handling in `main.go`

**File: `main.go`**

The note-key block currently fires unconditionally for any `1`–`7` / shifted key. Wrap it with a CursorCol guard:

```go
if base, ok := noteKeyToName[msg.String()]; ok {
    tr := m.trackerModel()
    if m.activeScreen == trackerScreenIdx && tr.CursorCol == tracker.ColNote {
        // ... existing note entry + audio preview logic ...
    }
    // When ColVolume: fall through so tracker.Update handles digit below
}
```

Digit keys `"0"` and `"8"`, `"9"` are not in `NoteKeys`, so they already fall through. Keys `"1"`–`"7"` are in `NoteKeys`, and those overlap the tens-digit range for volume values 10–79. By guarding with `ColNote`, pressing `3` in volume mode means "value starts with 3" not "note E".

---

### Step 4 — Split cell rendering for sub-cursor highlight

**File: `ui/tracker/tracker.go`**, inside `View()`

Current rendering (one style for the whole cell):

```go
cellContent := fmt.Sprintf("%-3s %2s %3s", ...)
if row == m.CursorRow && trackIdx == m.CursorTrack {
    tracks.WriteString(cursorCellStyle.Render(cellContent))
```

Replace with per-part rendering for the cursor cell. Add a second cursor style (or reuse `cursorCellStyle` with a different foreground):

```go
volCursorStyle = lipgloss.NewStyle().
    Background(common.ColorSurface).
    Foreground(common.ColorAccentEnvelope). // distinct from note cursor
    Padding(0, 1)
```

In `View()`, extract a helper (or inline) that builds the cell as three separately-styled tokens:

```go
notePart := fmt.Sprintf("%-3s", formatNote(trackRow.Note))
volRaw   := formatVolume(trackRow.Volume)
// Show pending digit: e.g., if volDigitPending and this is the cursor cell,
// display "3_" instead of "03" to signal partial entry.
if isCursorCell && m.volDigitPending {
    volRaw = fmt.Sprintf("%d_", m.volFirstDigit)
}
volPart  := fmt.Sprintf("%2s", volRaw)
arpPart  := fmt.Sprintf("%3s", formatArpeggio(trackRow.Arpeggio))

if isCursorCell {
    switch m.CursorCol {
    case ColNote:
        tracks.WriteString(cursorCellStyle.Render(notePart) + " " +
            cellStyle.Render(volPart) + " " + cellStyle.Render(arpPart))
    case ColVolume:
        tracks.WriteString(cellStyle.Render(notePart) + " " +
            volCursorStyle.Render(volPart) + " " + cellStyle.Render(arpPart))
    }
} else {
    tracks.WriteString(cellStyle.Render(notePart + " " + volPart + " " + arpPart))
}
```

The padding on `cellStyle` is `(0, 1)` which adds one space on each side. The above code must ensure total column width stays consistent regardless of which sub-column is highlighted to avoid layout jitter. Measure total width before and after the refactor with a fixed test string.

---

### Step 5 — Update the footer

**File: `ui/tracker/screen.go`**, `Footer()` method

Add `V: Volume col` to the hint string. The string is already long; it could be restructured or the new item appended.

---

### Step 6 — (Optional) Add volume to `RowEffectsDialog`

The feature description focuses on inline editing, but for completeness the effects dialog could also expose volume. If this is in scope:

- Add `volume int` field to `RowEffectsDialog` (pre-populated from `row.Volume`).
- Add a `Volume` field to `RowEffectsApplied`.
- Apply `msg.Volume` in the `tracker.RowEffectsApplied` handler in `main.go`.
- This is independent of the inline sub-cursor work and can be done separately.

---

## Affected Files

| File | What changes |
|---|---|
| `ui/tracker/tracker.go` | Add `ColNote`/`ColVolume` constants; add `CursorCol`, `volDigitPending`, `volFirstDigit` to `TrackerModel`; extend `Update()` for `V`, digit keys, navigation cancellation; add `SetVolume()`; refactor cell rendering in `View()` to per-part highlight; add `volCursorStyle` |
| `main.go` | Guard note-key block with `tr.CursorCol == tracker.ColNote` |
| `ui/tracker/screen.go` | Update `Footer()` string; guard `delete` → `SetNote` with `ColNote` check |
| `ui/tracker/roweffectsdialog.go` | (optional) add volume field and wire through `RowEffectsApplied` |

---

## Sub-cursor vs. Note-entry Cursor Interaction

| Situation | Behaviour |
|---|---|
| User presses `V` | `CursorCol` toggles (`ColNote` ↔ `ColVolume`); pending digit buffer cleared |
| User navigates up/down | Row moves; `CursorCol` preserved (stay in whichever column); pending digit cleared |
| User navigates left/right (track) | Track moves; `CursorCol` preserved; pending digit cleared |
| User presses note key while `CursorCol == ColVolume` | `main.go` guard prevents note entry and audio preview; `TrackerModel.Update` sees the digit and treats it as the tens digit of a volume value (if it is 0–6 mod 10) |
| User presses `8` or `9` while `CursorCol == ColVolume` | Not a note key; falls through to tracker Update; treated as a tens digit (clamped on second digit if resulting value > 64) |
| User presses `delete` while `CursorCol == ColVolume` | Calls `SetVolume(0)`; note is NOT cleared |
| User presses `delete` while `CursorCol == ColNote` | Existing behaviour: `SetNote(audio.Off())`; volume NOT cleared |
| Mid-first-digit and user presses `V` | Pending digit discarded, cursor moves back to note column |
| Mid-first-digit and user navigates | Pending digit discarded, row/track moves |
| Playback active | `CursorCol` / `volDigitPending` state is display-only; playback row highlight still obeys `PlaybackRow`; no interaction |

---

## Refactoring Risks and Considerations

1. **Cell width consistency** — The current cell render is a single `lipgloss.Style.Render()` call on a fixed-width string. Splitting into three separate styled segments may accumulate slightly different padding/margins if not done carefully. Verify that all tracks still align under their headers by running the app at typical window widths and checking that columns don't drift.

2. **`Padding(0, 1)` on `cellStyle`** — Each sub-part inheriting `cellStyle` would add `2` characters per segment instead of `2` total for the cell. The per-part rendering must either remove cell padding and add explicit spaces between parts, or use a zero-padding inner style and a single outer spacing character between them. The simplest approach: remove `Padding` from `cellStyle` in the per-part path and add a literal `" "` between each part.

3. **Note keys `1`–`7` as volume tens-digits** — Pressing `1` in `ColVolume` mode means the user wants a volume starting with `1x`. The `main.go` guard prevents note entry, but the digit `1` must then be routed to `TrackerModel.Update()`. Currently, once `main.go` does not match a note key, it falls through to `m.screens[m.activeScreen].Update(msg)`. That means `TrackerModel.Update` (called from `TrackerScreen.Update`) will see the key. However `TrackerScreen.Update` currently passes non-matching keys via `t.Tracker.Update(msg)` — verify this forwarding path exists for ALL digit keys including `1`–`7`.

4. **Auto-advance on second digit** — Auto-advancing `CursorRow` after a completed two-digit volume entry is the UX expectation from classic trackers, but it may surprise users who only type one digit via `Delete`-then-digit. Consider making auto-advance consistent: advance only after a full two-digit commit, not after cancel/delete.

5. **Volume range 0–64** — The storage type is `int`. The valid printed range is `00`–`64`. If a user types `7x` (tens=7) the resulting value is at minimum 70 which exceeds 64. The clamping rule on the second digit must clamp to 64. If the first digit alone is `> 6`, it could be rejected immediately (beep or ignore) rather than waiting for the second digit. This avoids a confusing partial-entry state.

6. **`RowEffectsApplied` does not write volume** — Currently no code path writes `TrackRow.Volume` via the dialog (the field was added to the model but the dialog has no volume row). Until Step 6 is implemented, the only write path is the new inline editor. The existing effects dialog does not override volumes, which is safe.

7. **Persistence** — `TrackRow.Volume` is already serialised by the `persistence` package (verify field is exported, which it is). No changes needed there.

8. **Tests** — `TrackerModel.Update` is currently not directly unit-tested; the tracker tests focus on audio. Adding a small table-driven test in `tracker_test.go` (or a new `tracker_input_test.go`) covering: (a) `V` toggles column, (b) single digit followed by second digit commits correct value, (c) digit > 64 after second digit clamps, (d) delete clears volume, would guard against regressions.
