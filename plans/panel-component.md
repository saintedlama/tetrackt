# Panel Component

As lipgloss 2 features a compositor that allows for more complex layouts, we can consider implementing a panel component that can be used to create UI elements for Oscillators, Filters, and other synthesizer modules.

## Title-in-Border Technique

The compositor allows layering arbitrary content on top of rendered strings. To embed a title in the top border:

1. Render the panel content inside a bordered style → produces a multi-line string where row 0 is the top border (e.g. `╭──────────────────╮`).
2. Render the title as a separate styled string with a background matching `ColorSurface` so it "erases" the border characters behind it: `" Oscillator 1 "`.
3. Composite the title layer at `X=2, Y=0` (top-left border row, 2 cells in to clear the corner rune) with `Z=1` on top of the border layer at `Z=0`.
4. Use `lipgloss.NewCanvas(width, height).Compose(lipgloss.NewCompositor(bgLayer, titleLayer)).Render()`.

## API

```go
// RenderPanel renders content inside a rounded border with an optional title
// embedded in the top border line.
// titleColor is the foreground color for the title text.
// active controls whether the active or inactive border color is used.
func RenderPanel(title string, titleColor lipgloss.Color, content string, active bool) string
```

## Implementation Plan

### Step 1 — Create `ui/panel.go`

```go
package ui

import "charm.land/lipgloss/v2"

func RenderPanel(title string, titleColor lipgloss.Color, content string, active bool) string {
    borderColor := ColorBorder
    if active {
        borderColor = ColorBorderActive
    }

    borderStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(borderColor).
        Padding(0, 2)

    bordered := borderStyle.Render(content)

    if title == "" {
        return bordered
    }

    titleStyle := lipgloss.NewStyle().
        Foreground(titleColor).
        Background(ColorSurface)
    titleRendered := titleStyle.Render(" " + title + " ")

    w := lipgloss.Width(bordered)
    h := lipgloss.Height(bordered)

    bgLayer := lipgloss.NewLayer(bordered).Z(0)
    titleLayer := lipgloss.NewLayer(titleRendered).X(2).Y(0).Z(1)

    return lipgloss.NewCanvas(w, h).
        Compose(lipgloss.NewCompositor(bgLayer, titleLayer)).
        Render()
}
```

### Step 2 — Remove `PanelBorderStyle` / `ActivePanelBorderStyle` from `ui/styles.go`

The shared border styles in `styles.go` are superseded by `RenderPanel`. Remove `PanelBorderStyle` and `ActivePanelBorderStyle` from `styles.go`.

### Step 3 — Migrate `main.go`

Replace every call-site in `synthView()` and `View()` that applies `panelBorderStyle.Render(...)` or `activePanelBorderStyle.Render(...)` with `ui.RenderPanel(title, titleColor, content, active)`.

Concrete mappings:

| Current                                     | `RenderPanel` call                                                                                         |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `oscillator1Border.Render(oscillatorView1)` | `ui.RenderPanel("Oscillator 1", ui.ColorAccentOscillator, oscillatorView1, m.mode == Oscillator1EditMode)` |
| `envelope1Border.Render(envelopeView1)`     | `ui.RenderPanel("Envelope 1", ui.ColorAccentEnvelope, envelopeView1, m.mode == Envelope1EditMode)`         |
| `oscillator2Border.Render(oscillatorView2)` | `ui.RenderPanel("Oscillator 2", ui.ColorAccentOscillator, oscillatorView2, m.mode == Oscillator2EditMode)` |
| `envelope2Border.Render(envelopeView2)`     | `ui.RenderPanel("Envelope 2", ui.ColorAccentEnvelope, envelopeView2, m.mode == Envelope2EditMode)`         |
| `mixerBorder.Render(m.mixer.View())`        | `ui.RenderPanel("Mixer", ui.ColorAccentModulation, m.mixer.View(), m.mode == MixerEditMode)`               |
| `instrumentBorder.Render(instrumentView)`   | `ui.RenderPanel("Instruments", ui.ColorAccentInstrument, instrumentView, m.mode == InstrumentMode)`        |
| `trackerBorder.Render(trackerView)`         | `ui.RenderPanel("Tracker", ui.ColorAccentPrimary, trackerView, m.mode == TrackMode)`                       |

Also remove the `oscillator1Border`, `envelope1Border`, … local variables and the `switch` block that selects between active/inactive styles, as `RenderPanel` handles that internally.

### Step 4 — Update `ui/dialog.go`

Replace `dialogBorderStyle.Render(...)` with `RenderPanel("", ...)` (no title) — or keep the existing inline style for the dialog since dialog borders use `ColorAccentWarning` and are a different context. Leave `dialog.go` unchanged for now; the dialog border is semantically different.

### Step 5 — Build and test

```bash
go build ./...
go test ./...
```
