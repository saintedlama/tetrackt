package synth

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
)

type lfoField int

const (
	lfoFieldWaveform lfoField = iota
	lfoFieldRate
	lfoFieldDepth
	lfoFieldDelay
	lfoFieldDest
	lfoFieldCount
)

var lfoDestNames = []string{"Pitch", "Volume", "Cutoff", "PulseWidth"}

// LFOModel is the UI component for editing a single LFO and its modulation destination.
type LFOModel struct {
	LFO          audio.LFO // Dest field inside LFO selects the modulation target
	editField    lfoField
	waveformList []audio.LFOWaveform
}

// LFOUpdated is emitted when the LFO parameters or destination change.
type LFOUpdated struct {
	LFO audio.LFO // Dest field inside LFO carries the modulation destination
}

func NewLFOModel(lfo audio.LFO) *LFOModel {
	return &LFOModel{
		LFO:          lfo,
		waveformList: []audio.LFOWaveform{audio.LFOSine, audio.LFOTriangle, audio.LFOSquare, audio.LFOSawtooth},
	}
}

func (m *LFOModel) Init() tea.Cmd { return nil }

func (m *LFOModel) View() string {
	if m.LFO.Depth == 0 {
		var sb strings.Builder
		sb.WriteString(renderFieldSelected(fmt.Sprintf("Wave:  %-10s", string(m.LFO.Waveform)), m.editField == lfoFieldWaveform))
		sb.WriteString("\n")
		sb.WriteString(renderFieldSelected("Rate:  (disabled)", m.editField == lfoFieldRate))
		sb.WriteString("\n")
		sb.WriteString(renderFieldSelected(fmt.Sprintf("Depth: %3d%%", int(m.LFO.Depth*100)), m.editField == lfoFieldDepth))
		sb.WriteString("\n")
		sb.WriteString(renderFieldSelected("Delay: (disabled)", m.editField == lfoFieldDelay))
		sb.WriteString("\n")
		sb.WriteString(renderFieldSelected("Dest:  (disabled)", m.editField == lfoFieldDest))
		return sb.String()
	}

	var sb strings.Builder

	destName := lfoDestNames[int(m.LFO.Dest)%len(lfoDestNames)]

	sb.WriteString(renderFieldSelected(fmt.Sprintf("Wave:  %-10s", string(m.LFO.Waveform)), m.editField == lfoFieldWaveform))
	sb.WriteString("\n")
	sb.WriteString(renderFieldSelected(fmt.Sprintf("Rate:  %5.2f Hz", m.LFO.Rate), m.editField == lfoFieldRate))
	sb.WriteString("\n")
	sb.WriteString(renderFieldSelected(fmt.Sprintf("Depth: %3d%%", int(m.LFO.Depth*100)), m.editField == lfoFieldDepth))
	sb.WriteString("\n")
	sb.WriteString(renderFieldSelected(fmt.Sprintf("Delay: %.2f s", m.LFO.Delay), m.editField == lfoFieldDelay))
	sb.WriteString("\n")
	sb.WriteString(renderFieldSelected(fmt.Sprintf("Dest:  %-10s", destName), m.editField == lfoFieldDest))

	return sb.String()
}

func (m *LFOModel) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			m.editField = (m.editField - 1 + lfoFieldCount) % lfoFieldCount
		case "down":
			m.editField = (m.editField + 1) % lfoFieldCount
		case "left":
			m.adjustField(-1)
			cmd = func() tea.Msg { return LFOUpdated{LFO: m.LFO} }
		case "shift+left":
			m.adjustField(-5)
			cmd = func() tea.Msg { return LFOUpdated{LFO: m.LFO} }
		case "right":
			m.adjustField(1)
			cmd = func() tea.Msg { return LFOUpdated{LFO: m.LFO} }
		case "shift+right":
			m.adjustField(5)
			cmd = func() tea.Msg { return LFOUpdated{LFO: m.LFO} }
		}
	}
	return m, cmd
}

func (m *LFOModel) adjustField(steps int) {
	switch m.editField {
	case lfoFieldWaveform:
		n := len(m.waveformList)
		idx := 0
		for i, w := range m.waveformList {
			if w == m.LFO.Waveform {
				idx = i
				break
			}
		}
		m.LFO.Waveform = m.waveformList[(idx+steps+n*10)%n]
	case lfoFieldRate:
		// Each step = 0.1 Hz; shift = 0.5 Hz per step
		delta := float64(steps) * 0.1
		m.LFO.Rate = max(0.01, min(20.0, m.LFO.Rate+delta))
	case lfoFieldDepth:
		delta := float64(steps) * 0.01
		m.LFO.Depth = max(0.0, min(1.0, m.LFO.Depth+delta))
	case lfoFieldDelay:
		delta := float64(steps) * 0.1
		m.LFO.Delay = max(0.0, min(10.0, m.LFO.Delay+delta))
	case lfoFieldDest:
		n := len(lfoDestNames)
		m.LFO.Dest = audio.ModDest((int(m.LFO.Dest) + steps + n*10) % n)
	}
}
