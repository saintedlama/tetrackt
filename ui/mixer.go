package ui

import (
	"fmt"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui/widgets"
)

type Mixer struct {
	Bar1     widgets.Bar
	Bar2     widgets.Bar
	Mixer    audio.Mixer
	selected int // 0=osc1, 1=osc2
}

type MixerUpdated struct {
	Mixer audio.Mixer
}

func NewMixer(vol1, vol2 float64) *Mixer {
	return &Mixer{
		Mixer: audio.Mixer{Volume1: vol1, Volume2: vol2},
		Bar1:  widgets.NewBar(0, 1, vol1, 10),
		Bar2:  widgets.NewBar(0, 1, vol2, 10),
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

func (m *Mixer) View() string {
	var sb strings.Builder
	marker1, marker2 := " ", " "
	if m.selected == 0 {
		marker1 = ">"
	} else {
		marker2 = ">"
	}
	fmt.Fprintf(&sb, "%s Osc1: %s %3d%%\n", marker1, m.Bar1.View(), int(math.Round(m.Mixer.Volume1*100)))
	fmt.Fprintf(&sb, "%s Osc2: %s %3d%%", marker2, m.Bar2.View(), int(math.Round(m.Mixer.Volume2*100)))
	return sb.String()
}

func (m *Mixer) Update(msg tea.Msg) (Component, tea.Cmd) {
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
