package synth

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	ui "github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

type Mixer struct {
	Bar1          common.Bar
	Bar2          common.Bar
	Mixer         audio.Mixer
	Portamento    float64
	portamentoBar common.Bar
	selected      int // 0=osc1, 1=osc2, 2=portamento
}

type MixerUpdated struct {
	Mixer audio.Mixer
}

func NewMixer(vol1, vol2, portamento float64) *Mixer {
	return &Mixer{
		Mixer:         audio.Mixer{Volume1: vol1, Volume2: vol2},
		Bar1:          common.NewBar(0, 1, vol1, 10),
		Bar2:          common.NewBar(0, 1, vol2, 10),
		Portamento:    portamento,
		portamentoBar: common.NewBar(0, 2, portamento, 10),
	}
}

// SetMixer updates the audio.Mixer value and syncs the bars.
func (m *Mixer) SetMixer(mixer audio.Mixer) {
	m.Mixer = mixer
	m.Bar1.Value = mixer.Volume1
	m.Bar2.Value = mixer.Volume2
}

// SetPortamento updates the portamento value and syncs the bar.
func (m *Mixer) SetPortamento(portamento float64) {
	m.Portamento = portamento
	m.portamentoBar.Value = portamento
}

func (m *Mixer) Init() tea.Cmd {
	return nil
}

var mixerSelectedStyle = lipgloss.NewStyle().
	Background(common.ColorGrayDark).
	Foreground(common.ColorAccentPrimary)

func (m *Mixer) View() string {
	var sb strings.Builder
	renderRow := func(label string, bar common.Bar, vol float64, selected bool) string {
		labelPart := label + ":"
		if selected {
			labelPart = mixerSelectedStyle.Render(labelPart)
		}
		return fmt.Sprintf("%s %s %3d%%", labelPart, bar.View(), int(math.Round(vol*100)))
	}
	sb.WriteString(renderRow("Osc1", m.Bar1, m.Mixer.Volume1, m.selected == 0))
	sb.WriteString("\n")
	sb.WriteString(renderRow("Osc2", m.Bar2, m.Mixer.Volume2, m.selected == 1))
	sb.WriteString("\n")
	sb.WriteString("\n")
	glideLabel := "Glide:"
	if m.selected == 2 {
		glideLabel = mixerSelectedStyle.Render(glideLabel)
	}
	sb.WriteString(fmt.Sprintf("%s %s %4.2fs", glideLabel, m.portamentoBar.View(), m.Portamento))
	return sb.String()
}

func (m *Mixer) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			if m.selected > 0 {
				m.selected--
			}
		case "down":
			if m.selected < 2 {
				m.selected++
			}
		case "left":
			m.adjustSelected(-0.01)
		case "shift+left":
			m.adjustSelected(-0.1)
		case "right":
			m.adjustSelected(0.01)
		case "shift+right":
			m.adjustSelected(0.1)
		}
	}

	m.Bar1.Value = m.Mixer.Volume1
	m.Bar2.Value = m.Mixer.Volume2
	m.portamentoBar.Value = m.Portamento

	return m, func() tea.Msg {
		return MixerUpdated{Mixer: m.Mixer}
	}
}

func (m *Mixer) adjustSelected(delta float64) {
	switch m.selected {
	case 0:
		m.Mixer.Volume1 = clampVolume(math.Round((m.Mixer.Volume1+delta)*100) / 100)
	case 1:
		m.Mixer.Volume2 = clampVolume(math.Round((m.Mixer.Volume2+delta)*100) / 100)
	case 2:
		m.Portamento = clampPortamento(math.Round((m.Portamento+delta)*100) / 100)
	}
}

func clampPortamento(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 2 {
		return 2
	}
	return v
}

func clampVolume(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
