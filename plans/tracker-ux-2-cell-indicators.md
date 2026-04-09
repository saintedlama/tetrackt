# Tracker UX: Show Ticks and Continuous in the Grid Cell

## Feature Description

Currently each tracker cell renders `NOTE VOL ARP`. The `Ticks` and `Continuous`
fields on `TrackRow` are invisible once set — the only way to see them is to
re-open the row effects dialog. Add a 4th column indicator (`T4` for a custom
tick count, `C.` for continuous) so this data is surfaced in the grid without
extra interaction.

---

## Column Layout Design

### Current format (10 chars of content)

```text
%-3s %2s %3s
NOTE VOL ARP
C4-  4 A47
---  0 ---
```

| Field     | Width  | Format  | Examples            |
| --------- | ------ | ------- | ------------------- |
| Note      | 3      | `%-3s`  | `C4-`, `C#4`, `---` |
| (space)   | 1      | literal |                     |
| Volume    | 2      | `%2s`   | `4`, `..`           |
| (space)   | 1      | literal |                     |
| Arpeggio  | 3      | `%3s`   | `A47`, `---`        |
| **Total** | **10** |         |                     |

Each track column occupies **13 chars** on screen: 1 (left pad) + 10 (content) +
1 (right pad) from `cellStyle.Padding(0, 1)`, plus 1 trailing space written
after each cell in `View()`.

The separator and header trailing spaces are sized to match:

- Separator: `strings.Repeat("─", 10)` + `"   "` = 13 chars
- Header: `"Track N"` (7) + `Padding(0,1)` (9 rendered) + `"    "` (4 spaces) = 13 chars

### New format (13 chars of content)

```text
%-3s %2s %3s %2s
NOTE VOL ARP TK
C4-  4 A47 T6
---  0 --- ..
C4-  0 --- C.
C4-  4 A47 C6
```

| Field      | Width  | Format  | Examples                           |
| ---------- | ------ | ------- | ---------------------------------- |
| Note       | 3      | `%-3s`  | `C4-`, `C#4`, `---`                |
| (space)    | 1      | literal |                                    |
| Volume     | 2      | `%2s`   | `4`, `..`                          |
| (space)    | 1      | literal |                                    |
| Arpeggio   | 3      | `%3s`   | `A47`, `---`                       |
| (space)    | 1      | literal |                                    |
| Ticks/Cont | 2      | `%2s`   | `..`, `T6`, `C.`, `C6`, `T+`, `C+` |
| **Total**  | **13** |         |                                    |

Each track column will now be **16 chars** on screen.

### Indicator encoding (2-char field)

| State                          | Display | Meaning                         |
| ------------------------------ | ------- | ------------------------------- |
| Ticks==0, Continuous==false    | `..`    | Default (global Speed, no loop) |
| Ticks 1–9, Continuous==false   | `T{N}`  | Custom tick count N             |
| Ticks 10–32, Continuous==false | `T+`    | Custom tick count ≥ 10          |
| Ticks==0, Continuous==true     | `C.`    | Continuous, default tick count  |
| Ticks 1–9, Continuous==true    | `C{N}`  | Continuous, custom tick count N |
| Ticks 10–32, Continuous==true  | `C+`    | Continuous, tick count ≥ 10     |

The `T` prefix distinguishes a tick-only setting from a continuous one. When both
are active the `C` prefix takes priority (continuous is the more significant flag)
and the digit conveys the tick count in the same slot.

---

## Step-by-Step Implementation

### Step 1 — Add `formatTicksContinuous` helper

In `ui/tracker/tracker.go`, add a new pure formatting function alongside the
existing `formatNote`, `formatVolume`, and `formatArpeggio` helpers:

```go
// formatTicksContinuous formats the Ticks/Continuous row-effects indicator (2 chars).
//   ".."  – default (no custom ticks, not continuous)
//   "T{N}" – custom tick count 1–9
//   "T+"  – custom tick count 10+
//   "C."  – continuous, default ticks
//   "C{N}" – continuous + custom tick count 1–9
//   "C+"  – continuous + custom tick count 10+
func formatTicksContinuous(ticks int, continuous bool) string {
    if !continuous && ticks == 0 {
        return ".."
    }
    prefix := "T"
    if continuous {
        prefix = "C"
    }
    if ticks == 0 {
        return prefix + "."
    }
    if ticks < 10 {
        return fmt.Sprintf("%s%d", prefix, ticks)
    }
    return prefix + "+"
}
```

### Step 2 — Update the cell format string in `View()`

Find the single line in `TrackerModel.View()` that builds `cellContent`:

```go
// before
cellContent := fmt.Sprintf("%-3s %2s %3s",
    formatNote(trackRow.Note),
    formatVolume(trackRow.Volume),
    formatArpeggio(trackRow.Arpeggio))
```

Replace with:

```go
// after
cellContent := fmt.Sprintf("%-3s %2s %3s %2s",
    formatNote(trackRow.Note),
    formatVolume(trackRow.Volume),
    formatArpeggio(trackRow.Arpeggio),
    formatTicksContinuous(trackRow.Ticks, trackRow.Continuous))
```

### Step 3 — Update the separator width

In the same `View()` method, the separator loop uses a hard-coded repeat count:

```go
// before
tracks.WriteString(strings.Repeat("─", 10))
```

```go
// after
tracks.WriteString(strings.Repeat("─", 13))
```

This keeps the separator aligned with the wider cell content (10 → 13 content
chars, same 3-char gap after each cell).

### Step 4 — Update the header trailing padding

The header is assembled immediately above the separator loop:

```go
// before
tracks.WriteString(trackHeader)
tracks.WriteString("    ")   // 4 spaces
```

```go
// after
tracks.WriteString(trackHeader)
tracks.WriteString("       ") // 7 spaces
```

`"Track N"` renders to 9 chars (7 chars + `Padding(0,1)`).
Old total: 9 + 4 = 13. New total: 9 + 7 = 16.
Separator total: 13 + 3 = 16. ✓

---

## Affected Files

| File                    | Location              | Change                                                     |
| ----------------------- | --------------------- | ---------------------------------------------------------- |
| `ui/tracker/tracker.go` | `TrackerModel.View()` | Update `cellContent` format string — add 4th `%2s` column  |
| `ui/tracker/tracker.go` | `TrackerModel.View()` | `strings.Repeat("─", 10)` → `strings.Repeat("─", 13)`      |
| `ui/tracker/tracker.go` | `TrackerModel.View()` | Header trailing `"    "` → `"       "`                     |
| `ui/tracker/tracker.go` | (new function)        | `formatTicksContinuous(ticks int, continuous bool) string` |

No other files require modification:

- **`audio/effects.go`** — `TrackRow.Ticks` and `TrackRow.Continuous` already exist; no audio-layer changes needed.
- **`persistence/song.go`** — `SavedTrackRow` already serializes both fields (`row_ticks`, `continuous`); round-trip is complete.
- **`ui/tracker/roweffectsdialog.go`** — emits `RowEffectsApplied` with `Ticks` and `Continuous`; no change needed.
- **`ui/common/styles.go`** — no new styles required for the plain-text indicator.

---

## Risks and Considerations

### Column width is a hard-coded coupling

The separator repeat count, header padding, and cell content width are all
co-dependent magic numbers inside a single function. All three must be updated
atomically. If any one is missed the grid will visually misalign. Consider
extracting a `cellContentWidth = 13` constant at the top of the function to make
this coupling explicit.

### Two-digit tick counts lose precision in the display

`maxTicks = 32` (defined in `roweffectsdialog.go`). Counts 10–32 will all show as
`T+` or `C+`. This is a deliberate trade-off to keep the indicator at 2 chars.
If precise display of two-digit ticks is later required, either widen the column
to 3 chars (`T32`) or use hex encoding (`Ta` for 10, `Tf` for 15, etc.). The
plan as described accepts `T+` as sufficient for now.

### No per-field colour on the indicator

The existing cell rendering builds a single plain string and passes it to
`cellStyle.Render()` / `cursorCellStyle.Render()`. Adding distinct colour to only
the new indicator would require restructuring each cell from a single formatted
string into lipgloss-composed substrings (e.g. concatenating
`lipgloss.NewStyle().Foreground(common.ColorAccentWarning).Render(tk)` onto the
rest of the cell). That is out of scope here but is a natural follow-up.

### No test files in `ui/tracker/`

The tracker package currently has no `*_test.go` files, so there are no snapshot
or unit tests that would break. The new `formatTicksContinuous` function is a
pure string formatter and is straightforward to unit-test independently if
coverage is desired — a test table covering all six indicator cases (see encoding
table above) would be sufficient.

### `ui/tracker/screen.go` is unaffected

`TrackerScreen` delegates all grid rendering to `TrackerModel.View()` and holds
no column-width assumptions of its own.
