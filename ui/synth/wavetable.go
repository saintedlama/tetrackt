package synth

import (
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/persistence/akwf"
)

// WavetableEntry describes a single-cycle waveform in the bank.
type WavetableEntry struct {
	ID   string    // e.g. "builtin:Organ" or "akwf:AKWF_flute/AKWF_flute_0001"
	Name string    // display name
	Bank string    // e.g. "Built-in", "AKWF_flute", …
	Data []float64 // one cycle of normalised samples
}

var wavetableBank []WavetableEntry

func init() {
	// Built-in waveforms computed from the audio package.
	builtins := []WavetableEntry{
		{ID: "builtin:SoftSaw", Name: "SoftSaw", Bank: "Built-in", Data: audio.WavetableSoftSaw},
		{ID: "builtin:SoftSquare", Name: "SoftSquare", Bank: "Built-in", Data: audio.WavetableSoftSquare},
		{ID: "builtin:Organ", Name: "Organ", Bank: "Built-in", Data: audio.WavetableOrgan},
		{ID: "builtin:Glass", Name: "Glass", Bank: "Built-in", Data: audio.WavetableGlass},
		{ID: "builtin:Bass", Name: "Bass", Bank: "Built-in", Data: audio.WavetableBass},
		{ID: "builtin:Strings", Name: "Strings", Bank: "Built-in", Data: audio.WavetableStrings},
		{ID: "builtin:Flute", Name: "Flute", Bank: "Built-in", Data: audio.WavetableFlute},
		{ID: "builtin:Brass", Name: "Brass", Bank: "Built-in", Data: audio.WavetableBrass},
		{ID: "builtin:Chime", Name: "Chime", Bank: "Built-in", Data: audio.WavetableChime},
		{ID: "builtin:Voice", Name: "Voice", Bank: "Built-in", Data: audio.WavetableVoice},
	}
	wavetableBank = append(wavetableBank, builtins...)

	// AKWF waveforms loaded from embedded JSON files.
	for _, e := range akwf.LoadAll() {
		wavetableBank = append(wavetableBank, WavetableEntry{
			ID:   "akwf:" + e.Bank + "/" + e.Name,
			Name: e.Name,
			Bank: e.Bank,
			Data: e.Data,
		})
	}
}

// WavetableBanksInOrder returns unique bank names in the order they
// were first added to the bank.
func WavetableBanksInOrder() []string {
	seen := map[string]bool{}
	var banks []string
	for _, e := range wavetableBank {
		if !seen[e.Bank] {
			seen[e.Bank] = true
			banks = append(banks, e.Bank)
		}
	}
	return banks
}

// WavetableEntriesForBank returns all entries belonging to the given
// bank, in registration order.
func WavetableEntriesForBank(bank string) []WavetableEntry {
	var out []WavetableEntry
	for _, e := range wavetableBank {
		if e.Bank == bank {
			out = append(out, e)
		}
	}
	return out
}

// LookupWavetableByID returns the sample data for the given wavetable ID.
// Returns (nil, false) if not found.
func LookupWavetableByID(id string) ([]float64, bool) {
	for _, e := range wavetableBank {
		if e.ID == id {
			return e.Data, true
		}
	}
	return nil, false
}

// resolveWavetable returns the sample data for the given ID, or nil if not
// found. Convenience wrapper for use in patch preset literals.
func resolveWavetable(id string) []float64 {
	data, _ := LookupWavetableByID(id)
	return data
}

// resolveWavetableMeta returns the Metadata for the given legacy wavetable ID.
// Used in patch preset literals to populate Meta alongside Wavetable data.
func resolveWavetableMeta(id string) audio.Metadata {
	for _, e := range wavetableBank {
		if e.ID == id {
			return audio.Metadata{Bank: e.Bank, Name: e.Name}
		}
	}
	return audio.Metadata{}
}
