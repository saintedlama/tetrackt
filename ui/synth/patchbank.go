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

// SynthPatch represents a complete synth patch configuration.
type SynthPatch struct {
	Name     string
	Category string
	Custom   bool // true = user-saved patch
	Synth    *audio.Synth
}

// focusRow identifies which row in the patch bank UI has keyboard focus.
type focusRow int

const (
	focusCategory focusRow = iota // category filter row
	focusCustom                   // custom-only filter row
	focusList                     // patch list
)

// SynthPatchBankView is the UI component for browsing and managing synth patches.
// Navigation: up/down moves focus between the two filter rows and the patch list.
// Left/right on a filter row changes that filter's value.
type SynthPatchBankView struct {
	Patches       []SynthPatch
	SelectedPatch int
	MaxHeight     int

	// Category filter
	Categories     []string
	CategoryIndex  int
	CategoryCounts map[string]int

	// Custom filter: when true, only custom (user-saved) patches are shown.
	CustomOnly bool

	// Which row has focus.
	focus focusRow
}

// NewSynthPatchBankView initializes a new patch bank view with built-in patches.
func NewSynthPatchBankView() *SynthPatchBankView {
	patches := builtinPatches()
	slices.SortFunc(patches, func(i, j SynthPatch) int {
		return strings.Compare(i.Name, j.Name)
	})
	categories, categoryCounts := buildCategories(patches)

	return &SynthPatchBankView{
		Patches:        patches,
		Categories:     categories,
		CategoryCounts: categoryCounts,
	}
}

// SetUserPatches replaces all custom patches and rebuilds the category list.
func (v *SynthPatchBankView) SetUserPatches(patches []SynthPatch) {
	filtered := make([]SynthPatch, 0, len(v.Patches))
	for _, p := range v.Patches {
		if !p.Custom {
			filtered = append(filtered, p)
		}
	}
	filtered = append(filtered, patches...)
	slices.SortFunc(filtered, func(i, j SynthPatch) int {
		return strings.Compare(i.Name, j.Name)
	})
	v.Patches = filtered
	v.Categories, v.CategoryCounts = buildCategories(v.Patches)
	v.snapSelectionToFilter()
}

// GetPatch returns the patch at the specified index.
func (v *SynthPatchBankView) GetPatch(index int) *SynthPatch {
	if index >= 0 && index < len(v.Patches) {
		return &v.Patches[index]
	}
	return nil
}

func (v *SynthPatchBankView) Init() tea.Cmd { return nil }

// Update handles navigation and selection.
func (v *SynthPatchBankView) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			v.moveUp()
		case "down":
			v.moveDown()
		case "left":
			v.filterLeft()
		case "right":
			v.filterRight()
		case "enter":
			if v.focus == focusList {
				indexes := v.filteredIndexes()
				if len(indexes) == 0 {
					return v, nil
				}
				patch := v.Patches[v.SelectedPatch]
				return v, func() tea.Msg { return ui.SynthUpdated{Synth: patch.Synth} }
			}
		}
	}
	return v, nil
}

// moveUp moves focus up: list → custom filter → category filter.
// Within the list, moves to the previous patch; wraps up into the filter area.
func (v *SynthPatchBankView) moveUp() {
	switch v.focus {
	case focusCategory:
		// already at top
	case focusCustom:
		v.focus = focusCategory
	case focusList:
		indexes := v.filteredIndexes()
		if len(indexes) == 0 {
			v.focus = focusCustom
			return
		}
		pos := v.selectionIndex(indexes)
		if pos == 0 {
			v.focus = focusCustom
		} else {
			v.SelectedPatch = indexes[pos-1]
		}
	}
}

// moveDown moves focus down: category filter → custom filter → patch list.
// Within the list, moves to the next patch.
func (v *SynthPatchBankView) moveDown() {
	switch v.focus {
	case focusCategory:
		v.focus = focusCustom
	case focusCustom:
		if len(v.filteredIndexes()) > 0 {
			v.focus = focusList
			v.snapSelectionToFilter()
		}
	case focusList:
		indexes := v.filteredIndexes()
		if len(indexes) == 0 {
			return
		}
		pos := v.selectionIndex(indexes)
		if pos < len(indexes)-1 {
			v.SelectedPatch = indexes[pos+1]
		}
	}
}

func (v *SynthPatchBankView) filterLeft() {
	switch v.focus {
	case focusCategory:
		if len(v.Categories) > 0 {
			v.CategoryIndex = (v.CategoryIndex - 1 + len(v.Categories)) % len(v.Categories)
			v.snapSelectionToFilter()
		}
	case focusCustom:
		v.CustomOnly = !v.CustomOnly
		v.snapSelectionToFilter()
	}
}

func (v *SynthPatchBankView) filterRight() {
	switch v.focus {
	case focusCategory:
		if len(v.Categories) > 0 {
			v.CategoryIndex = (v.CategoryIndex + 1) % len(v.Categories)
			v.snapSelectionToFilter()
		}
	case focusCustom:
		v.CustomOnly = !v.CustomOnly
		v.snapSelectionToFilter()
	}
}

// buildCategories returns the list of category names (always starting with "All")
// and a map of name → count.
func buildCategories(patches []SynthPatch) ([]string, map[string]int) {
	categories := []string{"All"}
	seen := map[string]bool{"All": true}
	counts := map[string]int{"All": len(patches)}
	for _, p := range patches {
		if p.Category == "" {
			continue
		}
		counts[p.Category]++
		if !seen[p.Category] {
			categories = append(categories, p.Category)
			seen[p.Category] = true
		}
	}
	return categories, counts
}

func (v *SynthPatchBankView) currentCategory() string {
	if v.CategoryIndex < 0 || v.CategoryIndex >= len(v.Categories) {
		v.CategoryIndex = 0
	}
	if len(v.Categories) == 0 {
		return "All"
	}
	return v.Categories[v.CategoryIndex]
}

// filteredIndexes returns the indexes of patches that pass both active filters.
func (v *SynthPatchBankView) filteredIndexes() []int {
	if len(v.Patches) == 0 {
		return nil
	}
	category := v.currentCategory()
	indexes := make([]int, 0, len(v.Patches))
	for i, p := range v.Patches {
		if v.CustomOnly && !p.Custom {
			continue
		}
		if category != "All" && p.Category != category {
			continue
		}
		indexes = append(indexes, i)
	}
	return indexes
}

func (v *SynthPatchBankView) selectionIndex(indexes []int) int {
	for i, idx := range indexes {
		if idx == v.SelectedPatch {
			return i
		}
	}
	return 0
}

func (v *SynthPatchBankView) snapSelectionToFilter() {
	indexes := v.filteredIndexes()
	if len(indexes) == 0 {
		if v.focus == focusList {
			v.focus = focusCustom
		}
		return
	}
	if !slices.Contains(indexes, v.SelectedPatch) {
		v.SelectedPatch = indexes[0]
	}
}

// View renders the two filter rows followed by the patch list.
func (v *SynthPatchBankView) View() string {
	var b strings.Builder

	// Category filter row
	cat := v.currentCategory()
	catVal := fmt.Sprintf("◀ %s ▶", cat)
	catLine := fmt.Sprintf("Category: %s", catVal)
	if v.focus == focusCategory {
		catLine = common.StyleSelected.Render(catLine)
	}
	fmt.Fprintln(&b, catLine)

	// Custom filter row
	customVal := "All"
	if v.CustomOnly {
		customVal = "Custom only"
	}
	customLine := fmt.Sprintf("Custom:   ◀ %s ▶", customVal)
	if v.focus == focusCustom {
		customLine = common.StyleSelected.Render(customLine)
	}
	fmt.Fprintln(&b, customLine)

	fmt.Fprintln(&b, "")

	// Patch list
	indexes := v.filteredIndexes()
	if len(indexes) == 0 {
		fmt.Fprintln(&b, "(no patches)")
		return b.String()
	}

	selectedIndex := v.selectionIndex(indexes)
	start := 0
	end := len(indexes)
	if v.MaxHeight > 0 {
		visibleItems := max(v.MaxHeight-1, 1)
		if selectedIndex >= visibleItems {
			start = selectedIndex - visibleItems + 1
		}
		maxStart := max(len(indexes)-visibleItems, 0)
		start = min(start, maxStart)
		end = min(start+visibleItems, len(indexes))
	}

	for idx := start; idx < end; idx++ {
		patchIndex := indexes[idx]
		patch := v.Patches[patchIndex]
		name := patch.Name
		if patch.Custom {
			name = "★ " + name
		}
		if v.focus == focusList && patchIndex == v.SelectedPatch {
			name = common.StyleSelected.Render(name)
		}
		fmt.Fprintln(&b, name)
	}

	return b.String()
}

func builtinPatches() []SynthPatch {
	return []SynthPatch{}
}
