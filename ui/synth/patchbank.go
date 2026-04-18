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
	Name  string
	Bank  string
	Tags  []string // e.g. ["Custom", "NES", "C64"]
	Synth *audio.Synth
}

// IsCustom reports whether this patch was saved by the user.
func (p SynthPatch) IsCustom() bool {
	return slices.Contains(p.Tags, "Custom")
}

// focusRow identifies which row in the patch bank UI has keyboard focus.
type focusRow int

const (
	focusBank focusRow = iota // bank filter row
	focusTag                  // tag filter row
	focusList                 // patch list
)

// SynthPatchBankView is the UI component for browsing and managing synth patches.
type SynthPatchBankView struct {
	Patches       []SynthPatch
	SelectedPatch int
	MaxHeight     int

	// Bank filter
	Banks      []string
	BankIndex  int
	BankCounts map[string]int

	// Tag filter
	Tags      []string
	TagIndex  int
	TagCounts map[string]int

	// Which row has focus.
	focus focusRow
}

// NewSynthPatchBankView initializes a new patch bank view with built-in patches.
func NewSynthPatchBankView() *SynthPatchBankView {
	patches := builtinPatches()
	slices.SortFunc(patches, func(i, j SynthPatch) int {
		return strings.Compare(i.Name, j.Name)
	})
	for i := range patches {
		populateSynthMeta(&patches[i])
	}
	banks, bankCounts := buildBanks(patches)
	tags, tagCounts := buildTags(patches)

	return &SynthPatchBankView{
		Patches:    patches,
		Banks:      banks,
		BankCounts: bankCounts,
		Tags:       tags,
		TagCounts:  tagCounts,
	}
}

// SetUserPatches replaces all custom patches and rebuilds the filter lists.
func (v *SynthPatchBankView) SetUserPatches(patches []SynthPatch) {
	filtered := make([]SynthPatch, 0, len(v.Patches))
	for _, p := range v.Patches {
		if !p.IsCustom() {
			filtered = append(filtered, p)
		}
	}
	for i := range patches {
		populateSynthMeta(&patches[i])
	}
	filtered = append(filtered, patches...)
	slices.SortFunc(filtered, func(i, j SynthPatch) int {
		return strings.Compare(i.Name, j.Name)
	})
	v.Patches = filtered
	v.Banks, v.BankCounts = buildBanks(v.Patches)
	v.Tags, v.TagCounts = buildTags(v.Patches)
	v.snapSelectionToFilter()
}

// populateSynthMeta copies patch-level display metadata into the synth so
// the metadata travels with the synth when it is assigned to a track.
func populateSynthMeta(p *SynthPatch) {
	if p.Synth == nil {
		return
	}
	p.Synth.Meta = audio.Metadata{Bank: p.Bank, Name: p.Name, Tags: p.Tags}
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

// moveUp moves focus up: list → tag filter → bank filter.
func (v *SynthPatchBankView) moveUp() {
	switch v.focus {
	case focusBank:
		// already at top
	case focusTag:
		v.focus = focusBank
	case focusList:
		indexes := v.filteredIndexes()
		if len(indexes) == 0 {
			v.focus = focusTag
			return
		}
		pos := v.selectionIndex(indexes)
		if pos == 0 {
			v.focus = focusTag
		} else {
			v.SelectedPatch = indexes[pos-1]
		}
	}
}

// moveDown moves focus down: bank filter → tag filter → patch list.
func (v *SynthPatchBankView) moveDown() {
	switch v.focus {
	case focusBank:
		v.focus = focusTag
	case focusTag:
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
	case focusBank:
		if len(v.Banks) > 0 {
			v.BankIndex = (v.BankIndex - 1 + len(v.Banks)) % len(v.Banks)
			v.snapSelectionToFilter()
		}
	case focusTag:
		if len(v.Tags) > 0 {
			v.TagIndex = (v.TagIndex - 1 + len(v.Tags)) % len(v.Tags)
			v.snapSelectionToFilter()
		}
	}
}

func (v *SynthPatchBankView) filterRight() {
	switch v.focus {
	case focusBank:
		if len(v.Banks) > 0 {
			v.BankIndex = (v.BankIndex + 1) % len(v.Banks)
			v.snapSelectionToFilter()
		}
	case focusTag:
		if len(v.Tags) > 0 {
			v.TagIndex = (v.TagIndex + 1) % len(v.Tags)
			v.snapSelectionToFilter()
		}
	}
}

// buildBanks returns ["All", ...unique banks] with counts.
func buildBanks(patches []SynthPatch) ([]string, map[string]int) {
	banks := []string{"All"}
	seen := map[string]bool{"All": true}
	counts := map[string]int{"All": len(patches)}
	for _, p := range patches {
		if p.Bank == "" {
			continue
		}
		counts[p.Bank]++
		if !seen[p.Bank] {
			banks = append(banks, p.Bank)
			seen[p.Bank] = true
		}
	}
	return banks, counts
}

// buildTags returns ["All", ...unique tags] with counts.
func buildTags(patches []SynthPatch) ([]string, map[string]int) {
	tags := []string{"All"}
	seen := map[string]bool{"All": true}
	counts := map[string]int{"All": len(patches)}
	for _, p := range patches {
		for _, t := range p.Tags {
			counts[t]++
			if !seen[t] {
				tags = append(tags, t)
				seen[t] = true
			}
		}
	}
	return tags, counts
}

func (v *SynthPatchBankView) currentBank() string {
	if v.BankIndex < 0 || v.BankIndex >= len(v.Banks) {
		v.BankIndex = 0
	}
	if len(v.Banks) == 0 {
		return "All"
	}
	return v.Banks[v.BankIndex]
}

func (v *SynthPatchBankView) currentTag() string {
	if v.TagIndex < 0 || v.TagIndex >= len(v.Tags) {
		v.TagIndex = 0
	}
	if len(v.Tags) == 0 {
		return "All"
	}
	return v.Tags[v.TagIndex]
}

// filteredIndexes returns indexes of patches passing both active filters.
func (v *SynthPatchBankView) filteredIndexes() []int {
	if len(v.Patches) == 0 {
		return nil
	}
	bank := v.currentBank()
	tag := v.currentTag()
	indexes := make([]int, 0, len(v.Patches))
	for i, p := range v.Patches {
		if tag != "All" && !slices.Contains(p.Tags, tag) {
			continue
		}
		if bank != "All" && p.Bank != bank {
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
			v.focus = focusTag
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

	// Bank filter row
	bank := v.currentBank()
	bankLine := fmt.Sprintf("Bank:     ◀ %s ▶", bank)
	if v.focus == focusBank {
		bankLine = common.StyleSelected.Render(bankLine)
	}
	fmt.Fprintln(&b, bankLine)

	// Tag filter row
	tag := v.currentTag()
	tagLine := fmt.Sprintf("Tag:      ◀ %s ▶", tag)
	if v.focus == focusTag {
		tagLine = common.StyleSelected.Render(tagLine)
	}
	fmt.Fprintln(&b, tagLine)

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
		if patch.IsCustom() {
			name = "★ " + name
		}
		if v.focus == focusList && patchIndex == v.SelectedPatch {
			name = common.StyleSelected.Render(name)
		}
		fmt.Fprintln(&b, name)
	}

	return b.String()
}
