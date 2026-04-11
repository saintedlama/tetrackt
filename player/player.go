package player

import (
	"math"
	"time"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"

	"github.com/gopxl/beep/v2/speaker"
)

// TODO: This file cries for a refactoring!

// channelEffectState holds per-tick mutable state for a single track's active effect.
type channelEffectState struct {
	vibratoPhase float64
	volume       float64 // current output scalar; 1.0 = unity
}

// Player owns sequencer playback state: sub-tick clock and active patches.
type Player struct {
	subTickCount    int
	activePatches   []*audio.Patch
	previewPatch    *audio.Patch
	prevFrequencies []float64 // last triggered frequency per track; 0 = no prior note
	arpTickIdx      []int     // current arpeggio step per track; -1 = inactive
	effectStates    []channelEffectState

	// Preview arp state
	previewArp      audio.ArpeggioEffect
	previewBaseFreq float64
	previewSubTick  int
	previewSpeed    int
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
	p.prevFrequencies = nil
	p.arpTickIdx = nil
	p.effectStates = nil
	p.previewArp = audio.ArpeggioEffect{}
	p.previewBaseFreq = 0
	p.previewSubTick = 0
	p.previewSpeed = 0
}

// StartPreview plays a single note preview using the given synth.
// When arp is active it applies the first step immediately and returns true
// so the caller schedules previewTick calls to cycle through remaining steps.
func (p *Player) StartPreview(note audio.Note, arp audio.ArpeggioEffect, s *audio.Synth, duration time.Duration, sampleRate audio.SampleRate, globalVolume float64, speed int) bool {
	noteSamples := sampleRate.N(duration)
	patch := s.NewPatch(sampleRate, note.Frequency(), noteSamples)
	p.previewPatch = patch

	if arp.IsActive() {
		if speed <= 0 {
			speed = tracker.DefaultSpeed
		}
		p.previewArp = arp
		p.previewBaseFreq = note.Frequency()
		p.previewSubTick = 0
		p.previewSpeed = speed
		// Apply step 0 immediately (mirrors Tick() which also applies step 0 on sub-tick 0).
		mult := math.Pow(2, float64(arp.Offsets[0])/12)
		patch.SetFrequency(p.previewBaseFreq * mult)
		speaker.Play(audio.NewVolume(globalVolume).Streamer(patch))
		return true
	}

	p.previewArp = audio.ArpeggioEffect{}
	speaker.Play(audio.NewVolume(globalVolume).Streamer(patch))
	return false
}

// TickPreview advances the arp preview one sub-tick and updates the patch frequency.
// Returns true while there are more sub-ticks to cycle, false when done.
func (p *Player) TickPreview() bool {
	if p.previewPatch == nil || !p.previewArp.IsActive() {
		return false
	}
	p.previewSubTick++
	if p.previewSubTick >= p.previewSpeed {
		p.previewArp = audio.ArpeggioEffect{}
		return false
	}
	idx := p.previewSubTick % len(p.previewArp.Offsets)
	mult := math.Pow(2, float64(p.previewArp.Offsets[idx])/12)
	p.previewPatch.SetFrequency(p.previewBaseFreq * mult)
	return true
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

	for trackIdx, patch := range p.activePatches {
		if patch == nil || trackIdx >= len(trackerModel.Tracks) {
			continue
		}
		trackRow := trackerModel.Tracks[trackIdx].Rows[row]

		// Arpeggio cycling
		if trackIdx < len(p.arpTickIdx) && trackRow.Arpeggio.IsActive() {
			p.arpTickIdx[trackIdx]++
			idx := p.arpTickIdx[trackIdx] % len(trackRow.Arpeggio.Offsets)
			mult := math.Pow(2, float64(trackRow.Arpeggio.Offsets[idx])/12)
			if trackIdx < len(p.prevFrequencies) {
				patch.SetFrequency(p.prevFrequencies[trackIdx] * mult)
			}
		}

		// Per-tick effects
		if trackIdx < len(p.effectStates) {
			state := &p.effectStates[trackIdx]
			switch trackRow.Effect.Type {
			case tracker.EffectVibrato:
				vibratoSpeed := (trackRow.Effect.Param >> 4) & 0xF
				vibratoDepth := float64(trackRow.Effect.Param & 0xF)
				if vibratoSpeed > 0 {
					state.vibratoPhase += (2 * math.Pi) / float64(vibratoSpeed)
				}
				semitones := (vibratoDepth / 4.0) * math.Sin(state.vibratoPhase)
				mult := math.Pow(2, semitones/12)
				if trackIdx < len(p.prevFrequencies) {
					patch.SetFrequency(p.prevFrequencies[trackIdx] * mult)
				}
			case tracker.EffectVolumeSlide:
				state.volume = math.Max(0, math.Min(1, state.volume+float64(trackRow.Effect.Param)/64.0))
				patch.SetVolume(state.volume)
			case tracker.EffectNoteCut:
				if p.subTickCount == trackRow.Effect.Param {
					patch.SetVolume(0)
				}
			case tracker.EffectNoteDelay:
				if p.subTickCount == trackRow.Effect.Param {
					patch.NoteOn()
				}
			}
		}

		patch.TickPortamento()
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
func (p *Player) playRowNotes(trackerModel *tracker.TrackerModel, row int, sampleRate audio.SampleRate, globalVolume float64) []*audio.Patch {
	if row < 0 || row >= trackerModel.NumRows {
		return nil
	}

	duration := trackerModel.BPMDuration()
	patches := make([]*audio.Patch, trackerModel.NumTracks)
	var streamers []audio.Streamer

	// Grow prevFrequencies if the track count increased
	if len(p.prevFrequencies) < trackerModel.NumTracks {
		grown := make([]float64, trackerModel.NumTracks)
		copy(grown, p.prevFrequencies)
		p.prevFrequencies = grown
	}

	// Grow arpTickIdx and effectStates if the track count increased
	if len(p.arpTickIdx) < trackerModel.NumTracks {
		grown := make([]int, trackerModel.NumTracks)
		copy(grown, p.arpTickIdx)
		p.arpTickIdx = grown
	}
	if len(p.effectStates) < trackerModel.NumTracks {
		grown := make([]channelEffectState, trackerModel.NumTracks)
		copy(grown, p.effectStates)
		p.effectStates = grown
	}

	for trackIdx := 0; trackIdx < trackerModel.NumTracks; trackIdx++ {
		track := trackerModel.Tracks[trackIdx]
		trackRow := track.Rows[row]

		// Fire NoteOff on the previous patch for this track so its release plays out.
		if len(p.activePatches) > trackIdx && p.activePatches[trackIdx] != nil {
			p.activePatches[trackIdx].NoteOff()
		}

		if audio.IsOff(trackRow.Note) {
			p.prevFrequencies[trackIdx] = 0
			continue
		}

		targetFrequency := trackRow.Note.Frequency()
		noteSamples := sampleRate.N(duration)
		patch := track.Synth.NewGatedPatch(sampleRate, targetFrequency)

		if track.Synth.Portamento > 0 && p.prevFrequencies[trackIdx] > 0 {
			ticks := int(math.Round(track.Synth.Portamento * float64(sampleRate) / float64(noteSamples)))
			patch.StartPortamento(p.prevFrequencies[trackIdx], targetFrequency, ticks)
		}
		p.prevFrequencies[trackIdx] = targetFrequency

		// Reset per-channel effect state for the new row
		p.arpTickIdx[trackIdx] = -1
		p.effectStates[trackIdx] = channelEffectState{volume: 1.0}

		// NoteDelay suppresses immediate NoteOn; it fires in Tick() at the target sub-tick
		if trackRow.Effect.Type != tracker.EffectNoteDelay {
			patch.NoteOn()
		}
		patches[trackIdx] = patch
		streamers = append(streamers, patch)
	}

	speaker.Play(audio.NewVolume(globalVolume).Streamer(streamers...))
	return patches
}
