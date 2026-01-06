package audio

import (
	"fmt"
	"math"

	"github.com/gopxl/beep/v2"
)

type Stages int

const (
	StageOff Stages = iota
	StageAttack
	StageDecay
	StageSustain
	StageRelease
)

type envelopeGenerator struct {
	samples  int // total number of samples for the envelope
	idx      int // current sample index
	Streamer beep.Streamer

	currentStage      Stages
	currentLevel      float64
	currentMultiplier float64
	sustain           float64

	attackSamples  int
	decaySamples   int
	sustainSamples int
	releaseSamples int
}

// NewWaveform creates a beep.Streamer that generates the specified waveform
func (s *Synth) NewADSREnvelope(streamer beep.Streamer, samples int, attack, decay, sustain, release float64) beep.Streamer {
	attackSamples := int(attack * float64(samples))
	decaySamples := int(decay * float64(samples))
	releaseSamples := int(release * float64(samples))
	sustainSamples := samples - (attackSamples + decaySamples + releaseSamples)

	fmt.Printf("sample counts:\n%d attack\n%d decay\n%d sustain\n%d release\n", attackSamples, decaySamples, sustainSamples, releaseSamples)
	return &envelopeGenerator{
		samples:  samples,
		idx:      -1,
		Streamer: streamer,

		currentStage:      StageOff,
		currentLevel:      0, // start with minimum level greater than 0 for multiplicative increase
		currentMultiplier: 1.0,
		sustain:           sustain,
		attackSamples:     attackSamples,
		decaySamples:      decaySamples,
		sustainSamples:    sustainSamples,
		releaseSamples:    releaseSamples,
	}
}

func (e *envelopeGenerator) nextSample() {
	e.idx++

	if e.idx < e.attackSamples {
		if e.currentStage != StageAttack {
			e.currentStage = StageAttack
			e.currentLevel = 0.0001 // TODO: Extract mininimum level constant
			e.currentMultiplier = calculateMultiplier(e.currentLevel, 1, e.attackSamples)

			fmt.Printf("Entering attack stage at sample %d/%d with level %f and multiplier %f\n", e.idx, e.samples, e.currentLevel, e.currentMultiplier)
		}
	} else if e.idx < e.attackSamples+e.decaySamples {
		if e.currentStage != StageDecay {
			e.currentStage = StageDecay
			e.currentLevel = 1.0
			e.currentMultiplier = calculateMultiplier(e.currentLevel, e.sustain, e.decaySamples)

			fmt.Printf("Entering decay stage at sample %d/%d with level %f and multiplier %f\n", e.idx, e.samples, e.currentLevel, e.currentMultiplier)
		}
	} else if e.idx < e.attackSamples+e.decaySamples+e.sustainSamples {
		if e.currentStage != StageSustain {
			e.currentStage = StageSustain
			e.currentLevel = e.sustain
			e.currentMultiplier = 1.0

			fmt.Printf("Entering sustain stage at sample %d/%d with level %f and multiplier %f\n", e.idx, e.samples, e.currentLevel, e.currentMultiplier)
		}
	} else if e.idx < e.attackSamples+e.decaySamples+e.sustainSamples+e.releaseSamples {
		if e.currentStage != StageRelease {
			e.currentStage = StageRelease
			e.currentLevel = e.sustain
			e.currentMultiplier = calculateMultiplier(e.currentLevel, 0.0001, e.releaseSamples)

			fmt.Printf("Entering release stage at sample %d/%d with level %f and multiplier %f\n", e.idx, e.samples, e.currentLevel, e.currentMultiplier)
		}
	} else {
		if e.currentStage != StageOff {
			e.currentStage = StageOff
			e.currentLevel = 0.0
			e.currentMultiplier = 1.0

			fmt.Printf("Entering off stage at sample %d/%d with level %f and multiplier %f\n", e.idx, e.samples, e.currentLevel, e.currentMultiplier)
		}
	}
}

func (e *envelopeGenerator) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = e.Streamer.Stream(samples)

	fmt.Printf("Received %d samples in batch\n", len(samples))

	// Process samples from streamer in context of a note
	for i := 0; i < n; i++ {
		e.nextSample()

		samples[i][0] *= e.currentLevel
		samples[i][1] *= e.currentLevel

		e.currentLevel *= e.currentMultiplier
	}

	return n, ok
}

func calculateMultiplier(startLevel float64, endLevel float64, lengthInSamples int) float64 {
	return 1.0 + (math.Log(endLevel)-math.Log(startLevel))/float64(lengthInSamples)
}

func (e *envelopeGenerator) Err() error {
	return nil
}
