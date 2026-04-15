# Plan: Page Up / Page Down Navigation

## Status: Done

## Feature Description

Jump a full viewport-height at a time in the tracker pattern editor. `Home`/`End` are already implemented, making Page Up / Page Down the obvious gap in row navigation. The feature adds two new key bindings to the tracker panel that move the cursor (and viewport) by one full visible page at a time.

---

## Implementation

Both steps were completed. Key names in Bubbletea v2 are `"pgup"` / `"pgdown"` as expected.

### Step 1 — `pgup` / `pgdown` cases in `TrackerModel.Update()`

File: `ui/tracker/tracker.go`

The two cases delegate to the existing `m.nav` navigation abstraction, keeping them consistent with `home`/`end`/`up`/`down`:

```go
case "pgup":
    m.nav.Move(0, -m.nav.ViewportHeight())
    m.clearNibbleBuffer()
case "pgdown":
    m.nav.Move(0, m.nav.ViewportHeight())
    m.clearNibbleBuffer()
```

### Step 2 — Footer help text

File: `ui/tracker/screen.go`

Added `PgUp / PgDn: Jump by viewport height` to the help entries alongside the other navigation hints.

---

## Affected Files

| File                    | Symbol                   | Change                                                           |
| ----------------------- | ------------------------ | ---------------------------------------------------------------- |
| `ui/tracker/tracker.go` | `TrackerModel.Update()`  | Added `"pgup"` and `"pgdown"` cases delegating to `m.nav.Move()` |
| `ui/tracker/screen.go`  | `TrackerScreen.Footer()` | Added PgUp/PgDn hint to the help entries                         |
