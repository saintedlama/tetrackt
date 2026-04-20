package render

import (
	"fmt"

	"github.com/tetrackt/tetrackt/audio"
)

type RenderConfig struct {
	SampleRate   audio.SampleRate
	GlobalVolume float64
	LoopCount    int
	// EndRow is the exclusive end row index for rendering (0 means all rows).
	EndRow int
}

type RenderEngine struct {
	pattern         *Pattern
	cfg             RenderConfig
	activeVoices    []audio.Streamer
	prevFrequencies []float64
}

func NewRenderEngine(pattern *Pattern, cfg RenderConfig) *RenderEngine {
	return &RenderEngine{pattern: pattern, cfg: cfg}
}

func (e *RenderEngine) Run(sink RenderSink) error {
	if e.pattern == nil {
		return fmt.Errorf("render: pattern is nil")
	}
	if e.pattern.NumRows <= 0 || e.pattern.NumTracks <= 0 {
		return fmt.Errorf("render: pattern is empty")
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

	endRow := e.pattern.NumRows
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

	if err := e.drainVoices(sink); err != nil {
		return err
	}

	return sink.End()
}

// RenderToStream renders the pattern fully offline and returns a Streamer ready
// for playback. If loop is true, the stream will repeat indefinitely.
// Returns nil, nil when the render produces no audio (e.g. all tracks empty).
func RenderToStream(pattern *Pattern, cfg RenderConfig, loop bool) (audio.Streamer, error) {
	collector := &bufferSink{}
	engine := NewRenderEngine(pattern, cfg)
	if err := engine.Run(collector); err != nil {
		return nil, err
	}
	if len(collector.frames) == 0 {
		return nil, nil
	}
	return &sampleStreamer{samples: collector.frames, loop: loop}, nil
}

func (e *RenderEngine) resetState() {
	e.activeVoices = nil
	e.prevFrequencies = make([]float64, e.pattern.NumTracks)
}

func (e *RenderEngine) renderRow(rowIdx int, sink RenderSink) error {
	e.startRow(rowIdx)
	rowSamples := int(e.cfg.SampleRate.N(e.pattern.RowDuration))
	return e.mixActiveVoices(rowSamples, sink)
}

func (e *RenderEngine) startRow(rowIdx int) {
	ticks := e.pattern.RowTicks(rowIdx)
	durationMs := float64(e.pattern.RowDuration.Milliseconds())

	for trackIdx := 0; trackIdx < e.pattern.NumTracks; trackIdx++ {
		track := e.pattern.Tracks[trackIdx]
		row := track.Rows[rowIdx]

		if row.Frequency == 0 {
			e.prevFrequencies[trackIdx] = 0
			continue
		}

		subticks := ticks
		if subticks <= 0 {
			subticks = 1
		}
		fx := rowToEffectDefs(row, subticks)
		ep := audio.NewEffectsPatch(track.Synth, fx, durationMs, subticks)
		streamer := ep.Streamer(e.cfg.SampleRate, row.Frequency, e.prevFrequencies[trackIdx])
		e.prevFrequencies[trackIdx] = row.Frequency

		vol := e.cfg.GlobalVolume
		if row.Volume > 0 {
			vol *= row.Volume
		}

		if vol != 1.0 {
			e.activeVoices = append(e.activeVoices, &scaledStreamer{s: streamer, scale: vol})
		} else {
			e.activeVoices = append(e.activeVoices, streamer)
		}
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
	for _, voice := range e.activeVoices {
		voiceBuf := make([][2]float64, samplesPerTick)
		n, ok := voice.Stream(voiceBuf)
		for i := range n {
			mixBuf[i][0] += voiceBuf[i][0]
			mixBuf[i][1] += voiceBuf[i][1]
		}
		if ok {
			remaining = append(remaining, voice)
		}
	}
	e.activeVoices = remaining

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

// scaledStreamer wraps an audio.Streamer and applies a constant volume scale.
type scaledStreamer struct {
	s     audio.Streamer
	scale float64
}

func (ss *scaledStreamer) Stream(samples [][2]float64) (int, bool) {
	n, ok := ss.s.Stream(samples)
	for i := range n {
		samples[i][0] *= ss.scale
		samples[i][1] *= ss.scale
	}
	return n, ok
}

func (ss *scaledStreamer) Err() error { return ss.s.Err() }
