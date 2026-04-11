package audio

import (
	"math"
	"math/rand/v2"

	"github.com/gopxl/beep/v2"
)

type Oscillator struct {
	Type        OscillatorType
	Phase       float64   // normalized initial phase [0..1)
	PulseWidth  float64   // duty cycle [0.01..0.99]; only used by Square; 0 defaults to 0.5
	Detune      float64   // fine tuning in cents (±1200 = ±1 octave); 0 = no detune
	Wavetable   []float64 // one cycle of samples; only used by Wavetable type; nil = silent
	NoisePeriod int       // LFSR clock period in samples; 0 = derive from frequency; only used by NoisePeriodic
}

// OscillatorType represents the type of oscillator waveform to generate
type OscillatorType string

const (
	Sine            OscillatorType = "sine"
	Square          OscillatorType = "square"
	Triangle        OscillatorType = "triangle"
	Sawtooth        OscillatorType = "sawtooth"
	SawtoothReverse OscillatorType = "sawtooth_reverse"
	Noise           OscillatorType = "noise"
	NoisePeriodic   OscillatorType = "noise_periodic"
	Silent          OscillatorType = "silent"
	Wavetable       OscillatorType = "wavetable"
)

// NewOscillator creates an oscillatorGenerator for the specified waveform.
// pulseWidth is used only by Square; a zero value defaults to 0.5 (50% duty).
// detuneCents shifts the oscillator frequency by the given number of cents
// (100 cents = 1 semitone, 1200 cents = 1 octave); 0 disables detune.
func NewOscillator(oscillatorType OscillatorType, frequency float64, sampleRate beep.SampleRate, initialPhase float64, pulseWidth float64, detuneCents float64, wavetable []float64, noisePeriod int) *oscillatorGenerator {
	pw := pulseWidth
	if pw == 0 {
		pw = 0.5
	}
	mult := 1.0
	if detuneCents != 0 {
		mult = math.Pow(2, detuneCents/1200.0)
	}
	return &oscillatorGenerator{
		oscillatorType:   oscillatorType,
		frequency:        frequency,
		detuneMultiplier: mult,
		sampleRate:       sampleRate,
		phase:            math.Mod(initialPhase, 1.0),
		pulseWidth:       pw,
		wavetable:        wavetable,
		noisePeriod:      noisePeriod,
		lfsrState:        0x7FFF,
	}
}

// oscillatorGenerator implements beep.Streamer for oscillator waveform generation
type oscillatorGenerator struct {
	oscillatorType   OscillatorType
	frequency        float64
	detuneMultiplier float64 // pre-computed 2^(cents/1200); applied by SetFrequency during arp retune
	sampleRate       beep.SampleRate
	phase            float64
	pulseWidth       float64   // resolved duty cycle [0.01..0.99]
	wavetable        []float64 // single-cycle samples for Wavetable type
	noisePeriod      int       // LFSR clock period in samples; 0 = derive from frequency
	lfsrState        uint16    // 15-bit shift register; non-zero seed guarantees non-trivial sequence
	lfsrCounter      int       // samples remaining until next LFSR clock
}

// Stream fills the samples buffer with oscillator waveform data
func (g *oscillatorGenerator) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		phaseIncrement := g.frequency * g.detuneMultiplier / float64(g.sampleRate)
		var sample float64

		switch g.oscillatorType {
		case Sine:
			sample = math.Sin(2 * math.Pi * g.phase)

		case Square:
			if g.phase < g.pulseWidth {
				sample = 1.0
			} else {
				sample = -1.0
			}

		case Triangle:
			if g.phase < 0.5 {
				sample = 4*g.phase - 1
			} else {
				sample = -4*g.phase + 3
			}

		case Sawtooth:
			sample = 2*g.phase - 1

		case SawtoothReverse:
			sample = 1 - 2*g.phase

		case Noise:
			sample = rand.Float64()*2 - 1

		case NoisePeriodic:
			if g.lfsrCounter <= 0 {
				feedback := (g.lfsrState & 1) ^ ((g.lfsrState >> 1) & 1)
				g.lfsrState = (g.lfsrState >> 1) | (feedback << 14)
				period := g.noisePeriod
				if period <= 0 {
					period = int(float64(g.sampleRate) / (g.frequency * g.detuneMultiplier))
					if period < 1 {
						period = 1
					}
				}
				g.lfsrCounter = period
			}
			g.lfsrCounter--
			if (g.lfsrState & 1) == 0 {
				sample = 1.0
			} else {
				sample = -1.0
			}

		case Silent:
			sample = 0

		case Wavetable:
			if len(g.wavetable) > 0 {
				n := float64(len(g.wavetable))
				pos := g.phase * n
				i0 := int(pos) % len(g.wavetable)
				i1 := (i0 + 1) % len(g.wavetable)
				frac := pos - math.Floor(pos)
				sample = g.wavetable[i0]*(1-frac) + g.wavetable[i1]*frac
			}
		}

		samples[i][0] = sample
		samples[i][1] = sample

		g.phase += phaseIncrement
		if g.phase >= 1.0 {
			g.phase -= 1.0
		}
	}

	return len(samples), true
}

// Err returns any error that occurred during streaming
func (g *oscillatorGenerator) Err() error {
	return nil
}

// SetFrequency retunes the oscillator to the given frequency in Hz.
// The detune multiplier is applied at stream time, so frequency stores the raw value.
// Safe to call between audio blocks via speaker.Lock/Unlock.
func (g *oscillatorGenerator) SetFrequency(frequency float64) {
	g.frequency = frequency
}
