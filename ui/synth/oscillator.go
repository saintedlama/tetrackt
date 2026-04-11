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

type editField int

const (
	oscillatorType editField = iota
	oscillatorPhase
	oscillatorPulseWidth
	oscillatorDetune
	oscillatorWavetable
	oscillatorNoisePeriod
	oscillatorFieldCount
)

type wavetablePreset struct {
	name string
	data []float64
}

var builtinWavetables = []wavetablePreset{
	{"SoftSaw", audio.WavetableSoftSaw},
	{"SoftSquare", audio.WavetableSoftSquare},
	{"Organ", audio.WavetableOrgan},
	{"Glass", audio.WavetableGlass},
	{"Bass", audio.WavetableBass},
	{"Strings", audio.WavetableStrings},
	{"Flute", audio.WavetableFlute},
	{"Brass", audio.WavetableBrass},
	{"Chime", audio.WavetableChime},
	{"Voice", audio.WavetableVoice},
}

type OscillatorModel struct {
	Oscillator          audio.Oscillator
	oscillatorList      []audio.OscillatorType
	oscillatorTypeStyle lipgloss.Style
	editField           editField
	wavetableIdx        int // index into builtinWavetables
}

type OscillatorUpdated struct {
	Oscillator audio.Oscillator
}

func NewOscillatorModel(oscillator audio.Oscillator) *OscillatorModel {
	oscillatorList := []audio.OscillatorType{audio.Sine, audio.Square, audio.Triangle, audio.Sawtooth, audio.SawtoothReverse, audio.Noise, audio.NoisePeriodic, audio.Wavetable, audio.Silent}

	oscillatorTypeStyle := lipgloss.NewStyle().Width(calcOscWidth(oscillatorList))

	// match wavetableIdx to current Oscillator.Wavetable if possible
	wtIdx := 0
	for i, p := range builtinWavetables {
		if len(oscillator.Wavetable) == len(p.data) {
			wtIdx = i
			break
		}
	}

	return &OscillatorModel{
		Oscillator:          oscillator,
		oscillatorList:      oscillatorList,
		oscillatorTypeStyle: oscillatorTypeStyle,
		wavetableIdx:        wtIdx,
	}
}

func (m *OscillatorModel) Init() tea.Cmd {
	return nil
}

func (m *OscillatorModel) View() string {
	var oscillatorView strings.Builder
	oscType := renderFieldSelected(string(m.Oscillator.Type), m.editField == oscillatorType)
	oscillatorView.WriteString(m.oscillatorTypeStyle.Render(oscType))

	oscillatorView.WriteString("\n")
	oscillatorView.WriteString(renderFieldSelected(common.RenderKnob("Phase", m.Oscillator.Phase), m.editField == oscillatorPhase))

	pw := m.Oscillator.PulseWidth
	if pw == 0 {
		pw = 0.5
	}
	oscillatorView.WriteString("\n")
	oscillatorView.WriteString(renderFieldSelected(fmt.Sprintf("PW:   %3d%%", int(pw*100)), m.editField == oscillatorPulseWidth))

	oscillatorView.WriteString("\n")
	oscillatorView.WriteString(renderFieldSelected(fmt.Sprintf("Dtune:%+5.0fc", m.Oscillator.Detune), m.editField == oscillatorDetune))

	if m.Oscillator.Type == audio.Wavetable {
		wt := builtinWavetables[m.wavetableIdx]
		oscillatorView.WriteString("\n")
		oscillatorView.WriteString(renderFieldSelected(fmt.Sprintf("Wave: %-10s", wt.name), m.editField == oscillatorWavetable))
	}

	if m.Oscillator.Type == audio.NoisePeriodic {
		periodStr := "Auto"
		if m.Oscillator.NoisePeriod > 0 {
			periodStr = fmt.Sprintf("%4d", m.Oscillator.NoisePeriod)
		}
		oscillatorView.WriteString("\n")
		oscillatorView.WriteString(renderFieldSelected(fmt.Sprintf("Period:%-6s", periodStr), m.editField == oscillatorNoisePeriod))
	}

	return oscillatorView.String()
}

func (m *OscillatorModel) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			// Move to previous oscillator field; skip oscillatorWavetable unless type is Wavetable
			m.editField = m.prevField()
		case "down":
			// Move to next oscillator field; skip oscillatorWavetable unless type is Wavetable
			m.editField = m.nextField()
		case "left":
			switch m.editField {
			case oscillatorType:
				m.Oscillator.Type = cycle(m.oscillatorList, m.Oscillator.Type, -1)
				if m.Oscillator.Type == audio.Wavetable {
					m.Oscillator.Wavetable = builtinWavetables[m.wavetableIdx].data
					m.Oscillator.NoisePeriod = 0
				} else {
					m.Oscillator.Wavetable = nil
				}
				if m.Oscillator.Type != audio.NoisePeriodic {
					m.Oscillator.NoisePeriod = 0
				}
			case oscillatorPhase:
				// decrease phase
				m.Oscillator.Phase -= 0.05
				if m.Oscillator.Phase < 0.0 {
					m.Oscillator.Phase = 0.0
				}
			case oscillatorPulseWidth:
				pw := m.Oscillator.PulseWidth
				if pw == 0 {
					pw = 0.5
				}
				m.Oscillator.PulseWidth = max(0.01, pw-0.05)
			case oscillatorDetune:
				m.Oscillator.Detune = clampDetune(m.Oscillator.Detune - 1)
			case oscillatorWavetable:
				m.wavetableIdx = (m.wavetableIdx - 1 + len(builtinWavetables)) % len(builtinWavetables)
				m.Oscillator.Wavetable = builtinWavetables[m.wavetableIdx].data
			case oscillatorNoisePeriod:
				if m.Oscillator.NoisePeriod > 0 {
					m.Oscillator.NoisePeriod--
				}
			}
			cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
		case "shift+left":
			switch m.editField {
			case oscillatorDetune:
				m.Oscillator.Detune = clampDetune(m.Oscillator.Detune - 10)
				cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
			case oscillatorNoisePeriod:
				if m.Oscillator.NoisePeriod >= 10 {
					m.Oscillator.NoisePeriod -= 10
				} else {
					m.Oscillator.NoisePeriod = 0
				}
				cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
			}
		case "right":
			switch m.editField {
			case oscillatorType:
				m.Oscillator.Type = cycle(m.oscillatorList, m.Oscillator.Type, 1)
				if m.Oscillator.Type == audio.Wavetable {
					m.Oscillator.Wavetable = builtinWavetables[m.wavetableIdx].data
					m.Oscillator.NoisePeriod = 0
				} else {
					m.Oscillator.Wavetable = nil
				}
				if m.Oscillator.Type != audio.NoisePeriodic {
					m.Oscillator.NoisePeriod = 0
				}
			case oscillatorPhase:
				// decrease phase
				m.Oscillator.Phase += 0.05
				if m.Oscillator.Phase > 1.0 {
					m.Oscillator.Phase = 1.0
				}
			case oscillatorPulseWidth:
				pw := m.Oscillator.PulseWidth
				if pw == 0 {
					pw = 0.5
				}
				m.Oscillator.PulseWidth = min(0.99, pw+0.05)
			case oscillatorDetune:
				m.Oscillator.Detune = clampDetune(m.Oscillator.Detune + 1)
			case oscillatorWavetable:
				m.wavetableIdx = (m.wavetableIdx + 1) % len(builtinWavetables)
				m.Oscillator.Wavetable = builtinWavetables[m.wavetableIdx].data
			case oscillatorNoisePeriod:
				if m.Oscillator.NoisePeriod < 2048 {
					m.Oscillator.NoisePeriod++
				}
			}
			cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
		case "shift+right":
			switch m.editField {
			case oscillatorDetune:
				m.Oscillator.Detune = clampDetune(m.Oscillator.Detune + 10)
				cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
			case oscillatorNoisePeriod:
				m.Oscillator.NoisePeriod = min(2048, m.Oscillator.NoisePeriod+10)
				cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
			}
		}
	}

	return m, cmd
}

func renderFieldSelected(content string, selected bool) string {
	if selected {
		return common.StyleSelected.Render(content)
	}

	return content
}

func cycle(oscillatorList []audio.OscillatorType, current audio.OscillatorType, step int) audio.OscillatorType {
	nextIdx := slices.Index(oscillatorList, current) + step

	if nextIdx < 0 {
		return oscillatorList[len(oscillatorList)-1]
	}

	return oscillatorList[nextIdx%len(oscillatorList)]
}

func calcOscWidth(oscillatorList []audio.OscillatorType) int {
	oscWidth := 0
	for _, osc := range oscillatorList {
		oscWidth = max(oscWidth, len(osc))
	}
	return oscWidth
}

func clampDetune(v float64) float64 {
	if v < -1200 {
		return -1200
	}
	if v > 1200 {
		return 1200
	}
	return v
}

func (m *OscillatorModel) nextField() editField {
	f := (m.editField + 1) % oscillatorFieldCount
	if f == oscillatorWavetable && m.Oscillator.Type != audio.Wavetable {
		f = (f + 1) % oscillatorFieldCount
	}
	if f == oscillatorNoisePeriod && m.Oscillator.Type != audio.NoisePeriodic {
		f = (f + 1) % oscillatorFieldCount
	}
	return f
}

func (m *OscillatorModel) prevField() editField {
	f := (m.editField - 1 + oscillatorFieldCount) % oscillatorFieldCount
	if f == oscillatorWavetable && m.Oscillator.Type != audio.Wavetable {
		f = (f - 1 + oscillatorFieldCount) % oscillatorFieldCount
	}
	if f == oscillatorNoisePeriod && m.Oscillator.Type != audio.NoisePeriodic {
		f = (f - 1 + oscillatorFieldCount) % oscillatorFieldCount
	}
	return f
}
