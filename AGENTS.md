# Tetrackt — Agent Reference

## Commands

```sh
go test ./...      # run all tests
go build .         # build binary
go vet ./...       # lint
go run .           # run locally
```

Or via Make: `make test`, `make build`, `make lint`, `make run`.

## Stack

- **Go 1.25**
- **bubbletea/v2** — TUI framework (Elm-style: Model/Update/View)
- **lipgloss/v2** — terminal styling
- **beep/v2** — audio streaming and synthesis
- **go-yaml** — song file serialization

## Layout

```text
audio/          synthesis engine (oscillators, envelopes, LFO, filter, synth)
ui/             TUI components (screens, panels, dialogs, tracker)
  common/       shared styles and colours
  stateless/    stateless rendering helpers
persistence/    save/load YAML song files
main.go         app entry point, message routing, audio playback
plans/          design docs (not code)
```

## Key types

| Type                      | Package       | Purpose                                                                                                |
| ------------------------- | ------------- | ------------------------------------------------------------------------------------------------------ |
| `audio.Synth`             | `audio`       | Full synthesis engine; call `.Streamer(sr, note, dur)` to render audio                                 |
| `audio.LFO`               | `audio`       | Modulation source; `Dest` field selects target (`ModPitch`, `ModVolume`, `ModCutoff`, `ModPulseWidth`) |
| `ui.TrackerModel`         | `ui`          | Pattern editor state; each `Track` owns a `*audio.Synth`                                               |
| `ui.SynthScreen`          | `ui`          | Panel editor; `GetSynth()` returns the current `*audio.Synth`                                          |
| `ui.SynthPreset`          | `ui`          | Named synth snapshot; applied via `SynthPresetApplied` message                                         |
| `persistence.SavedModule` | `persistence` | JSON wire format; convert with `TracksToModule` / `ModuleToTracks`                                     |

## Conventions

- Message types live next to the component that emits them.
- `main.go` owns speaker init, global key bindings, and message routing between screens.
- `audio.Synth` fields are exported — set them directly, no setters needed.
- LFO depth 0 means disabled; `GetSynth()` in `SynthScreen` passes `nil` for zero-depth LFOs to `NewSynth`.
- Tests live alongside source (`*_test.go`). No external test framework.

## Code Style

- Never use `// ── Block` comment. In case a block comment would be needed, extract a function or file instead.
