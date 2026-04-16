package persistence

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetrackt/tetrackt/audio"
)

func TestPatchBankRoundTrip(t *testing.T) {
	// Build a patch with all fields populated.
	synth := audio.NewSynth(
		audio.Oscillator{Type: audio.Square, PulseWidth: 0.3},
		audio.Envelope{Attack: 10 * time.Millisecond, Decay: 50 * time.Millisecond, Sustain: 0.7, Release: 100 * time.Millisecond},
		audio.Oscillator{Type: audio.Sine},
		audio.Envelope{Attack: 5 * time.Millisecond, Decay: 20 * time.Millisecond, Sustain: 0.5, Release: 80 * time.Millisecond},
		audio.Oscillator{Type: audio.Silent},
		audio.Envelope{Sustain: 1.0},
		audio.Mixer{Volume1: 0.8, Volume2: 0.4},
		audio.Filter{Type: audio.FilterLowPass, Cutoff: 1000, Resonance: 0.5},
		audio.LFO{Rate: 2.0, Depth: 0.3},
		audio.LFO{},
		audio.LFO{},
	)

	bank := &PatchBank{
		Version: 1,
		SynthPatches: []SavedPatch{
			{
				Name:     "Cool Bass",
				Category: "Bass",
				Tags:     []string{"Custom", "C64"},
				Synth:    ToSavedSynth(synth),
			},
		},
	}

	// Save to a temp file.
	tmpPath := t.TempDir() + "/test_patchbank.tetrackt"
	data, err := json.MarshalIndent(bank, "", "  ")
	require.NoError(t, err, "marshal failed")
	require.NoError(t, os.WriteFile(tmpPath, data, 0600), "write failed")

	// Load back.
	raw, err := os.ReadFile(tmpPath)
	require.NoError(t, err, "read failed")
	var loaded PatchBank
	require.NoError(t, json.Unmarshal(raw, &loaded), "unmarshal failed")

	require.Len(t, loaded.SynthPatches, 1, "expected 1 patch")
	assert.Equal(t, 1, loaded.Version, "expected Version=1")

	p := loaded.SynthPatches[0]
	assert.Equal(t, "Cool Bass", p.Name, "expected Name='Cool Bass'")
	assert.Equal(t, "Bass", p.Category, "expected Category='Bass'")
	assert.True(t, p.IsCustom(), "expected IsCustom()=true")
	assert.Len(t, p.Tags, 2, "expected 2 tags")

	// Reconstruct synth and check oscillator type round-trips.
	restored := SynthFromSavedPatch(p)
	assert.Equal(t, audio.Square, restored.Oscillator1.Type, "expected Oscillator1=Square")
	assert.Equal(t, 0.3, restored.Oscillator1.PulseWidth, "expected PulseWidth=0.3")
}

func TestLoadPatchBankMissingFile(t *testing.T) {
	// Point the lookup at a non-existent path by temporarily overriding HOME.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir()) // Windows

	bank, err := LoadPatchBank()
	require.NoError(t, err, "expected no error for missing file")
	assert.Equal(t, 1, bank.Version, "expected Version=1")
	assert.Empty(t, bank.SynthPatches, "expected empty patches")
}

func TestLoadPatchBankCorruptJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	require.NoError(t, os.WriteFile(home+"/.tetrackt", []byte("{corrupt json"), 0600), "setup failed")

	_, err := LoadPatchBank()
	require.Error(t, err, "expected error for corrupt JSON")
}

func TestPatchBankSaveAtomic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	bank := &PatchBank{Version: 1, SynthPatches: []SavedPatch{{Name: "Test", Tags: []string{"Custom"}}}}
	require.NoError(t, bank.Save(), "Save failed")

	// Temp file should not exist after successful save.
	_, err := os.Stat(home + "/.tetrackt.tmp")
	assert.True(t, os.IsNotExist(err), "expected temp file to be removed after save")

	// Final file should exist and be valid JSON.
	loaded, err := LoadPatchBank()
	require.NoError(t, err, "LoadPatchBank after Save failed")
	require.Len(t, loaded.SynthPatches, 1)
	assert.Equal(t, "Test", loaded.SynthPatches[0].Name)
}
