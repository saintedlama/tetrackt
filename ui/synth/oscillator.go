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
	oscillatorWavetableCategory
	oscillatorWavetableEntry
	oscillatorNoisePeriod
	oscillatorFieldCount
)

type OscillatorModel struct {
	Oscillator          audio.Oscillator
	oscillatorList      []audio.OscillatorType
	oscillatorTypeStyle lipgloss.Style
	editField           editField
	wtBank              int // index into WavetableBanksInOrder()
	wtIndex             int // index within the category
}

type OscillatorUpdated struct {
	Oscillator audio.Oscillator
}

func normalizeOscillator(o audio.Oscillator) audio.Oscillator {
	if o.Type == "" {
		o.Type = audio.Silent
	}
	return o
}

func NewOscillatorModel(oscillator audio.Oscillator) *OscillatorModel {
	oscillatorList := []audio.OscillatorType{audio.Sine, audio.Square, audio.Triangle, audio.Sawtooth, audio.SawtoothReverse, audio.Noise, audio.NoisePeriodic, audio.Wavetable, audio.Silent}

	oscillator = normalizeOscillator(oscillator)

	oscillatorTypeStyle := lipgloss.NewStyle().Width(calcOscWidth(oscillatorList))

	// Match wtBank / wtIndex to the oscillator's current wavetable via Meta.
	bankIdx, wIdx := resolveWavetablePosition(oscillator.Meta.Bank, oscillator.Meta.Name)

	return &OscillatorModel{
		Oscillator:          oscillator,
		oscillatorList:      oscillatorList,
		oscillatorTypeStyle: oscillatorTypeStyle,
		wtBank:              bankIdx,
		wtIndex:             wIdx,
	}
}

// resolveWavetablePosition finds the bank and within-bank index for
// the given wavetable bank and name. Returns (0, 0) if not found.
func resolveWavetablePosition(bank, name string) (bankIdx, wIdx int) {
	if bank == "" && name == "" {
		return 0, 0
	}
	banks := WavetableBanksInOrder()
	for bi, b := range banks {
		if b != bank {
			continue
		}
		entries := WavetableEntriesForBank(b)
		for ei, e := range entries {
			if e.Name == name {
				return bi, ei
			}
		}
	}
	return 0, 0
}

// currentEntry returns the WavetableEntry at (wtBank, wtIndex).
func (m *OscillatorModel) currentEntry() WavetableEntry {
	banks := WavetableBanksInOrder()
	if len(banks) == 0 {
		return WavetableEntry{}
	}
	bi := m.wtBank % len(banks)
	entries := WavetableEntriesForBank(banks[bi])
	if len(entries) == 0 {
		return WavetableEntry{}
	}
	wi := m.wtIndex % len(entries)
	return entries[wi]
}

// applyCurrentWavetable copies the current entry's data and metadata into the Oscillator.
func (m *OscillatorModel) applyCurrentWavetable() {
	e := m.currentEntry()
	m.Oscillator.Wavetable = e.Data
	m.Oscillator.Meta = audio.Metadata{Bank: e.Bank, Name: e.Name}
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

	switch m.Oscillator.Type {
	case audio.Wavetable:
		banks := WavetableBanksInOrder()
		bankCount := len(banks)
		var bankName, wavName string
		var wIdx, wCount int
		if bankCount > 0 {
			bi := m.wtBank % bankCount
			bankName = banks[bi]
			entries := WavetableEntriesForBank(bankName)
			wCount = len(entries)
			if wCount > 0 {
				wIdx = (m.wtIndex % wCount) + 1
				wavName = entries[wIdx-1].Name
			}
		}
		displayBank := bankName
		displayWav := strings.TrimPrefix(wavName, "AKWF_")
		const maxBank, maxWav = 16, 16
		if len(displayBank) > maxBank {
			displayBank = displayBank[:maxBank]
		}
		if len(displayWav) > maxWav {
			displayWav = displayWav[:maxWav]
		}
		oscillatorView.WriteString("\n")
		oscillatorView.WriteString(renderFieldSelected(fmt.Sprintf("%-16s", displayBank), m.editField == oscillatorWavetableCategory))
		oscillatorView.WriteString("\n")
		oscillatorView.WriteString(renderFieldSelected(fmt.Sprintf("%-16s", displayWav), m.editField == oscillatorWavetableEntry))
	case audio.NoisePeriodic:
		periodStr := "Auto"
		if m.Oscillator.NoisePeriod > 0 {
			periodStr = fmt.Sprintf("%4d", m.Oscillator.NoisePeriod)
		}
		oscillatorView.WriteString("\n")
		oscillatorView.WriteString(renderFieldSelected(fmt.Sprintf("Period:%-6s", periodStr), m.editField == oscillatorNoisePeriod))
		oscillatorView.WriteString("\n")
	default:
		// Two blank lines keep all oscillator panels the same height.
		oscillatorView.WriteString("\n")
		oscillatorView.WriteString("\n")
	}

	return oscillatorView.String()
}

func (m *OscillatorModel) Update(msg tea.Msg) (ui.Component, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up":
			m.editField = m.prevField()
		case "down":
			m.editField = m.nextField()
		case "enter":
			if m.editField == oscillatorWavetableCategory || m.editField == oscillatorWavetableEntry {
				cmd = func() tea.Msg { return OpenWavetableDialogMsg{BankIdx: m.wtBank, EntryIdx: m.wtIndex} }
			}
		case "left":
			switch m.editField {
			case oscillatorType:
				m.Oscillator.Type = cycle(m.oscillatorList, m.Oscillator.Type, -1)
				if m.Oscillator.Type == audio.Wavetable {
					m.applyCurrentWavetable()
					m.Oscillator.NoisePeriod = 0
				} else {
					m.Oscillator.Wavetable = nil
					m.Oscillator.Meta = audio.Metadata{}
				}
				if m.Oscillator.Type != audio.NoisePeriodic {
					m.Oscillator.NoisePeriod = 0
				}
			case oscillatorPhase:
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
			case oscillatorWavetableCategory:
				banks := WavetableBanksInOrder()
				if len(banks) > 0 {
					m.wtBank = (m.wtBank - 1 + len(banks)) % len(banks)
					m.wtIndex = 0
					m.applyCurrentWavetable()
				}
			case oscillatorWavetableEntry:
				banks := WavetableBanksInOrder()
				if len(banks) > 0 {
					bi := m.wtBank % len(banks)
					entries := WavetableEntriesForBank(banks[bi])
					if len(entries) > 0 {
						m.wtIndex = (m.wtIndex - 1 + len(entries)) % len(entries)
						m.applyCurrentWavetable()
					}
				}
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
			case oscillatorWavetableEntry:
				banks := WavetableBanksInOrder()
				if len(banks) > 0 {
					bi := m.wtBank % len(banks)
					entries := WavetableEntriesForBank(banks[bi])
					if len(entries) > 0 {
						m.wtIndex = max(0, m.wtIndex-10)
						m.applyCurrentWavetable()
						cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
					}
				}
			}
		case "right":
			switch m.editField {
			case oscillatorType:
				m.Oscillator.Type = cycle(m.oscillatorList, m.Oscillator.Type, 1)
				if m.Oscillator.Type == audio.Wavetable {
					m.applyCurrentWavetable()
					m.Oscillator.NoisePeriod = 0
				} else {
					m.Oscillator.Wavetable = nil
					m.Oscillator.Meta = audio.Metadata{}
				}
				if m.Oscillator.Type != audio.NoisePeriodic {
					m.Oscillator.NoisePeriod = 0
				}
			case oscillatorPhase:
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
			case oscillatorWavetableCategory:
				banks := WavetableBanksInOrder()
				if len(banks) > 0 {
					m.wtBank = (m.wtBank + 1) % len(banks)
					m.wtIndex = 0
					m.applyCurrentWavetable()
				}
			case oscillatorWavetableEntry:
				banks := WavetableBanksInOrder()
				if len(banks) > 0 {
					bi := m.wtBank % len(banks)
					entries := WavetableEntriesForBank(banks[bi])
					if len(entries) > 0 {
						m.wtIndex = (m.wtIndex + 1) % len(entries)
						m.applyCurrentWavetable()
					}
				}
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
			case oscillatorWavetableEntry:
				banks := WavetableBanksInOrder()
				if len(banks) > 0 {
					bi := m.wtBank % len(banks)
					entries := WavetableEntriesForBank(banks[bi])
					if len(entries) > 0 {
						m.wtIndex = min(len(entries)-1, m.wtIndex+10)
						m.applyCurrentWavetable()
						cmd = func() tea.Msg { return OscillatorUpdated{Oscillator: m.Oscillator} }
					}
				}
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
	if f == oscillatorWavetableCategory && m.Oscillator.Type != audio.Wavetable {
		f = (f + 2) % oscillatorFieldCount
	}
	if f == oscillatorWavetableEntry && m.Oscillator.Type != audio.Wavetable {
		f = (f + 1) % oscillatorFieldCount
	}
	if f == oscillatorNoisePeriod && m.Oscillator.Type != audio.NoisePeriodic {
		f = (f + 1) % oscillatorFieldCount
	}
	return f
}

func (m *OscillatorModel) prevField() editField {
	f := (m.editField - 1 + oscillatorFieldCount) % oscillatorFieldCount
	if f == oscillatorWavetableEntry && m.Oscillator.Type != audio.Wavetable {
		f = (f - 2 + oscillatorFieldCount) % oscillatorFieldCount
	}
	if f == oscillatorWavetableCategory && m.Oscillator.Type != audio.Wavetable {
		f = (f - 1 + oscillatorFieldCount) % oscillatorFieldCount
	}
	if f == oscillatorNoisePeriod && m.Oscillator.Type != audio.NoisePeriodic {
		f = (f - 1 + oscillatorFieldCount) % oscillatorFieldCount
	}
	return f
}
