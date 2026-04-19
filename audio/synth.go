package audio

import (
	"math"

	"github.com/gopxl/beep/v2"
)

// SampleRate is the number of audio samples per second.
// It is an alias for beep.SampleRate, exposed so callers outside of audio
// don't need to import beep directly.
type SampleRate = beep.SampleRate

// Streamer is the interface for pulling audio samples.
// It is an alias for beep.Streamer, exposed so callers outside of audio
// don't need to import beep directly.
type Streamer = beep.Streamer

type StreamerFunc = beep.StreamerFunc

// Synth represents the audio synthesis engine (instrument patch definition).
type Synth struct {
	Oscillator1    Oscillator
	Envelope1      Envelope
	Oscillator2    Oscillator
	Envelope2      Envelope
	Oscillator3    Oscillator
	Envelope3      Envelope
	Mixer          Mixer
	Filter         Filter
	FilterEnvelope FilterEnvelope // ADSR-driven additive cutoff offset; Depth 0 = disabled
	LFO1           LFO
	LFO2           LFO
	LFO3           LFO
	Portamento     float64  // glide duration in seconds; 0 = snap
	Meta           Metadata // display-level patch metadata (Bank, Name, Tags)
}

// NewSynth creates a new synthesis engine
func NewSynth(oscillator1 Oscillator, envelope1 Envelope, oscillator2 Oscillator, envelope2 Envelope, oscillator3 Oscillator, envelope3 Envelope, mixer Mixer, filter Filter, lfo1, lfo2, lfo3 LFO) *Synth {
	return &Synth{
		Oscillator1: oscillator1,
		Envelope1:   envelope1,
		Oscillator2: oscillator2,
		Envelope2:   envelope2,
		Oscillator3: oscillator3,
		Envelope3:   envelope3,
		Mixer:       mixer,
		Filter:      filter,
		LFO1:        lfo1,
		LFO2:        lfo2,
		LFO3:        lfo3,
	}
}

// Patch is a live synthesis instance created from a Synth by calling NewPatch.
// It holds the complete signal pipeline and exposes controls for the caller.
//
// Typical usage:
//
//	patch := synth.NewPatch(sampleRate, 440.0, noteSamples)
//	patch.Stream(buf)          // pull samples; returns false after noteSamples
//	patch.SetFrequency(880.0)  // retune without restarting the envelope
//	patch.Reset()              // restart ADSR and LFOs (envelope gate)
//	patch.Stream(buf)
//
// Patch implements beep.Streamer. It automatically stops after noteSamples
// samples, so no external beep.Take wrapping is needed.
type Patch struct {
	osc1         *oscillatorGenerator
	osc2         *oscillatorGenerator
	osc3         *oscillatorGenerator
	modOsc1      *modulatedOscillatorStreamer // nil when no oscillator LFOs are active
	modOsc2      *modulatedOscillatorStreamer
	modOsc3      *modulatedOscillatorStreamer
	env1         *envelopeGenerator
	env2         *envelopeGenerator
	env3         *envelopeGenerator
	filterEnvGen *filterEnvelopeGenerator // nil when FilterEnvelope.Depth == 0 or filter off
	lfos         []*lfoGenerator
	pipeline     beep.Streamer
	noteSamples  int
	remaining    int
	volume       float64 // output scalar; 1.0 = unity, 0 = silent
	portamento   portamento
}

type portamento struct {
	fromFrequency float64
	toFrequency   float64
	step          int
	steps         int
}

// NewPatch instantiates a synthesis pipeline at the given frequency.
// noteSamples defines the total ADSR timeline length; the envelope silences
// naturally after that many samples, but the caller may keep streaming beyond it.
func (s *Synth) NewPatch(sampleRate beep.SampleRate, frequency float64, noteSamples int) *Patch {
	sr := float64(sampleRate)

	osc1 := NewOscillator(s.Oscillator1.Type, frequency, sampleRate, s.Oscillator1.Phase, s.Oscillator1.PulseWidth, s.Oscillator1.Detune, s.Oscillator1.Wavetable, s.Oscillator1.NoisePeriod)
	osc2 := NewOscillator(s.Oscillator2.Type, frequency, sampleRate, s.Oscillator2.Phase, s.Oscillator2.PulseWidth, s.Oscillator2.Detune, s.Oscillator2.Wavetable, s.Oscillator2.NoisePeriod)
	osc3 := NewOscillator(s.Oscillator3.Type, frequency, sampleRate, s.Oscillator3.Phase, s.Oscillator3.PulseWidth, s.Oscillator3.Detune, s.Oscillator3.Wavetable, s.Oscillator3.NoisePeriod)

	var lfos []*lfoGenerator
	makeLFO := func(dest ModDest) *lfoGenerator {
		if s.LFO1.Depth > 0 && s.LFO1.Dest == dest {
			g := newLFOGenerator(s.LFO1, sr)
			lfos = append(lfos, g)
			return g
		}
		if s.LFO2.Depth > 0 && s.LFO2.Dest == dest {
			g := newLFOGenerator(s.LFO2, sr)
			lfos = append(lfos, g)
			return g
		}
		if s.LFO3.Depth > 0 && s.LFO3.Dest == dest {
			g := newLFOGenerator(s.LFO3, sr)
			lfos = append(lfos, g)
			return g
		}
		return nil
	}

	raw1 := newModulatedOscillatorStreamer(osc1, osc1.frequency, osc1.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth), makeLFO(ModDetune))
	raw2 := newModulatedOscillatorStreamer(osc2, osc2.frequency, osc2.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth), makeLFO(ModDetune))
	raw3 := newModulatedOscillatorStreamer(osc3, osc3.frequency, osc3.pulseWidth, makeLFO(ModPitch), makeLFO(ModPulseWidth), makeLFO(ModDetune))

	env1 := NewEnvelope(raw1, sampleRate, noteSamples, s.Envelope1).(*envelopeGenerator)
	env2 := NewEnvelope(raw2, sampleRate, noteSamples, s.Envelope2).(*envelopeGenerator)
	env3 := NewEnvelope(raw3, sampleRate, noteSamples, s.Envelope3).(*envelopeGenerator)

	mod1 := newModulatedVolumeStreamer(env1, makeLFO(ModVolume))
	mod2 := newModulatedVolumeStreamer(env2, makeLFO(ModVolume))
	mod3 := newModulatedVolumeStreamer(env3, makeLFO(ModVolume))

	mixed := s.Mixer.Mix(mod1, mod2, mod3)
	var filterEnvGen *filterEnvelopeGenerator
	var pipeline beep.Streamer
	if s.FilterEnvelope.Depth > 0 && s.Filter.Type != FilterOff {
		pipeline = newFilterEnvelopeGenerator(mixed, sampleRate, noteSamples, s.Filter, s.FilterEnvelope, makeLFO(ModCutoff))
	} else {
		pipeline = NewModulatedFilterStreamer(mixed, sampleRate, s.Filter, makeLFO(ModCutoff))
	}

	var modOsc1, modOsc2, modOsc3 *modulatedOscillatorStreamer
	if mos, ok := raw1.(*modulatedOscillatorStreamer); ok {
		modOsc1 = mos
	}
	if mos, ok := raw2.(*modulatedOscillatorStreamer); ok {
		modOsc2 = mos
	}
	if mos, ok := raw3.(*modulatedOscillatorStreamer); ok {
		modOsc3 = mos
	}

	return &Patch{
		osc1:         osc1,
		osc2:         osc2,
		osc3:         osc3,
		modOsc1:      modOsc1,
		modOsc2:      modOsc2,
		modOsc3:      modOsc3,
		env1:         env1,
		env2:         env2,
		env3:         env3,
		filterEnvGen: filterEnvGen,
		lfos:         lfos,
		pipeline:     pipeline,
		noteSamples:  noteSamples,
		remaining:    noteSamples,
		volume:       1.0,
	}
}

// SetVolume sets the output amplitude scalar applied to all streamed samples.
// 1.0 = unity gain, 0 = silent. Use for per-tick effects such as VolumeSlide and NoteCut.
func (p *Patch) SetVolume(v float64) {
	p.volume = v
}

// SetFrequency retunes both oscillators to the given frequency in Hz. The detune offset configured in
// the Synth is preserved. When a pitch LFO is active, its base frequency is also
// updated so modulation remains relative to the new pitch.
func (p *Patch) SetFrequency(frequency float64) {
	p.osc1.SetFrequency(frequency)
	p.osc2.SetFrequency(frequency)
	p.osc3.SetFrequency(frequency)
	if p.modOsc1 != nil {
		p.modOsc1.baseFreq = p.osc1.frequency
	}
	if p.modOsc2 != nil {
		p.modOsc2.baseFreq = p.osc2.frequency
	}
	if p.modOsc3 != nil {
		p.modOsc3.baseFreq = p.osc3.frequency
	}
}

// StartPortamento begins a stepped frequency glide from fromFrequency to toFrequency
// over ticks player sub-ticks. Each call to TickPortamento advances
// one step. Calling with ticks <= 0 or fromFrequency <= 0 is a no-op.
func (p *Patch) StartPortamento(fromFrequency, toFrequency float64, ticks int) {
	if ticks <= 0 || fromFrequency <= 0 {
		return
	}
	p.portamento.fromFrequency = fromFrequency
	p.portamento.toFrequency = toFrequency
	p.portamento.step = 0
	p.portamento.steps = ticks
	p.SetFrequency(fromFrequency)
}

// TickPortamento advances the portamento glide by one step and updates the
// oscillator frequency. Call once per player sub-tick. Does nothing when no
// glide is in progress or when the glide has completed.
func (p *Patch) TickPortamento() {
	if p.portamento.steps == 0 || p.portamento.step >= p.portamento.steps {
		return
	}
	p.portamento.step++
	t := float64(p.portamento.step) / float64(p.portamento.steps)
	// Exponential interpolation = perceptually linear (equal semitones per tick)
	freq := p.portamento.fromFrequency * math.Pow(p.portamento.toFrequency/p.portamento.fromFrequency, t)
	p.SetFrequency(freq)
}

// Reset restarts the ADSR envelopes and all LFOs from the beginning.
// Oscillator phases are preserved to avoid audible clicks.
func (p *Patch) Reset() {
	p.env1.reset()
	p.env2.reset()
	p.env3.reset()
	if p.filterEnvGen != nil {
		p.filterEnvGen.reset()
	}
	for _, lfo := range p.lfos {
		lfo.reset()
	}
	p.remaining = p.noteSamples
	p.portamento = portamento{}
}

// Stream implements beep.Streamer — pulls the next samples from the pipeline.
// It returns false (drained) once noteSamples have been emitted.
func (p *Patch) Stream(samples [][2]float64) (int, bool) {
	if p.remaining <= 0 {
		return 0, false
	}
	if len(samples) > p.remaining {
		samples = samples[:p.remaining]
	}
	n, ok := p.pipeline.Stream(samples)
	if p.volume != 1.0 {
		for i := range n {
			samples[i][0] *= p.volume
			samples[i][1] *= p.volume
		}
	}
	p.remaining -= n
	if p.remaining <= 0 {
		return n, false
	}
	return n, ok
}

// Err implements beep.Streamer.
func (p *Patch) Err() error {
	return p.pipeline.Err()
}
