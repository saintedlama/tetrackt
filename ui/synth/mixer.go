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
	Mixer         audio.Mixer
	Portamento    float64
	volBar1       common.Bar
	panBar1       common.Bar
	volBar2       common.Bar
	panBar2       common.Bar
	volBar3       common.Bar
	panBar3       common.Bar
	masterBar     common.Bar
	portamentoBar common.Bar
	selected      int // 0=vol1, 1=pan1, 2=vol2, 3=pan2, 4=vol3, 5=pan3, 6=master, 7=mode, 8=portamento
}

const mixerNumRows = 9

type MixerUpdated struct {
	Mixer audio.Mixer
}

func NewMixer(mixer audio.Mixer, portamento float64) *Mixer {
	mv := mixer.MasterVolume
	if mv == 0 {
		mv = 1.0
	}
	return &Mixer{
		Mixer:         mixer,
		Portamento:    portamento,
		volBar1:       common.NewBar(0, 1, mixer.Volume1, 10),
		panBar1:       common.NewBar(-1, 1, mixer.Pan1, 10),
		volBar2:       common.NewBar(0, 1, mixer.Volume2, 10),
		panBar2:       common.NewBar(-1, 1, mixer.Pan2, 10),
		volBar3:       common.NewBar(0, 1, mixer.Volume3, 10),
		panBar3:       common.NewBar(-1, 1, mixer.Pan3, 10),
		masterBar:     common.NewBar(0, 1, mv, 10),
		portamentoBar: common.NewBar(0, 2, portamento, 10),
	}
}

// SetMixer updates the audio.Mixer value and syncs all bars.
func (m *Mixer) SetMixer(mixer audio.Mixer) {
	m.Mixer = mixer
	m.volBar1.Value = mixer.Volume1
	m.panBar1.Value = mixer.Pan1
	m.volBar2.Value = mixer.Volume2
	m.panBar2.Value = mixer.Pan2
	m.volBar3.Value = mixer.Volume3
	m.panBar3.Value = mixer.Pan3
	mv := mixer.MasterVolume
	if mv == 0 {
		mv = 1.0
	}
	m.masterBar.Value = mv
	m.Mixer.MasterVolume = mv
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

var mixerMutedStyle = lipgloss.NewStyle().Foreground(common.ColorAccentWarning)
var mixerActiveStyle = lipgloss.NewStyle().Foreground(common.ColorAccentPlay)

func muteIndicator(muted bool, volume float64) string {
	if muted || volume == 0 {
		return mixerMutedStyle.Render("[M]")
	}
	return mixerActiveStyle.Render("[ ]")
}

func (m *Mixer) View() string {
	var sb strings.Builder
	lbl := func(label string, sel bool) string {
		if sel {
			return mixerSelectedStyle.Render(label + ":")
		}
		return label + ":"
	}

	mv := m.Mixer.MasterVolume
	if mv == 0 {
		mv = 1.0
	}

	// Channel 1
	sb.WriteString(fmt.Sprintf("%s %s %3d%%  %s\n",
		lbl("Osc1", m.selected == 0),
		m.volBar1.View(),
		int(math.Round(m.Mixer.Volume1*100)),
		muteIndicator(m.Mixer.Mute1, m.Mixer.Volume1),
	))
	sb.WriteString(fmt.Sprintf("%s %s %+.2f\n",
		lbl("Pan1", m.selected == 1),
		m.panBar1.View(),
		m.Mixer.Pan1,
	))
	sb.WriteString("\n")
	// Channel 2
	sb.WriteString(fmt.Sprintf("%s %s %3d%%  %s\n",
		lbl("Osc2", m.selected == 2),
		m.volBar2.View(),
		int(math.Round(m.Mixer.Volume2*100)),
		muteIndicator(m.Mixer.Mute2, m.Mixer.Volume2),
	))
	sb.WriteString(fmt.Sprintf("%s %s %+.2f\n",
		lbl("Pan2", m.selected == 3),
		m.panBar2.View(),
		m.Mixer.Pan2,
	))
	sb.WriteString("\n")
	// Channel 3
	sb.WriteString(fmt.Sprintf("%s %s %3d%%  %s\n",
		lbl("Osc3", m.selected == 4),
		m.volBar3.View(),
		int(math.Round(m.Mixer.Volume3*100)),
		muteIndicator(m.Mixer.Mute3, m.Mixer.Volume3),
	))
	sb.WriteString(fmt.Sprintf("%s %s %+.2f\n",
		lbl("Pan3", m.selected == 5),
		m.panBar3.View(),
		m.Mixer.Pan3,
	))
	sb.WriteString("\n")
	// Output
	sb.WriteString(fmt.Sprintf("%s %s %3d%%\n",
		lbl("Master", m.selected == 6),
		m.masterBar.View(),
		int(math.Round(mv*100)),
	))
	sb.WriteString(fmt.Sprintf("%s %s\n",
		lbl("Mode", m.selected == 7),
		m.Mixer.Mode.String(),
	))
	sb.WriteString("\n")
	// Glide / portamento (no trailing newline)
	sb.WriteString(fmt.Sprintf("%s %s %4.2fs",
		lbl("Glide", m.selected == 8),
		m.portamentoBar.View(),
		m.Portamento,
	))
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
			if m.selected < mixerNumRows-1 {
				m.selected++
			}
		case "enter":
			switch m.selected {
			case 0:
				m.Mixer.Mute1 = !m.Mixer.Mute1
			case 2:
				m.Mixer.Mute2 = !m.Mixer.Mute2
			case 4:
				m.Mixer.Mute3 = !m.Mixer.Mute3
			}
		case "left", "shift+left":
			if m.selected == 7 {
				m.Mixer.Mode = (m.Mixer.Mode - 1 + audio.MixMode(audio.MixModeCount())) % audio.MixMode(audio.MixModeCount())
				break
			}
			delta := -0.01
			if msg.String() == "shift+left" {
				delta = -0.1
			}
			m.adjustSelected(delta)
		case "right", "shift+right":
			if m.selected == 7 {
				m.Mixer.Mode = (m.Mixer.Mode + 1) % audio.MixMode(audio.MixModeCount())
				break
			}
			delta := 0.01
			if msg.String() == "shift+right" {
				delta = 0.1
			}
			m.adjustSelected(delta)
		}
	}

	m.volBar1.Value = m.Mixer.Volume1
	m.panBar1.Value = m.Mixer.Pan1
	m.volBar2.Value = m.Mixer.Volume2
	m.panBar2.Value = m.Mixer.Pan2
	m.volBar3.Value = m.Mixer.Volume3
	m.panBar3.Value = m.Mixer.Pan3
	mv := m.Mixer.MasterVolume
	if mv == 0 {
		mv = 1.0
		m.Mixer.MasterVolume = mv
	}
	m.masterBar.Value = mv
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
		m.Mixer.Pan1 = clampPan(math.Round((m.Mixer.Pan1+delta)*100) / 100)
	case 2:
		m.Mixer.Volume2 = clampVolume(math.Round((m.Mixer.Volume2+delta)*100) / 100)
	case 3:
		m.Mixer.Pan2 = clampPan(math.Round((m.Mixer.Pan2+delta)*100) / 100)
	case 4:
		m.Mixer.Volume3 = clampVolume(math.Round((m.Mixer.Volume3+delta)*100) / 100)
	case 5:
		m.Mixer.Pan3 = clampPan(math.Round((m.Mixer.Pan3+delta)*100) / 100)
	case 6:
		m.Mixer.MasterVolume = clampVolume(math.Round((m.Mixer.MasterVolume+delta)*100) / 100)
	case 7:
		// Mode is cycled via left/right in Update, not here
	case 8:
		m.Portamento = clampPortamento(math.Round((m.Portamento+delta)*100) / 100)
	}
}

func clampPan(v float64) float64 {
	if v < -1 {
		return -1
	}
	if v > 1 {
		return 1
	}
	return v
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
