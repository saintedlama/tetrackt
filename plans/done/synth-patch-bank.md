# Synth Patch Bank

> Status: Done

Now we introduce a .tetrackt file in the users home! The .tetrackt file can save synths a user has arranged - the cool patches, you know!

The .tetrackt file will be a JSON file with a list of synth patches. Each patch will have a name and the settings for each parameter of the synth. The user can save the current synth as a patch, and load patches from the .tetrackt file.

The user uses the "p" key in the synth screen the preset dialog opens. The user has the option to save the current synth as a patch to the patch bank. They are prompted to enter a name for the patch and select a category. The patch is then saved to the .tetrackt file.

The saved patches are displayed in the synth preset dialog implemented in "presets.go". In the preset dialog, the user can filter for custom patches and load them into the current synth.

The user can also manage their patches in a new "Patch Bank" screen. In the Patch Bank, they can see all their saved patches, edit them, and delete them. The Patch Bank is accessible from the main menu.

## Design Decisions

- **Key**: `B` opens the Patch Bank (play/pause stays on `p`).
- **Saving**: Opening the bank shows a "Save current patch" option at the top of the list; selecting it prompts for a name and category, then sets `Custom = true`.
- **Filtering**: `Custom` flag drives a "My Patches" filter; category is a second filter layer on top.
- **Patch Bank screen**: Full screen with browse, save, rename (`R`), and delete (`D`).
- **Patch shape**: `Name` (string) + `Category` (string, optional) + `Custom` (bool).

---

## Detailed Implementation Plan

### Step 0 — Rename "presets" → "patchbank" across the codebase

Before adding any new functionality, rename all existing preset-related identifiers and files to use consistent "patch bank" / "patchbank" terminology.

**File renames** (git mv):

| Old                          | New                            |
| ---------------------------- | ------------------------------ |
| `ui/synth/presets.go`        | `ui/synth/patchbank.go`        |
| `ui/synth/presets_dialog.go` | `ui/synth/patchbank_dialog.go` |

**Type / function renames** (update all references in `ui/synth/`, `main.go`):

| Old name                          | New name                  |
| --------------------------------- | ------------------------- |
| `SynthPreset`                     | `SynthPatch`              |
| `SynthPresetView`                 | `SynthPatchBankView`      |
| `NewSynthPresetView`              | `NewSynthPatchBankView`   |
| `SynthPresetsDialog`              | `SynthPatchBankDialog`    |
| `NewSynthPresetsDialog`           | `NewSynthPatchBankDialog` |
| `OpenSynthPresetDialogMsg`        | `OpenPatchBankMsg`        |
| `PlaySynthPresetNoteMsg`          | `PlayPatchNoteMsg`        |
| `builtinPresets()`                | `builtinPatches()`        |
| `SynthPresetApplied` (if present) | `SynthPatchApplied`       |

**String literals** in `View()` and footer help text:

- `"Synth Presets"` → `"Patch Bank"`
- `"preset"` / `"presets"` in comments → `"patch"` / `"patches"`

After this step: `go build ./...` and `go test ./...` must pass with no other behaviour changes.

---

### Step 1 — `persistence/patchbank.go` (new file)

New types and functions for the `~/.tetrackt` user file.

```go
// PatchBank is the root JSON structure for ~/.tetrackt.
// The Version field allows future format migrations.
type PatchBank struct {
    Version      int          `json:"version"`
    SynthPatches []SavedPatch `json:"synthPatches"`
}

// SavedPatch is one entry in the patch bank.
type SavedPatch struct {
    Name     string     `json:"name"`
    Category string     `json:"category,omitempty"`
    Custom   bool       `json:"custom"`
    Synth    SavedSynth `json:"synth"`
}
```

Functions:

- `LoadPatchBank() (*PatchBank, error)` — reads `~/.tetrackt`; returns empty bank (version=1) if the file does not exist. Errors on malformed JSON.
- `(b *PatchBank) Save() error` — marshals to JSON and writes atomically (write temp file, rename).
- `patchBankPath() (string, error)` — returns `filepath.Join(os.UserHomeDir(), ".tetrackt")`.
- `ToSynthPresets(patches []SavedPatch) []SynthPreset` — converts `[]SavedPatch` → `[]ui/synth.SynthPreset` (needs to live in `persistence` or be a free function bridging packages; see note below).

> **Note on layering**: `persistence` must not import `ui/synth`. Conversion helpers will live in `main.go` (or a thin bridge in `persistence` that only imports `audio`). `SavedPatch.Synth` reuses the existing `SavedSynth` type.

Tests (`persistence/patchbank_test.go`):

- Round-trip: save and reload a patch with all fields set.
- Missing file returns empty bank (no error).
- Corrupt JSON returns an error.
- Atomic write: temp file is cleaned up.

---

### Step 2 — `ui/synth/presets.go` — extend `SynthPreset` and `SynthPresetView`

**`SynthPreset` changes**

Add `Custom bool` field:

```go
type SynthPreset struct {
    Name     string
    Category string
    Custom   bool       // true = user-saved patch
    Synth    *audio.Synth
}
```

**`SynthPresetView` changes**

- `SetUserPatches(patches []SynthPreset)` — replaces the user-patch subset in `Presets`, rebuilds category list. Called at startup and after every save/delete.
- Category building: `buildCategories` adds a `"My Patches"` pseudo-category for presets where `Custom=true`, independent of the regular `Category` string.
- Two-layer filtering:
  1. If current top-level filter is `"My Patches"`, restrict to `Custom=true` entries.
  2. Category filter then applies within that restricted set.

---

### Step 3 — `ui/synth/patchbank_dialog.go` (new file, replaces `presets_dialog.go` role)

A new `PatchBankDialog` model with three internal modes:

| Mode               | Description                                                             |
| ------------------ | ----------------------------------------------------------------------- |
| `modeBrowse`       | Navigate the preset/patch list (current `SynthPresetsDialog` behaviour) |
| `modeSaveName`     | Text input for the new patch name                                       |
| `modeSaveCategory` | Text input for the new patch category                                   |
| `modeRename`       | Text input for renaming a custom patch                                  |

**Key bindings (browse mode)**

| Key       | Action                                                  |
| --------- | ------------------------------------------------------- |
| `↑` / `↓` | Move selection                                          |
| `←` / `→` | Switch category filter                                  |
| `Enter`   | Apply selected preset to the current track              |
| `S`       | Begin saving current synth (→ `modeSaveName`)           |
| `R`       | Rename selected _custom_ patch (→ `modeRename`)         |
| `D`       | Delete selected _custom_ patch (emit `PatchDeletedMsg`) |
| `1–7`     | Preview note with selected preset                       |
| `Esc`     | Close dialog                                            |

**Messages emitted**

```go
// PatchSaveRequestedMsg is emitted when the user confirms Save in the dialog.
type PatchSaveRequestedMsg struct {
    Name     string
    Category string
    Synth    *audio.Synth   // snapshot of the synth at save time
}

// PatchDeleteRequestedMsg is emitted when the user confirms Delete.
type PatchDeleteRequestedMsg struct {
    PatchName string
}

// PatchRenameRequestedMsg is emitted when the user confirms Rename.
type PatchRenameRequestedMsg struct {
    OldName string
    NewName string
}
```

`main.go` handles these messages: mutates the `PatchBank`, writes it to disk, calls `SynthPresetView.SetUserPatches` with fresh data.

**Footer help text** adapts to the current mode:

- Browse: `↑↓: Navigate | ←→: Category | S: Save patch | R: Rename | D: Delete | Enter: Apply | Esc: Close`
- Save/Rename: `Type name | Enter: Confirm | Esc: Cancel`

---

### Step 4 — `ui/synth/screen.go`

- Add `B` key case inside `Update` (joins the existing `tab`/`shift+tab`/`ctrl+arrow` block):
  ```go
  case "b":
      return s, func() tea.Msg { return OpenSynthPresetDialogMsg{} }
  ```
- Update `Footer()` to mention `B: Patch Bank`.
- `OpenSynthPresetDialogMsg` already exists in `presets_dialog.go`; no new message needed.

---

### Step 5 — `main.go`

**At startup** (after `NewSynthScreen`):

1. `bank, err := persistence.LoadPatchBank()` — log/ignore error (app still usable without the file).
2. Convert `bank.SynthPatches` → `[]synth.SynthPreset` (helper in `main.go`).
3. `synthPresetView.SetUserPatches(userPatches)`.

**Message handling** (add cases inside the `Update` switch):

```go
case synth.PatchSaveRequestedMsg:
    patch := persistence.SavedPatch{
        Name:     msg.Name,
        Category: msg.Category,
        Custom:   true,
        Synth:    persistence.ToSavedSynth(msg.Synth),
    }
    bank.SynthPatches = append(bank.SynthPatches, patch)
    if err := bank.Save(); err != nil { /* log */ }
    synthPresetView.SetUserPatches(bankToPresets(bank))

case synth.PatchDeleteRequestedMsg:
    bank.SynthPatches = slices.DeleteFunc(bank.SynthPatches, func(p persistence.SavedPatch) bool {
        return p.Name == msg.PatchName && p.Custom
    })
    if err := bank.Save(); err != nil { /* log */ }
    synthPresetView.SetUserPatches(bankToPresets(bank))

case synth.PatchRenameRequestedMsg:
    for i := range bank.SynthPatches {
        if bank.SynthPatches[i].Name == msg.OldName && bank.SynthPatches[i].Custom {
            bank.SynthPatches[i].Name = msg.NewName
            break
        }
    }
    if err := bank.Save(); err != nil { /* log */ }
    synthPresetView.SetUserPatches(bankToPresets(bank))
```

---

### Step 6 — Tests

| File                                      | Tests                                                                                      |
| ----------------------------------------- | ------------------------------------------------------------------------------------------ |
| `persistence/patchbank_test.go`           | Round-trip, missing file, corrupt file, atomic write                                       |
| `ui/synth/presets_test.go` (new)          | `SetUserPatches` merges correctly; "My Patches" filter; category two-layer filter          |
| `ui/synth/patchbank_dialog_test.go` (new) | Save flow emits correct msg; delete only works on custom patches; rename emits correct msg |

---

### File Summary

| File                            | Change                                                                        |
| ------------------------------- | ----------------------------------------------------------------------------- |
| `persistence/patchbank.go`      | **New** — `PatchBank`, `SavedPatch`, load/save                                |
| `persistence/patchbank_test.go` | **New** — persistence tests                                                   |
| `ui/synth/presets.go`           | Add `Custom` to `SynthPreset`; add `SetUserPatches`; update `buildCategories` |
| `ui/synth/patchbank_dialog.go`  | **New** — replaces `SynthPresetsDialog` with multi-mode `PatchBankDialog`     |
| `ui/synth/presets_dialog.go`    | Remove (superseded by `patchbank_dialog.go`) or keep as wrapper               |
| `ui/synth/screen.go`            | `B` key → `OpenSynthPresetDialogMsg`; footer update                           |
| `main.go`                       | Load bank at startup; handle 3 new messages; `bankToPresets` helper           |
