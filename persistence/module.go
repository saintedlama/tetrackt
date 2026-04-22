package persistence

import (
	"encoding/json"
	"os"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/notes"
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
// WavetableBank, WavetableName, and WavetableData carry the wavetable inline.
type SavedOscillator struct {
	Type          string    `json:"type"`
	Phase         float64   `json:"phase,omitempty"`
	PulseWidth    float64   `json:"pulse_width,omitempty"`
	Detune        float64   `json:"detune,omitempty"`
	WavetableBank string    `json:"wavetable_bank,omitempty"`
	WavetableName string    `json:"wavetable_name,omitempty"`
	WavetableData []float64 `json:"wavetable_data,omitempty"`
}

func toSavedOscillator(o audio.Oscillator) SavedOscillator {
	return SavedOscillator{
		Type:          string(o.Type),
		Phase:         o.Phase,
		PulseWidth:    o.PulseWidth,
		Detune:        o.Detune,
		WavetableBank: o.Meta.Bank,
		WavetableName: o.Meta.Name,
		WavetableData: o.Wavetable,
	}
}

func fromSavedOscillator(s SavedOscillator) audio.Oscillator {
	o := audio.Oscillator{
		Type:       audio.OscillatorType(s.Type),
		Phase:      s.Phase,
		PulseWidth: s.PulseWidth,
		Detune:     s.Detune,
	}
	if len(s.WavetableData) > 0 {
		o.Wavetable = s.WavetableData
		o.Meta = audio.Metadata{Bank: s.WavetableBank, Name: s.WavetableName}
	}
	return o
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

// SavedFilterEnvelope is the JSON-serializable form of audio.FilterEnvelope.
// Attack, Decay, Release are stored as seconds (float64). All fields use
// omitempty so existing modules without this field load as zero (disabled).
type SavedFilterEnvelope struct {
	Attack  float64 `json:"attack,omitempty"`
	Decay   float64 `json:"decay,omitempty"`
	Sustain float64 `json:"sustain,omitempty"`
	Release float64 `json:"release,omitempty"`
	Depth   float64 `json:"depth,omitempty"`
}

func toSavedFilterEnvelope(fe audio.FilterEnvelope) SavedFilterEnvelope {
	return SavedFilterEnvelope{
		Attack:  fe.Attack.Seconds(),
		Decay:   fe.Decay.Seconds(),
		Sustain: fe.Sustain,
		Release: fe.Release.Seconds(),
		Depth:   fe.Depth,
	}
}

func fromSavedFilterEnvelope(s SavedFilterEnvelope) audio.FilterEnvelope {
	return audio.FilterEnvelope{
		Attack:  time.Duration(s.Attack * float64(time.Second)),
		Decay:   time.Duration(s.Decay * float64(time.Second)),
		Sustain: s.Sustain,
		Release: time.Duration(s.Release * float64(time.Second)),
		Depth:   s.Depth,
	}
}

// SavedSynth is the JSON-serializable form of audio.Synth.
type SavedSynth struct {
	Oscillator1    SavedOscillator     `json:"oscillator1"`
	Envelope1      SavedEnvelope       `json:"envelope1"`
	LFO1           SavedLFO            `json:"lfo1"`
	Oscillator2    SavedOscillator     `json:"oscillator2"`
	Envelope2      SavedEnvelope       `json:"envelope2"`
	LFO2           SavedLFO            `json:"lfo2"`
	Oscillator3    SavedOscillator     `json:"oscillator3"`
	Envelope3      SavedEnvelope       `json:"envelope3"`
	LFO3           SavedLFO            `json:"lfo3"`
	Mixer          SavedMixer          `json:"mixer"`
	Filter         SavedFilter         `json:"filter"`
	FilterEnvelope SavedFilterEnvelope `json:"filter_envelope,omitempty"`
}

func toSavedSynth(s *audio.Synth) SavedSynth {
	return SavedSynth{
		Oscillator1:    toSavedOscillator(s.Oscillator1),
		Envelope1:      toSavedEnvelope(s.Envelope1),
		LFO1:           toSavedLFO(s.LFO1),
		Oscillator2:    toSavedOscillator(s.Oscillator2),
		Envelope2:      toSavedEnvelope(s.Envelope2),
		LFO2:           toSavedLFO(s.LFO2),
		Oscillator3:    toSavedOscillator(s.Oscillator3),
		Envelope3:      toSavedEnvelope(s.Envelope3),
		LFO3:           toSavedLFO(s.LFO3),
		Mixer:          toSavedMixer(s.Mixer),
		Filter:         toSavedFilter(s.Filter),
		FilterEnvelope: toSavedFilterEnvelope(s.FilterEnvelope),
	}
}

func fromSavedSynth(s SavedSynth) *audio.Synth {
	synth := audio.NewSynth(
		fromSavedOscillator(s.Oscillator1),
		fromSavedEnvelope(s.Envelope1),
		fromSavedOscillator(s.Oscillator2),
		fromSavedEnvelope(s.Envelope2),
		fromSavedOscillator(s.Oscillator3),
		fromSavedEnvelope(s.Envelope3),
		fromSavedMixer(s.Mixer),
		fromSavedFilter(s.Filter),
		fromSavedLFO(s.LFO1),
		fromSavedLFO(s.LFO2),
		fromSavedLFO(s.LFO3),
	)
	synth.FilterEnvelope = fromSavedFilterEnvelope(s.FilterEnvelope)
	return synth
}

// SavedFXEffects is the JSON-serializable form of audio.EffectDefinitions.
// It is stored as an optional object on SavedTrackRow; omitted when nil.
type SavedFXEffects struct {
	Ticks        int      `json:"Ticks,omitempty"`
	ArpeggioOffsets []int    `json:"arpeggio_offsets,omitempty"`
	PortamentoStart int      `json:"portamento_start,omitempty"`
	PortamentoTicks int      `json:"portamento_ticks,omitempty"`
	VibratoDepth    float64  `json:"vibrato_depth,omitempty"`
	VibratoRate     float64  `json:"vibrato_rate,omitempty"`
	Retrigger       bool     `json:"retrigger,omitempty"`
	VolumeLevel     *float64 `json:"volume_level,omitempty"`
	VolSlideDelta   float64  `json:"vol_slide_delta,omitempty"`
	NoteCutTick     int      `json:"note_cut_tick,omitempty"`
	NoteDelayTick   int      `json:"note_delay_tick,omitempty"`
}

// SavedTrackRow is the JSON-serializable form of TrackRow.
// FX is the sole source of truth; legacy compact fields have been removed.
type SavedTrackRow struct {
	Base   string          `json:"base"`
	Octave int             `json:"octave"`
	FX     *SavedFXEffects `json:"fx,omitempty"`
}

// SavedMeta holds optional display metadata for a patch loaded from the bank.
type SavedMeta struct {
	Bank string   `json:"bank,omitempty"`
	Name string   `json:"name,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// SavedTrack is the JSON-serializable form of Track
type SavedTrack struct {
	Synth SavedSynth      `json:"synth"`
	Meta  *SavedMeta      `json:"meta,omitempty"`
	Rows  []SavedTrackRow `json:"rows"`
}

// SavedModule is the complete module structure for JSON serialization
type SavedModule struct {
	NumRows   int          `json:"num_rows"`
	NumTracks int          `json:"num_tracks"`
	BPM       int          `json:"bpm"`
	Tracks    []SavedTrack `json:"tracks"`
}

// TracksToModule converts the runtime TrackerModel to a SavedModule for serialization.
func TracksToModule(tracker *utracker.TrackerModel) *SavedModule {
	saved := &SavedModule{
		NumRows:   tracker.NumRows,
		NumTracks: tracker.NumTracks,
		BPM:       tracker.BPM.Value(),
		Tracks:    make([]SavedTrack, tracker.NumTracks),
	}

	for i, track := range tracker.Tracks {
		rows := make([]SavedTrackRow, len(track.Rows))
		for j, row := range track.Rows {
			sr := SavedTrackRow{
				Base:   string(row.Note.Base),
				Octave: int(row.Note.Octave),
			}
			fx := row.FX
			sfx := &SavedFXEffects{
				Ticks:      fx.Ticks,
				Retrigger:     fx.RetriggerEnvelope,
				VolSlideDelta: fx.VolumeSlide.Delta,
				NoteCutTick:   fx.NoteCut.Tick,
				NoteDelayTick: fx.NoteDelay.Tick,
			}
			if fx.Volume.Active {
				lvl := fx.Volume.Level
				sfx.VolumeLevel = &lvl
			}
			if fx.Pitch.Arpeggio != nil {
				sfx.ArpeggioOffsets = fx.Pitch.Arpeggio.Offsets
			}
			if fx.Pitch.Portamento != nil {
				sfx.PortamentoStart = fx.Pitch.Portamento.StartTick
				sfx.PortamentoTicks = fx.Pitch.Portamento.Ticks
			}
			if fx.Pitch.Vibrato != nil {
				sfx.VibratoDepth = fx.Pitch.Vibrato.Depth
				sfx.VibratoRate = fx.Pitch.Vibrato.Rate
			}
			// Only attach FX when there is something non-default to store.
			if sfx.Ticks != 0 || sfx.Retrigger || sfx.VolSlideDelta != 0 ||
				sfx.NoteCutTick != 0 || sfx.NoteDelayTick != 0 || sfx.VolumeLevel != nil ||
				len(sfx.ArpeggioOffsets) > 0 || sfx.PortamentoStart != 0 || sfx.PortamentoTicks != 0 ||
				sfx.VibratoDepth != 0 {
				sr.FX = sfx
			}
			rows[j] = sr
		}
		st := SavedTrack{
			Synth: toSavedSynth(track.Synth),
			Rows:  rows,
		}
		if track.Synth != nil && track.Synth.Meta.Name != "" {
			m := track.Synth.Meta
			st.Meta = &SavedMeta{Bank: m.Bank, Name: m.Name, Tags: m.Tags}
		}
		saved.Tracks[i] = st
	}
	return saved
}

// ModuleToTracks updates an existing TrackerModel with data from a SavedModule.
func ModuleToTracks(mod *SavedModule, tracker *utracker.TrackerModel) {
	const minVisibleTracks = 8
	const minVisibleRows = 64

	// Update tracker dimensions
	tracker.NumRows = mod.NumRows
	if tracker.NumRows < minVisibleRows {
		tracker.NumRows = minVisibleRows
	}
	tracker.NumTracks = mod.NumTracks
	if tracker.NumTracks < minVisibleTracks {
		tracker.NumTracks = minVisibleTracks
	}

	// Restore BPM; fall back to default for old saves that omit it
	if mod.BPM > 0 {
		tracker.BPM = utracker.NewBPM(mod.BPM)
	} else {
		tracker.BPM = utracker.NewBPM(utracker.DefaultBPM)
	}

	// Reset tracks to tracker defaults so any missing tracks/rows after loading
	// are valid and playable (default synth + empty note rows).
	defaults := utracker.NewTracker(tracker.NumTracks, tracker.NumRows, tracker.Viewport.Width, tracker.Viewport.Height)
	tracker.Tracks = defaults.Tracks

	// Update each track with saved data
	for i, savedTrack := range mod.Tracks {
		if i >= len(tracker.Tracks) {
			break
		}
		track := &tracker.Tracks[i]
		track.Synth = fromSavedSynth(savedTrack.Synth)
		if savedTrack.Meta != nil {
			track.Synth.Meta = audio.Metadata{Bank: savedTrack.Meta.Bank, Name: savedTrack.Meta.Name, Tags: savedTrack.Meta.Tags}
		}

		// Update each row with saved data
		for j, row := range savedTrack.Rows {
			if j < len(track.Rows) {
				tr := utracker.TrackRow{
					Note: notes.Note{Base: notes.Base(row.Base), Octave: notes.Octave(row.Octave)},
				}
				if row.FX != nil {
					fx := audio.EffectDefinitions{
						Ticks:          row.FX.Ticks,
						RetriggerEnvelope: row.FX.Retrigger,
						VolumeSlide:       audio.VolumeSlideEffect{Delta: row.FX.VolSlideDelta},
						NoteCut:           audio.NoteCutEffect{Tick: row.FX.NoteCutTick},
						NoteDelay:         audio.NoteDelayEffect{Tick: row.FX.NoteDelayTick},
					}
					if row.FX.VolumeLevel != nil {
						fx.Volume = audio.VolumeEffect{Active: true, Level: *row.FX.VolumeLevel}
					}
					if len(row.FX.ArpeggioOffsets) > 0 {
						a := audio.ArpeggioEffect{Offsets: row.FX.ArpeggioOffsets}
						fx.Pitch.Arpeggio = &a
					} else if row.FX.PortamentoStart > 0 || row.FX.PortamentoTicks > 0 {
						p := audio.PortamentoEffect{StartTick: row.FX.PortamentoStart, Ticks: row.FX.PortamentoTicks}
						fx.Pitch.Portamento = &p
					} else if row.FX.VibratoDepth > 0 {
						v := audio.VibratoEffect{Depth: row.FX.VibratoDepth, Rate: row.FX.VibratoRate}
						fx.Pitch.Vibrato = &v
					}
					tr.FX = fx
				}
				track.Rows[j] = tr
			}
		}
	}

	// Reset cursor to safe position
	cursorRow := 0
	cursorTrack := 0
	if tracker.NumTracks > 0 && cursorTrack >= tracker.NumTracks {
		cursorTrack = 0
	}
	if tracker.NumRows > 0 && cursorRow >= tracker.NumRows {
		cursorRow = 0
	}
	tracker.SetCursorPosition(cursorRow, cursorTrack)
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
	return LoadFromBytes(data)
}

// LoadFromBytes reads JSON bytes and returns a SavedModule.
func LoadFromBytes(data []byte) (*SavedModule, error) {
	var saved SavedModule
	err := json.Unmarshal(data, &saved)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}
