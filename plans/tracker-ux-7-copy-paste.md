# Tracker UX: Copy / Paste / Batch Editing

**Status: Done**

## Feature Description

A rectangular block clipboard for the tracker grid. The user selects a region with `Shift+Arrows` or `Ctrl+A`, copies or cuts it with `Alt+C` / `Alt+X`, and pastes it with `Alt+V`. An effects-only paste (`Alt+Shift+V`) overwrites all columns except Note.

---

## Implementation

### Clipboard data model

`trackerClipboard` is a package-private struct in `ui/tracker/tracker.go`:

```go
type trackerClipboard struct {
    HasData bool
    Cells   [][]TrackRow // [row][track]
}
```

`TrackerModel` holds a single `clipboard trackerClipboard` field (unexported). There is no persistent Clipboard kind enum; the shape of `Cells` determines what was copied (single cell, row, multi-row block, etc.).

### Selection model

Selection is managed entirely by `navigation.Grid` in `ui/tracker/navigation/navigation.go`. Key API:

| Method | Behaviour |
|---|---|
| `MoveExtending(dTrack, dRow)` | Starts or extends a rectangular selection while moving |
| `SelectAll()` | Selects the entire grid (row 0..N-1, track 0..T-1) |
| `ClearSelection()` | Collapses selection |
| `SelectionBounds()` | Returns `(minRow, maxRow, minTrack, maxTrack, hasSelection bool)` |
| `IsSelected(row, track)` | True if the cell is within the current selection |
| `SelectedCells()` | All `Cell{Row, Track}` in the selection; cursor only if no selection |

`Move()` (plain navigation) always calls `selection.clear()`, so navigating without Shift collapses the selection.

### Key bindings

All clipboard and selection operations are handled in `TrackerModel.handleGlobalEditingKey()` (called from `Update()` before column-specific edit dispatch, in both navigate and edit mode):

| Key | Action |
|---|---|
| `Ctrl+A` | `nav.SelectAll()` |
| `Alt+C` | `copySelectionToClipboard()` |
| `Alt+X` | `copySelectionToClipboard()` then `clearSelectedCells()` |
| `Alt+V` | `pasteClipboard()` |
| `Alt+Shift+V` | `pasteClipboardEffectsOnly()` |
| `Shift+↑↓←→` | `nav.MoveExtending(...)` — extends rectangular selection |
| `Insert` | `insertTrackSpace()` |
| `Shift+Insert` | `insertGlobalRowSpace()` |

Note: the original plan proposed lowercase single-key bindings (`c`, `v`, `d`, `x`, `M`, `F`, etc.). The implementation uses `Alt+` modifier combinations to avoid conflicts with note entry and existing single-key bindings.

### `copySelectionToClipboard()`

Reads `nav.SelectionBounds()` to get `(r0, r1, t0, t1)`, allocates a `[][]TrackRow` of size `(r1-r0+1) × (t1-t0+1)`, then iterates `nav.SelectedCells()` to populate it. Stores the result in `m.clipboard`.

### `clearSelectedCells()`

Iterates `nav.SelectedCells()` and sets each `TrackRow` to `TrackRow{Note: audio.Off()}`.

### `pasteClipboard()`

Pastes `m.clipboard.Cells` starting at the cursor position `(cursorRow, cursorTrack)`, skipping rows/tracks that would fall outside `[0, NumRows)` / `[0, NumTracks)`. Returns `bool` indicating whether anything was pasted. Does **not** advance the cursor after paste.

### `pasteClipboardEffectsOnly()`

Same geometry as `pasteClipboard()` but only copies non-Note fields (Volume, Ticks, Continuous, Arpeggio, Effect) from each source cell, leaving the destination's `Note` unchanged.

### Selection rendering

`nav.IsSelected(row, track)` is called per-cell in `View()`. Selected cells are rendered with `common.StyleSelected` (cyan foreground on dark gray background). This applies to all columns of a selected cell uniformly. The cursor cell overrides the selection style.

Transpose operations (`Alt+↑/↓`, `Shift+Alt+↑/↓`, `F7`/`F8`, `Shift+F7`/`Shift+F8`) also iterate `nav.SelectedCells()` and operate on the selection or cursor cell.

---

## Affected Files

| File | Change |
|---|---|
| `ui/tracker/tracker.go` | `trackerClipboard` struct; `clipboard` field on `TrackerModel`; `copySelectionToClipboard`, `clearSelectedCells`, `pasteClipboard`, `pasteClipboardEffectsOnly`, `transposeSelection`, `insertTrackSpace`, `insertGlobalRowSpace` methods; `handleGlobalEditingKey` dispatcher; selection style in `View()` |
| `ui/tracker/navigation/navigation.go` | `Grid` — `MoveExtending`, `SelectAll`, `ClearSelection`, `SelectionBounds`, `IsSelected`, `SelectedCells`, `HasSelection` |
| `ui/tracker/navigation/selection.go` | Internal `selection` type implementing the rectangular selection anchor/extent logic |
| `ui/tracker/screen.go` | Help entries for selection / clipboard keys |

---

## Divergences from Original Plan

The implementation differed from the original plan in several ways:

- **No typed clipboard kinds** — the plan proposed a `ClipboardKind` enum (`ClipboardCell`, `ClipboardRow`, `ClipboardTrack`, `ClipboardSynth`). The implementation uses a single flat `[][]TrackRow` block; the "kind" is implicit in the shape. Synth copy/paste was not implemented.
- **Key bindings use Alt modifiers** — the plan proposed single lowercase/uppercase keys (`c`, `C`, `v`, `d`, `D`, `x`, `M`, `F`, `X`). The implementation uses `Alt+C`, `Alt+X`, `Alt+V`, `Alt+Shift+V` to avoid conflicts with note entry and existing global bindings.
- **Selection is rectangular and multi-track** — the plan described a single-track row-range selection with a mark/anchor toggled by `M`. The implementation delegates all selection state to `navigation.Grid`, which manages a full 2D rectangular selection across rows and tracks.
- **`pasteClipboardEffectsOnly`** — not in the original plan; added as an extra paste mode that preserves notes.
- **No duplicate / swap / fill operations** — `DuplicateCell`, `DuplicateRow`, `SwapCell`, `FillSelection` from the plan were not implemented.
- **`Ctrl+A`** selects the entire pattern (all tracks); the plan did not mention a select-all binding.
- **Transpose on selection** — transpose operations (`Alt+↑/↓`, `F7`/`F8`) act on all selected cells, not just the cursor cell.
- **Insert/Shift+Insert** — row insertion (`insertTrackSpace`, `insertGlobalRowSpace`) is implemented here, not covered by the original plan.
# Tracker UX: Copy / Paste / Batch Editing

## Feature Description

A unified clipboard that holds whatever was last copied. Paste (`v`) inspects the clipboard type and acts accordingly:

| Clipboard contents | Paste behaviour |
|--------------------|-----------------|
| Cell               | Overwrite the cursor cell; advance cursor down |
| Row                | Overwrite all tracks at the cursor row; advance cursor down |
| Track              | Overwrite all rows of the cursor track |
| Synth              | Apply synth settings to the cursor track |

**Copy operations:**
- Copy cell (`c`): copy the cell under the cursor (note + volume + ticks + continuous + arpeggio).
- Copy row (`C`): copy the full row across all tracks at the current cursor row.
- Copy track (`R`): copy all rows of the cursor track. *(see key conflict notes below)*
- Copy synth (`S`): copy the synth settings of the cursor track. *(see key conflict notes below)*

**Paste:** (`v`): paste whatever is in the clipboard; behaviour determined by clipboard type.

**Convenience shortcuts:**
- Duplicate cell (`d`): copy current cell + paste to the row below, then advance.
- Duplicate row (`D`): copy current row + paste to the next row, then advance.
- Swap cell with clipboard (`x`): exchange the cursor cell with the clipboard cell. *(see key conflict notes below)*

**Selection + range fill:**
- Mark selection (`M`): toggle a selection anchor at the current row.
- Fill selection (`F`): fill all selected rows in the current track with the clipboard.
- Clear selection (`X`): clear note/effects on all selected rows in the current track.

---

## Clipboard Data Model

Define in `ui/tracker/tracker.go` (alongside `TrackerModel`).

### Kind enum

```go
// ClipboardKind identifies what type of data the clipboard holds.
type ClipboardKind int

const (
    ClipboardEmpty ClipboardKind = iota
    ClipboardCell
    ClipboardRow
    ClipboardTrack
    ClipboardSynth
)
```

### Clipboard struct

Use a single flat struct rather than an interface{} tagged union — all fields are small value types and this avoids runtime type-assertions in `Paste`.

```go
// Clipboard holds the most recently copied data.
// Only the field matching Kind is valid.
type Clipboard struct {
    Kind  ClipboardKind
    Cell  TrackRow     // valid when Kind == ClipboardCell
    Row   []TrackRow   // valid when Kind == ClipboardRow; len == NumTracks
    Track []TrackRow   // valid when Kind == ClipboardTrack; len == NumRows
    Synth audio.Synth  // valid when Kind == ClipboardSynth; value copy (not pointer)
}
```

Store `audio.Synth` by value (not pointer) so the clipboard is a snapshot that can't be mutated by later synth edits.

### SelectionState

```go
// SelectionState tracks the optional row-range selection anchor.
type SelectionState struct {
    Active    bool
    AnchorRow int
}

// SelectedRange returns the inclusive [lo, hi] row indices for the
// current selection. Returns (CursorRow, CursorRow) when not active.
func (m *TrackerModel) SelectedRange() (lo, hi int) {
    if !m.Selection.Active {
        return m.CursorRow, m.CursorRow
    }
    a, b := m.Selection.AnchorRow, m.CursorRow
    if a > b {
        a, b = b, a
    }
    return a, b
}
```

### TrackerModel additions

```go
type TrackerModel struct {
    // ... existing fields ...
    Clipboard Clipboard
    Selection SelectionState
}
```

---

## Step-by-Step Implementation

### Step 1 — Define types in `ui/tracker/tracker.go`

Add `ClipboardKind`, `Clipboard`, `SelectionState` types and the `SelectedRange()` helper directly above or below the `TrackRow` definition.

Add `Clipboard` and `Selection` fields to `TrackerModel`.

No initialisation required — zero values are correct (`ClipboardEmpty`, `Active: false`).

### Step 2 — Copy methods on `TrackerModel`

All copy methods are pure mutations of `m.Clipboard`; they return nothing.

```go
func (m *TrackerModel) CopyCell() {
    m.Clipboard = Clipboard{
        Kind: ClipboardCell,
        Cell: m.Tracks[m.CursorTrack].Rows[m.CursorRow],
    }
}

func (m *TrackerModel) CopyRow() {
    rows := make([]TrackRow, m.NumTracks)
    for i, t := range m.Tracks {
        rows[i] = t.Rows[m.CursorRow]
    }
    m.Clipboard = Clipboard{Kind: ClipboardRow, Row: rows}
}

func (m *TrackerModel) CopyTrack() {
    track := make([]TrackRow, m.NumRows)
    copy(track, m.Tracks[m.CursorTrack].Rows)
    m.Clipboard = Clipboard{Kind: ClipboardTrack, Track: track}
}

func (m *TrackerModel) CopySynth() {
    m.Clipboard = Clipboard{
        Kind:  ClipboardSynth,
        Synth: *m.Tracks[m.CursorTrack].Synth, // value copy
    }
}
```

### Step 3 — Paste method on `TrackerModel`

`Paste` returns a `tea.Cmd` (which may be `nil`) so that synth paste can emit a `ui.TrackChanged` message to synchronise the synth screen.

```go
func (m *TrackerModel) Paste() tea.Cmd {
    switch m.Clipboard.Kind {
    case ClipboardCell:
        m.Tracks[m.CursorTrack].Rows[m.CursorRow] = m.Clipboard.Cell
        m.advanceCursorDown()
    case ClipboardRow:
        if len(m.Clipboard.Row) == m.NumTracks {
            for i := range m.Tracks {
                m.Tracks[i].Rows[m.CursorRow] = m.Clipboard.Row[i]
            }
            m.advanceCursorDown()
        }
    case ClipboardTrack:
        if len(m.Clipboard.Track) == m.NumRows {
            copy(m.Tracks[m.CursorTrack].Rows, m.Clipboard.Track)
        }
    case ClipboardSynth:
        snap := m.Clipboard.Synth
        m.Tracks[m.CursorTrack].Synth = &snap
        synth := m.Tracks[m.CursorTrack].Synth
        return func() tea.Msg { return ui.TrackChanged{Synth: synth} }
    }
    return nil
}

// advanceCursorDown moves the cursor one row down, clamped to NumRows-1.
func (m *TrackerModel) advanceCursorDown() {
    if m.CursorRow < m.NumRows-1 {
        m.CursorRow++
        visibleRows := m.visibleRows()
        if m.CursorRow >= m.viewportRow+visibleRows {
            m.viewportRow = m.CursorRow - visibleRows + 1
        }
    }
}
```

### Step 4 — Duplicate methods

```go
func (m *TrackerModel) DuplicateCell() tea.Cmd {
    m.CopyCell()
    return m.Paste()
}

func (m *TrackerModel) DuplicateRow() tea.Cmd {
    m.CopyRow()
    return m.Paste()
}
```

### Step 5 — Swap cell with clipboard

Only meaningful when Kind == ClipboardCell; silently no-op otherwise.

```go
func (m *TrackerModel) SwapCell() {
    if m.Clipboard.Kind != ClipboardCell {
        return
    }
    cur := m.Tracks[m.CursorTrack].Rows[m.CursorRow]
    m.Tracks[m.CursorTrack].Rows[m.CursorRow] = m.Clipboard.Cell
    m.Clipboard.Cell = cur
}
```

### Step 6 — Selection methods

```go
func (m *TrackerModel) ToggleMarkSelection() {
    if m.Selection.Active && m.Selection.AnchorRow == m.CursorRow {
        // Pressing M at the anchor deactivates selection.
        m.Selection.Active = false
    } else {
        m.Selection = SelectionState{Active: true, AnchorRow: m.CursorRow}
    }
}

func (m *TrackerModel) FillSelection() {
    if m.Clipboard.Kind != ClipboardCell {
        return // only cell clipboard makes sense for fill
    }
    lo, hi := m.SelectedRange()
    for row := lo; row <= hi; row++ {
        m.Tracks[m.CursorTrack].Rows[row] = m.Clipboard.Cell
    }
}

func (m *TrackerModel) ClearSelection() {
    lo, hi := m.SelectedRange()
    empty := TrackRow{Note: audio.Off(), Volume: 0}
    for row := lo; row <= hi; row++ {
        m.Tracks[m.CursorTrack].Rows[row] = empty
    }
}
```

### Step 7 — Wire key bindings in `TrackerScreen.Update`

In `screen.go`, inside the `if t.activePanel == 0` branch (tracker panel active), add a new case block **before** the fallthrough to `t.Tracker.Update(msg)`:

```go
// Clipboard / selection keys (tracker panel only)
switch msg.String() {
case "c":
    t.Tracker.CopyCell()
    return t, nil
case "C":
    t.Tracker.CopyRow()
    return t, nil
case "R":
    t.Tracker.CopyTrack()
    return t, nil
case "S":
    t.Tracker.CopySynth()
    return t, nil
case "v":
    cmd := t.Tracker.Paste()
    return t, cmd
case "d":
    cmd := t.Tracker.DuplicateCell()
    return t, cmd
case "D":
    cmd := t.Tracker.DuplicateRow()
    return t, cmd
case "x":
    t.Tracker.SwapCell()
    return t, nil
case "M":
    t.Tracker.ToggleMarkSelection()
    return t, nil
case "F":
    t.Tracker.FillSelection()
    return t, nil
case "X":
    t.Tracker.ClearSelection()
    return t, nil
}
```

### Step 8 — Render selection highlight in `TrackerModel.View()`

In the row-rendering loop in `tracker.go`, compute `selLo, selHi := m.SelectedRange()` once before the loop. Add a new `selectionRowStyle` and apply it to the row number when the row is within the selection range (and selection is active).

```go
// New style
selectionRowStyle = lipgloss.NewStyle().
    Foreground(common.ColorAccentModulation).
    Bold(true)
```

In the loop body, after the existing cursor/playback row number colour logic, add:

```go
} else if m.Selection.Active && row >= selLo && row <= selHi {
    tracks.WriteString(selectionRowStyle.Render(rowNumStr))
}
```

For cells within the selection on the active track, apply a dim background to give a visual hint without being as prominent as the cursor:

```go
selectedCellStyle = lipgloss.NewStyle().
    Background(common.ColorSurface). // or a distinct selection colour
    Foreground(common.ColorAccentModulation).
    Padding(0, 1)
```

Apply this style when `trackIdx == m.CursorTrack && row >= selLo && row <= selHi && !(row == m.CursorRow && trackIdx == m.CursorTrack)`.

### Step 9 — Update `TrackerScreen.Footer()`

Add new bindings to the footer string:

```
c: Copy cell | C: Copy row | R: Copy track | S: Copy synth | v: Paste | d: Dup cell | D: Dup row
x: Swap | M: Mark | F: Fill | X: Clear sel
```

---

## Affected Files

| File | What changes |
|------|-------------|
| `ui/tracker/tracker.go` | New types `ClipboardKind`, `Clipboard`, `SelectionState`; add fields to `TrackerModel`; new methods `CopyCell`, `CopyRow`, `CopyTrack`, `CopySynth`, `Paste`, `DuplicateCell`, `DuplicateRow`, `SwapCell`, `ToggleMarkSelection`, `FillSelection`, `ClearSelection`, `SelectedRange`, `advanceCursorDown`; update `View()` for selection rendering |
| `ui/tracker/screen.go` | Add clipboard/selection key dispatch in `Update()`; update `Footer()` |
| `ui/msgs.go` | No change needed — `ui.TrackChanged` already exists and is the right message for synth paste |
| `main.go` | No change needed — it already routes `ui.TrackChanged` to the synth screen via `m.synth().ApplyTrackChange(msg)` |

---

## Key Binding Conflict Analysis

### Existing bindings (all layers)

| Key string | Handler | Action |
|-----------|---------|--------|
| `"s"` | `main.go` | Open save dialog |
| `"l"` | `main.go` | Open load dialog |
| `"i"` | `main.go` | Open synth presets dialog |
| `"e"` | `main.go` | Open row effects dialog |
| `"t"` | `main.go` | Switch screen |
| `"+"` / `"-"` | `main.go` | Octave up/down |
| `"p"` / `"P"` | `main.go` | Play / loop-play |
| `"q"` / `"ctrl+c"` | `main.go` | Quit |
| `"1"`–`"7"`, `"!"`, `"@"`, `"$"`, `"%"`, `"^"`, `"&"`, `"\""` | `main.go` | Note input |
| `"tab"` / `"shift+tab"` | `TrackerScreen` | Panel switching |
| `"up"` / `"down"` / `"left"` / `"right"` | `TrackerScreen` / `TrackerModel` | Navigation |
| `"shift+left"` / `"shift+right"` | `TrackerScreen` | Volume/BPM coarse adjust |
| `"delete"` | `TrackerScreen` | Clear note |
| `"home"` / `"end"` | `TrackerModel` | Jump to first/last row |

### New bindings and their safety

| Proposed key | Key string | Status | Notes |
|-------------|-----------|--------|-------|
| Copy cell | `"c"` | **Safe** | Unused |
| Copy row | `"C"` | **Safe** | Unused |
| Copy track | `"R"` | **Safe** | Unused — see conflict below |
| Copy synth | `"S"` | **Safe** | `"s"` is used by main.go (save) but `"S"` (shift+s) is not — main.go's switch checks `"s"` literally and unhandled keys fall through to the screen |
| Paste | `"v"` | **Safe** | Unused |
| Duplicate cell | `"d"` | **Safe** | Unused |
| Duplicate row | `"D"` | **Safe** | Unused |
| Swap cell | `"x"` | **Safe** | Unused |
| Mark selection | `"M"` | **Safe** | `"m"` unused, `"M"` unused |
| Fill selection | `"F"` | **Safe** | `"f"` unused, `"F"` unused |
| Clear selection | `"X"` | **Safe** | Unused |

### Conflicts in the original spec (resolved above)

1. **`Shift+C` vs `C`** — In bubbletea v2 terminal handling, holding Shift and pressing `c` produces the key string `"C"` (the uppercase character), **not** `"shift+c"`. Therefore the spec's distinction between "Copy row (`C`)" and "Copy track (`Shift+C`)" is **impossible to implement** as separate bindings on most terminals. Resolution: assign Copy track to `"R"` (shift+r), which is unambiguous and unused.

2. **`Shift+X` vs `X`** — Same terminal limitation: shift+x = `"X"`. The spec lists both "Clear selection (`X`)" and "Swap (`Shift+X`)". Resolution: use `"x"` (lowercase) for swap cell and `"X"` (shift+x) for clear selection, inverting the shift sense so both are reachable.

3. **`Shift+S` for copy synth** — `"S"` (shift+s) is safe because main.go only consumes `"s"`. Verified by inspecting main.go's key switch — unmatched keys fall through to the `m.screens[m.activeScreen].Update(msg)` call. Apply this binding only when the tracker panel is active inside `TrackerScreen.Update()`.

---

## Selection State — Where It Lives and How It's Rendered

**Location:** `SelectionState` field on `TrackerModel`. This keeps selection entirely within the tracker subsystem; no message needs to be emitted.

**Lifecycle:**
- Created/toggled by `ToggleMarkSelection()` (key `M`).
- Automatically deactivated when the tracker model is reset (load, new song).
- Navigation does **not** clear the selection (intentional; lets the user set anchor then move cursor to define range).
- After `FillSelection`, `ClearSelection`, or `DuplicateRow`, the selection remains (user can paste repeatedly).

**Rendering:**
- Row number column: colour the row number in `selectionRowStyle` (distinct from cursor and playback).
- Active-track cells in selection: apply `selectedCellStyle` (dim highlight).
- Other-track rows in selection range: no extra styling (selection is per-track for fill/clear operations).

---

## Refactoring Risks and Considerations

1. **`advanceCursorDown` extraction** — The cursor+viewport advancement logic currently exists inline in `TrackerModel.Update` for the `"down"` key. Extracting it into a helper avoids duplication but touches existing behaviour. Verify the viewport scrolling thresholds match the inline version exactly.

2. **Clipboard size mismatch on paste** — If a song is loaded with more/fewer tracks or rows than when the clipboard was filled, `Paste` for `ClipboardRow` or `ClipboardTrack` may have the wrong size. The guard `if len(m.Clipboard.Row) == m.NumTracks` / `len(m.Clipboard.Track) == m.NumRows` silently skips the paste rather than panicking. Consider whether the UX should warn or attempt a partial paste (truncate/pad).

3. **Synth paste and pointer aliasing** — `CopySynth` stores `*m.Tracks[m.CursorTrack].Synth` by value in `Clipboard.Synth`. `Paste` for `ClipboardSynth` creates a new pointer to a copy of that snapshot. This is correct, but reviewers should confirm no other subsystem caches the old `*audio.Synth` pointer (main.go and the synth screen always receive a new pointer via `ui.TrackChanged`).

4. **Key dispatch ordering** — New clipboard keys must be added **before** the `t.Tracker.Update(msg)` call in `TrackerScreen.Update`. If `TrackerModel.Update` ever gains a handler for `"c"`, `"v"`, etc., this will shadow the clipboard binding. Prefer keeping all clipboard/selection logic in `TrackerScreen.Update` (or entirely in `TrackerModel.Update`) to avoid split ownership.

5. **Settings panel** — Clipboard and selection bindings should only be active when `t.activePanel == 0` (tracker). The `x`, `c`, `v` keys should not interfere with the settings panel (BPM/volume). The existing code structure already handles this via the panel guard.

6. **Persistence** — The clipboard is ephemeral (session memory). It should **not** be serialised to `song.yaml`. No changes to `persistence/` are needed.

7. **Testing** — Clipboard methods (`CopyCell`, `Paste`, etc.) and `SelectedRange` are pure or nearly-pure mutations of `TrackerModel`, making them straightforward to unit test without bubbletea's `Update` loop. Add tests in `ui/tracker/tracker_test.go` (create if absent) verifying round-trip copy/paste for each clipboard kind, swap semantics, and selection range calculation.
