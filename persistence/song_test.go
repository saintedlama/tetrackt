package persistence

import (
	"os"
	"testing"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"
)

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
	tmpFile := "test_song.yaml"
	defer os.Remove(tmpFile)

	song := TracksToSong(trackerModel)
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
	newTracker := tracker.NewTracker(8, 64, 0, 0) // Different dimensions initially
	SongToTracks(loadedSong, newTracker)

	// Verify dimensions were updated
	if newTracker.NumRows != 16 {
		t.Errorf("Expected NumRows=16, got %d", newTracker.NumRows)
	}
	if newTracker.NumTracks != 4 {
		t.Errorf("Expected NumTracks=4, got %d", newTracker.NumTracks)
	}

	// Verify track data
	if newTracker.Tracks[0].Synth.Oscillator1 != (audio.Oscillator{Type: audio.Sine}) {
		t.Errorf("Expected Oscillator1=Sine, got %v", newTracker.Tracks[0].Synth.Oscillator1)
	}
	if newTracker.Tracks[0].Synth.Oscillator2 != (audio.Oscillator{Type: audio.Square}) {
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
