package audio

// Metadata holds display-level information for audio entities such as synth
// patches and wavetables. It is intentionally separate from synthesis
// parameters so that the audio engine never needs to know about UI concerns.
//
// For Synth, Tags carries patch-bank tags (e.g. "Custom", "NES").
// For Oscillator, Tags is always nil — only Bank and Name are meaningful.
type Metadata struct {
	Bank string
	Name string
	Tags []string
}
