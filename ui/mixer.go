package ui

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tetrackt/tetrackt/audio"
)

type Mixer struct {
	BalanceBar   Bar
	Mixer        audio.Mixer
	GlobalVolume float64 // Global output volume (0.0 to 1.0), set by main
}

type MixerUpdated struct {
	Mixer audio.Mixer
}

func NewMixer(balance float64) *Mixer {
	return &Mixer{
		Mixer:        audio.Mixer{Balance: balance},
		BalanceBar:   NewBar(0, 1, balance, 10),
		GlobalVolume: 1.0,
	}
}

func (m *Mixer) Init() tea.Cmd {
	return nil
}

func (m *Mixer) View() string {
	envView := strings.Builder{}
	envView.WriteString("Mixer:\n")

	v := m.Mixer.Balance

	fmt.Fprintf(&envView, "%3d%% ", int(math.Round((1-v)*100)))
	envView.WriteString(m.BalanceBar.View())
	fmt.Fprintf(&envView, " %3d%%", int(math.Round(v*100)))
	envView.WriteString("\n")
	envView.WriteString(fmt.Sprintf("Volume:  %3d%%", int(m.GlobalVolume*100)))

	return envView.String()
}

func (m *Mixer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			m.Mixer.Balance -= 0.01
		case "shift+left":
			m.Mixer.Balance -= 0.1
		case "right":
			m.Mixer.Balance += 0.01
		case "shift+right":
			m.Mixer.Balance += 0.1
		}
	}

	m.Mixer.Balance = math.Round(m.Mixer.Balance*100) / 100
	if m.Mixer.Balance < 0 {
		m.Mixer.Balance = 0
	} else if m.Mixer.Balance > 1 {
		m.Mixer.Balance = 1
	}

	m.BalanceBar.Value = m.Mixer.Balance

	// TODO: Optimize to only send update when value changes
	return m, func() tea.Msg {
		return MixerUpdated{
			Mixer: m.Mixer,
		}
	}
}
