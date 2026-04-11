package synth

import (
	"fmt"
	"slices"
	"strings"
	"time"

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
	filterFieldEnvDepth
	filterFieldEnvAttack
	filterFieldEnvDecay
	filterFieldEnvSustain
	filterFieldEnvRelease
)

// FilterModel is the UI component for editing the filter parameters.
type FilterModel struct {
	Filter          audio.Filter
	FilterEnvelope  audio.FilterEnvelope
	filterList      []audio.FilterType
	filterTypeStyle lipgloss.Style
	editField       filterField
	cutoffBar       common.Bar
	resonanceBar    common.Bar
	envDepthBar     common.Bar
	envSustainBar   common.Bar
}

// FilterUpdated is emitted when any filter parameter changes.
type FilterUpdated struct {
	Filter         audio.Filter
	FilterEnvelope audio.FilterEnvelope
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
		envDepthBar:     common.NewBar(0, 1, 0, 10),
		envSustainBar:   common.NewBar(0, 1, 0, 10),
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
	sb.WriteString(renderBarRow("Cutoff    ", m.cutoffBar.View(), int(m.Filter.Cutoff*100), m.editField == filterFieldCutoff))
	sb.WriteString("\n")
	sb.WriteString(renderBarRow("Resonance ", m.resonanceBar.View(), int(m.Filter.Resonance*100), m.editField == filterFieldResonance))
	if m.Filter.Type != audio.FilterOff {
		fe := m.FilterEnvelope
		sb.WriteString("\n")
		sb.WriteString(renderBarRow("Depth     ", m.envDepthBar.View(), int(fe.Depth*100), m.editField == filterFieldEnvDepth))
		sb.WriteString("\n")
		sb.WriteString(common.RenderKnobDurationSelected("Attack    ", fe.Attack, m.editField == filterFieldEnvAttack))
		sb.WriteString("\n")
		sb.WriteString(common.RenderKnobDurationSelected("Decay     ", fe.Decay, m.editField == filterFieldEnvDecay))
		sb.WriteString("\n")
		sb.WriteString(renderBarRow("Sustain   ", m.envSustainBar.View(), int(fe.Sustain*100), m.editField == filterFieldEnvSustain))
		sb.WriteString("\n")
		sb.WriteString(common.RenderKnobDurationSelected("Release   ", fe.Release, m.editField == filterFieldEnvRelease))
	}
	return sb.String()
}

func (m *FilterModel) numFields() filterField {
	if m.Filter.Type != audio.FilterOff {
		return 8
	}
	return 3
}

func (m *FilterModel) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			n := m.numFields()
			m.editField = (m.editField - 1 + n) % n
		case "down":
			n := m.numFields()
			m.editField = (m.editField + 1) % n
		case "left", "shift+left":
			delta := 0.05
			timeDelta := 0.01
			if msg.String() == "shift+left" {
				delta = 0.1
				timeDelta = 0.10
			}
			m.adjustField(-delta, -timeDelta)
			return m, func() tea.Msg { return FilterUpdated{Filter: m.Filter, FilterEnvelope: m.FilterEnvelope} }
		case "right", "shift+right":
			delta := 0.05
			timeDelta := 0.01
			if msg.String() == "shift+right" {
				delta = 0.1
				timeDelta = 0.10
			}
			m.adjustField(delta, timeDelta)
			return m, func() tea.Msg { return FilterUpdated{Filter: m.Filter, FilterEnvelope: m.FilterEnvelope} }
		}
	}
	return m, nil
}

func (m *FilterModel) adjustField(delta, timeDelta float64) {
	switch m.editField {
	case filterFieldType:
		dir := 1
		if delta < 0 {
			dir = -1
		}
		m.Filter.Type = cycleFilter(m.filterList, m.Filter.Type, dir)
		if m.Filter.Type == audio.FilterOff && m.editField > filterFieldResonance {
			m.editField = filterFieldType
		}
	case filterFieldCutoff:
		m.Filter.Cutoff = clampFilter(m.Filter.Cutoff + delta)
		m.cutoffBar.Value = m.Filter.Cutoff
	case filterFieldResonance:
		m.Filter.Resonance = clampFilter(m.Filter.Resonance + delta)
		m.resonanceBar.Value = m.Filter.Resonance
	case filterFieldEnvDepth:
		m.FilterEnvelope.Depth = clampFilter(m.FilterEnvelope.Depth + delta)
		m.envDepthBar.Value = m.FilterEnvelope.Depth
	case filterFieldEnvAttack:
		d := m.FilterEnvelope.Attack + time.Duration(timeDelta*float64(time.Second))
		if d < 0 {
			d = 0
		}
		m.FilterEnvelope.Attack = d
	case filterFieldEnvDecay:
		d := m.FilterEnvelope.Decay + time.Duration(timeDelta*float64(time.Second))
		if d < 0 {
			d = 0
		}
		m.FilterEnvelope.Decay = d
	case filterFieldEnvSustain:
		m.FilterEnvelope.Sustain = clampFilter(m.FilterEnvelope.Sustain + delta)
		m.envSustainBar.Value = m.FilterEnvelope.Sustain
	case filterFieldEnvRelease:
		d := m.FilterEnvelope.Release + time.Duration(timeDelta*float64(time.Second))
		if d < 0 {
			d = 0
		}
		m.FilterEnvelope.Release = d
	}
}

// SyncBars updates bar values to match the current Filter and FilterEnvelope state.
func (m *FilterModel) SyncBars() {
	m.cutoffBar.Value = m.Filter.Cutoff
	m.resonanceBar.Value = m.Filter.Resonance
	m.envDepthBar.Value = m.FilterEnvelope.Depth
	m.envSustainBar.Value = m.FilterEnvelope.Sustain
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
