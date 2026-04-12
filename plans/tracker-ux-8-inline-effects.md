# Tracker UX: Fully Inline Effects Editing

Status: Planned

## Feature Description

Make tracker effect editing fully inline in the pattern grid, matching classic tracker workflows:

- No dialog required for day-to-day effect entry.
- Effect command + parameter typed directly in effect columns.
- ARP, ticks/continuous, and other row-level playback modifiers editable from inline columns.
- Keep the row-effects dialog as an optional advanced inspector, not the primary entry path.

This preserves speed and muscle-memory for tracker users while keeping advanced controls available.

---

## Current State (Gap Analysis)

Current inline support already exists for:

- Note entry in Note column.
- Volume hex entry in Volume column.
- Arpeggio two-nibble shortcut in Arpeggio column.
- Effect type + param in Effect/Param columns.

Current gaps vs. classic fully-inline behavior:

- Per-row tick count and continuous mode are mostly dialog-driven.
- ARP presets/step/manual-per-tick are dialog-driven.
- Effect command entry is numeric-only in the Effect column, not tracker-command oriented.
- Inline discoverability is weak (help does not describe a full effect command language).

---

## Design Goals

1. Inline-first workflow: every commonly-used row effect can be authored from the grid.
2. Compact command model: typed commands map to deterministic row mutations.
3. Backward compatibility: existing songs and existing row data remain valid.
4. Progressive migration: keep dialog as fallback during rollout.
5. Testability: pure parse/apply helpers with table-driven tests.

---

## Proposed Inline Model

Use existing subcolumns with a tracker-like command/param model:

- Effect column: command code.
- Param column: two-hex-digit parameter.
- Arpeggio column: dedicated quick ARP entry stays available.

### Command Encoding

Define an inline command set (typed in Effect + Param):

- None: clear effect lane (`00`).
- Vibrato: speed/depth packed in param (`1xy`).
- Volume slide: signed delta encoding (`2xx`, decoder maps to signed range).
- Note cut: tick index (`3xx`).
- Note delay: tick index (`4xx`).
- Row ticks: per-row ticks override (`5xx`, 1..32, 0 clears override).
- Continuous toggle/mode (`6xx`, e.g. `00` off, `01` on).
- Arp preset+step (`7xy`, x=preset, y=step bucket).

Notes:

- Keep existing semantics for 0..4 commands to avoid regressions.
- Add 5..7 commands incrementally.
- ARP quick two-nibble column remains; if both ARP-column and effect command touch arp, last edit wins.

---

## Data Model Changes

Files: ui/tracker/tracker.go, ui/tracker/roweffectsdialog.go

1. Extend EffectType/EffectParam semantics to include row-level ops:
- Add EffectRowTicks.
- Add EffectContinuous.
- Add EffectArpPreset.

2. Add small helper types:

```go
type InlineEffectEdit struct {
    Type  EffectType
    Param int
}
```

3. Keep TrackRow as-is; command application mutates existing fields:
- Ticks
- Continuous
- Arpeggio
- Effect

---

## Parser + Applier (Core Refactor)

Create pure helpers in ui/tracker/tracker.go:

- parseEffectCommandNibble(v int) (EffectType, bool)
- decodeEffectParam(effectType EffectType, param int) (decoded struct, bool)
- applyInlineEffect(row *TrackRow, effectType EffectType, param int)

Rules:

- Parsing only validates syntax/range.
- Applying owns mutations and side effects (e.g. ARP implies continuous).
- Invalid command/param is ignored (no mutation), matching tracker robustness.

This isolates logic from UI key handling and enables focused tests.

---

## Input Handling Changes

File: ui/tracker/tracker.go

1. Keep existing cursor subcolumn flow.
2. Update Effect column input:
- Accept command nibble entry for new commands (0..7).
- Optionally accept letter aliases (V/S/C/D/T/O/A) as convenience mapping.

3. Update Param column input:
- Two-nibble capture remains.
- On second nibble, call applyInlineEffect for command types that target row-level behavior.

4. Update delete behavior:
- Delete on Effect clears command and any command-owned temporary state.
- Delete on Param clears param and re-applies default/cleared behavior safely.

---

## Rendering Changes

File: ui/tracker/tracker.go

1. Keep current column widths stable (no width expansion regression).
2. Ensure display of effect type/param reflects new inline commands.
3. Add clear visual markers for row-level states set by inline commands:
- ticks override marker in effect lane output,
- continuous marker,
- arp preset indicator where possible within existing width.

If space is insufficient, prioritize deterministic compact tokens over verbose text.

---

## Help + Documentation

Files: ui/tracker/screen.go, ui/helpdialog.go, README.md

1. Add inline command table to tracker help:
- command nibble,
- meaning,
- param format examples.

2. Clarify that Ctrl+E is optional advanced editor/fallback.
3. Add "classic inline effects workflow" examples to README.

---

## Compatibility + Migration

1. No persistence format changes required for base rollout.
2. Existing saved rows load unchanged.
3. New commands map to existing row fields; nothing breaks for old songs.
4. Keep dialog path active until inline feature stabilizes.

---

## Implementation Steps

### Step 1 — Command Model Scaffold

- Add new EffectType values and formatter support.
- Add parser/applier helpers with no behavioral wiring yet.

### Step 2 — Wire Inline Apply Path

- Route Param completion through applyInlineEffect.
- Preserve existing behavior for old command types.

### Step 3 — Add Row-Level Commands

- Implement row ticks and continuous commands.
- Implement arp preset command.

### Step 4 — Render + Help Updates

- Render compact new command states in cells.
- Update tracker/global help and README examples.

### Step 5 — Optional Dialog Integration

- When opening Ctrl+E, pre-select field inferred from current inline command.
- Keep Enter/apply behavior unchanged.

### Step 6 — Hardening

- Add/expand tests.
- Validate keyboard conflicts and mode behavior.

---

## Testing Plan

Files: ui/tracker/tracker_test.go (+ optionally roweffectsdialog tests)

1. Parser tests:
- valid/invalid command nibble,
- param range boundaries,
- alias mapping (if enabled).

2. Applier tests:
- each command mutates expected row fields,
- invalid params no-op,
- delete/clear behavior consistency.

3. Integration tests:
- EDIT mode key sequences produce expected row state,
- cursor advance/edit-step unaffected,
- selection copy/paste preserves new effect results.

4. Regression tests:
- old effect commands 0..4 remain identical.

Run:

- go test ./...

---

## Risks and Mitigations

1. Command ambiguity with current ARP column:
- Mitigation: define precedence (last write wins) and document it.

2. Width pressure in tracker grid:
- Mitigation: keep fixed-width tokens, avoid new columns in this phase.

3. User confusion during transition:
- Mitigation: keep Ctrl+E fallback and provide a compact inline command cheat-sheet.

4. Behavioral drift from classic trackers:
- Mitigation: lock command semantics in tests and document mapping explicitly.

---

## Acceptance Criteria

1. A user can author note + effect + param entirely from the grid in EDIT mode.
2. Ticks/continuous can be set and cleared inline without opening dialog.
3. ARP can be authored inline via dedicated ARP column and command model.
4. Ctrl+E remains optional, not required, for normal sequencing.
5. All tests pass and no tracker rendering width regressions are introduced.
