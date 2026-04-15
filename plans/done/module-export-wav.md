# Plan: Export Module to WAV

Status: Done

## Feature Description

Add an offline export path that renders a saved module to a `.wav` file without real-time playback. This should produce deterministic output and allow users to bounce songs for sharing/mastering.

---

## Goals

1. Export any valid module file to a stereo WAV.
2. Reuse existing tracker/audio behavior so playback and export sound consistent.
3. Avoid using live speaker playback APIs during export.
4. Integrate export into existing save workflow with minimal UI friction.

## Non-Goals (Phase 1)

1. MP3/FLAC export.
2. Stem export (per-track files).
3. Real-time progress UI in TUI.
4. Normalization/limiting/mastering DSP.

---

## Current State

1. Live playback is driven by `main.go -> player.Player.Tick(...)` and ultimately uses `speaker.Play(...)`.
2. Global volume is already available in model state and wrapped via `audio.NewVolume(...)` for playback.
3. Module load/save already works through `persistence`.
4. Tracker row volume exists in data model; current playback path does not yet apply it directly (only effect-driven runtime volume changes are applied).

---

## Proposed Architecture

Unify live playback and WAV export behind a single render engine with pluggable output sinks:

1. New core package (for example `render`) owns row/sub-tick sequencing, effect updates, and patch lifecycle.
2. Existing `player` becomes a thin runtime adapter that uses the core engine and writes to a speaker sink.
3. WAV export uses the same core engine and writes to a WAV sink/encoder.
4. This removes logic duplication and guarantees better parity between what users hear live and what is exported.

### Core Concepts

1. `RenderEngine`: deterministic tracker sequencer state machine.
2. `RenderSink`: output abstraction for where mixed audio frames go.
3. Two sink implementations in Phase 1:
4. `SpeakerSink` for real-time playback.
5. `WAVSink` for file export.

### API Sketch

```go
type WavExportOptions struct {
    SampleRate audio.SampleRate // default 44100
    GlobalVolume float64        // default 1.0
    LoopCount int               // default 1, exports one pattern pass
}

type RenderConfig struct {
    SampleRate audio.SampleRate
    GlobalVolume float64
    LoopCount int
}

type RenderSink interface {
    Begin(sampleRate audio.SampleRate) error
    Write(samples [][2]float64) error
    End() error
}

func NewRenderEngine(m *tracker.TrackerModel, cfg RenderConfig) *RenderEngine
func (e *RenderEngine) Run(sink RenderSink) error

func ExportModuleToWAV(modulePath, wavPath string, opts WavExportOptions) error
```

---

## Rendering Strategy

Use one deterministic sub-tick simulation for both live and export:

1. Move sequencing/effect/patch state from `player.Player` internals into `RenderEngine`.
2. Iterate rows and sub-ticks using tracker BPM + Speed rules.
3. For each row/track note event, create patch streamers via synth APIs.
4. Apply per-sub-tick effects in same order as live engine.
5. Mix active patches into stereo PCM frames.
6. Send mixed frames to the active sink:
7. speaker sink for real-time playback,
8. WAV sink for file export.

Key requirement: engine logic remains sink-agnostic and free from `speaker` coupling.

---

## Save Dialog UX Plan

Invoke WAV export from the existing save dialog flow instead of command-line options.

### User Flow

1. User opens save dialog as usual.
2. User chooses export target by filename extension:
3. `.json` (or existing module extension): normal module save behavior.
4. If `.wav` is detected, dialog reveals additional WAV export options inline.
5. User confirms save/export.
6. `.wav`: render current tracker state to WAV at the chosen path using selected options.
7. On success, close dialog and clear dirty state consistently with save behavior.
8. On failure, keep dialog/context and show an actionable error message.

### Conditional WAV Options (shown only for `.wav` filename)

1. Sample rate: `22050`, `44100` (default), `48000`.
2. Loop count: integer `1..64` (default `1`).
3. Export gain: `0.00..1.50` (default `1.00`) applied as final export scalar.

Notes:

1. Options are hidden for non-`.wav` saves and do not affect module save.
2. Options should persist for the current session once changed (optional for Phase 1).
3. If a value is invalid, show inline validation and block confirmation.

### Integration Points

1. `ui/filedialog.go`: extend save mode UI to include conditional WAV options panel.
2. `main.go` in `ui.FileDialogConfirmed` handling:
3. branch by extension and call module save vs WAV export.
4. Keep load/open paths unchanged.
5. Add fields to file-dialog confirmed payload for export options (used only for `.wav`).

### Filename Rules

1. If user enters name without extension in save mode, keep existing default module behavior.
2. WAV export is only triggered when extension is explicitly `.wav`.
3. Option panel visibility updates immediately as filename extension changes.
4. If needed later, add a separate "Export WAV" mode in dialog; Phase 1 remains extension-driven.

---

## Implementation Steps

### Step 1: Introduce Unified Engine + Sink Abstraction

1. Create `RenderEngine` and `RenderSink` abstractions.
2. Move sequencing/effect code out of `player` into engine.
3. Keep behavior stable by preserving current update order and defaults.

### Step 2: Adapt Live Player to Engine

1. Replace direct `player` sequencing with engine invocation.
2. Implement `SpeakerSink` and wire it into current play/tick path.
3. Verify no regressions in interactive playback and preview behavior.

### Step 3: Implement WAV Sink + Export Runner

1. Implement `WAVSink` using `beep/wav` encoding.
2. Run unified engine with `WAVSink` for one-or-more loops.
3. Write output atomically (temp + rename if needed).

### Step 4: Save Dialog Wiring

1. Add extension-based branching in `main.go` save-confirm handler.
2. Add conditional WAV options UI/state in `ui/filedialog.go` for `.wav` targets.
3. Route `.wav` targets to the offline exporter with current tracker model + global volume + selected WAV options.
4. Include options in save-confirmed message payload (ignored for non-`.wav`).
5. Preserve existing module-save behavior for non-`.wav` paths.
6. Ensure dirty-state updates remain consistent after successful save/export.

### Step 5: Hardening

1. Validate sample rate and output path.
2. Validate loop count and gain bounds from dialog input.
3. Guard against empty modules and invalid tracker dimensions.
4. Add parity checks between speaker and WAV rendering for the same fixture.
5. Add friendly error messages.

---

## Testing Plan

1. Unit tests for duration math (rows x ticks x BPM).
2. Engine tests for deterministic output length and row/sub-tick transitions.
3. Golden/snapshot tests for small fixture modules (hash/byte-size/sanity bounds).
4. Parity tests: same fixture through speaker-path engine and wav-path engine yields equivalent frame stream (within tolerance).
5. Regression tests ensuring live path still works.
6. Save-dialog integration tests covering extension-based routing and conditional option visibility.
7. Save-dialog validation tests for invalid loop count/gain/sample rate.
8. Error-path tests for invalid/unwritable `.wav` destination.
9. Option propagation tests: selected dialog options reach exporter correctly.

Suggested commands:

1. `go test ./...`
2. `go test ./... -run Export`

---

## Risks and Mitigations

1. Drift between live and offline sound due to duplicated logic.
2. Mitigation: shared unified engine + sink abstraction and parity tests.

3. Memory use for long songs when buffering all frames.
4. Mitigation: support chunked streaming writer (phase 2), begin with bounded fixtures.

5. Clicks at row transitions if patch lifecycle differs from live mode.
6. Mitigation: reuse existing patch NoteOn/NoteOff order and add transition tests.

7. UI complexity creep in the save dialog.
8. Mitigation: keep options compact and only visible for `.wav` filenames.

9. Refactor risk while migrating `player` to the engine.
10. Mitigation: do migration in small steps and keep compatibility tests green at each step.

---

## Open Decisions

1. Should export include only one pattern loop or configurable loop count?
2. Should row volume be applied at note trigger time for export parity with intended tracker UX?
3. Which exact sample-rate presets should be available in Phase 1?
4. Should export gain multiply or replace current global volume in export mode?
5. Should preview-note playback also be migrated to the unified engine in Phase 1 or Phase 2?

---

## Acceptance Criteria

1. Saving with a `.wav` filename from the save dialog creates a valid playable WAV file.
2. Saving with a non-`.wav` filename keeps current module-save behavior unchanged.
3. When `.wav` extension is present, WAV options are shown; when absent, they are hidden.
4. Invalid WAV option values are blocked with inline validation feedback.
5. Selected WAV options are applied to export output.
6. Output duration matches expected rows/BPM/speed timing.
7. Live playback and WAV export both use the shared render engine.
8. Existing live playback behavior remains stable and tests continue to pass.
9. Plan status can be moved to Done once implementation and tests land.
