# Pre-1.0 Tracker Editing UX (Modern Profile)

**Status:** Done

## Summary

Implement the pre-1.0 tracker editing UX with a single modern keybinding profile.
Do not add profile switching.

This plan also refactors panel navigation so Tab and Shift+Tab are no longer used for panel switching in the Tracker screen.
Panel switching will use Ctrl+Arrow.

## Key Decisions

1. Ship one profile only: Modern.
2. Keep editing semantics consistent across note, instrument, volume, effect, and parameter entry.
3. Reassign panel switching in Tracker from Tab / Shift+Tab to Ctrl+Left / Ctrl+Right.
4. Keep Synth panel navigation on Ctrl+Arrow (already implemented).
5. Reserve Tab / Shift+Tab for in-grid movement (subcolumn/track navigation) in Tracker editing.

## Ctrl+Arrow Availability Check

Current codebase check result:

1. Tracker screen does not currently use Ctrl+Arrow.
2. Synth screen already uses Ctrl+Arrow for panel-grid navigation.
3. No other in-repo key handlers consume Ctrl+Arrow.

Conclusion:

Ctrl+Arrow is available for the Tracker panel-navigation refactor and remains compatible with the pre-1.0 editing UX plan.

## Scope

### In Scope

1. Explicit Navigate mode and Edit mode in Tracker.
2. Subcolumn cursor model in Tracker row cells.
3. Edit-step setting and post-entry row advance.
4. Deterministic cell-entry pipeline (note, instrument, volume, effect, parameter).
5. Rectangular selection foundation.
6. Clipboard operations for block workflows.
7. Tracker panel-switching refactor to Ctrl+Arrow.
8. Help text and docs updates for the new controls.

### Out of Scope

1. Multiple keybinding profiles.
2. Legacy compatibility layer for old Tracker panel shortcuts.
3. Full remapping UI.

## Implementation Phases

## Phase 1: Input Model Foundation

Goals:

1. Add tracker edit state (Navigate vs Edit).
2. Add subcolumn cursor representation.
3. Define write targets for Note | Inst | Vol | FX | Param.

Primary files:

1. ui/tracker/tracker.go
2. ui/tracker/screen.go
3. ui/msgs.go (if additional message types are needed)

Acceptance:

1. Cursor can move between subcolumns.
2. Edit mode is visible in footer/help and stateful.

## Phase 2: Tab and Panel Navigation Refactor

Goals:

1. Move Tracker panel switching to Ctrl+Left / Ctrl+Right.
2. Free Tab / Shift+Tab for tracker-grid movement.
3. Keep Synth Ctrl+Arrow behavior unchanged.

Primary files:

1. ui/tracker/screen.go
2. ui/tracker/tracker.go
3. ui/synth/screen.go (help text review only, if needed)
4. ui/helpdialog.go

Acceptance:

1. Tracker: Ctrl+Left / Ctrl+Right switches Tracker and Settings panels.
2. Tracker: Tab / Shift+Tab moves in-grid as designed.
3. Synth: Ctrl+Arrow still navigates synth panel grid.
4. Help screens describe the new behavior.

## Phase 3: Cell Entry Semantics

Goals:

1. Note entry from musical keys.
2. Instrument and parameter hex nibble entry with predictable progression.
3. Delete behavior scoped by active subcolumn.
4. Edit-step row advance after successful write.

Primary files:

1. ui/tracker/tracker.go
2. ui/tracker/screen.go

Acceptance:

1. Partial hex entry is stable and does not corrupt adjacent fields.
2. Row advance follows edit-step rules.
3. Invalid keys are ignored without side effects.

## Phase 4: Selection and Clipboard

Goals:

1. Rectangular selection with Shift+Arrow.
2. Copy/cut/paste for selected blocks.
3. Insert/delete row-space operations.

Primary files:

1. ui/tracker/tracker.go
2. ui/tracker/screen.go

Acceptance:

1. Selection is visually obvious.
2. Copy/cut/paste preserves cell structure.
3. Insert/delete operations are deterministic and undo-safe if undo exists.

## Phase 5: Docs, Help, and Regression Coverage

Goals:

1. Update in-app help and footer text.
2. Add tests for input parsing, cursor movement, and keybinding routes.
3. Document modern profile controls in plan/docs.

Primary files:

1. ui/helpdialog.go
2. ui/tracker/screen.go
3. ui/tracker/*_test.go (new or updated)

Acceptance:

1. Keybinding docs match actual behavior.
2. Automated tests cover panel-navigation refactor and edit-step behavior.

## Risks and Mitigations

1. Risk: Keybinding collisions between tracker and synth mental models.
Mitigation: Keep Ctrl+Arrow as panel-navigation family in both screens, while Tab remains local to Tracker editing.

2. Risk: Regression in existing note-entry behavior.
Mitigation: Add focused tests for note insert, delete, and row advance before refactor completion.

3. Risk: Discoverability drop due to key changes.
Mitigation: Update help text and footer hints in same PR as behavior changes.

## Rollout Strategy

1. Implement Phase 1 and Phase 2 together in a small PR so key routing is coherent.
2. Land Phase 3 next with tests.
3. Land Phase 4 and Phase 5 after core input stability is verified.

## Definition of Done

1. Modern-only tracker UX is implemented with no profile toggle.
2. Tracker panel switching uses Ctrl+Arrow.
3. Tab / Shift+Tab is available for tracker-grid editing navigation.
4. Synth Ctrl+Arrow navigation remains functional.
5. Help text, docs, and tests are updated and passing.
