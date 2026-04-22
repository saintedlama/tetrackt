package audio

import (
	"math"
	"time"
)

// minEnvelopeLevel is the floor for exponential envelope calculations.
// math.Log(0) = -Inf, which corrupts the multiplier; using this small positive
// value instead produces ~-80 dB, which is inaudible.
const minEnvelopeLevel = 0.0001

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
	Streamer Streamer

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
	Attack  time.Duration
	Decay   time.Duration
	Sustain float64
	Release time.Duration
}

// Creates a Streamer that applies ADSR envelope to the provided streamer
func NewEnvelope(streamer Streamer, sampleRate SampleRate, noteSamples int, envelope Envelope) Streamer {
	sr := float64(sampleRate)
	attackSamples := int(envelope.Attack.Seconds() * sr)
	decaySamples := int(envelope.Decay.Seconds() * sr)
	releaseSamples := int(envelope.Release.Seconds() * sr)
	sustainSamples := max(0, noteSamples-(attackSamples+decaySamples+releaseSamples))

	return &envelopeGenerator{
		samples:  noteSamples,
		idx:      -1,
		Streamer: streamer,

		currentStage:      StageOff,
		currentLevel:      0, // start with minimum level greater than 0 for multiplicative increase
		currentMultiplier: 1.0,
		sustain:           math.Max(minEnvelopeLevel, envelope.Sustain),
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
			e.currentLevel = minEnvelopeLevel
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
			e.currentMultiplier = calculateMultiplier(e.currentLevel, minEnvelopeLevel, e.releaseSamples)
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

// reset restarts the envelope from the beginning (stage Off, level 0).
func (e *envelopeGenerator) reset() {
	e.idx = -1
	e.currentStage = StageOff
	e.currentLevel = 0
	e.currentMultiplier = 1.0
}
