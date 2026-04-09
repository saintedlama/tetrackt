package audio

import "math"

const wavetableSize = 256

// WavetableSoftSaw is a band-limited sawtooth wave built by additive synthesis
// (harmonics 1–16, amplitude 1/n). Alias-free at low and mid octaves.
var WavetableSoftSaw = computeSoftSaw()

// WavetableSoftSquare is a band-limited square wave built by additive synthesis
// (odd harmonics 1, 3, 5 … 31, amplitude 1/n).
var WavetableSoftSquare = computeSoftSquare()

// WavetableOrgan approximates a Hammond-style tone wheel organ
// (harmonics 1–8 with falling weights).
var WavetableOrgan = computeOrgan()

// WavetableGlass is a bright, glassy tone built from harmonics 1, 3, 5, 7
// with halving amplitudes.
var WavetableGlass = computeGlass()

// WavetableBass is a warm, round bass tone: strong fundamental with a few soft harmonics.
var WavetableBass = computeBass()

// WavetableStrings is a string ensemble tone: sawtooth-like spectrum with boosted
// even harmonics for a bowed character.
var WavetableStrings = computeStrings()

// WavetableFlute is a near-pure tone with faint upper harmonics.
var WavetableFlute = computeFlute()

// WavetableBrass is a bright brass tone with a rising-then-falling harmonic envelope.
var WavetableBrass = computeBrass()

// WavetableChime is a bell-like tone using inharmonic partials derived from bell acoustics.
var WavetableChime = computeChime()

// WavetableVoice approximates a vowel 'ah' formant with three weighted spectral peaks.
var WavetableVoice = computeVoice()

func computeSoftSaw() []float64 {
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		for h := 1; h <= 16; h++ {
			t[i] += math.Sin(2*math.Pi*float64(h)*phase) / float64(h)
		}
	}
	return normalizeWavetable(t)
}

func computeSoftSquare() []float64 {
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		for h := 1; h <= 31; h += 2 {
			t[i] += math.Sin(2*math.Pi*float64(h)*phase) / float64(h)
		}
	}
	return normalizeWavetable(t)
}

func computeOrgan() []float64 {
	weights := []float64{1.0, 0.8, 0.5, 0.3, 0.2, 0.15, 0.1, 0.05}
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		for h, w := range weights {
			t[i] += w * math.Sin(2*math.Pi*float64(h+1)*phase)
		}
	}
	return normalizeWavetable(t)
}

func computeGlass() []float64 {
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		t[i] = math.Sin(2*math.Pi*phase) +
			0.5*math.Sin(2*math.Pi*3*phase) +
			0.25*math.Sin(2*math.Pi*5*phase) +
			0.125*math.Sin(2*math.Pi*7*phase)
	}
	return normalizeWavetable(t)
}

func computeBass() []float64 {
	weights := []float64{1.0, 0.5, 0.2, 0.08, 0.03}
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		for h, w := range weights {
			t[i] += w * math.Sin(2*math.Pi*float64(h+1)*phase)
		}
	}
	return normalizeWavetable(t)
}

func computeStrings() []float64 {
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		for h := 1; h <= 20; h++ {
			amp := 1.0 / float64(h)
			if h%2 == 0 {
				amp *= 1.3 // boost even harmonics for bowed string character
			}
			t[i] += amp * math.Sin(2*math.Pi*float64(h)*phase)
		}
	}
	return normalizeWavetable(t)
}

func computeFlute() []float64 {
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		t[i] = math.Sin(2*math.Pi*phase) +
			0.12*math.Sin(2*math.Pi*2*phase) +
			0.03*math.Sin(2*math.Pi*3*phase)
	}
	return normalizeWavetable(t)
}

func computeBrass() []float64 {
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		for h := 1; h <= 12; h++ {
			var w float64
			if h <= 5 {
				w = float64(h) / 5.0
			} else {
				w = float64(13-h) / 8.0
			}
			t[i] += w * math.Sin(2*math.Pi*float64(h)*phase)
		}
	}
	return normalizeWavetable(t)
}

func computeChime() []float64 {
	ratios := []float64{1.0, 2.756, 5.404, 8.933, 13.35}
	weights := []float64{1.0, 0.5, 0.25, 0.12, 0.06}
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		for j, r := range ratios {
			t[i] += weights[j] * math.Sin(2*math.Pi*r*phase)
		}
	}
	return normalizeWavetable(t)
}

func computeVoice() []float64 {
	t := make([]float64, wavetableSize)
	for i := range t {
		phase := float64(i) / wavetableSize
		for h := 1; h <= 20; h++ {
			fh := float64(h)
			// Three formant peaks at harmonics ~3, ~7, ~15
			w := 0.8*math.Exp(-0.8*(fh-3)*(fh-3)) +
				0.5*math.Exp(-0.3*(fh-7)*(fh-7)) +
				0.25*math.Exp(-0.15*(fh-15)*(fh-15))
			t[i] += w * math.Sin(2*math.Pi*fh*phase)
		}
	}
	return normalizeWavetable(t)
}

// normalizeWavetable scales t so the peak amplitude is 1.0.
func normalizeWavetable(t []float64) []float64 {
	peak := 0.0
	for _, v := range t {
		if a := math.Abs(v); a > peak {
			peak = a
		}
	}
	if peak > 0 {
		for i := range t {
			t[i] /= peak
		}
	}
	return t
}
