package player

import (
	"math"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"

	"github.com/gopxl/beep/v2/speaker"
)

// TODO: This file cries for a refactoring!

// Player owns sequencer playback state: sub-tick clock and active voices.
type Player struct {
	subTickCount int
	activeVoices []audio.Streamer
	previewVoice audio.Streamer
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
	p.activeVoices = nil
	p.previewVoice = nil
}

// arpFrequencies converts an ArpeggioEffect into a frequency slice suitable for
// Synth.Streamer. Each offset is applied as a frequency multiplier 2^(offset/12),
// making the computation strictly frequency-based with no note-type involvement.
// Returns []float64{baseFreq} when the arp is inactive.
func arpFrequencies(baseFreq float64, arp audio.ArpeggioEffect) []float64 {
	if !arp.IsActive() {
		return []float64{baseFreq}
	}
	freqs := make([]float64, len(arp.Offsets))
	for i, offset := range arp.Offsets {
		freqs[i] = baseFreq * math.Pow(2.0, float64(offset)/12.0)
	}
	return freqs
}

// StartPreview plays a single note preview using the given synth and arpeggio.
// Arp cycling is handled internally by the streamer; always returns false.
func (p *Player) StartPreview(note audio.Note, arp audio.ArpeggioEffect, s *audio.Synth, duration time.Duration, sampleRate audio.SampleRate, globalVolume float64, speed int) bool {
	frequencies := arpFrequencies(note.Frequency(), arp)
	continuous := arp.IsActive()
	voice := s.Streamer(sampleRate, audio.PlayParams{
		Frequencies: frequencies,
		Continuous:  continuous,
		Duration:    duration,
	})
	p.previewVoice = voice
	speaker.Play(audio.NewVolume(globalVolume).Streamer(voice))
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
func (p *Player) playRowNotes(trackerModel *tracker.TrackerModel, row int, sampleRate audio.SampleRate, globalVolume float64) []audio.Streamer {
	if row < 0 || row >= trackerModel.NumRows {
		return nil
	}

	duration := trackerModel.BPMDuration()
	voices := make([]audio.Streamer, trackerModel.NumTracks)
	var streamers []audio.Streamer

	for trackIdx := 0; trackIdx < trackerModel.NumTracks; trackIdx++ {
		track := trackerModel.Tracks[trackIdx]
		trackRow := track.Rows[row]

		if audio.IsOff(trackRow.Note) {
			continue
		}

		frequencies := arpFrequencies(trackRow.Note.Frequency(), trackRow.Arpeggio)
		continuous := trackRow.Continuous || trackRow.Arpeggio.IsActive()
		voice := track.Synth.Streamer(sampleRate, audio.PlayParams{
			Frequencies: frequencies,
			Continuous:  continuous,
			Duration:    duration,
		})
		voices[trackIdx] = voice
		streamers = append(streamers, voice)
	}

	speaker.Play(audio.NewVolume(globalVolume).Streamer(streamers...))
	return voices
}
