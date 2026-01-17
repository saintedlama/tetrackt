package ui

import (
	"fmt"
	"math"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Mixer struct {
	BalanceBar   Bar
	MixBalance   float64
	GlobalVolume float64 // Global output volume (0.0 to 1.0), set by main
}

type MixerUpdated struct {
	Balance float64
}

func NewMixer(balance float64) *Mixer {
	return &Mixer{
		MixBalance:   balance,
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

	v := m.MixBalance

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
			m.MixBalance -= 0.01
		case "shift+left":
			m.MixBalance -= 0.1
		case "right":
			m.MixBalance += 0.01
		case "shift+right":
			m.MixBalance += 0.1
		}
	}

	m.MixBalance = math.Round(m.MixBalance*100) / 100

	if m.MixBalance < 0 {
		m.MixBalance = 0
	} else if m.MixBalance > 1 {
		m.MixBalance = 1
	}

	m.BalanceBar.Value = m.MixBalance

	// TODO: Optimize to only send update when value changes
	return m, func() tea.Msg {
		return MixerUpdated{
			Balance: m.MixBalance,
		}
	}
}
