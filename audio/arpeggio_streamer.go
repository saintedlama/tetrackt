package audio

import (
	"time"

	"github.com/gopxl/beep/v2"
)

// ArpeggioStreamer creates a streamer that changes frequency per tick for arpeggio effect
type ArpeggioStreamer struct {
	synth        *Synth
	note         Note
	effect       Effect
	duration     time.Duration
	tickDuration time.Duration
	currentTick  int
	tickSamples  int
	currentGen   beep.Streamer
	sampleCount  int
	totalSamples int
}

// NewArpeggioStreamer creates a new arpeggio streamer
func NewArpeggioStreamer(synth *Synth, note Note, effect Effect, rowDuration time.Duration, ticksPerRow int) *ArpeggioStreamer {
	tickDuration := rowDuration / time.Duration(ticksPerRow)
	tickSamples := synth.sampleRate.N(tickDuration)
	totalSamples := synth.sampleRate.N(rowDuration)

	as := &ArpeggioStreamer{
		synth:        synth,
		note:         note,
		effect:       effect,
		duration:     rowDuration,
		tickDuration: tickDuration,
		currentTick:  0,
		tickSamples:  tickSamples,
		totalSamples: totalSamples,
		sampleCount:  0,
	}

	// Generate first tick
	as.generateTick()

	return as
}

func (as *ArpeggioStreamer) generateTick() {
	// Calculate frequency for current tick
	offset := as.effect.GetArpeggioOffset(as.currentTick)
	freq := as.note.FrequencyWithSemitoneOffset(offset)

	// Create a streamer for this tick duration
	as.currentGen = as.synth.StreamerWithFrequency(as.note, freq, as.tickDuration)
}

func (as *ArpeggioStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if as.sampleCount >= as.totalSamples {
		return 0, false
	}

	filled := 0
	for filled < len(samples) {
		if as.sampleCount >= as.totalSamples {
			break
		}

		// Check if we need to move to next tick
		tickOffset := as.sampleCount % as.tickSamples
		if tickOffset == 0 && as.sampleCount > 0 {
			as.currentTick++
			as.generateTick()
		}

		// Stream from current generator
		remaining := len(samples) - filled
		samplesThisTick := as.tickSamples - tickOffset
		if samplesThisTick > remaining {
			samplesThisTick = remaining
		}
		if as.sampleCount+samplesThisTick > as.totalSamples {
			samplesThisTick = as.totalSamples - as.sampleCount
		}

		chunk := samples[filled : filled+samplesThisTick]
		n, ok := as.currentGen.Stream(chunk)
		if !ok || n == 0 {
			// Generator exhausted, might happen at tick boundaries
			// Fill rest with silence
			for i := filled; i < filled+samplesThisTick; i++ {
				samples[i][0] = 0
				samples[i][1] = 0
			}
			n = samplesThisTick
		}

		filled += n
		as.sampleCount += n

		if !ok && as.sampleCount < as.totalSamples {
			// Need to continue but current generator is done
			continue
		}
	}

	return filled, true
}

func (as *ArpeggioStreamer) Err() error {
	if as.currentGen != nil {
		return as.currentGen.Err()
	}
	return nil
}
