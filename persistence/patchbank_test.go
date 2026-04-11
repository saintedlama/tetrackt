package persistence

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/tetrackt/tetrackt/audio"
)

func TestPatchBankRoundTrip(t *testing.T) {
	// Build a patch with all fields populated.
	synth := audio.NewSynth(
		audio.Oscillator{Type: audio.Square, PulseWidth: 0.3},
		audio.Envelope{Attack: 10 * time.Millisecond, Decay: 50 * time.Millisecond, Sustain: 0.7, Release: 100 * time.Millisecond},
		audio.Oscillator{Type: audio.Sine},
		audio.Envelope{Attack: 5 * time.Millisecond, Decay: 20 * time.Millisecond, Sustain: 0.5, Release: 80 * time.Millisecond},
		audio.Mixer{Volume1: 0.8, Volume2: 0.4},
		audio.Filter{Type: audio.FilterLowPass, Cutoff: 1000, Resonance: 0.5},
		audio.LFO{Rate: 2.0, Depth: 0.3},
		audio.LFO{},
	)

	bank := &PatchBank{
		Version: 1,
		SynthPatches: []SavedPatch{
			{
				Name:     "Cool Bass",
				Category: "Bass",
				Custom:   true,
				Synth:    ToSavedSynth(synth),
			},
		},
	}

	// Save to a temp file.
	tmpPath := t.TempDir() + "/test_patchbank.tetrackt"
	data, err := json.MarshalIndent(bank, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Load back.
	raw, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	var loaded PatchBank
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.Version != 1 {
		t.Errorf("expected Version=1, got %d", loaded.Version)
	}
	if len(loaded.SynthPatches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(loaded.SynthPatches))
	}
	p := loaded.SynthPatches[0]
	if p.Name != "Cool Bass" {
		t.Errorf("expected Name='Cool Bass', got %q", p.Name)
	}
	if p.Category != "Bass" {
		t.Errorf("expected Category='Bass', got %q", p.Category)
	}
	if !p.Custom {
		t.Error("expected Custom=true")
	}

	// Reconstruct synth and check oscillator type round-trips.
	restored := SynthFromSavedPatch(p)
	if restored.Oscillator1.Type != audio.Square {
		t.Errorf("expected Oscillator1=Square, got %v", restored.Oscillator1.Type)
	}
	if restored.Oscillator1.PulseWidth != 0.3 {
		t.Errorf("expected PulseWidth=0.3, got %v", restored.Oscillator1.PulseWidth)
	}
}

func TestLoadPatchBankMissingFile(t *testing.T) {
	// Point the lookup at a non-existent path by temporarily overriding HOME.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir()) // Windows

	bank, err := LoadPatchBank()
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if bank.Version != 1 {
		t.Errorf("expected Version=1, got %d", bank.Version)
	}
	if len(bank.SynthPatches) != 0 {
		t.Errorf("expected empty patches, got %d", len(bank.SynthPatches))
	}
}

func TestLoadPatchBankCorruptJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if err := os.WriteFile(home+"/.tetrackt", []byte("{corrupt json"), 0600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, err := LoadPatchBank()
	if err == nil {
		t.Error("expected error for corrupt JSON, got nil")
	}
}

func TestPatchBankSaveAtomic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	bank := &PatchBank{Version: 1, SynthPatches: []SavedPatch{{Name: "Test", Custom: true}}}
	if err := bank.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Temp file should not exist after successful save.
	if _, err := os.Stat(home + "/.tetrackt.tmp"); !os.IsNotExist(err) {
		t.Error("expected temp file to be removed after save")
	}

	// Final file should exist and be valid JSON.
	loaded, err := LoadPatchBank()
	if err != nil {
		t.Fatalf("LoadPatchBank after Save failed: %v", err)
	}
	if len(loaded.SynthPatches) != 1 || loaded.SynthPatches[0].Name != "Test" {
		t.Errorf("unexpected loaded patches: %+v", loaded.SynthPatches)
	}
}
