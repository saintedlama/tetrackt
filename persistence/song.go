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
}

// SavedTrack is the YAML-serializable form of Track
type SavedTrack struct {
	Oscillator1           string          `yaml:"oscillator1"`
	Oscillator1Phase      float64         `yaml:"oscillator1_phase"`
	Oscillator1PulseWidth float64         `yaml:"oscillator1_pulse_width"`
	Envelope1             audio.Envelope  `yaml:"envelope1"`
	Oscillator2           string          `yaml:"oscillator2"`
	Oscillator2Phase      float64         `yaml:"oscillator2_phase"`
	Oscillator2PulseWidth float64         `yaml:"oscillator2_pulse_width"`
	Envelope2             audio.Envelope  `yaml:"envelope2"`
	MixerVolume1          float64         `yaml:"mixer_volume1"`
	MixerVolume2          float64         `yaml:"mixer_volume2"`
	FilterType            string          `yaml:"filter_type"`
	FilterCutoff          float64         `yaml:"filter_cutoff"`
	FilterResonance       float64         `yaml:"filter_resonance"`
	LFO1Waveform          string          `yaml:"lfo1_waveform"`
	LFO1Rate              float64         `yaml:"lfo1_rate"`
	LFO1Depth             float64         `yaml:"lfo1_depth"`
	LFO1Delay             float64         `yaml:"lfo1_delay"`
	LFO1Dest              int             `yaml:"lfo1_dest"`
	LFO2Waveform          string          `yaml:"lfo2_waveform"`
	LFO2Rate              float64         `yaml:"lfo2_rate"`
	LFO2Depth             float64         `yaml:"lfo2_depth"`
	LFO2Delay             float64         `yaml:"lfo2_delay"`
	LFO2Dest              int             `yaml:"lfo2_dest"`
	Rows                  []SavedTrackRow `yaml:"rows"`
}

// SavedSong is the complete song structure for YAML serialization
type SavedSong struct {
	NumRows   int          `yaml:"num_rows"`
	NumTracks int          `yaml:"num_tracks"`
	BPM       int          `yaml:"bpm"`
	Tracks    []SavedTrack `yaml:"tracks"`
}

// TracksToSong converts the runtime TrackerModel to a SavedSong for YAML serialization
func TracksToSong(tracker *ui.TrackerModel) *SavedSong {
	saved := &SavedSong{
		NumRows:   tracker.NumRows,
		NumTracks: tracker.NumTracks,
		BPM:       tracker.BPM,
		Tracks:    make([]SavedTrack, tracker.NumTracks),
	}

	for i, track := range tracker.Tracks {
		rows := make([]SavedTrackRow, len(track.Rows))
		for j, row := range track.Rows {
			rows[j] = SavedTrackRow{
				Base:   string(row.Note.Base),
				Octave: int(row.Note.Octave),
				Volume: row.Volume,
			}
		}
		s := track.Synth
		saved.Tracks[i] = SavedTrack{
			Oscillator1:           string(s.Oscillator1.Type),
			Oscillator1Phase:      s.Oscillator1.Phase,
			Oscillator1PulseWidth: s.Oscillator1.PulseWidth,
			Envelope1:             s.Envelope1,
			Oscillator2:           string(s.Oscillator2.Type),
			Oscillator2Phase:      s.Oscillator2.Phase,
			Oscillator2PulseWidth: s.Oscillator2.PulseWidth,
			Envelope2:             s.Envelope2,
			MixerVolume1:          s.Mixer.Volume1,
			MixerVolume2:          s.Mixer.Volume2,
			FilterType:            string(s.Filter.Type),
			FilterCutoff:          s.Filter.Cutoff,
			FilterResonance:       s.Filter.Resonance,
			LFO1Waveform:          string(s.LFO1.Waveform),
			LFO1Rate:              s.LFO1.Rate,
			LFO1Depth:             s.LFO1.Depth,
			LFO1Delay:             s.LFO1.Delay,
			LFO1Dest:              int(s.LFO1.Dest),
			LFO2Waveform:          string(s.LFO2.Waveform),
			LFO2Rate:              s.LFO2.Rate,
			LFO2Depth:             s.LFO2.Depth,
			LFO2Delay:             s.LFO2.Delay,
			LFO2Dest:              int(s.LFO2.Dest),
			Rows:                  rows,
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

	// Restore BPM; fall back to default for old saves that omit it
	if saved.BPM > 0 {
		tracker.BPM = saved.BPM
	} else {
		tracker.BPM = ui.DefaultBPM
	}

	// Resize tracks slice if needed
	if len(tracker.Tracks) != saved.NumTracks {
		tracker.Tracks = make([]ui.Track, saved.NumTracks)
	}

	// Update each track with saved data
	for i, savedTrack := range saved.Tracks {
		track := &tracker.Tracks[i]
		track.Synth = audio.NewSynth(
			audio.Oscillator{
				Type:       audio.OscillatorType(savedTrack.Oscillator1),
				Phase:      savedTrack.Oscillator1Phase,
				PulseWidth: savedTrack.Oscillator1PulseWidth,
			},
			savedTrack.Envelope1,
			audio.Oscillator{
				Type:       audio.OscillatorType(savedTrack.Oscillator2),
				Phase:      savedTrack.Oscillator2Phase,
				PulseWidth: savedTrack.Oscillator2PulseWidth,
			},
			savedTrack.Envelope2,
			audio.Mixer{Volume1: savedTrack.MixerVolume1, Volume2: savedTrack.MixerVolume2},
			audio.Filter{
				Type:      audio.FilterType(savedTrack.FilterType),
				Cutoff:    savedTrack.FilterCutoff,
				Resonance: savedTrack.FilterResonance,
			},
			audio.LFO{
				Waveform: audio.LFOWaveform(savedTrack.LFO1Waveform),
				Rate:     savedTrack.LFO1Rate,
				Depth:    savedTrack.LFO1Depth,
				Delay:    savedTrack.LFO1Delay,
				Dest:     audio.ModDest(savedTrack.LFO1Dest),
			},
			audio.LFO{
				Waveform: audio.LFOWaveform(savedTrack.LFO2Waveform),
				Rate:     savedTrack.LFO2Rate,
				Depth:    savedTrack.LFO2Depth,
				Delay:    savedTrack.LFO2Delay,
				Dest:     audio.ModDest(savedTrack.LFO2Dest),
			},
		)

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
