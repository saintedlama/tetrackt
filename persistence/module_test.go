package persistence

import (
	"os"
	"testing"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"
)

func TestQuickstartModuleLoads(t *testing.T) {
	const quickstartPath = "../modules/quickstart.json"

	if _, err := os.Stat(quickstartPath); err != nil {
		t.Fatalf("quickstart module is missing: %v", err)
	}

	mod, err := LoadFromFile(quickstartPath)
	if err != nil {
		t.Fatalf("failed to load quickstart module: %v", err)
	}

	if mod.NumTracks == 0 || mod.NumRows == 0 {
		t.Fatalf("quickstart module has invalid dimensions: tracks=%d rows=%d", mod.NumTracks, mod.NumRows)
	}

	m := tracker.NewTracker(1, 1, 0, 0)
	ModuleToTracks(mod, m)
	expectedTracks := max(8, mod.NumTracks)
	if len(m.Tracks) != expectedTracks {
		t.Fatalf("expected %d tracks after load, got %d", expectedTracks, len(m.Tracks))
	}
	expectedRows := max(64, mod.NumRows)
	if m.NumRows != expectedRows {
		t.Fatalf("expected %d rows after load, got %d", expectedRows, m.NumRows)
	}
}

func TestChiptuneDemoModuleLoads(t *testing.T) {
	const demoPath = "../modules/chiptune-demo.json"

	if _, err := os.Stat(demoPath); err != nil {
		t.Fatalf("chiptune demo module is missing: %v", err)
	}

	mod, err := LoadFromFile(demoPath)
	if err != nil {
		t.Fatalf("failed to load chiptune demo module: %v", err)
	}

	if mod.NumTracks != 8 {
		t.Fatalf("expected 8 tracks, got %d", mod.NumTracks)
	}
	if mod.NumRows < 16 {
		t.Fatalf("expected at least 16 rows, got %d", mod.NumRows)
	}

	m := tracker.NewTracker(1, 1, 0, 0)
	ModuleToTracks(mod, m)
	expectedTracks := max(8, mod.NumTracks)
	if len(m.Tracks) != expectedTracks {
		t.Fatalf("expected %d tracks after load, got %d", expectedTracks, len(m.Tracks))
	}
	expectedRows := max(64, mod.NumRows)
	if m.NumRows != expectedRows {
		t.Fatalf("expected %d rows after load, got %d", expectedRows, m.NumRows)
	}
}

func TestModuleToTracksPadsToEightWithDefaults(t *testing.T) {
	mod := &SavedModule{
		NumRows:   4,
		NumTracks: 2,
		BPM:       120,
		Speed:     6,
		Tracks: []SavedTrack{
			{
				Synth: ToSavedSynth(audio.NewSynth(
					audio.Oscillator{Type: audio.Square},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Oscillator{Type: audio.Silent},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Mixer{Volume1: 1.0, Volume2: 1.0},
					audio.NewFilter(),
					audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Dest: audio.ModPitch},
					audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Dest: audio.ModVolume},
				)),
				Rows: []SavedTrackRow{{Base: "C", Octave: 4, Volume: 64}},
			},
			{
				Synth: ToSavedSynth(audio.NewSynth(
					audio.Oscillator{Type: audio.Triangle},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Oscillator{Type: audio.Silent},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Mixer{Volume1: 1.0, Volume2: 1.0},
					audio.NewFilter(),
					audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Dest: audio.ModPitch},
					audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Dest: audio.ModVolume},
				)),
				Rows: []SavedTrackRow{{Base: "E", Octave: 3, Volume: 48}},
			},
		},
	}

	m := tracker.NewTracker(8, 16, 0, 0)
	ModuleToTracks(mod, m)

	if m.NumTracks != 8 {
		t.Fatalf("expected 8 tracks, got %d", m.NumTracks)
	}

	if m.Tracks[2].Synth == nil {
		t.Fatal("expected padded track synth to be default-initialized")
	}

	if len(m.Tracks[2].Rows) != 64 {
		t.Fatalf("expected padded track rows to be 64, got %d", len(m.Tracks[2].Rows))
	}

	if m.Tracks[2].Rows[0].Note != audio.Off() || m.Tracks[2].Rows[0].Volume != 0 {
		t.Fatalf("expected padded row to be empty note with zero volume, got note=%s volume=%d", m.Tracks[2].Rows[0].Note.String(), m.Tracks[2].Rows[0].Volume)
	}

	if m.Tracks[0].Rows[63].Note != audio.Off() || m.Tracks[0].Rows[63].Volume != 0 {
		t.Fatalf("expected padded tail row to be empty note with zero volume, got note=%s volume=%d", m.Tracks[0].Rows[63].Note.String(), m.Tracks[0].Rows[63].Volume)
	}
}

func TestLoadFromBytes(t *testing.T) {
	data := []byte(`{"num_rows":2,"num_tracks":1,"bpm":120,"speed":6,"tracks":[{"synth":{"oscillator1":{"type":"sine"},"envelope1":{"attack":0,"decay":0,"sustain":1,"release":0},"lfo1":{"waveform":"sine","rate":1,"dest":0},"oscillator2":{"type":"silent"},"envelope2":{"attack":0,"decay":0,"sustain":1,"release":0},"lfo2":{"waveform":"sine","rate":1,"dest":1},"oscillator3":{"type":"silent"},"envelope3":{"attack":0,"decay":0,"sustain":1,"release":0},"lfo3":{"waveform":"sine","rate":1,"dest":0},"mixer":{"volume1":1,"volume2":1},"filter":{"type":"off","cutoff":0.5,"resonance":0}},"rows":[{"base":"C","octave":4,"volume":64},{"base":"---","octave":0,"volume":0}]}]}`)

	mod, err := LoadFromBytes(data)
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	if mod.NumRows != 2 || mod.NumTracks != 1 {
		t.Fatalf("unexpected dimensions: rows=%d tracks=%d", mod.NumRows, mod.NumTracks)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create a test tracker with some data
	trackerModel := tracker.NewTracker(4, 16, 0, 0)

	// Add some test data
	trackerModel.Tracks[0].Synth.Oscillator1 = audio.Oscillator{Type: audio.Sine}
	trackerModel.Tracks[0].Synth.Oscillator2 = audio.Oscillator{Type: audio.Square}
	trackerModel.Tracks[0].Synth.Mixer = audio.Mixer{Volume1: 0.75, Volume2: 0.5}
	trackerModel.Tracks[0].Synth.Envelope1 = audio.Envelope{
		Attack:  100 * time.Millisecond,
		Decay:   200 * time.Millisecond,
		Sustain: 0.5,
		Release: 300 * time.Millisecond,
	}
	trackerModel.Tracks[0].Rows[0] = tracker.TrackRow{
		Note:   audio.NewNote("C", 4),
		Volume: 64,
	}
	trackerModel.Tracks[0].Rows[1] = tracker.TrackRow{
		Note:   audio.NewNote("E", 4),
		Volume: 80,
	}

	// Save to a temporary file
	tmpFile := "test_module.json"
	defer os.Remove(tmpFile)

	mod := TracksToModule(trackerModel)
	err := SaveToFile(tmpFile, mod)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load from file
	loadedMod, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Create a new tracker and load data into it
	newTracker := tracker.NewTracker(8, 64, 0, 0) // Different dimensions initially
	ModuleToTracks(loadedMod, newTracker)

	// Verify dimensions were updated
	if newTracker.NumRows != 64 {
		t.Errorf("Expected NumRows=64, got %d", newTracker.NumRows)
	}
	if newTracker.NumTracks != 8 {
		t.Errorf("Expected NumTracks=8, got %d", newTracker.NumTracks)
	}

	// Verify track data
	if newTracker.Tracks[0].Synth.Oscillator1.Type != audio.Sine {
		t.Errorf("Expected Oscillator1=Sine, got %v", newTracker.Tracks[0].Synth.Oscillator1)
	}
	if newTracker.Tracks[0].Synth.Oscillator2.Type != audio.Square {
		t.Errorf("Expected Oscillator2=Square, got %v", newTracker.Tracks[0].Synth.Oscillator2)
	}
	if newTracker.Tracks[0].Synth.Mixer != (audio.Mixer{Volume1: 0.75, Volume2: 0.5}) {
		t.Errorf("Expected Mixer={0.75, 0.5}, got %+v", newTracker.Tracks[0].Synth.Mixer)
	}

	// Verify row data
	if newTracker.Tracks[0].Rows[0].Note != audio.NewNote("C", 4) {
		t.Errorf("Expected Note=C-4, got %s", newTracker.Tracks[0].Rows[0].Note.String())
	}
	if newTracker.Tracks[0].Rows[0].Volume != 64 {
		t.Errorf("Expected Volume=64, got %d", newTracker.Tracks[0].Rows[0].Volume)
	}
	if newTracker.Tracks[0].Rows[1].Note != audio.NewNote("E", 4) {
		t.Errorf("Expected Note=E-4, got %s", newTracker.Tracks[0].Rows[1].Note.String())
	}
	if newTracker.Tracks[0].Rows[1].Volume != 80 {
		t.Errorf("Expected Volume=80, got %d", newTracker.Tracks[0].Rows[1].Volume)
	}

	// Verify envelope data
	if newTracker.Tracks[0].Synth.Envelope1.Attack != 100*time.Millisecond {
		t.Errorf("Expected Attack=100ms, got %v", newTracker.Tracks[0].Synth.Envelope1.Attack)
	}
}

func TestTrackPatchMetadataRoundTrip(t *testing.T) {
	trackerModel := tracker.NewTracker(8, 16, 0, 0)
	trackerModel.Tracks[0].PatchName = "Square Lead"
	trackerModel.Tracks[0].PatchCategory = "Lead"
	trackerModel.Tracks[0].PatchTags = []string{"NES", "Custom"}
	trackerModel.Tracks[0].Synth.Oscillator1 = audio.Oscillator{Type: audio.Square, PulseWidth: 0.25, Detune: 5}

	tmpFile := "test_patch_meta_module.json"
	defer os.Remove(tmpFile)

	mod := TracksToModule(trackerModel)
	if err := SaveToFile(tmpFile, mod); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	loadedMod, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	newTracker := tracker.NewTracker(8, 64, 0, 0)
	ModuleToTracks(loadedMod, newTracker)

	if newTracker.Tracks[0].PatchName != "Square Lead" {
		t.Fatalf("expected patch name to round-trip, got %q", newTracker.Tracks[0].PatchName)
	}
	if newTracker.Tracks[0].PatchCategory != "Lead" {
		t.Fatalf("expected patch category to round-trip, got %q", newTracker.Tracks[0].PatchCategory)
	}
	if len(newTracker.Tracks[0].PatchTags) != 2 || newTracker.Tracks[0].PatchTags[0] != "NES" || newTracker.Tracks[0].PatchTags[1] != "Custom" {
		t.Fatalf("expected patch tags to round-trip, got %+v", newTracker.Tracks[0].PatchTags)
	}

	if newTracker.Tracks[0].Synth.Oscillator1.Type != audio.Square || newTracker.Tracks[0].Synth.Oscillator1.PulseWidth != 0.25 || newTracker.Tracks[0].Synth.Oscillator1.Detune != 5 {
		t.Fatalf("expected synth oscillator data to round-trip, got %+v", newTracker.Tracks[0].Synth.Oscillator1)
	}
}
