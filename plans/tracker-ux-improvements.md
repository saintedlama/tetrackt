# Tracker UX Improvements

## 1. Cursor Advance on Note Entry

When the user enters a note (keys 1–7 or sharp variants), the cursor automatically moves down one row.
This is the single most impactful usability feature in any tracker — without it the user must manually press Down after every note.

## 2. Show Ticks and Continuous in the Grid Cell

Currently each cell renders `NOTE VOL ARP`. `Ticks` and `Continuous` are invisible once set — the only way to see them is to re-open the row effects dialog.
Add a 4th column indicator (e.g. `T4` for custom tick count, `C` for continuous) so this data is surfaced in the grid without extra interaction.

## 3. Volume Editing Directly in the Grid

Volume can currently only be set via the row effects dialog. In classic trackers the volume column is edited in-place by typing a two-digit value. Adding a volume sub-cursor (cycle to it with a key like `V`) would be significantly faster for composing.

## 4. Page Up / Page Down Navigation

Jump a full viewport-height at a time. `Home`/`End` are already implemented, making this the obvious gap in row navigation.

## 5. Speed Row in the Settings Panel

The Settings panel exposes Volume and BPM but the global `Speed` (sub-ticks per row) is neither visible nor editable there. It pairs naturally with BPM and should be added as a third settings row.

## 6. Per-Row Visual Markers for Non-Default State

Rows that have custom Ticks, ARP, or Continuous set look identical to plain rows in the current renderer. Apply a subtle accent to the row number (similar to the existing playback highlight) on any row that carries non-default effects, making patterns easy to scan at a glance.

## 7. Copy / Paste / Batch Editing

Working without copy-paste forces re-entering every note, effect, and synth setting by hand. The following actions would dramatically reduce that friction.

### Clipboard model

A single unified clipboard holds whatever was last copied. Paste (`v`) inspects the clipboard type and acts accordingly:

| Clipboard contents | Paste behaviour                                             |
| ------------------ | ----------------------------------------------------------- |
| Cell               | Overwrite the cursor cell; advance cursor down              |
| Row                | Overwrite all tracks at the cursor row; advance cursor down |
| Track              | Overwrite all rows of the cursor track                      |
| Synth              | Apply synth settings to the cursor track                    |

This means the user only needs to remember one paste key.

### Copy operations

- **Copy cell** (`c`): copy the cell under the cursor (note + volume + ticks + continuous + arpeggio).
- **Copy row** (`C`): copy the full row across all tracks at the current cursor row.
- **Copy track** (`Shift+C`): copy all rows of the cursor track.
- **Copy synth** (`Shift+S`): copy the synth settings of the cursor track.

### Paste

- **Paste** (`v`): paste whatever is in the clipboard; behaviour determined by clipboard type (see table above).

### Convenience shortcuts

- **Duplicate cell** (`d`): copy current cell + paste to the row below, then advance.
- **Duplicate row** (`D`): copy current row + paste to the next row, then advance.
- **Swap cell with clipboard** (`Shift+X`): exchange the cursor cell with the clipboard without moving the cursor — useful for reordering notes.

### Selection + range fill

- **Mark selection** (`M`): toggle a selection anchor at the current row; moving the cursor extends the selection highlight.
- **Fill selection** (`F`): fill all selected rows in the current track with the clipboard (must be a cell or row clipboard).
- **Clear selection** (`X`): clear note/effects on all selected rows.
