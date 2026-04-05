# Refactor Dialog Handling

> Status: **Done**

The dialog handling is currently implemented per instance, leading to code duplication and inconsistent behavior across different dialogs. To improve maintainability and user experience, we should refactor the dialog handling into a more centralized and reusable system.

pcloud-cli implements a generic dialog component here: <https://github.com/saintedlama/pcloud-cli/blob/master/internal/tui/dialog.go> and an overlay component to render dialogs on top of the main UI here: <https://github.com/saintedlama/pcloud-cli/blob/master/internal/tui/overlay.go>. We can take inspiration from this implementation to create a similar system for TeTrackT, allowing us to manage dialogs more effectively and provide a consistent user experience across the application.

## Decisions

- Use the **lipgloss v2 Canvas/Layer compositor** for overlay rendering (true compositing, supports dim background).
- Adopt the **`dialogModel` wrapper pattern**: when a dialog opens, the program's root model is replaced with a `dialogModel` that forwards input to the dialog content and renders it on top of the background.
- Dialogs signal closure via a **generic `CloseDialogMsg`** carrying an optional result payload. The `dialogModel` unwraps itself and forwards the payload to the background model.
- The **EnvelopeModel preset picker** (`ShowModal` / `PresetModel`) is extracted into a standalone dialog, consistent with the new system.
- The **background is visually dimmed** while a dialog is open.

## Implementation Plan

### Step 1 — Create `ui/overlay.go`

Port pcloud-cli's overlay helper using lipgloss v2 APIs.

```go
type OverlayOption func(*overlayOptions)
func WithDim() OverlayOption

// OverlayCenter places fg centered on bg using lipgloss Canvas/Layer compositor.
func OverlayCenter(width, height int, fg, bg string, opts ...OverlayOption) string
```

- `dimContent(bg)` renders every line through a dim foreground style (`lipgloss.Color("240")`).
- Uses `lipgloss.NewLayer`, `lipgloss.NewCanvas`, `lipgloss.NewCompositor` (lipgloss v2).

### Step 2 — Create `ui/dialog.go`

A generic `dialogModel` that acts as the Bubble Tea root while a dialog is visible.

```go
type CloseDialogMsg struct{ Payload tea.Msg }

type dialogModel struct {
    dialog tea.Model
    main   tea.Model
    width  int
    height int
}

func NewDialogModel(dialog, main tea.Model, width, height int) dialogModel
```

**`Update` routing:**

- `tea.WindowSizeMsg` → propagate to both `dialog` and `main`, update `width`/`height`.
- `CloseDialogMsg` with non-nil `Payload` → return `m.main.Update(Payload)` (main processes the result).
- `CloseDialogMsg` with nil `Payload` → return `m.main, nil` (cancelled, no result).
- All other messages → forward to `m.dialog`.

**`View() tea.View`:**

```go
v := tea.NewView(OverlayCenter(m.width, m.height, m.dialog.View().Content(), m.main.View().Content(), WithDim()))
v.AltScreen = true
return v
```

### Step 3 — Refactor `FileDialogModel`

Remove all show/hide lifecycle management; `FileDialogModel` is always "shown" when instantiated.

- **Remove**: `Mode ModeHidden`, `IsVisible()`, `Show()`, `Hide()`, the `borderStyle lipgloss.Style` constructor parameter.
- **Constructor**: `NewFileDialog(mode FileDialogMode, prefill string) *FileDialogModel`. Border style becomes a package-level var.
- **On Enter**: return `CloseDialogMsg{Payload: FileDialogConfirmed{Filename: ...}}`.
- **On Esc**: return `CloseDialogMsg{}` (nil payload = cancelled).
- **Remove** `FileDialogCancelled` message type (cancellation is implicit in nil payload).
- Keep `FileDialogConfirmed` as the result payload type.
- Update `filedialog_test.go` to match the new constructor and message shapes.

### Step 4 — Refactor `EnvelopeModel` preset picker

Extract `ShowModal` / `PresetModel` out of `EnvelopeModel` into a standalone dialog.

- **Remove** `ShowModal bool`, `PresetModel PresetModel`, and all modal-branching code from `EnvelopeModel.Update` and `EnvelopeModel.View`.
- Create `EnvelopePresetDialog` implementing `tea.Model`:
  - Wraps current `PresetModel` rendering and key handling.
  - **On Enter**: `CloseDialogMsg{Payload: EnvelopePresetSelected{Envelope: audio.Envelope{...}}}`.
  - **On Esc**: `CloseDialogMsg{}`.
- Define `type EnvelopePresetSelected struct { Envelope audio.Envelope }`.
- The `"."` key in `EnvelopeModel.Update` no longer toggles `ShowModal`; it instead returns a command that signals `main` to open the preset dialog (e.g. an `OpenPresetDialogMsg` carrying the current envelope, or handled inline in `main.Update`).

### Step 5 — Update `main.go`

Remove all manual dialog orchestration and replace with `dialogModel` wrapping.

**Model struct**: remove `fileDialog *ui.FileDialogModel`.

**`Update`**:

- Remove the `m.fileDialog.IsVisible()` early-return block.
- `"s"` key → `return ui.NewDialogModel(ui.NewFileDialog(ui.ModeSave, prefill), m, m.width, m.height), nil`
- `"l"` key → `return ui.NewDialogModel(ui.NewFileDialog(ui.ModeLoad, ""), m, m.width, m.height), nil`
- `"."` in envelope edit modes → `return ui.NewDialogModel(ui.NewEnvelopePresetDialog(), m, m.width, m.height), nil`
- Handle `ui.FileDialogConfirmed` (arrives as the forwarded payload from `CloseDialogMsg`) — same save/load logic as today.
- Handle `ui.EnvelopePresetSelected` — apply the returned envelope to the active envelope model.
- Remove `ui.FileDialogCancelled` case.

**`View()`**: remove the three `lipgloss.Place` override blocks (file dialog, envelope1 modal, envelope2 modal). The `dialogModel` handles rendering when active; `main.View()` only needs to render the normal UI.

### Step 6 — Clean up

- `go mod tidy` to confirm no stray imports.
- Run `go test ./...` — all `ui` and `persistence` tests must pass.
- Ensure `filedialog_test.go` covers the new constructor and `CloseDialogMsg` return shape.
