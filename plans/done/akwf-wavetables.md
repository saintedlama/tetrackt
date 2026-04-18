# AKWF Wavetable Integration

**Status: Done**

Integrate the complete [AKWF-FREE](https://github.com/KristofferKarlAxelEkstrand/AKWF-FREE)
single-cycle wavetable library into tetrackt so users can browse and select all ~4,800 waveforms
from the oscillator panel.

## Background

AKWF-FREE is a CC0-licensed collection of single-cycle WAV files (600 samples, 44 100 Hz, 16-bit
mono). The repo also ships a JavaScript float-array mirror (`AKWF-js/`) that is trivial to
convert to Go slices. The tetrackt oscillator already interpolates over arbitrary-length
`[]float64` at playback time, so no engine changes are needed.

## Problems to solve

### 1. Wavetable identity (fragile length-based matching)

`ui/synth/oscillator.go:72–78` identifies the active preset by comparing `len(oscillator.Wavetable)`
to `len(p.data)`. AKWF waveforms are 600 samples while builtins are 256 — they won't collide
today — but the approach is brittle. A name-based identity is needed.

### 2. Persistence bloat

`audio.Oscillator.Wavetable []float64` is serialised inline. A 600-sample waveform is 600 floats
in every saved patch. Storing a wavetable ID (string) and resolving it to `[]float64` at load
time is much leaner and makes saves human-readable.

### 3. UI: flat list doesn't scale to ~4,000 entries

The current `builtinWavetables` slice is navigated with plain left/right keys. That works for 10
entries; it is unusable for thousands. A category-aware browser is required.

### 4. Embedding vs. external files

The WAV files (or their JS float equivalents) must be available at runtime. Embedding a useful
curated subset with `//go:embed` is the right approach — embedding all ~4,000 would add ~14 MB
to the binary, which is acceptable but optional subsets (by category) may be preferred.

## Proposed solution

### Step 1 — Add a wavetable registry

Create `audio/wavetable_registry.go`:

```go
type WavetableEntry struct {
    ID       string    // e.g. "builtin:Organ" or "akwf:AKWF_flute/AKWF_flute_0001"
    Name     string    // display name
    Category string    // e.g. "Built-in", "Flute", "E-Guitar", …
    Data     []float64
}

var Registry []WavetableEntry        // populated at init()
func LookupByID(id string) ([]float64, bool)
```

Builtin waveforms get IDs like `"builtin:Organ"`. AKWF waveforms get IDs like
`"akwf:AKWF_flute/AKWF_flute_0001"`.

### Step 2 — Embed AKWF waveforms

Add the `AKWF-js/` folder from the AKWF-FREE repo as vendored JSON files under `audio/akwf/`.
Each file (e.g. `AKWF_flute.json`) is a JSON object mapping waveform names to pre-normalized
`[]float64` arrays of 600 samples — no WAV decoding, no normalization step needed.

Use `//go:embed *.json` to embed all category files. A loader (`audio/akwf/loader.go`) parses
each file at `init()` using `encoding/json` and registers every waveform in the registry.
The `manifest.json` file lists all category names and drives the registration order.

### Step 3 — Add `WavetableID` to `audio.Oscillator`

```go
type Oscillator struct {
    // existing fields …
    WavetableID string    // registry ID; takes precedence over inline Wavetable if non-empty
    Wavetable   []float64 // legacy inline data; kept for backward compat
}
```

`NewOscillator` resolves `WavetableID → []float64` via `LookupByID` at construction time.

### Step 4 — Update persistence

In `persistence/module.go`, `SavedOscillator` stores `WavetableID string` and omits the
`[]float64`. On load, if `WavetableID` is set the data is resolved from the registry; if only
inline `Wavetable []float64` is present (old saves) it is used as-is for backward compat.

### Step 5 — Wavetable browser in the oscillator panel

Replace the current flat left/right cycling with a two-level browser:

- First level: category list (`Built-in`, `Flute`, `E-Guitar`, `Organ`, …) — navigate with
  `[`/`]` or `Shift+←`/`Shift+→`
- Second level: waveform within category — navigate with `←`/`→` (existing keys)
- Display: `Category (n/N) | Name` in the wavetable row

The `wavetableIdx` field in `OscillatorModel` is replaced by `wtCategory int` and `wtIndex int`.
The matching logic in `NewOscillatorModel` is updated to match by `WavetableID` instead of
length.

### Step 6 — Update patch bank presets

Revisit `ui/synth/presets.go` (or wherever synth presets are defined) and update any
wavetable-based presets to use the new ID scheme. Consider adding new presets that demonstrate
AKWF waveforms (e.g. a flute patch using `akwf:AKWF_flute/AKWF_flute_0001`).

## File changes summary

| File | Change |
|---|---|
| `audio/wavetable_registry.go` | New — registry, `LookupByID` |
| `audio/akwf/loader.go` | New — JSON loader + `init()` registration |
| `audio/akwf/*.json`    | New — vendored AKWF-js JSON files (one per category) |
| `audio/wavetables.go` | Register builtins into registry at `init()` |
| `audio/oscilator.go` | Add `WavetableID` field; resolve on construction |
| `persistence/module.go` | Store `WavetableID`; backward-compat inline fallback |
| `ui/synth/oscillator.go` | Two-level browser; ID-based matching |
| `ui/synth/presets.go` | Update/add AKWF-based presets |

## Out of scope

- The `AKWF-js/` JSON format is used — values are pre-normalized `float64` arrays, so no WAV
  parsing or normalization is required. The files are small (~600 floats × waveform count per
  JSON file) and trivially decoded with `encoding/json`.
- No streaming or lazy loading — all categories are decoded at startup (~23 MB total, acceptable
  for a desktop app).

## Open questions

- Should the Git submodule be vendored or fetched at build time? Vendoring avoids a network
  dependency; submodule is easier to update.
