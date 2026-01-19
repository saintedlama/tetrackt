package persistence

import (
	"os"
	"testing"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui"
)

func TestSaveAndLoad(t *testing.T) {
	// Create a test tracker with some data
	tracker := ui.NewTracker(4, 16, 0, 0)

	// Add some test data
	tracker.Tracks[0].Oscillator1 = audio.Sine
	tracker.Tracks[0].Oscillator2 = audio.Square
	tracker.Tracks[0].Mixer = 0.75
	tracker.Tracks[0].Envelope1 = audio.Envelope{
		Attack:  0.1,
		Decay:   0.2,
		Sustain: 0.5,
		Release: 0.3,
	}
	tracker.Tracks[0].Rows[0] = ui.TrackRow{
		Note:   audio.NewNote("C", 4),
		Volume: 64,
		Effect: "---",
	}
	tracker.Tracks[0].Rows[1] = ui.TrackRow{
		Note:   audio.NewNote("E", 4),
		Volume: 80,
		Effect: "---",
	}

	// Save to a temporary file
	tmpFile := "test_song.yaml"
	defer os.Remove(tmpFile)

	song := TracksToSong(tracker)
	err := SaveToFile(tmpFile, song)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load from file
	loadedSong, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Create a new tracker and load data into it
	newTracker := ui.NewTracker(8, 64, 0, 0) // Different dimensions initially
	SongToTracks(loadedSong, newTracker)

	// Verify dimensions were updated
	if newTracker.NumRows != 16 {
		t.Errorf("Expected NumRows=16, got %d", newTracker.NumRows)
	}
	if newTracker.NumTracks != 4 {
		t.Errorf("Expected NumTracks=4, got %d", newTracker.NumTracks)
	}

	// Verify track data
	if newTracker.Tracks[0].Oscillator1 != audio.Sine {
		t.Errorf("Expected Oscillator1=Sine, got %v", newTracker.Tracks[0].Oscillator1)
	}
	if newTracker.Tracks[0].Oscillator2 != audio.Square {
		t.Errorf("Expected Oscillator2=Square, got %v", newTracker.Tracks[0].Oscillator2)
	}
	if newTracker.Tracks[0].Mixer != 0.75 {
		t.Errorf("Expected Mixer=0.75, got %f", newTracker.Tracks[0].Mixer)
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
	if newTracker.Tracks[0].Envelope1.Attack != 0.1 {
		t.Errorf("Expected Attack=0.1, got %f", newTracker.Tracks[0].Envelope1.Attack)
	}
}
