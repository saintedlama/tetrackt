# Plan: Page Up / Page Down Navigation

## Feature Description

Jump a full viewport-height at a time in the tracker pattern editor. `Home`/`End` are already implemented, making Page Up / Page Down the obvious gap in row navigation. The feature adds two new key bindings to the tracker panel that move the cursor (and viewport) by one full visible page at a time.

---

## Step-by-Step Implementation

### Step 1 — Add `pgup` / `pgdown` cases to `TrackerModel.Update()`

File: `ui/tracker/tracker.go`, inside the `tea.KeyPressMsg` block of `TrackerModel.Update()`.

Add the two new `case` branches immediately after the existing `"end"` case, mirroring its structure:

```go
case "pgdown":
    visibleRows := m.visibleRows()
    m.CursorRow = min(m.CursorRow+visibleRows, m.NumRows-1)
    if m.CursorRow >= m.viewportRow+visibleRows {
        m.viewportRow = m.CursorRow - visibleRows + 1
    }
case "pgup":
    visibleRows := m.visibleRows()
    m.CursorRow = max(m.CursorRow-visibleRows, 0)
    if m.CursorRow < m.viewportRow {
        m.viewportRow = m.CursorRow
    }
```

The viewport logic is deliberately consistent with the existing `"down"` and `"up"` cases: the viewport only moves when the cursor would go out of view.

> **Key name note**: Verify the exact key string against `charm.land/bubbletea/v2`. In Bubbletea v1 the names were `"pgup"` / `"pgdown"`. If v2 spells them differently (e.g. `"page_up"` / `"page_down"`), use whatever the library constant resolves to. A quick `fmt.Println(msg.String())` in `Update` or grepping the bubbletea v2 key tables will confirm it.

### Step 2 — Update the footer help text

File: `ui/tracker/screen.go`, `TrackerScreen.Footer()` method.

Append `PgUp/PgDn: Page` (or similar) to the returned string so users can discover the feature. The footer is a single long string; insert the new hint alongside the adjacent navigation hints (near `↑↓←→: Navigate`).

---

## Affected Files

| File | Symbol | Change |
|---|---|---|
| `ui/tracker/tracker.go` | `TrackerModel.Update()` | Add `"pgdown"` and `"pgup"` cases to the `tea.KeyPressMsg` switch |
| `ui/tracker/screen.go` | `TrackerScreen.Footer()` | Append PgUp/PgDn hint to the help string |

No new types, messages, or helper functions are needed. Both changes are self-contained.

---

## Edge Cases

### Cursor near the top, Page Up pressed
`CursorRow < visibleRows` — `max(CursorRow - visibleRows, 0)` clamps to `0`. The viewport guard `if m.CursorRow < m.viewportRow` then also sets `viewportRow = 0`. Result is identical to `Home`, which is correct.

### Cursor near the bottom, Page Down pressed
`CursorRow + visibleRows >= NumRows` — `min(CursorRow + visibleRows, NumRows-1)` clamps to `NumRows-1`. The viewport guard brings the last row into view. Result is identical to `End`, which is correct.

### Pattern shorter than viewport (`NumRows <= visibleRows`)
`visibleRows()` returns `Viewport.Height - 4`. If the entire pattern fits on screen, page up/down will simply clamp `CursorRow` to `0` or `NumRows-1` and will not move the viewport (it is already fully visible). This is sensible: PgUp becomes Home, PgDn becomes End.

### Very small terminal (`Viewport.Height < 4`)
`visibleRows()` returns a negative or zero number. `min`/`max` clamping on `CursorRow` keeps it in `[0, NumRows-1]`, so no out-of-bounds access occurs. The viewport may not update meaningfully, but the cursor will still land at a valid row. This degenerate case already affects `up`/`down` navigation equally, so no special handling is needed here either.

### Repeated key presses at boundary
Holding PgDn at the last row is a no-op after the first press: `CursorRow` is already `NumRows-1`, `min` returns the same value, and the viewport guard does not fire. No infinite loop or off-by-one risk.

---

## Refactoring Risks and Considerations

### Inline logic vs. helper extraction
The existing `home`/`end`/`up`/`down` viewport logic is all inline. Introducing a `clampViewport()` helper would be a nice-to-have but would be scope creep here. Keep the new cases inline to match the convention.

### No propagation to `TrackerScreen`
`TrackerScreen.Update()` already forwards all unhandled key events to `t.Tracker.Update(msg)` when the tracker panel is active (the `_, cmd := t.Tracker.Update(msg)` fallthrough). `pgup`/`pgdown` will be handled by that existing dispatch without any changes to `screen.go`'s `Update()`.

### Bubbletea key name verification
This is the only non-trivial risk. If the key string used in the `case` does not match what Bubbletea v2 produces, the keys will silently do nothing. Verify before shipping by either: (a) adding a temporary `default` log case, or (b) grepping `charm.land/bubbletea/v2` source for `pgdown`/`page_down`.

### No test coverage gap
`tracker.go` has a corresponding `*_test.go` (pattern is consistent across the package). A small table-driven test for `pgup`/`pgdown` — covering mid-pattern, near-top, and near-bottom start positions — should be added to `ui/tracker/tracker_test.go` (or created if absent) to prevent regressions.
