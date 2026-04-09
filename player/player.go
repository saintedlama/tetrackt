package player

import (
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"

	"github.com/gopxl/beep/v2/speaker"
)

// TODO: This file cries for a refactoring!

// Player owns sequencer playback state: sub-tick clock and active patches.
type Player struct {
	subTickCount  int
	activePatches []audio.Streamer
	previewPatch  audio.Streamer
}

// Init initialises the speaker hardware with the given sample rate.
// Call once at startup before any playback.
func Init(sampleRate audio.SampleRate) {
	bufferSize := sampleRate.N(time.Millisecond * 100)
	speaker.Init(sampleRate, bufferSize)
}

// Clear stops all currently playing audio.
func (p *Player) Clear() {
	speaker.Clear()
}

// Play wraps streamer with volume and submits it to the speaker.
func (p *Player) Play(streamer audio.Streamer, globalVolume float64) {
	speaker.Play(audio.NewVolume(globalVolume).Streamer(streamer))
}

// Reset clears playback state. Call when playback starts or restarts.
func (p *Player) Reset() {
	p.subTickCount = 0
	p.activePatches = nil
	p.previewPatch = nil
}

// StartPreview plays a single note preview using the given synth.
func (p *Player) StartPreview(note audio.Note, arp audio.ArpeggioEffect, s *audio.Synth, duration time.Duration, sampleRate audio.SampleRate, globalVolume float64, speed int) bool {
	noteSamples := sampleRate.N(duration)
	patch := s.NewPatch(sampleRate, note.Frequency(), noteSamples)
	p.previewPatch = patch
	speaker.Play(audio.NewVolume(globalVolume).Streamer(patch))
	return false
}

// TickPreview is a no-op; arp cycling is handled internally by the streamer.
func (p *Player) TickPreview() bool {
	return false
}

// Tick processes one sub-tick of playback:
//   - On the first sub-tick of a row, starts audio for all notes in that row.
//   - Advances the row counter once all sub-ticks for the row are consumed.
func (p *Player) Tick(trackerModel *tracker.TrackerModel, sampleRate audio.SampleRate, globalVolume float64) {
	speed := trackerModel.Speed
	if speed <= 0 {
		speed = tracker.DefaultSpeed
	}

	row := trackerModel.PlaybackRow

	if p.subTickCount == 0 {
		p.activePatches = p.playRowNotes(trackerModel, row, sampleRate, globalVolume)
	}

	// Advance sub-tick counter; advance row when all sub-ticks are consumed
	p.subTickCount++
	if p.subTickCount >= speed {
		p.subTickCount = 0
		trackerModel.PlaybackRow++
		if trackerModel.LoopToRow {
			if trackerModel.PlaybackRow > trackerModel.LoopEndRow {
				trackerModel.PlaybackRow = 0
			}
		} else {
			if trackerModel.PlaybackRow >= trackerModel.NumRows {
				trackerModel.PlaybackRow = 0
			}
		}
	}
}

// playRowNotes starts audio for all non-empty notes in the given row and returns
// a slice of active patches indexed by track (nil for empty/off rows).
func (p *Player) playRowNotes(trackerModel *tracker.TrackerModel, row int, sampleRate audio.SampleRate, globalVolume float64) []audio.Streamer {
	if row < 0 || row >= trackerModel.NumRows {
		return nil
	}

	duration := trackerModel.BPMDuration()
	patches := make([]audio.Streamer, trackerModel.NumTracks)
	var streamers []audio.Streamer

	for trackIdx := 0; trackIdx < trackerModel.NumTracks; trackIdx++ {
		track := trackerModel.Tracks[trackIdx]
		trackRow := track.Rows[row]

		if audio.IsOff(trackRow.Note) {
			continue
		}

		noteSamples := sampleRate.N(duration)
		patch := track.Synth.NewPatch(sampleRate, trackRow.Note.Frequency(), noteSamples)
		patches[trackIdx] = patch
		streamers = append(streamers, patch)
	}

	speaker.Play(audio.NewVolume(globalVolume).Streamer(streamers...))
	return patches
}
