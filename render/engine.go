package render

import (
	"fmt"
	"math"

	"github.com/tetrackt/tetrackt/audio"
)

type channelEffectState struct {
	vibratoPhase float64
	volume       float64
}

type RenderConfig struct {
	SampleRate   audio.SampleRate
	GlobalVolume float64
	LoopCount    int
	// EndRow is the exclusive end row index for rendering (0 means all rows).
	EndRow int
}

type RenderEngine struct {
	song            *Pattern
	cfg             RenderConfig
	currentPatches  []*audio.Patch
	activeVoices    []*audio.Patch
	prevFrequencies []float64
	arpTickIdx      []int
	effectStates    []channelEffectState
}

func NewRenderEngine(song *Pattern, cfg RenderConfig) *RenderEngine {
	return &RenderEngine{song: song, cfg: cfg}
}

func (e *RenderEngine) Run(sink RenderSink) error {
	if e.song == nil {
		return fmt.Errorf("render: song is nil")
	}
	if e.song.NumRows <= 0 || e.song.NumTracks <= 0 {
		return fmt.Errorf("render: song is empty")
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
	if err := sink.Begin(e.cfg.SampleRate); err != nil {
		return err
	}

	endRow := e.song.NumRows
	if e.cfg.EndRow > 0 && e.cfg.EndRow < endRow {
		endRow = e.cfg.EndRow
	}

	for loopIdx := 0; loopIdx < e.cfg.LoopCount; loopIdx++ {
		for rowIdx := 0; rowIdx < endRow; rowIdx++ {
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

// RenderToStream renders the song fully offline and returns a Streamer ready
// for playback. If loop is true, the stream will repeat indefinitely.
// Returns nil, nil when the render produces no audio (e.g. all tracks empty).
func RenderToStream(song *Pattern, cfg RenderConfig, loop bool) (audio.Streamer, error) {
	collector := &bufferSink{}
	engine := NewRenderEngine(song, cfg)
	if err := engine.Run(collector); err != nil {
		return nil, err
	}
	if len(collector.frames) == 0 {
		return nil, nil
	}
	return &sampleStreamer{samples: collector.frames, loop: loop}, nil
}

func (e *RenderEngine) resetState() {
	e.currentPatches = make([]*audio.Patch, e.song.NumTracks)
	e.activeVoices = nil
	e.prevFrequencies = make([]float64, e.song.NumTracks)
	e.arpTickIdx = make([]int, e.song.NumTracks)
	e.effectStates = make([]channelEffectState, e.song.NumTracks)
	for i := range e.arpTickIdx {
		e.arpTickIdx[i] = -1
	}
}

func (e *RenderEngine) renderRow(rowIdx int, sink RenderSink) error {
	e.startRow(rowIdx)
	ticks := e.song.RowTicks(rowIdx)

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
	noteSamples := int(e.cfg.SampleRate.N(e.song.RowDuration))
	e.releaseCurrentPatches()

	for trackIdx := 0; trackIdx < e.song.NumTracks; trackIdx++ {
		track := e.song.Tracks[trackIdx]
		row := track.Rows[rowIdx]

		e.currentPatches[trackIdx] = nil
		if row.Frequency == 0 {
			e.prevFrequencies[trackIdx] = 0
			e.arpTickIdx[trackIdx] = -1
			e.effectStates[trackIdx] = channelEffectState{volume: 1.0}
			continue
		}

		patch := track.Synth.NewGatedPatch(e.cfg.SampleRate, row.Frequency)
		if row.Volume > 0 {
			patch.SetVolume(row.Volume)
		}
		if track.Synth.Portamento > 0 && e.prevFrequencies[trackIdx] > 0 {
			ticks := int(math.Round(track.Synth.Portamento * float64(e.cfg.SampleRate) / float64(noteSamples)))
			patch.StartPortamento(e.prevFrequencies[trackIdx], row.Frequency, ticks)
		}
		e.prevFrequencies[trackIdx] = row.Frequency
		e.arpTickIdx[trackIdx] = -1
		e.effectStates[trackIdx] = channelEffectState{volume: 1.0}
		if row.Effect.Type != EffectNoteDelay {
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
		if patch == nil || trackIdx >= len(e.song.Tracks) {
			continue
		}
		row := e.song.Tracks[trackIdx].Rows[rowIdx]

		if row.Arpeggio.IsActive() {
			e.arpTickIdx[trackIdx]++
			idx := e.arpTickIdx[trackIdx] % len(row.Arpeggio.Offsets)
			mult := math.Pow(2, float64(row.Arpeggio.Offsets[idx])/12)
			patch.SetFrequency(e.prevFrequencies[trackIdx] * mult)
		}

		state := &e.effectStates[trackIdx]
		switch row.Effect.Type {
		case EffectVibrato:
			vibratoSpeed := (row.Effect.Param >> 4) & 0xF
			vibratoDepth := float64(row.Effect.Param & 0xF)
			if vibratoSpeed > 0 {
				state.vibratoPhase += (2 * math.Pi) / float64(vibratoSpeed)
			}
			semitones := (vibratoDepth / 4.0) * math.Sin(state.vibratoPhase)
			mult := math.Pow(2, semitones/12)
			patch.SetFrequency(e.prevFrequencies[trackIdx] * mult)
		case EffectVolumeSlide:
			delta := row.Effect.Param
			if delta > 127 {
				delta = int(int8(uint8(delta)))
			}
			state.volume = math.Max(0, math.Min(1, state.volume+float64(delta)/64.0))
			patch.SetVolume(state.volume)
		case EffectNoteCut:
			if subTick == row.Effect.Param {
				patch.SetVolume(0)
			}
		case EffectNoteDelay:
			if subTick == row.Effect.Param {
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
	ticks := e.song.RowTicks(rowIdx)
	rowSamples := int(e.cfg.SampleRate.N(e.song.RowDuration))
	baseTickSamples := rowSamples / ticks
	remainder := rowSamples % ticks
	if subTick < remainder {
		return baseTickSamples + 1
	}
	return baseTickSamples
}
