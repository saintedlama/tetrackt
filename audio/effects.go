package audio

import (
	"math"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
)

// minEnvelopeLevel is the floor for exponential envelope calculations.
// math.Log(0) = -Inf, which corrupts the multiplier; using this small positive
// value instead produces ~-80 dB, which is inaudible.
const minEnvelopeLevel = 0.0001

type gateState int

const (
	gateIdle gateState = iota
	gateAttack
	gateDecay
	gateSustain
	gateRelease
	gateDone
)

type gatedEnvelopeGenerator struct {
	Streamer beep.Streamer

	attackSamples  int
	decaySamples   int
	releaseSamples int
	sustainLevel   float64

	mu           sync.Mutex
	state        gateState
	pos          int
	currentLevel float64
	multiplier   float64
}

func newGatedEnvelopeGenerator(streamer beep.Streamer, sampleRate beep.SampleRate, env Envelope) *gatedEnvelopeGenerator {
	sr := float64(sampleRate)
	return &gatedEnvelopeGenerator{
		Streamer:       streamer,
		attackSamples:  int(env.Attack.Seconds() * sr),
		decaySamples:   int(env.Decay.Seconds() * sr),
		releaseSamples: int(env.Release.Seconds() * sr),
		sustainLevel:   math.Max(minEnvelopeLevel, env.Sustain),
		state:          gateIdle,
		currentLevel:   0,
		multiplier:     1.0,
	}
}

func (g *gatedEnvelopeGenerator) NoteOn() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pos = 0
	if g.attackSamples > 0 {
		g.state = gateAttack
		g.currentLevel = minEnvelopeLevel
		g.multiplier = calculateMultiplier(g.currentLevel, 1.0, g.attackSamples)
	} else if g.decaySamples > 0 {
		g.state = gateDecay
		g.currentLevel = 1.0
		g.multiplier = calculateMultiplier(1.0, g.sustainLevel, g.decaySamples)
	} else {
		g.state = gateSustain
		g.currentLevel = g.sustainLevel
		g.multiplier = 1.0
	}
}

func (g *gatedEnvelopeGenerator) NoteOff() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state == gateIdle {
		g.state = gateDone
		return
	}
	if g.releaseSamples > 0 {
		g.state = gateRelease
		g.pos = 0
		g.multiplier = calculateMultiplier(math.Max(g.currentLevel, minEnvelopeLevel), minEnvelopeLevel, g.releaseSamples)
	} else {
		g.state = gateDone
		g.currentLevel = 0
		g.multiplier = 1.0
	}
}

func (g *gatedEnvelopeGenerator) Stream(samples [][2]float64) (int, bool) {
	n, _ := g.Streamer.Stream(samples)

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == gateIdle {
		for i := range n {
			samples[i][0] = 0
			samples[i][1] = 0
		}
		return n, true
	}
	if g.state == gateDone {
		return 0, false
	}

	for i := range n {
		samples[i][0] *= g.currentLevel
		samples[i][1] *= g.currentLevel
		g.currentLevel *= g.multiplier
		g.pos++

		switch g.state {
		case gateAttack:
			if g.pos >= g.attackSamples {
				g.pos = 0
				g.currentLevel = 1.0
				if g.decaySamples > 0 {
					g.state = gateDecay
					g.multiplier = calculateMultiplier(1.0, g.sustainLevel, g.decaySamples)
				} else {
					g.state = gateSustain
					g.currentLevel = g.sustainLevel
					g.multiplier = 1.0
				}
			}
		case gateDecay:
			if g.pos >= g.decaySamples {
				g.state = gateSustain
				g.pos = 0
				g.currentLevel = g.sustainLevel
				g.multiplier = 1.0
			}
		case gateRelease:
			if g.pos >= g.releaseSamples {
				g.state = gateDone
				g.currentLevel = 0
				g.multiplier = 1.0
				return n, false
			}
		}
	}

	return n, true
}

func (g *gatedEnvelopeGenerator) Err() error {
	return nil
}

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
	Attack  time.Duration
	Decay   time.Duration
	Sustain float64
	Release time.Duration
}

// ArpeggioEffect cycles through frequency offsets, one per tick.
// Offsets are semitone values relative to the base frequency and are converted
// to multipliers via 2^(offset/12), making the effect purely frequency-based.
// An empty Offsets slice means the effect is inactive.
type ArpeggioEffect struct {
	Offsets []int // one semitone offset per tick, e.g. [0, 4, 7, 4] for a 4-tick major arp
}

// IsActive reports whether the arpeggio effect will produce any retuning.
func (a ArpeggioEffect) IsActive() bool {
	return len(a.Offsets) > 0
}

// Creates a beep.Streamer that applies ADSR envelope to the provided streamer
func NewEnvelope(streamer beep.Streamer, sampleRate beep.SampleRate, noteSamples int, envelope Envelope) beep.Streamer {
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
