package player

import (
	"math"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
)

// TODO: This file cries for a refactoring!

// Player owns sequencer playback state: sub-tick clock and active voices.
type Player struct {
	subTickCount int
	activeVoices []beep.Streamer
	previewVoice beep.Streamer
}

// Reset clears playback state. Call when playback starts or restarts.
func (p *Player) Reset() {
	p.subTickCount = 0
	p.activeVoices = nil
	p.previewVoice = nil
}

// arpFrequencies converts an ArpeggioEffect into a frequency slice suitable for
// Synth.Streamer. Each offset is applied as a frequency multiplier 2^(offset/12),
// making the computation strictly frequency-based with no note-type involvement.
// Returns ([]float64{baseFreq}, 1) when the arp is inactive.
func arpFrequencies(baseFreq float64, arp audio.ArpeggioEffect) ([]float64, int) {
	if !arp.IsActive() {
		return []float64{baseFreq}, 1
	}
	freqs := make([]float64, len(arp.Offsets))
	for i, offset := range arp.Offsets {
		freqs[i] = baseFreq * math.Pow(2.0, float64(offset)/12.0)
	}
	return freqs, len(arp.Offsets)
}

// StartPreview plays a single note preview using the given synth and arpeggio.
// Arp cycling is handled internally by the streamer; always returns false.
func (p *Player) StartPreview(note audio.Note, arp audio.ArpeggioEffect, s *audio.Synth, duration time.Duration, sampleRate beep.SampleRate, globalVolume float64, speed int) bool {
	frequencies, tickCount := arpFrequencies(note.Frequency(), arp)
	voice := s.Streamer(sampleRate, frequencies, tickCount, true, duration)
	p.previewVoice = voice

	volumeAdjusted := &effects.Volume{
		Streamer: voice,
		Base:     2,
		Volume:   volumeToDecibels(globalVolume),
		Silent:   globalVolume == 0,
	}
	speaker.Play(volumeAdjusted)
	return false
}

// TickPreview is a no-op; arp cycling is handled internally by the streamer.
func (p *Player) TickPreview() bool {
	return false
}

// Tick processes one sub-tick of playback:
//   - On the first sub-tick of a row, starts audio for all notes in that row.
//   - Advances the row counter once all sub-ticks for the row are consumed.
func (p *Player) Tick(trackerModel *tracker.TrackerModel, sampleRate beep.SampleRate, globalVolume float64) {
	speed := trackerModel.Speed
	if speed <= 0 {
		speed = tracker.DefaultSpeed
	}

	row := trackerModel.PlaybackRow

	if p.subTickCount == 0 {
		p.activeVoices = p.playRowNotes(trackerModel, row, sampleRate, globalVolume)
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
// a slice of active voices indexed by track (nil for empty/off rows).
func (p *Player) playRowNotes(trackerModel *tracker.TrackerModel, row int, sampleRate beep.SampleRate, globalVolume float64) []beep.Streamer {
	if row < 0 || row >= trackerModel.NumRows {
		return nil
	}

	duration := trackerModel.BPMDuration()
	voices := make([]beep.Streamer, trackerModel.NumTracks)
	var streamers []beep.Streamer

	for trackIdx := 0; trackIdx < trackerModel.NumTracks; trackIdx++ {
		track := trackerModel.Tracks[trackIdx]
		trackRow := track.Rows[row]

		if audio.IsOff(trackRow.Note) {
			continue
		}

		frequencies, tickCount := arpFrequencies(trackRow.Note.Frequency(), trackRow.Arpeggio)
		voice := track.Synth.Streamer(sampleRate, frequencies, tickCount, true, duration)
		voices[trackIdx] = voice
		streamers = append(streamers, voice)
	}

	if len(streamers) > 0 {
		mixed := beep.Mix(streamers...)
		volumeAdjusted := &effects.Volume{
			Streamer: mixed,
			Base:     2,
			Volume:   volumeToDecibels(globalVolume),
			Silent:   globalVolume == 0,
		}
		speaker.Play(volumeAdjusted)
	}
	return voices
}

func volumeToDecibels(volume float64) float64 {
	if volume <= 0 {
		return -999
	}
	return math.Log2(volume) * 6
}
