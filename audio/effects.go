package audio

import (
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

type Envelope struct {
	Attack, Decay, Sustain, Release float64
}

// Creates a beep.Streamer that applies ADSR envelope to the provided streamer
func NewEnvelope(streamer beep.Streamer, samples int, envelope Envelope) beep.Streamer {
	attackSamples := int(envelope.Attack * float64(samples))
	decaySamples := int(envelope.Decay * float64(samples))
	releaseSamples := int(envelope.Release * float64(samples))
	sustainSamples := samples - (attackSamples + decaySamples + releaseSamples)

	return &envelopeGenerator{
		samples:  samples,
		idx:      -1,
		Streamer: streamer,

		currentStage:      StageOff,
		currentLevel:      0, // start with minimum level greater than 0 for multiplicative increase
		currentMultiplier: 1.0,
		sustain:           envelope.Sustain,
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
		}
	} else if e.idx < e.attackSamples+e.decaySamples {
		if e.currentStage != StageDecay {
			e.currentStage = StageDecay
			e.currentLevel = 1.0
			e.currentMultiplier = calculateMultiplier(e.currentLevel, e.sustain, e.decaySamples)
		}
	} else if e.idx < e.attackSamples+e.decaySamples+e.sustainSamples {
		if e.currentStage != StageSustain {
			e.currentStage = StageSustain
			e.currentLevel = e.sustain
			e.currentMultiplier = 1.0
		}
	} else if e.idx < e.attackSamples+e.decaySamples+e.sustainSamples+e.releaseSamples {
		if e.currentStage != StageRelease {
			e.currentStage = StageRelease
			e.currentLevel = e.sustain
			e.currentMultiplier = calculateMultiplier(e.currentLevel, 0.0001, e.releaseSamples)
		}
	} else {
		if e.currentStage != StageOff {
			e.currentStage = StageOff
			e.currentLevel = 0.0
			e.currentMultiplier = 1.0
		}
	}
}

func (e *envelopeGenerator) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = e.Streamer.Stream(samples)

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
