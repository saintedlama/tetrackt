package persistence

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"
)

func TestQuickstartModuleLoads(t *testing.T) {
	const quickstartPath = "../modules/quickstart.json"

	_, err := os.Stat(quickstartPath)
	require.NoError(t, err, "quickstart module is missing")

	mod, err := LoadFromFile(quickstartPath)
	require.NoError(t, err, "failed to load quickstart module")

	require.True(t, mod.NumTracks != 0 && mod.NumRows != 0,
		"quickstart module has invalid dimensions: tracks=%d rows=%d", mod.NumTracks, mod.NumRows)

	m := tracker.NewTracker(1, 1, 0, 0)
	ModuleToTracks(mod, m)
	expectedTracks := max(8, mod.NumTracks)
	require.Equal(t, expectedTracks, len(m.Tracks), "expected %d tracks after load", expectedTracks)
	expectedRows := max(64, mod.NumRows)
	require.Equal(t, expectedRows, m.NumRows, "expected %d rows after load", expectedRows)
}

func TestChiptuneDemoModuleLoads(t *testing.T) {
	const demoPath = "../modules/chiptune-demo.json"

	_, err := os.Stat(demoPath)
	require.NoError(t, err, "chiptune demo module is missing")

	mod, err := LoadFromFile(demoPath)
	require.NoError(t, err, "failed to load chiptune demo module")

	require.Equal(t, 8, mod.NumTracks, "expected 8 tracks")
	require.GreaterOrEqual(t, mod.NumRows, 16, "expected at least 16 rows")

	m := tracker.NewTracker(1, 1, 0, 0)
	ModuleToTracks(mod, m)
	expectedTracks := max(8, mod.NumTracks)
	require.Equal(t, expectedTracks, len(m.Tracks), "expected %d tracks after load", expectedTracks)
	expectedRows := max(64, mod.NumRows)
	require.Equal(t, expectedRows, m.NumRows, "expected %d rows after load", expectedRows)
}

func TestModuleToTracksPadsToEightWithDefaults(t *testing.T) {
	mod := &SavedModule{
		NumRows:   4,
		NumTracks: 2,
		BPM:       120,
		Tracks: []SavedTrack{
			{
				Synth: ToSavedSynth(audio.NewSynth(
					audio.Oscillator{Type: audio.Square},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Oscillator{Type: audio.Silent},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Oscillator{Type: audio.Silent},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Mixer{Volume1: 1.0, Volume2: 1.0},
					audio.NewFilter(),
					audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Dest: audio.ModPitch},
					audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Dest: audio.ModVolume},
					audio.LFO{},
				)),
				Rows: []SavedTrackRow{{Base: "C", Octave: 4, Volume: 64}},
			},
			{
				Synth: ToSavedSynth(audio.NewSynth(
					audio.Oscillator{Type: audio.Triangle},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Oscillator{Type: audio.Silent},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Oscillator{Type: audio.Silent},
					audio.Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0},
					audio.Mixer{Volume1: 1.0, Volume2: 1.0},
					audio.NewFilter(),
					audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Dest: audio.ModPitch},
					audio.LFO{Waveform: audio.LFOSine, Rate: 1.0, Dest: audio.ModVolume},
					audio.LFO{},
				)),
				Rows: []SavedTrackRow{{Base: "E", Octave: 3, Volume: 48}},
			},
		},
	}

	m := tracker.NewTracker(8, 16, 0, 0)
	ModuleToTracks(mod, m)

	require.Equal(t, 8, m.NumTracks, "expected 8 tracks")
	require.NotNil(t, m.Tracks[2].Synth, "expected padded track synth to be default-initialized")
	require.Equal(t, 64, len(m.Tracks[2].Rows), "expected padded track rows to be 64")

	assert.Equal(t, audio.Off(), m.Tracks[2].Rows[0].Note,
		"expected padded row to be empty note")
	assert.Equal(t, 0, m.Tracks[2].Rows[0].Volume,
		"expected padded row to have zero volume")

	assert.Equal(t, audio.Off(), m.Tracks[0].Rows[63].Note,
		"expected padded tail row to be empty note")
	assert.Equal(t, 0, m.Tracks[0].Rows[63].Volume,
		"expected padded tail row to have zero volume")
}

func TestLoadFromBytes(t *testing.T) {
	data := []byte(`{"num_rows":2,"num_tracks":1,"bpm":120,"speed":6,"tracks":[{"synth":{"oscillator1":{"type":"sine"},"envelope1":{"attack":0,"decay":0,"sustain":1,"release":0},"lfo1":{"waveform":"sine","rate":1,"dest":0},"oscillator2":{"type":"silent"},"envelope2":{"attack":0,"decay":0,"sustain":1,"release":0},"lfo2":{"waveform":"sine","rate":1,"dest":1},"oscillator3":{"type":"silent"},"envelope3":{"attack":0,"decay":0,"sustain":1,"release":0},"lfo3":{"waveform":"sine","rate":1,"dest":0},"mixer":{"volume1":1,"volume2":1},"filter":{"type":"off","cutoff":0.5,"resonance":0}},"rows":[{"base":"C","octave":4,"volume":64},{"base":"---","octave":0,"volume":0}]}]}`)

	mod, err := LoadFromBytes(data)
	require.NoError(t, err, "LoadFromBytes failed")
	assert.Equal(t, 2, mod.NumRows, "unexpected rows")
	assert.Equal(t, 1, mod.NumTracks, "unexpected tracks")
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
	require.NoError(t, err, "SaveToFile failed")

	// Load from file
	loadedMod, err := LoadFromFile(tmpFile)
	require.NoError(t, err, "LoadFromFile failed")

	// Create a new tracker and load data into it
	newTracker := tracker.NewTracker(8, 64, 0, 0) // Different dimensions initially
	ModuleToTracks(loadedMod, newTracker)

	// Verify dimensions were updated
	assert.Equal(t, 64, newTracker.NumRows, "expected NumRows=64")
	assert.Equal(t, 8, newTracker.NumTracks, "expected NumTracks=8")

	// Verify track data
	assert.Equal(t, audio.Sine, newTracker.Tracks[0].Synth.Oscillator1.Type, "expected Oscillator1=Sine")
	assert.Equal(t, audio.Square, newTracker.Tracks[0].Synth.Oscillator2.Type, "expected Oscillator2=Square")
	assert.Equal(t, audio.Mixer{Volume1: 0.75, Volume2: 0.5}, newTracker.Tracks[0].Synth.Mixer, "expected Mixer={0.75, 0.5}")

	// Verify row data
	assert.Equal(t, audio.NewNote("C", 4), newTracker.Tracks[0].Rows[0].Note, "expected Note=C-4")
	assert.Equal(t, 64, newTracker.Tracks[0].Rows[0].Volume, "expected Volume=64")
	assert.Equal(t, audio.NewNote("E", 4), newTracker.Tracks[0].Rows[1].Note, "expected Note=E-4")
	assert.Equal(t, 80, newTracker.Tracks[0].Rows[1].Volume, "expected Volume=80")

	// Verify envelope data
	assert.Equal(t, 100*time.Millisecond, newTracker.Tracks[0].Synth.Envelope1.Attack, "expected Attack=100ms")
}

func TestTrackPatchMetadataRoundTrip(t *testing.T) {
	trackerModel := tracker.NewTracker(8, 16, 0, 0)
	trackerModel.Tracks[0].Synth.Meta = audio.Metadata{Name: "Square Lead", Bank: "Lead"}
	trackerModel.Tracks[0].Synth.Oscillator1 = audio.Oscillator{Type: audio.Square, PulseWidth: 0.25, Detune: 5}

	tmpFile := "test_patch_meta_module.json"
	defer os.Remove(tmpFile)

	mod := TracksToModule(trackerModel)
	err := SaveToFile(tmpFile, mod)
	require.NoError(t, err, "SaveToFile failed")

	loadedMod, err := LoadFromFile(tmpFile)
	require.NoError(t, err, "LoadFromFile failed")

	newTracker := tracker.NewTracker(8, 64, 0, 0)
	ModuleToTracks(loadedMod, newTracker)

	require.Equal(t, "Square Lead", newTracker.Tracks[0].Synth.Meta.Name, "expected patch name to round-trip")
	require.Equal(t, "Lead", newTracker.Tracks[0].Synth.Meta.Bank, "expected patch bank to round-trip")

	osc := newTracker.Tracks[0].Synth.Oscillator1
	require.Equal(t, audio.Square, osc.Type, "expected oscillator type to round-trip")
	require.Equal(t, 0.25, osc.PulseWidth, "expected pulse width to round-trip")
	require.Equal(t, 5.0, osc.Detune, "expected detune to round-trip")
}
