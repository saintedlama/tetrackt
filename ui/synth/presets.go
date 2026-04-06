package synth

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

// SynthPreset represents a complete synth preset configuration
type SynthPreset struct {
	Name     string
	Category string
	Synth    *audio.Synth
}

// SynthPresetView represents the UI component for managing synth presets
type SynthPresetView struct {
	Presets         []SynthPreset
	SelectedPreset  int
	CurrentTrackNum int
	MaxHeight       int
	Categories      []string
	CategoryIndex   int
	CategoryCounts  map[string]int
}

// NewSynthPresetView initializes a new synth preset view
func NewSynthPresetView() *SynthPresetView {
	presets := builtinPresets()
	slices.SortFunc(presets, func(i, j SynthPreset) int {
		return strings.Compare(i.Name, j.Name)
	})
	categories, categoryCounts := buildCategories(presets)

	return &SynthPresetView{
		Presets:         presets,
		SelectedPreset:  0,
		CurrentTrackNum: 0,
		Categories:      categories,
		CategoryIndex:   0,
		CategoryCounts:  categoryCounts,
	}
}

// GetPreset returns the synth preset at the specified index
func (ip *SynthPresetView) GetPreset(index int) *SynthPreset {
	if index >= 0 && index < len(ip.Presets) {
		return &ip.Presets[index]
	}
	return nil
}

func (ip *SynthPresetView) Init() tea.Cmd {
	return nil
}

// Update handles input for selecting and applying presets.
func (ip *SynthPresetView) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
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
				return ui.SynthUpdated{Synth: preset.Synth}
			}
		}
	}

	return ip, nil
}

func buildCategories(presets []SynthPreset) ([]string, map[string]int) {
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

func (ip *SynthPresetView) currentCategory() string {
	if len(ip.Categories) == 0 {
		return "All"
	}
	if ip.CategoryIndex < 0 || ip.CategoryIndex >= len(ip.Categories) {
		ip.CategoryIndex = 0
	}
	return ip.Categories[ip.CategoryIndex]
}

func (ip *SynthPresetView) categoryCount(name string) int {
	if ip.CategoryCounts == nil {
		return 0
	}
	return ip.CategoryCounts[name]
}

func (ip *SynthPresetView) categoryLabel(name string) string {
	return fmt.Sprintf("%s (%d)", name, ip.categoryCount(name))
}

func (ip *SynthPresetView) filteredIndexes() []int {
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

func (ip *SynthPresetView) selectionIndex(indexes []int) int {
	for i, idx := range indexes {
		if idx == ip.SelectedPreset {
			return i
		}
	}
	return 0
}

func (ip *SynthPresetView) moveSelection(step int) {
	indexes := ip.filteredIndexes()
	if len(indexes) == 0 {
		return
	}

	current := ip.selectionIndex(indexes)
	next := (current + step + len(indexes)) % len(indexes)
	ip.SelectedPreset = indexes[next]
}

func (ip *SynthPresetView) snapSelectionToFilter() {
	indexes := ip.filteredIndexes()
	if len(indexes) == 0 {
		return
	}

	if slices.Contains(indexes, ip.SelectedPreset) {
		return
	}

	ip.SelectedPreset = indexes[0]
}

// View renders the synth preset list.
func (ip *SynthPresetView) View() string {
	var view strings.Builder
	fmt.Fprintf(&view, "Synth Presets [%s]\n", ip.categoryLabel(ip.currentCategory()))

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
			name = common.StyleSelected.Render(name)
		}
		fmt.Fprintf(&view, "%s\n", name)
	}

	return view.String()
}

func builtinPresets() []SynthPreset {
	return []SynthPreset{}
}
