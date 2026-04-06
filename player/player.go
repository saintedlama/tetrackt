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

// Player owns sequencer playback state: sub-tick clock, active voices, and arp stepping.
// It is the single place that understands the relationship between BPM, Speed, rows, and arpeggio.
type Player struct {
	subTickCount int
	activeVoices []*audio.ActiveVoice

	// preview state for live note-key arp preview
	previewVoice   *audio.ActiveVoice
	previewNote    audio.Note
	previewArp     audio.ArpeggioEffect
	previewSubTick int
	previewSpeed   int
}

// Reset clears playback state. Call when playback starts or restarts.
func (p *Player) Reset() {
	p.subTickCount = 0
	p.activeVoices = nil
	p.previewVoice = nil
}

// StartPreview plays a single note preview using the given synth and arpeggio.
// Returns true if the arp is active, indicating the caller should drive
// TickPreview calls via a timer.
func (p *Player) StartPreview(note audio.Note, arp audio.ArpeggioEffect, s *audio.Synth, duration time.Duration, sampleRate beep.SampleRate, globalVolume float64, speed int) bool {
	voice := s.Streamer(sampleRate, note.Frequency(), duration)
	p.previewVoice = voice
	p.previewNote = note
	p.previewArp = arp
	p.previewSubTick = 0
	if speed <= 0 {
		speed = tracker.DefaultSpeed
	}
	p.previewSpeed = speed

	volumeAdjusted := &effects.Volume{
		Streamer: voice,
		Base:     2,
		Volume:   volumeToDecibels(globalVolume),
		Silent:   globalVolume == 0,
	}
	speaker.Play(volumeAdjusted)
	return arp.IsActive()
}

// TickPreview advances the arp by one sub-tick for the current live preview.
// Returns true while the preview is still running; false when done.
func (p *Player) TickPreview() bool {
	if p.previewVoice == nil || !p.previewArp.IsActive() {
		return false
	}

	p.previewSubTick++
	if p.previewSubTick >= p.previewSpeed {
		p.previewVoice = nil
		return false
	}

	step := (p.previewSubTick / p.previewArp.TicksPerStep) % len(p.previewArp.Offsets)
	transposed, ok := p.previewNote.Transpose(p.previewArp.Offsets[step])
	if ok {
		speaker.Lock()
		p.previewVoice.SetFrequency(transposed.Frequency())
		speaker.Unlock()
	}
	return true
}

// Tick processes one sub-tick of playback:
//   - On the first sub-tick of a row, starts audio for all notes in that row.
//   - On every sub-tick, applies arpeggio retuning to active voices.
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

	// Apply arpeggio retuning for each active voice on this sub-tick
	for trackIdx, voice := range p.activeVoices {
		if voice == nil || trackIdx >= len(trackerModel.Tracks) {
			continue
		}
		trackRows := trackerModel.Tracks[trackIdx].Rows
		if row >= len(trackRows) {
			continue
		}
		arp := trackRows[row].Arpeggio
		if !arp.IsActive() {
			continue
		}
		step := (p.subTickCount / arp.TicksPerStep) % len(arp.Offsets)
		transposed, ok := trackRows[row].Note.Transpose(arp.Offsets[step])
		if ok {
			speaker.Lock()
			voice.SetFrequency(transposed.Frequency())
			speaker.Unlock()
		}
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
func (p *Player) playRowNotes(trackerModel *tracker.TrackerModel, row int, sampleRate beep.SampleRate, globalVolume float64) []*audio.ActiveVoice {
	if row < 0 || row >= trackerModel.NumRows {
		return nil
	}

	duration := trackerModel.BPMDuration()
	voices := make([]*audio.ActiveVoice, trackerModel.NumTracks)
	var streamers []beep.Streamer

	for trackIdx := 0; trackIdx < trackerModel.NumTracks; trackIdx++ {
		track := trackerModel.Tracks[trackIdx]
		trackRow := track.Rows[row]

		if audio.IsOff(trackRow.Note) {
			continue
		}

		voice := track.Synth.Streamer(sampleRate, trackRow.Note.Frequency(), duration)
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
