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
	Bar1     common.Bar
	Bar2     common.Bar
	Mixer    audio.Mixer
	selected int // 0=osc1, 1=osc2
}

type MixerUpdated struct {
	Mixer audio.Mixer
}

func NewMixer(vol1, vol2 float64) *Mixer {
	return &Mixer{
		Mixer: audio.Mixer{Volume1: vol1, Volume2: vol2},
		Bar1:  common.NewBar(0, 1, vol1, 10),
		Bar2:  common.NewBar(0, 1, vol2, 10),
	}
}

// SetMixer updates the audio.Mixer value and syncs the bars.
func (m *Mixer) SetMixer(mixer audio.Mixer) {
	m.Mixer = mixer
	m.Bar1.Value = mixer.Volume1
	m.Bar2.Value = mixer.Volume2
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
			if m.selected < 1 {
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

	return m, func() tea.Msg {
		return MixerUpdated{Mixer: m.Mixer}
	}
}

func (m *Mixer) adjustSelected(delta float64) {
	if m.selected == 0 {
		m.Mixer.Volume1 = clampVolume(math.Round((m.Mixer.Volume1+delta)*100) / 100)
	} else {
		m.Mixer.Volume2 = clampVolume(math.Round((m.Mixer.Volume2+delta)*100) / 100)
	}
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
