package render

import (
	"fmt"
	"math"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/tracker"
)

type channelEffectState struct {
	vibratoPhase float64
	volume       float64
}

type RenderConfig struct {
	SampleRate   audio.SampleRate
	GlobalVolume float64
	LoopCount    int
}

type RenderEngine struct {
	trackerModel    *tracker.TrackerModel
	cfg             RenderConfig
	subTickCount    int
	currentPatches  []*audio.Patch
	activeVoices    []*audio.Patch
	prevFrequencies []float64
	arpTickIdx      []int
	effectStates    []channelEffectState
}

func NewRenderEngine(m *tracker.TrackerModel, cfg RenderConfig) *RenderEngine {
	return &RenderEngine{trackerModel: m, cfg: cfg}
}

func (e *RenderEngine) Run(sink RenderSink) error {
	if e.trackerModel == nil {
		return fmt.Errorf("render: tracker model is nil")
	}
	if e.trackerModel.NumRows <= 0 || e.trackerModel.NumTracks <= 0 {
		return fmt.Errorf("render: tracker is empty")
	}
	if e.cfg.SampleRate <= 0 {
		return fmt.Errorf("render: invalid sample rate %d", e.cfg.SampleRate)
	}
	if e.cfg.LoopCount <= 0 {
		e.cfg.LoopCount = 1
	}
	if e.cfg.GlobalVolume < 0 {
		e.cfg.GlobalVolume = 1.0
	}

	e.resetState()
	e.subTickCount = 0
	if err := sink.Begin(e.cfg.SampleRate); err != nil {
		return err
	}

	for loopIdx := 0; loopIdx < e.cfg.LoopCount; loopIdx++ {
		for rowIdx := 0; rowIdx < e.trackerModel.NumRows; rowIdx++ {
			if err := e.renderRow(rowIdx, sink); err != nil {
				return err
			}
		}
	}

	e.releaseCurrentPatches()
	if err := e.drainVoices(sink); err != nil {
		return err
	}

	return sink.End()
}

func (e *RenderEngine) StartLive(sink RenderSink) error {
	if err := e.validateConfig(); err != nil {
		return err
	}
	e.resetState()
	e.subTickCount = 0
	return sink.Begin(e.cfg.SampleRate)
}

func (e *RenderEngine) TickLive(sink RenderSink, globalVolume float64) error {
	if err := e.validateConfig(); err != nil {
		return err
	}
	if globalVolume >= 0 {
		e.cfg.GlobalVolume = globalVolume
	}
	if e.subTickCount == 0 {
		e.startRow(e.trackerModel.PlaybackRow)
	}

	rowIdx := e.trackerModel.PlaybackRow
	e.applyTick(rowIdx, e.subTickCount)
	if err := e.mixActiveVoices(e.tickSampleCount(rowIdx, e.subTickCount), sink); err != nil {
		return err
	}
	e.advancePlaybackRow()
	return nil
}

func (e *RenderEngine) StopLive(sink RenderSink) error {
	e.releaseCurrentPatches()
	return sink.End()
}

func (e *RenderEngine) validateConfig() error {
	if e.trackerModel == nil {
		return fmt.Errorf("render: tracker model is nil")
	}
	if e.trackerModel.NumRows <= 0 || e.trackerModel.NumTracks <= 0 {
		return fmt.Errorf("render: tracker is empty")
	}
	if e.cfg.SampleRate <= 0 {
		return fmt.Errorf("render: invalid sample rate %d", e.cfg.SampleRate)
	}
	if e.cfg.LoopCount <= 0 {
		e.cfg.LoopCount = 1
	}
	if e.cfg.GlobalVolume < 0 {
		e.cfg.GlobalVolume = 1.0
	}
	return nil
}

func (e *RenderEngine) resetState() {
	e.currentPatches = make([]*audio.Patch, e.trackerModel.NumTracks)
	e.activeVoices = nil
	e.prevFrequencies = make([]float64, e.trackerModel.NumTracks)
	e.arpTickIdx = make([]int, e.trackerModel.NumTracks)
	e.effectStates = make([]channelEffectState, e.trackerModel.NumTracks)
	for i := range e.arpTickIdx {
		e.arpTickIdx[i] = -1
	}
}

func (e *RenderEngine) renderRow(rowIdx int, sink RenderSink) error {
	e.startRow(rowIdx)
	e.subTickCount = 0
	ticks := e.trackerModel.RowTicks(rowIdx)

	for subTick := 0; subTick < ticks; subTick++ {
		tickSamples := e.tickSampleCount(rowIdx, subTick)
		e.applyTick(rowIdx, subTick)
		if err := e.mixActiveVoices(tickSamples, sink); err != nil {
			return err
		}
	}

	return nil
}

func (e *RenderEngine) startRow(rowIdx int) {
	noteSamples := int(e.cfg.SampleRate.N(e.trackerModel.BPMDuration()))
	e.releaseCurrentPatches()

	for trackIdx := 0; trackIdx < e.trackerModel.NumTracks; trackIdx++ {
		track := e.trackerModel.Tracks[trackIdx]
		trackRow := track.Rows[rowIdx]

		e.currentPatches[trackIdx] = nil
		if audio.IsOff(trackRow.Note) {
			e.prevFrequencies[trackIdx] = 0
			e.arpTickIdx[trackIdx] = -1
			e.effectStates[trackIdx] = channelEffectState{volume: 1.0}
			continue
		}

		targetFrequency := trackRow.Note.Frequency()
		patch := track.Synth.NewGatedPatch(e.cfg.SampleRate, targetFrequency)
		if trackRow.Volume > 0 {
			patch.SetVolume(float64(trackRow.Volume) / 64.0)
		}
		if track.Synth.Portamento > 0 && e.prevFrequencies[trackIdx] > 0 {
			ticks := int(math.Round(track.Synth.Portamento * float64(e.cfg.SampleRate) / float64(noteSamples)))
			patch.StartPortamento(e.prevFrequencies[trackIdx], targetFrequency, ticks)
		}
		e.prevFrequencies[trackIdx] = targetFrequency
		e.arpTickIdx[trackIdx] = -1
		e.effectStates[trackIdx] = channelEffectState{volume: 1.0}
		if trackRow.Effect.Type != tracker.EffectNoteDelay {
			patch.NoteOn()
		}
		e.currentPatches[trackIdx] = patch
		e.activeVoices = append(e.activeVoices, patch)
	}
}

func (e *RenderEngine) releaseCurrentPatches() {
	for trackIdx, patch := range e.currentPatches {
		if patch == nil {
			continue
		}
		patch.NoteOff()
		e.currentPatches[trackIdx] = nil
	}
}

func (e *RenderEngine) applyTick(rowIdx, subTick int) {
	for trackIdx, patch := range e.currentPatches {
		if patch == nil || trackIdx >= len(e.trackerModel.Tracks) {
			continue
		}
		trackRow := e.trackerModel.Tracks[trackIdx].Rows[rowIdx]

		if trackRow.Arpeggio.IsActive() {
			e.arpTickIdx[trackIdx]++
			idx := e.arpTickIdx[trackIdx] % len(trackRow.Arpeggio.Offsets)
			mult := math.Pow(2, float64(trackRow.Arpeggio.Offsets[idx])/12)
			patch.SetFrequency(e.prevFrequencies[trackIdx] * mult)
		}

		state := &e.effectStates[trackIdx]
		switch trackRow.Effect.Type {
		case tracker.EffectVibrato:
			vibratoSpeed := (trackRow.Effect.Param >> 4) & 0xF
			vibratoDepth := float64(trackRow.Effect.Param & 0xF)
			if vibratoSpeed > 0 {
				state.vibratoPhase += (2 * math.Pi) / float64(vibratoSpeed)
			}
			semitones := (vibratoDepth / 4.0) * math.Sin(state.vibratoPhase)
			mult := math.Pow(2, semitones/12)
			patch.SetFrequency(e.prevFrequencies[trackIdx] * mult)
		case tracker.EffectVolumeSlide:
			delta := trackRow.Effect.Param
			if delta > 127 {
				delta = int(int8(uint8(delta)))
			}
			state.volume = math.Max(0, math.Min(1, state.volume+float64(delta)/64.0))
			patch.SetVolume(state.volume)
		case tracker.EffectNoteCut:
			if subTick == trackRow.Effect.Param {
				patch.SetVolume(0)
			}
		case tracker.EffectNoteDelay:
			if subTick == trackRow.Effect.Param {
				patch.NoteOn()
			}
		}

		patch.TickPortamento()
	}
}

func (e *RenderEngine) mixActiveVoices(samplesPerTick int, sink RenderSink) error {
	if samplesPerTick <= 0 {
		return nil
	}

	mixBuf := make([][2]float64, samplesPerTick)
	if len(e.activeVoices) == 0 {
		return sink.Write(mixBuf)
	}

	remaining := e.activeVoices[:0]
	for _, patch := range e.activeVoices {
		voiceBuf := make([][2]float64, samplesPerTick)
		n, ok := patch.Stream(voiceBuf)
		for i := 0; i < n; i++ {
			mixBuf[i][0] += voiceBuf[i][0]
			mixBuf[i][1] += voiceBuf[i][1]
		}
		if ok {
			remaining = append(remaining, patch)
		}
	}
	e.activeVoices = remaining

	if e.cfg.GlobalVolume != 1.0 {
		for i := range mixBuf {
			mixBuf[i][0] *= e.cfg.GlobalVolume
			mixBuf[i][1] *= e.cfg.GlobalVolume
		}
	}

	return sink.Write(mixBuf)
}

func (e *RenderEngine) drainVoices(sink RenderSink) error {
	for len(e.activeVoices) > 0 {
		if err := e.mixActiveVoices(1024, sink); err != nil {
			return err
		}
	}
	return nil
}

func (e *RenderEngine) tickSampleCount(rowIdx, subTick int) int {
	ticks := e.trackerModel.RowTicks(rowIdx)
	rowSamples := int(e.cfg.SampleRate.N(e.trackerModel.BPMDuration()))
	baseTickSamples := rowSamples / ticks
	remainder := rowSamples % ticks
	if subTick < remainder {
		return baseTickSamples + 1
	}
	return baseTickSamples
}

func (e *RenderEngine) advancePlaybackRow() {
	e.subTickCount++
	if e.subTickCount < e.trackerModel.RowTicks(e.trackerModel.PlaybackRow) {
		return
	}
	e.subTickCount = 0
	e.trackerModel.PlaybackRow++
	if e.trackerModel.LoopToRow {
		if e.trackerModel.PlaybackRow > e.trackerModel.LoopEndRow {
			e.trackerModel.PlaybackRow = 0
		}
		return
	}
	if e.trackerModel.PlaybackRow >= e.trackerModel.NumRows {
		e.trackerModel.PlaybackRow = 0
	}
}
