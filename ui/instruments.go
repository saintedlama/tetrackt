package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
)

// Instrument represents a complete instrument configuration
type Instrument struct {
	Name        string
	Category    string
	Oscillator1 audio.Oscillator
	Envelope1   audio.Envelope
	Oscillator2 audio.Oscillator
	Envelope2   audio.Envelope
	Mixer       audio.Mixer
}

// InstrumentView represents the UI component for managing instrument presets
type InstrumentView struct {
	Presets         []Instrument
	SelectedPreset  int
	CurrentTrackNum int
	MaxHeight       int
	Categories      []string
	CategoryIndex   int
	CategoryCounts  map[string]int
	SelectedStyle   lipgloss.Style
}

type InstrumentApplied struct {
	Instrument Instrument
}

// NewInstrumentView initializes a new instrument view with chiptune presets
func NewInstrumentView(selectedStyle lipgloss.Style) *InstrumentView {
	presets := presets()
	slices.SortFunc(presets, func(i, j Instrument) int {
		return strings.Compare(i.Name, j.Name)
	})
	categories, categoryCounts := buildCategories(presets)

	return &InstrumentView{
		Presets:         presets,
		SelectedPreset:  0,
		CurrentTrackNum: 0,
		Categories:      categories,
		CategoryIndex:   0,
		CategoryCounts:  categoryCounts,
		SelectedStyle:   selectedStyle,
	}
}

// GetPreset returns the instrument at the specified index
func (ip *InstrumentView) GetPreset(index int) *Instrument {
	if index >= 0 && index < len(ip.Presets) {
		return &ip.Presets[index]
	}
	return nil
}

func (ip *InstrumentView) Init() tea.Cmd {
	return nil
}

// Update handles input for selecting and applying presets.
func (ip *InstrumentView) Update(msg tea.Msg) (Component, tea.Cmd) {
	if len(ip.Presets) == 0 {
		return ip, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			ip.moveSelection(-1)
		case "down":
			ip.moveSelection(1)
		case "left":
			if len(ip.Categories) > 0 {
				ip.CategoryIndex = (ip.CategoryIndex - 1 + len(ip.Categories)) % len(ip.Categories)
				ip.snapSelectionToFilter()
			}
		case "right":
			if len(ip.Categories) > 0 {
				ip.CategoryIndex = (ip.CategoryIndex + 1) % len(ip.Categories)
				ip.snapSelectionToFilter()
			}
		case "enter":
			preset := ip.Presets[ip.SelectedPreset]
			return ip, func() tea.Msg {
				return InstrumentApplied{Instrument: preset}
			}
		}
	}

	return ip, nil
}

func buildCategories(presets []Instrument) ([]string, map[string]int) {
	categories := []string{"All"}
	seen := map[string]bool{"All": true}
	counts := map[string]int{"All": len(presets)}
	for _, preset := range presets {
		if preset.Category == "" {
			continue
		}
		counts[preset.Category]++
		if !seen[preset.Category] {
			categories = append(categories, preset.Category)
			seen[preset.Category] = true
		}
	}
	return categories, counts
}

func (ip *InstrumentView) currentCategory() string {
	if len(ip.Categories) == 0 {
		return "All"
	}
	if ip.CategoryIndex < 0 || ip.CategoryIndex >= len(ip.Categories) {
		ip.CategoryIndex = 0
	}
	return ip.Categories[ip.CategoryIndex]
}

func (ip *InstrumentView) categoryCount(name string) int {
	if ip.CategoryCounts == nil {
		return 0
	}
	return ip.CategoryCounts[name]
}

func (ip *InstrumentView) categoryLabel(name string) string {
	return fmt.Sprintf("%s (%d)", name, ip.categoryCount(name))
}

func (ip *InstrumentView) filteredIndexes() []int {
	if len(ip.Presets) == 0 {
		return nil
	}

	category := ip.currentCategory()
	if category == "All" {
		indexes := make([]int, len(ip.Presets))
		for i := range ip.Presets {
			indexes[i] = i
		}
		return indexes
	}

	indexes := make([]int, 0, len(ip.Presets))
	for i, preset := range ip.Presets {
		if preset.Category == category {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

func (ip *InstrumentView) selectionIndex(indexes []int) int {
	for i, idx := range indexes {
		if idx == ip.SelectedPreset {
			return i
		}
	}
	return 0
}

func (ip *InstrumentView) moveSelection(step int) {
	indexes := ip.filteredIndexes()
	if len(indexes) == 0 {
		return
	}

	current := ip.selectionIndex(indexes)
	next := (current + step + len(indexes)) % len(indexes)
	ip.SelectedPreset = indexes[next]
}

func (ip *InstrumentView) snapSelectionToFilter() {
	indexes := ip.filteredIndexes()
	if len(indexes) == 0 {
		return
	}

	if slices.Contains(indexes, ip.SelectedPreset) {
		return
	}

	ip.SelectedPreset = indexes[0]
}

// View renders the instrument view list.
func (ip *InstrumentView) View() string {
	var view strings.Builder
	fmt.Fprintf(&view, "Instruments [%s]\n", ip.categoryLabel(ip.currentCategory()))

	indexes := ip.filteredIndexes()
	start := 0
	end := len(indexes)
	selectedIndex := ip.selectionIndex(indexes)
	if ip.MaxHeight > 0 {
		visibleItems := max(ip.MaxHeight-1, 1)
		if selectedIndex >= visibleItems {
			start = selectedIndex - visibleItems + 1
		}
		maxStart := max(len(indexes)-visibleItems, 0)
		start = min(start, maxStart)
		end = min(start+visibleItems, len(indexes))
	}

	for idx := start; idx < end; idx++ {
		presetIndex := indexes[idx]
		preset := ip.Presets[presetIndex]
		name := preset.Name
		if presetIndex == ip.SelectedPreset {
			name = ip.SelectedStyle.Render(name)
		}
		fmt.Fprintf(&view, "%s\n", name)
	}

	return view.String()
}

func presets() []Instrument {
	return []Instrument{
		{
			Name:        "8-Bit Square Lead",
			Category:    "Lead",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.1, Sustain: 0.7, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Silent, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 0, Release: 0},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Classic Triangle",
			Category:    "Lead",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.02, Decay: 0.15, Sustain: 0.6, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Silent, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 0, Release: 0},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Retro Sawtooth",
			Category:    "Lead",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.005, Decay: 0.2, Sustain: 0.5, Release: 0.25},
			Oscillator2: audio.Oscillator{Type: audio.Silent, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 0, Release: 0},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Noise Percussion",
			Category:    "Percussion",
			Oscillator1: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0, Release: 0.05},
			Oscillator2: audio.Oscillator{Type: audio.Silent, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0, Decay: 0, Sustain: 0, Release: 0},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chiptune Bass",
			Category:    "Bass",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.1, Sustain: 0.9, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.1, Sustain: 0.6, Release: 0.1},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Synth Lead",
			Category:    "Lead",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.7, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.25},
			Envelope2:   audio.Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.5, Release: 0.3},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Vibrato Pad",
			Category:    "Pad",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.3, Decay: 0.2, Sustain: 0.8, Release: 0.5},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.1},
			Envelope2:   audio.Envelope{Attack: 0.3, Decay: 0.2, Sustain: 0.8, Release: 0.5},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Arpeggiated Chords",
			Category:    "Chords",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.4, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.33},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.4, Release: 0.1},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Bit Crusher",
			Category:    "FX",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.SawtoothReverse, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "PWM Lead",
			Category:    "Lead",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.15, Sustain: 0.7, Release: 0.25},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.15, Sustain: 0.7, Release: 0.25},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Vocal Synth",
			Category:    "Vocal",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.8, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Sawtooth, Phase: 0.25},
			Envelope2:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chiptune Strings",
			Category:    "Strings",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.2, Decay: 0.3, Sustain: 0.7, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.2, Decay: 0.3, Sustain: 0.5, Release: 0.4},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Glitchy FX",
			Category:    "FX",
			Oscillator1: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.3, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.75},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.2, Release: 0.1},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Retro Organ",
			Category:    "Organ",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.05, Sustain: 0.9, Release: 0.15},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.05, Sustain: 0.8, Release: 0.15},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Digital Flute",
			Category:    "Wind",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.08, Decay: 0.1, Sustain: 0.6, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.3},
			Envelope2:   audio.Envelope{Attack: 0.08, Decay: 0.1, Sustain: 0.4, Release: 0.3},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Synth Brass",
			Category:    "Brass",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.8, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chiptune Bells",
			Category:    "Bells",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.3, Sustain: 0.2, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.25},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.3, Sustain: 0.1, Release: 0.4},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Funky Guitar",
			Category:    "Guitar",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0.4, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Sawtooth, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0.3, Release: 0.2},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Ambient Pads",
			Category:    "Pad",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.5, Decay: 0.3, Sustain: 0.9, Release: 0.8},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.6},
			Envelope2:   audio.Envelope{Attack: 0.5, Decay: 0.3, Sustain: 0.7, Release: 0.8},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Epic Lead",
			Category:    "Lead",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.03, Decay: 0.2, Sustain: 0.8, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.3},
			Envelope2:   audio.Envelope{Attack: 0.03, Decay: 0.2, Sustain: 0.6, Release: 0.4},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Game Over Sound",
			Category:    "FX",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.5, Sustain: 0, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.3, Sustain: 0, Release: 0.2},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Retro Kick Drum",
			Category:    "Percussion",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.1, Sustain: 0, Release: 0.05},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.08, Sustain: 0, Release: 0.03},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Synthesized Choir",
			Category:    "Choir",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.2, Decay: 0.2, Sustain: 0.9, Release: 0.5},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.2, Decay: 0.2, Sustain: 0.8, Release: 0.5},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chiptune Harp",
			Category:    "Strings",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.2, Sustain: 0.3, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.2},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.2, Sustain: 0.2, Release: 0.4},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Vocoder Effect",
			Category:    "FX",
			Oscillator1: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.02, Decay: 0.1, Sustain: 0.7, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.02, Decay: 0.1, Sustain: 0.6, Release: 0.2},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Retro Synth Bass",
			Category:    "Bass",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.9, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0.7, Release: 0.1},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chiptune Flanger",
			Category:    "FX",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.15, Sustain: 0.7, Release: 0.25},
			Oscillator2: audio.Oscillator{Type: audio.SawtoothReverse, Phase: 0.1},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.15, Sustain: 0.7, Release: 0.25},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Synthesized Trumpet",
			Category:    "Brass",
			Oscillator1: audio.Oscillator{Type: audio.Sawtooth, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.8, Release: 0.3},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.05, Decay: 0.1, Sustain: 0.6, Release: 0.3},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chiptune Organ",
			Category:    "Organ",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.05, Sustain: 0.9, Release: 0.2},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.05, Sustain: 0.8, Release: 0.2},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Retro Soundtrack",
			Category:    "Pad",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.1, Decay: 0.2, Sustain: 0.8, Release: 0.4},
			Oscillator2: audio.Oscillator{Type: audio.Sawtooth, Phase: 0.3},
			Envelope2:   audio.Envelope{Attack: 0.1, Decay: 0.2, Sustain: 0.6, Release: 0.4},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Laser Zap",
			Category:    "FX",
			Oscillator1: audio.Oscillator{Type: audio.SawtoothReverse, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.08, Sustain: 0, Release: 0.05},
			Oscillator2: audio.Oscillator{Type: audio.Sine, Phase: 0.2},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.06, Sustain: 0, Release: 0.04},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Sub Pulse Bass",
			Category:    "Bass",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.01, Decay: 0.12, Sustain: 0.9, Release: 0.12},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.5},
			Envelope2:   audio.Envelope{Attack: 0.01, Decay: 0.12, Sustain: 0.6, Release: 0.12},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Pluck Lead",
			Category:    "Lead",
			Oscillator1: audio.Oscillator{Type: audio.Square, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.12, Sustain: 0.3, Release: 0.1},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.3},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.12, Sustain: 0.2, Release: 0.1},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Lo-Fi Keys",
			Category:    "Keys",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.7, Release: 0.25},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.4},
			Envelope2:   audio.Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.5, Release: 0.25},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chip Snare",
			Category:    "Percussion",
			Oscillator1: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.09, Sustain: 0, Release: 0.06},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.6},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.05, Sustain: 0, Release: 0.04},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chip Hi-Hat",
			Category:    "Percussion",
			Oscillator1: audio.Oscillator{Type: audio.Noise, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.04, Sustain: 0, Release: 0.03},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.8},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.03, Sustain: 0, Release: 0.02},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Crystal Pad",
			Category:    "Pad",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.25, Decay: 0.2, Sustain: 0.85, Release: 0.6},
			Oscillator2: audio.Oscillator{Type: audio.Triangle, Phase: 0.2},
			Envelope2:   audio.Envelope{Attack: 0.25, Decay: 0.2, Sustain: 0.7, Release: 0.6},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Metal Bell",
			Category:    "Bells",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.001, Decay: 0.4, Sustain: 0.15, Release: 0.5},
			Oscillator2: audio.Oscillator{Type: audio.Sawtooth, Phase: 0.2},
			Envelope2:   audio.Envelope{Attack: 0.001, Decay: 0.35, Sustain: 0.1, Release: 0.45},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Chiptune Piano",
			Category:    "Keys",
			Oscillator1: audio.Oscillator{Type: audio.Triangle, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.005, Decay: 0.2, Sustain: 0.5, Release: 0.25},
			Oscillator2: audio.Oscillator{Type: audio.Square, Phase: 0.25},
			Envelope2:   audio.Envelope{Attack: 0.005, Decay: 0.2, Sustain: 0.35, Release: 0.25},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
		{
			Name:        "Airy Vox",
			Category:    "Vocal",
			Oscillator1: audio.Oscillator{Type: audio.Sine, Phase: 0},
			Envelope1:   audio.Envelope{Attack: 0.08, Decay: 0.2, Sustain: 0.7, Release: 0.35},
			Oscillator2: audio.Oscillator{Type: audio.Sawtooth, Phase: 0.15},
			Envelope2:   audio.Envelope{Attack: 0.08, Decay: 0.2, Sustain: 0.5, Release: 0.35},
			Mixer:       audio.Mixer{Volume1: 1.0, Volume2: 1.0},
		},
	}
}
