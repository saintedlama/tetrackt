package persistence

import (
	"encoding/json"
	"os"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	utracker "github.com/tetrackt/tetrackt/ui/tracker"
)

// SavedEnvelope is the JSON-serializable form of audio.Envelope.
// Attack, Decay, and Release are stored in seconds as float64.
type SavedEnvelope struct {
	Attack  float64 `json:"attack"`
	Decay   float64 `json:"decay"`
	Sustain float64 `json:"sustain"`
	Release float64 `json:"release"`
}

func toSavedEnvelope(e audio.Envelope) SavedEnvelope {
	return SavedEnvelope{
		Attack:  e.Attack.Seconds(),
		Decay:   e.Decay.Seconds(),
		Sustain: e.Sustain,
		Release: e.Release.Seconds(),
	}
}

func fromSavedEnvelope(s SavedEnvelope) audio.Envelope {
	return audio.Envelope{
		Attack:  time.Duration(s.Attack * float64(time.Second)),
		Decay:   time.Duration(s.Decay * float64(time.Second)),
		Sustain: s.Sustain,
		Release: time.Duration(s.Release * float64(time.Second)),
	}
}

// SavedOscillator is the JSON-serializable form of audio.Oscillator.
type SavedOscillator struct {
	Type       string  `json:"type"`
	Phase      float64 `json:"phase,omitempty"`
	PulseWidth float64 `json:"pulse_width,omitempty"`
	Detune     float64 `json:"detune,omitempty"`
}

func toSavedOscillator(o audio.Oscillator) SavedOscillator {
	return SavedOscillator{
		Type:       string(o.Type),
		Phase:      o.Phase,
		PulseWidth: o.PulseWidth,
		Detune:     o.Detune,
	}
}

func fromSavedOscillator(s SavedOscillator) audio.Oscillator {
	return audio.Oscillator{
		Type:       audio.OscillatorType(s.Type),
		Phase:      s.Phase,
		PulseWidth: s.PulseWidth,
		Detune:     s.Detune,
	}
}

// SavedLFO is the JSON-serializable form of audio.LFO.
type SavedLFO struct {
	Waveform string  `json:"waveform,omitempty"`
	Rate     float64 `json:"rate,omitempty"`
	Depth    float64 `json:"depth,omitempty"`
	Delay    float64 `json:"delay,omitempty"`
	Dest     int     `json:"dest,omitempty"`
}

func toSavedLFO(l audio.LFO) SavedLFO {
	return SavedLFO{
		Waveform: string(l.Waveform),
		Rate:     l.Rate,
		Depth:    l.Depth,
		Delay:    l.Delay,
		Dest:     int(l.Dest),
	}
}

func fromSavedLFO(s SavedLFO) audio.LFO {
	return audio.LFO{
		Waveform: audio.LFOWaveform(s.Waveform),
		Rate:     s.Rate,
		Depth:    s.Depth,
		Delay:    s.Delay,
		Dest:     audio.ModDest(s.Dest),
	}
}

// SavedMixer is the JSON-serializable form of audio.Mixer.
type SavedMixer struct {
	Volume1      float64 `json:"volume1"`
	Volume2      float64 `json:"volume2"`
	Volume3      float64 `json:"volume3,omitempty"`
	Pan1         float64 `json:"pan1,omitempty"`
	Pan2         float64 `json:"pan2,omitempty"`
	Pan3         float64 `json:"pan3,omitempty"`
	MasterVolume float64 `json:"master_volume,omitempty"`
	Mute1        bool    `json:"mute1,omitempty"`
	Mute2        bool    `json:"mute2,omitempty"`
	Mute3        bool    `json:"mute3,omitempty"`
	Mode         int     `json:"mode,omitempty"`
}

func toSavedMixer(m audio.Mixer) SavedMixer {
	return SavedMixer{
		Volume1:      m.Volume1,
		Volume2:      m.Volume2,
		Volume3:      m.Volume3,
		Pan1:         m.Pan1,
		Pan2:         m.Pan2,
		Pan3:         m.Pan3,
		MasterVolume: m.MasterVolume,
		Mute1:        m.Mute1,
		Mute2:        m.Mute2,
		Mute3:        m.Mute3,
		Mode:         int(m.Mode),
	}
}

func fromSavedMixer(s SavedMixer) audio.Mixer {
	return audio.Mixer{
		Volume1:      s.Volume1,
		Volume2:      s.Volume2,
		Volume3:      s.Volume3,
		Pan1:         s.Pan1,
		Pan2:         s.Pan2,
		Pan3:         s.Pan3,
		MasterVolume: s.MasterVolume,
		Mute1:        s.Mute1,
		Mute2:        s.Mute2,
		Mute3:        s.Mute3,
		Mode:         audio.MixMode(s.Mode),
	}
}

// SavedFilter is the JSON-serializable form of audio.Filter.
type SavedFilter struct {
	Type      string  `json:"type"`
	Cutoff    float64 `json:"cutoff"`
	Resonance float64 `json:"resonance"`
}

func toSavedFilter(f audio.Filter) SavedFilter {
	return SavedFilter{
		Type:      string(f.Type),
		Cutoff:    f.Cutoff,
		Resonance: f.Resonance,
	}
}

func fromSavedFilter(s SavedFilter) audio.Filter {
	return audio.Filter{
		Type:      audio.FilterType(s.Type),
		Cutoff:    s.Cutoff,
		Resonance: s.Resonance,
	}
}

// SavedSynth is the JSON-serializable form of audio.Synth.
type SavedSynth struct {
	Oscillator1 SavedOscillator `json:"oscillator1"`
	Envelope1   SavedEnvelope   `json:"envelope1"`
	LFO1        SavedLFO        `json:"lfo1"`
	Oscillator2 SavedOscillator `json:"oscillator2"`
	Envelope2   SavedEnvelope   `json:"envelope2"`
	LFO2        SavedLFO        `json:"lfo2"`
	Oscillator3 SavedOscillator `json:"oscillator3"`
	Envelope3   SavedEnvelope   `json:"envelope3"`
	LFO3        SavedLFO        `json:"lfo3"`
	Mixer       SavedMixer      `json:"mixer"`
	Filter      SavedFilter     `json:"filter"`
	Portamento  float64         `json:"portamento,omitempty"`
}

func toSavedSynth(s *audio.Synth) SavedSynth {
	return SavedSynth{
		Oscillator1: toSavedOscillator(s.Oscillator1),
		Envelope1:   toSavedEnvelope(s.Envelope1),
		LFO1:        toSavedLFO(s.LFO1),
		Oscillator2: toSavedOscillator(s.Oscillator2),
		Envelope2:   toSavedEnvelope(s.Envelope2),
		LFO2:        toSavedLFO(s.LFO2),
		Oscillator3: toSavedOscillator(s.Oscillator3),
		Envelope3:   toSavedEnvelope(s.Envelope3),
		LFO3:        toSavedLFO(s.LFO3),
		Mixer:       toSavedMixer(s.Mixer),
		Filter:      toSavedFilter(s.Filter),
		Portamento:  s.Portamento,
	}
}

func fromSavedSynth(s SavedSynth) *audio.Synth {
	synth := audio.NewSynth(
		fromSavedOscillator(s.Oscillator1),
		fromSavedEnvelope(s.Envelope1),
		fromSavedOscillator(s.Oscillator2),
		fromSavedEnvelope(s.Envelope2),
		fromSavedMixer(s.Mixer),
		fromSavedFilter(s.Filter),
		fromSavedLFO(s.LFO1),
		fromSavedLFO(s.LFO2),
	)
	synth.Portamento = s.Portamento
	synth.Oscillator3 = fromSavedOscillator(s.Oscillator3)
	synth.Envelope3 = fromSavedEnvelope(s.Envelope3)
	synth.LFO3 = fromSavedLFO(s.LFO3)
	return synth
}

// SavedTrackRow is the JSON-serializable form of TrackRow
type SavedTrackRow struct {
	Base            string `json:"base"`
	Octave          int    `json:"octave"`
	Volume          int    `json:"volume"`
	RowTicks        int    `json:"row_ticks,omitempty"`
	Continuous      bool   `json:"continuous,omitempty"`
	ArpeggioOffsets []int  `json:"arpeggio_offsets,omitempty"`
	EffectType      int    `json:"effect_type,omitempty"`
	EffectParam     int    `json:"effect_param,omitempty"`
}

// SavedTrack is the JSON-serializable form of Track
type SavedTrack struct {
	Synth SavedSynth      `json:"synth"`
	Rows  []SavedTrackRow `json:"rows"`
}

// SavedModule is the complete module structure for JSON serialization
type SavedModule struct {
	NumRows   int          `json:"num_rows"`
	NumTracks int          `json:"num_tracks"`
	BPM       int          `json:"bpm"`
	Speed     int          `json:"speed,omitempty"`
	Tracks    []SavedTrack `json:"tracks"`
}

// TracksToModule converts the runtime TrackerModel to a SavedModule for serialization.
func TracksToModule(tracker *utracker.TrackerModel) *SavedModule {
	saved := &SavedModule{
		NumRows:   tracker.NumRows,
		NumTracks: tracker.NumTracks,
		BPM:       tracker.BPM,
		Speed:     tracker.Speed,
		Tracks:    make([]SavedTrack, tracker.NumTracks),
	}

	for i, track := range tracker.Tracks {
		rows := make([]SavedTrackRow, len(track.Rows))
		for j, row := range track.Rows {
			rows[j] = SavedTrackRow{
				Base:            string(row.Note.Base),
				Octave:          int(row.Note.Octave),
				Volume:          row.Volume,
				RowTicks:        row.Ticks,
				Continuous:      row.Continuous,
				ArpeggioOffsets: row.Arpeggio.Offsets,
				EffectType:      int(row.Effect.Type),
				EffectParam:     row.Effect.Param,
			}
		}
		saved.Tracks[i] = SavedTrack{
			Synth: toSavedSynth(track.Synth),
			Rows:  rows,
		}
	}
	return saved
}

// ModuleToTracks updates an existing TrackerModel with data from a SavedModule.
func ModuleToTracks(mod *SavedModule, tracker *utracker.TrackerModel) {
	// Update tracker dimensions
	tracker.NumRows = mod.NumRows
	tracker.NumTracks = mod.NumTracks

	// Restore BPM; fall back to default for old saves that omit it
	if mod.BPM > 0 {
		tracker.BPM = mod.BPM
	} else {
		tracker.BPM = utracker.DefaultBPM
	}

	// Restore Speed; fall back to default for old saves that omit it
	if mod.Speed > 0 {
		tracker.Speed = mod.Speed
	} else {
		tracker.Speed = utracker.DefaultSpeed
	}

	// Resize tracks slice if needed
	if len(tracker.Tracks) != mod.NumTracks {
		tracker.Tracks = make([]utracker.Track, mod.NumTracks)
	}

	// Update each track with saved data
	for i, savedTrack := range mod.Tracks {
		track := &tracker.Tracks[i]
		track.Synth = fromSavedSynth(savedTrack.Synth)

		// Resize rows slice if needed
		if len(track.Rows) != mod.NumRows {
			track.Rows = make([]utracker.TrackRow, mod.NumRows)
		}

		// Update each row with saved data
		for j, row := range savedTrack.Rows {
			if j < len(track.Rows) {
				track.Rows[j] = utracker.TrackRow{
					Note:       audio.Note{Base: audio.Base(row.Base), Octave: audio.Octave(row.Octave)},
					Volume:     row.Volume,
					Ticks:      row.RowTicks,
					Continuous: row.Continuous,
					Arpeggio: audio.ArpeggioEffect{
						Offsets: row.ArpeggioOffsets,
					},
					Effect: utracker.TrackerEffect{
						Type:  utracker.EffectType(row.EffectType),
						Param: row.EffectParam,
					},
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

// SaveToFile writes a SavedModule to a JSON file
func SaveToFile(filename string, mod *SavedModule) error {
	data, err := json.MarshalIndent(mod, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// LoadFromFile reads a JSON file and returns a SavedModule
func LoadFromFile(filename string) (*SavedModule, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var saved SavedModule
	err = json.Unmarshal(data, &saved)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}
