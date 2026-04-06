package synth

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

type filterField int

const (
	filterFieldType filterField = iota
	filterFieldCutoff
	filterFieldResonance
)

// FilterModel is the UI component for editing the filter parameters.
type FilterModel struct {
	Filter          audio.Filter
	filterList      []audio.FilterType
	filterTypeStyle lipgloss.Style
	editField       filterField
	cutoffBar       common.Bar
	resonanceBar    common.Bar
}

// FilterUpdated is emitted when any filter parameter changes.
type FilterUpdated struct {
	Filter audio.Filter
}

// NewFilterModel creates a FilterModel with the given initial filter state.
func NewFilterModel(filter audio.Filter) *FilterModel {
	filterList := []audio.FilterType{audio.FilterOff, audio.FilterLowPass, audio.FilterHighPass, audio.FilterBandPass}
	maxWidth := 0
	for _, t := range filterList {
		if len(string(t)) > maxWidth {
			maxWidth = len(string(t))
		}
	}
	return &FilterModel{
		Filter:          filter,
		filterList:      filterList,
		filterTypeStyle: lipgloss.NewStyle().Width(maxWidth),
		cutoffBar:       common.NewBar(0, 1, filter.Cutoff, 10),
		resonanceBar:    common.NewBar(0, 1, filter.Resonance, 10),
	}
}

func (m *FilterModel) Init() tea.Cmd { return nil }

func (m *FilterModel) View() string {
	var sb strings.Builder
	renderBarRow := func(label, barView string, pct int, selected bool) string {
		l := label
		if selected {
			l = common.StyleSelected.Render(label)
		}
		return fmt.Sprintf("%s %s %3d%%", l, barView, pct)
	}
	typeStr := renderFieldSelected(m.filterTypeStyle.Render(string(m.Filter.Type)), m.editField == filterFieldType)
	sb.WriteString(typeStr)
	sb.WriteString("\n")
	sb.WriteString(renderBarRow("Cutoff", m.cutoffBar.View(), int(m.Filter.Cutoff*100), m.editField == filterFieldCutoff))
	sb.WriteString("\n")
	sb.WriteString(renderBarRow("Reso  ", m.resonanceBar.View(), int(m.Filter.Resonance*100), m.editField == filterFieldResonance))
	return sb.String()
}

func (m *FilterModel) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			m.editField = (m.editField - 1 + 3) % 3
		case "down":
			m.editField = (m.editField + 1) % 3
		case "left", "shift+left":
			delta := 0.05
			if msg.String() == "shift+left" {
				delta = 0.1
			}
			switch m.editField {
			case filterFieldType:
				m.Filter.Type = cycleFilter(m.filterList, m.Filter.Type, -1)
			case filterFieldCutoff:
				m.Filter.Cutoff = clampFilter(m.Filter.Cutoff - delta)
				m.cutoffBar.Value = m.Filter.Cutoff
			case filterFieldResonance:
				m.Filter.Resonance = clampFilter(m.Filter.Resonance - delta)
				m.resonanceBar.Value = m.Filter.Resonance
			}
			return m, func() tea.Msg { return FilterUpdated{Filter: m.Filter} }
		case "right", "shift+right":
			delta := 0.05
			if msg.String() == "shift+right" {
				delta = 0.1
			}
			switch m.editField {
			case filterFieldType:
				m.Filter.Type = cycleFilter(m.filterList, m.Filter.Type, 1)
			case filterFieldCutoff:
				m.Filter.Cutoff = clampFilter(m.Filter.Cutoff + delta)
				m.cutoffBar.Value = m.Filter.Cutoff
			case filterFieldResonance:
				m.Filter.Resonance = clampFilter(m.Filter.Resonance + delta)
				m.resonanceBar.Value = m.Filter.Resonance
			}
			return m, func() tea.Msg { return FilterUpdated{Filter: m.Filter} }
		}
	}
	return m, nil
}

// SyncBars updates bar values to match the current Filter state.
func (m *FilterModel) SyncBars() {
	m.cutoffBar.Value = m.Filter.Cutoff
	m.resonanceBar.Value = m.Filter.Resonance
}

func cycleFilter(list []audio.FilterType, current audio.FilterType, step int) audio.FilterType {
	idx := slices.Index(list, current) + step
	if idx < 0 {
		return list[len(list)-1]
	}
	return list[idx%len(list)]
}

func clampFilter(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
