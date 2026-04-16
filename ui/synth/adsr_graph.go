package synth

import (
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/sparkline"

	"github.com/tetrackt/tetrackt/audio"
)

// RenderADSRGraph renders an ADSR envelope shape using an ntcharts braille sparkline.
func RenderADSRGraph(envelope audio.Envelope, width, height int) string {
	if width <= 10 || height <= 0 {
		return ""
	}

	sustainDuration := max(envelope.Attack, envelope.Decay, envelope.Release)
	totalDuration := (envelope.Attack + envelope.Decay + sustainDuration + envelope.Release).Seconds()
	sr := width
	if totalDuration > 0 {
		sr = int(float64(width) / totalDuration)
		if sr <= 0 {
			sr = 1
		}
	} else {
		totalDuration = float64(width)
		sr = 1
	}
	notesSamples := int(totalDuration * float64(sr))

	streamer := audio.NewEnvelope(audio.LinearStreamer(), audio.SampleRate(sr), notesSamples, envelope)
	values := renderSparklineValues(streamer, notesSamples, width)

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7700"))

	sl := sparkline.New(width, height,
		sparkline.WithStyle(style),
		sparkline.WithMaxValue(1.0),
		sparkline.WithNoAutoMaxValue(),
	)
	sl.PushAll(values)
	sl.DrawBraille()
	return sl.View()
}

func renderSparklineValues(streamer audio.Streamer, streamSamples, width int) []float64 {
	if streamer == nil || streamSamples <= 0 {
		return make([]float64, max(width, 0))
	}

	buffer := make([][2]float64, streamSamples)
	streamer.Stream(buffer)

	values := make([]float64, len(buffer))

	for i, sample := range buffer {
		values[i] = sample[0]
	}

	return values
}
