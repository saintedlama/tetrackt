package persistence

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/tetrackt/tetrackt/audio"
)

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

// LoadPatchBank reads ~/.tetrackt and returns the stored PatchBank.
// If the file does not exist an empty bank (Version=1) is returned without error.
func LoadPatchBank() (*PatchBank, error) {
	path, err := patchBankPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &PatchBank{Version: 1}, nil
		}
		return nil, err
	}
	var bank PatchBank
	if err := json.Unmarshal(data, &bank); err != nil {
		return nil, err
	}
	return &bank, nil
}

// Save writes the PatchBank to ~/.tetrackt atomically (temp file + rename).
func (b *PatchBank) Save() error {
	path, err := patchBankPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func patchBankPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tetrackt"), nil
}

// ToSavedSynth converts an *audio.Synth to a SavedSynth.
// Exported as a bridge for main.go to build SavedPatch values.
func ToSavedSynth(s *audio.Synth) SavedSynth {
	return toSavedSynth(s)
}

// SynthFromSavedPatch reconstructs an *audio.Synth from a SavedPatch.
func SynthFromSavedPatch(p SavedPatch) *audio.Synth {
	return fromSavedSynth(p.Synth)
}
