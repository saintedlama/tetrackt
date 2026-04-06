package ui

import "github.com/tetrackt/tetrackt/audio"

// TrackChanged is emitted when the user moves to a different track in the
// tracker, so the synth panel can display that track's settings.
type TrackChanged struct {
	Synth *audio.Synth
}

// SynthUpdated is emitted when any parameter of the current track's synth
// changes. It is consumed by SynthScreen (to sync panels on preset load) and
// TrackerScreen (to keep track.Synth in sync).
type SynthUpdated struct {
	Synth *audio.Synth
}
