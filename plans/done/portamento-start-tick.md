# Portamento StartTick

## Status: Done

## Goal

Extend `PortamentoEffect` with a `StartTick` field so portamento glides can be
delayed within a row's sub-tick range. This enables effects like:

- Fast glide at the start then stable pitch
- Delayed glide (hold prev pitch, then sweep into the note)
- Delayed snap (hold prev pitch, then jump)

## Design

### `PortamentoEffect` (audio/effects.go)

```go
type PortamentoEffect struct {
    StartTick int // sub-tick at which glide begins; 0 = immediate
    Ticks     int // number of sub-ticks for the glide; 0 = snap (no glide)
}
```

| StartTick | Ticks | Behaviour |
|-----------|-------|-----------|
| 0         | 0     | Snap immediately (no glide) — unchanged |
| 0         | N     | Glide immediately over N sub-ticks — unchanged |
| S         | 0     | Hold prevFreq, snap to noteFreq at tick S |
| S         | N     | Hold prevFreq, glide over N ticks starting at tick S |

### Tracker encoding (render/model.go)

`Row.Portamento` stays a single packed int — no UI or persistence changes:

```
high nibble = StartTick (0–15)
low  nibble = Ticks     (0–15)
```

Backwards compatible: existing values `0x0N` (immediate glide, ≤15 ticks) decode
identically since hi nibble = 0.

### Files changed

| File | Change |
|------|--------|
| `audio/effects.go` | Add `StartTick int`; update `gliding` condition; update `applySubtickEffects` |
| `audio/effects_test.go` | Tests for delayed glide, delayed snap, partial glide |
| `render/model.go` | Decode packed byte: `StartTick = hi nibble`, `Ticks = lo nibble` |
