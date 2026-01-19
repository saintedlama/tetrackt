package persistence

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui"
)

// SavedTrackRow is the YAML-serializable form of TrackRow
type SavedTrackRow struct {
	Base   string `yaml:"base"`
	Octave int    `yaml:"octave"`
	Volume int    `yaml:"volume"`
	Effect string `yaml:"effect"`
}

// SavedTrack is the YAML-serializable form of Track
type SavedTrack struct {
	Oscillator1 string          `yaml:"oscillator1"`
	Envelope1   audio.Envelope  `yaml:"envelope1"`
	Oscillator2 string          `yaml:"oscillator2"`
	Envelope2   audio.Envelope  `yaml:"envelope2"`
	Mixer       float64         `yaml:"mixer"`
	Rows        []SavedTrackRow `yaml:"rows"`
}

// SavedSong is the complete song structure for YAML serialization
type SavedSong struct {
	NumRows   int          `yaml:"num_rows"`
	NumTracks int          `yaml:"num_tracks"`
	Tracks    []SavedTrack `yaml:"tracks"`
}

// TracksToSong converts the runtime TrackerModel to a SavedSong for YAML serialization
func TracksToSong(tracker *ui.TrackerModel) *SavedSong {
	saved := &SavedSong{
		NumRows:   tracker.NumRows,
		NumTracks: tracker.NumTracks,
		Tracks:    make([]SavedTrack, tracker.NumTracks),
	}

	for i, track := range tracker.Tracks {
		rows := make([]SavedTrackRow, len(track.Rows))
		for j, row := range track.Rows {
			rows[j] = SavedTrackRow{
				Base:   string(row.Note.Base),
				Octave: int(row.Note.Octave),
				Volume: row.Volume,
				Effect: row.Effect,
			}
		}
		saved.Tracks[i] = SavedTrack{
			Oscillator1: string(track.Oscillator1),
			Envelope1:   track.Envelope1,
			Oscillator2: string(track.Oscillator2),
			Envelope2:   track.Envelope2,
			Mixer:       track.Mixer,
			Rows:        rows,
		}
	}
	return saved
}

// SongToTracks updates an existing TrackerModel with data from a SavedSong
// This fixes the TODO: instead of creating a new model, it updates the existing one
func SongToTracks(saved *SavedSong, tracker *ui.TrackerModel) {
	// Update tracker dimensions
	tracker.NumRows = saved.NumRows
	tracker.NumTracks = saved.NumTracks

	// Resize tracks slice if needed
	if len(tracker.Tracks) != saved.NumTracks {
		tracker.Tracks = make([]ui.Track, saved.NumTracks)
	}

	// Update each track with saved data
	for i, savedTrack := range saved.Tracks {
		track := &tracker.Tracks[i]
		track.Oscillator1 = audio.OscillatorType(savedTrack.Oscillator1)
		track.Envelope1 = savedTrack.Envelope1
		track.Oscillator2 = audio.OscillatorType(savedTrack.Oscillator2)
		track.Envelope2 = savedTrack.Envelope2
		track.Mixer = savedTrack.Mixer

		// Resize rows slice if needed
		if len(track.Rows) != saved.NumRows {
			track.Rows = make([]ui.TrackRow, saved.NumRows)
		}

		// Update each row with saved data
		for j, row := range savedTrack.Rows {
			if j < len(track.Rows) {
				track.Rows[j] = ui.TrackRow{
					Note:   audio.Note{Base: audio.Base(row.Base), Octave: audio.Octave(row.Octave)},
					Volume: row.Volume,
					Effect: row.Effect,
				}
			}
		}
	}

	// Reset cursor to safe position
	if tracker.CursorTrack >= tracker.NumTracks {
		tracker.CursorTrack = 0
	}
	if tracker.CursorRow >= tracker.NumRows {
		tracker.CursorRow = 0
	}
}

// SaveToFile writes a SavedSong to a YAML file
func SaveToFile(filename string, song *SavedSong) error {
	data, err := yaml.Marshal(song)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// LoadFromFile reads a YAML file and returns a SavedSong
func LoadFromFile(filename string) (*SavedSong, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var saved SavedSong
	err = yaml.Unmarshal(data, &saved)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}
