# Tracker Synth Screens

> Status: **In Progress**

To allow for more flexibility in the UI, including display of additional information such as the current BPM (future), we consider implementing separate screens for the tracker and the synthesizer. This would allow us to display more information and controls without cluttering the main tracker interface.

Basically splitting synthesizer and pattern editor into separate screens.

Synth components:

- Oscillator 1
- Envelope 1
- Oscillator 2
- Envelope 2
- Mixer
- Preset picker (InstrumentView)

Tracker components:

- Pattern editor

Switching between screens is done by "t" key, which toggles between the tracker and synth views. The state of each screen is preserved when switching.

## Implementation Plan

### Overview

Replace the current single-layout rendering (synth panels stacked above tracker) with a two-screen model backed by a `Screen` interface and a `[]Screen` slice. The active screen is selected by an integer index. `View()` and `Update()` in `main.go` dispatch to `screens[activeScreen]` — no branching on screen type. Each screen owns its own internal navigation state.

### 1. Define a `Screen` interface in `ui/`

```go
// ui/screen.go
type Screen interface {
    Init() tea.Cmd
    Update(tea.Msg) (Screen, tea.Cmd)
    View() string
    Footer() string   // screen-specific help text shown in the footer bar
}
```

This mirrors the existing `Component` interface and keeps screen logic self-contained.

### 2. Implement `SynthScreen`

`SynthScreen` wraps the six existing `Panel` components and owns the active-panel index (currently carried as `InputMode` in `main.go`).

```go
// ui/synthscreen.go
type SynthScreen struct {
    panels      []Panel
    activePanel int   // 0–5, replaces modes Oscillator1EditMode…InstrumentMode
}
```

- `Update()` routes keyboard events to `panels[activePanel]`; handles `tab`/`shift+tab` to move between panels, and `o`/`e`/`m`/`i` shortcuts.
- `View()` renders all panels in the existing horizontal layout, expanded to fill the full terminal height.
- `Footer()` returns synth-specific hints (`tab` to switch panels, `←→` to edit, `t` = Tracker).
- Still emits `OscillatorUpdated`, `EnvelopeUpdated`, `MixerUpdated`, `InstrumentApplied` messages so the audio engine and tracker sync is unchanged.

### 3. Implement `TrackerScreen`

`TrackerScreen` wraps the existing `TrackerModel`:

```go
// ui/trackerscreen.go
type TrackerScreen struct {
    tracker *TrackerModel
}
```

- `Update()` forwards all keyboard events to `tracker.Update()`.
- `View()` renders the tracker grid at full terminal height.
- `Footer()` returns tracker-specific hints (note entry, cursor keys, playback, `t` = Synth).
- Still emits `TrackChanged` messages so synth panels stay in sync.

### 4. Simplify `main.go` model

Remove `synthPanels`, `tracker`, `mode`, and `lastSynthMode` fields. Replace with:

```go
screens      []ui.Screen
activeScreen int
```

Initialize as:
```go
m.screens = []ui.Screen{
    ui.NewSynthScreen(panels),
    ui.NewTrackerScreen(tracker),
}
m.activeScreen = 0  // start on SynthScreen
```

### 5. Update `t` key handler

```go
case "t":
    m.activeScreen = (m.activeScreen + 1) % len(m.screens)
```

A simple index increment replaces the old if/else toggle. All other mode-switching keys (`o`, `e`, `tab`, etc.) are forwarded into the active screen's own `Update()` — `main.go` no longer interprets them.

### 6. Update `Update()` routing

```go
var cmd tea.Cmd
m.screens[m.activeScreen], cmd = m.screens[m.activeScreen].Update(msg)
return m, cmd
```

Tick and playback messages are broadcast to all screens so audio stays in sync regardless of which screen is visible.

### 7. Update `View()` and footer

```go
func (m model) View() string {
    header := m.renderHeader()
    body   := m.screens[m.activeScreen].View()
    footer := m.screens[m.activeScreen].Footer()
    return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
```

No per-screen branching.

### 8. File changes

| File                   | Change                                                                                    |
| ---------------------- | ----------------------------------------------------------------------------------------- |
| `ui/screen.go`         | New — `Screen` interface                                                                  |
| `ui/synthscreen.go`    | New — `SynthScreen` wrapping existing panels; owns panel navigation                      |
| `ui/trackerscreen.go`  | New — `TrackerScreen` wrapping existing `TrackerModel`                                    |
| `main.go`              | Replace `synthPanels`/`tracker`/`mode` with `screens`/`activeScreen`; simplify key logic |

No new files required. All synth panel and tracker component code remains unchanged.
