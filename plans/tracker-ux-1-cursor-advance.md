# Tracker UX: Cursor Advance on Note Entry

## Feature Description

When the user enters a note (keys 1–7 or sharp variants), the cursor automatically moves down one row. This is the single most impactful usability feature in any tracker — without it the user must manually press Down after every note.

---

## Step-by-Step Implementation Approach

### Step 1 — Add `MoveCursorDown()` to `TrackerModel` (`ui/tracker/tracker.go`)

The cursor-down logic already exists inline in `TrackerModel.Update` for the `"down"` key. Extract it into a dedicated method so it can be called from outside the model without duplicating the viewport-adjustment arithmetic.

```go
// MoveCursorDown advances the cursor one row, clamping at the last row
// and scrolling the viewport if necessary.
func (m *TrackerModel) MoveCursorDown() {
    if m.CursorRow < m.NumRows-1 {
        m.CursorRow++
        visibleRows := m.visibleRows()
        if m.CursorRow >= m.viewportRow+visibleRows {
            m.viewportRow = m.CursorRow - visibleRows + 1
        }
    }
}
```

Optionally replace the identical inline block inside `Update`'s `"down"` case with a call to `m.MoveCursorDown()` to keep the logic in one place.

### Step 2 — Call `MoveCursorDown()` after note entry in `main.go`

In `main.go`, the note-key handling block (the `noteKeyToName` lookup) already gates on `m.activeScreen == trackerScreenIdx`. After the existing `tr.SetNote(note)` call, add one call to the new method:

```go
if base, ok := noteKeyToName[msg.String()]; ok {
    note := audio.Note{Base: base, Octave: audio.Octave(m.octave)}

    if m.activeScreen == trackerScreenIdx {
        tr := m.trackerModel()
        tr.SetNote(note)
        tr.MoveCursorDown()                                    // ← NEW
        row := tr.Tracks[tr.CursorTrack].Rows[tr.CursorRow]
        if m.player.StartPreview(note, row.Arpeggio, tr.Tracks[tr.CursorTrack].Synth,
            tr.BPMDuration(), m.sampleRate, m.globalVolume, tr.Speed) {
            return m, m.previewTick()
        }
        return m, nil
    }

    m.playNote(note)
    return m, nil
}
```

> Note: `row` is read **after** the cursor moves so that the preview plays the note that was just written, not the note now under the new cursor position. The variable is derived from the track/row that was written (`tr.CursorTrack` / the old row index via the `SetNote` call), but since `SetNote` writes to the cell and we then advance, `row` must be captured before `MoveCursorDown()` or re-read from the previous index. See the risk section below.

**Safer ordering** — capture `row` before advancing:

```go
tr.SetNote(note)
row := tr.Tracks[tr.CursorTrack].Rows[tr.CursorRow]   // capture from the written cell
tr.MoveCursorDown()
if m.player.StartPreview(note, row.Arpeggio, ...) { ... }
```

This is the correct order: write the note, read the row's arpeggio for preview, then advance.

---

## Affected Files

| File | Symbol | Change |
|------|--------|--------|
| `ui/tracker/tracker.go` | `TrackerModel` | Add `MoveCursorDown()` method |
| `ui/tracker/tracker.go` | `TrackerModel.Update` | (Optional) replace inline down-logic with `m.MoveCursorDown()` |
| `main.go` | `model.Update` | Call `tr.MoveCursorDown()` after `tr.SetNote(note)` in the note-key branch |

---

## Refactoring Risks and Considerations

### 1. Preview row capture ordering
The existing code reads `row` after `SetNote` but before any cursor movement. If `MoveCursorDown()` is inserted between them, `row` will be read from the **new** cursor position, which is the **next** empty row — its `Arpeggio` field would be the default zero value, silently breaking arpeggio preview. **Fix**: capture `row` before calling `MoveCursorDown()`.

### 2. `delete` key must NOT advance the cursor
`TrackerScreen.Update` handles `"delete"` by calling `t.Tracker.SetNote(audio.Off())` directly, without going through `main.go`. This path does **not** call `MoveCursorDown()` and should continue to not do so. No change needed there.

### 3. Octave `+`/`-` keys must NOT advance the cursor
The `"+"` and `"-"` handlers in `main.go` also call `tr.SetNote(...)` (for a transposed note), but these are pitch-transpose operations, not note-entry events. They do not go through the `noteKeyToName` branch and are unaffected by this change.

### 4. Cursor clamping at last row
`MoveCursorDown()` (matching existing "down" key behaviour) silently does nothing when already at the last row. This is the correct tracker convention — the cursor stops at the bottom rather than wrapping.

### 5. No new message type required
The advance is a direct synchronous state mutation on `*TrackerModel`. No `tea.Cmd` is needed because the viewport/cursor state is not communicated to other components and is rendered on the next `View()` call.

### 6. Synth screen isolation
The note-key branch already guards on `m.activeScreen == trackerScreenIdx`. When the user is on the synth screen, `m.playNote(note)` is called instead, and `MoveCursorDown()` is never invoked.

---

## Key Bindings / Message Flow

```
User presses note key (e.g. "1", "!", "2", …)
  └─ main.go: model.Update (KeyPressMsg)
       └─ noteKeyToName lookup → audio.Note constructed
            └─ activeScreen == trackerScreenIdx ?
                 yes:
                   tr.SetNote(note)          writes note to CursorTrack/CursorRow
                   tr.MoveCursorDown()       ← NEW: advances CursorRow, scrolls viewport
                   row := ...               reads Arpeggio from written cell
                   player.StartPreview(...) starts audio preview
                 no:
                   m.playNote(note)          plays note without writing to grid
```

The `"down"` arrow key continues to work independently via `TrackerModel.Update` → `MoveCursorDown()` (or the existing inline logic). The two paths are orthogonal.
