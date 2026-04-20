package synth

import (
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
	"github.com/tetrackt/tetrackt/ui"
	"github.com/tetrackt/tetrackt/ui/common"
)

type panelDef struct {
	Title string
	Color color.Color
	Child ui.Component
}

// SynthScreen is the synthesizer editor screen containing the five synth panels.
// It owns panel navigation state (which panel is active) that was previously
// held as InputMode in main.go.
type SynthScreen struct {
	panels        []panelDef
	ActivePanel   int // 0=Osc1, 1=Env1, 2=Osc2, 3=Env2, 4=Mixer
	patchBankView *SynthPatchBankView
}

// NewSynthScreen creates a new SynthScreen for the given synth instance.
// Panels are constructed internally from the synth's initial state.
func NewSynthScreen(synth *audio.Synth) *SynthScreen {
	panels := []panelDef{
		{Title: "Oscillator 1", Color: common.ColorAccentPrimary, Child: NewOscillatorModel(synth.Oscillator1)},
		{Title: "Envelope 1", Color: common.ColorAccentPrimary, Child: NewEnvelopeModel(synth.Envelope1)},
		{Title: "LFO 1", Color: common.ColorAccentPrimary, Child: NewLFOModel(synth.LFO1)},
		{Title: "Oscillator 2", Color: common.ColorAccentEnvelope, Child: NewOscillatorModel(synth.Oscillator2)},
		{Title: "Envelope 2", Color: common.ColorAccentEnvelope, Child: NewEnvelopeModel(synth.Envelope2)},
		{Title: "LFO 2", Color: common.ColorAccentEnvelope, Child: NewLFOModel(synth.LFO2)},
		{Title: "Oscillator 3", Color: common.ColorAccentPlay, Child: NewOscillatorModel(synth.Oscillator3)},
		{Title: "Envelope 3", Color: common.ColorAccentPlay, Child: NewEnvelopeModel(synth.Envelope3)},
		{Title: "LFO 3", Color: common.ColorAccentPlay, Child: NewLFOModel(synth.LFO3)},
		{Title: "Mixer", Color: common.ColorAccentModulation, Child: NewMixer(synth.Mixer)},
		{Title: "Filter", Color: common.ColorAccentModulation, Child: NewFilterModel(synth.Filter)},
	}
	return &SynthScreen{
		panels:        panels,
		ActivePanel:   0,
		patchBankView: NewSynthPatchBankView(),
	}
}

// PatchBankView returns the persistent patch bank view owned by this screen.
func (s *SynthScreen) PatchBankView() *SynthPatchBankView {
	return s.patchBankView
}

// SetUserPatches refreshes the patch bank view with the current user patches.
func (s *SynthScreen) SetUserPatches(patches []SynthPatch) {
	s.patchBankView.SetUserPatches(patches)
}

// Update handles keyboard navigation between panels and forwards all other key
// events to the active panel's Update method.
func (s *SynthScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+right":
			s.navigateGrid(0, 1)
			return s, nil
		case "ctrl+left":
			s.navigateGrid(0, -1)
			return s, nil
		case "ctrl+down":
			s.navigateGrid(1, 0)
			return s, nil
		case "ctrl+up":
			s.navigateGrid(-1, 0)
			return s, nil
		}

		// Forward to active panel; convert any component update into SynthUpdated.
		var cmd tea.Cmd
		s.panels[s.ActivePanel].Child, cmd = s.panels[s.ActivePanel].Child.Update(msg)
		return s, s.toSynthUpdated(cmd)

	case ui.SynthUpdated:
		s.ApplyTrackChange(ui.TrackChanged{Synth: msg.Synth})
		return s, nil

	case WavetablePickedMsg:
		osc := s.activeOscillator()
		if osc != nil {
			osc.wtBank = msg.BankIdx
			osc.wtIndex = msg.EntryIdx
			osc.Oscillator.Wavetable = msg.Entry.Data
			osc.Oscillator.Meta = audio.Metadata{Bank: msg.Entry.Bank, Name: msg.Entry.Name}
			return s, func() tea.Msg { return ui.SynthUpdated{Synth: s.GetSynth()} }
		}
		return s, nil
	}

	return s, nil
}

// navigateGrid moves the active panel by (dRow, dCol) in the 2-D panel layout:
//
//	col: 0=Osc  1=Env  2=LFO  3=right-column
//	row: 0=Voice1  1=Voice2  2=Voice3
//
// The right column has Mixer at row 0 and Filter at row 1; row 2 is empty.
func (s *SynthScreen) navigateGrid(dRow, dCol int) {
	grid := [3][4]int{
		{0, 1, 2, 9},
		{3, 4, 5, 10},
		{6, 7, 8, -1},
	}

	curRow, curCol := 0, 0
outer:
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c] == s.ActivePanel {
				curRow, curCol = r, c
				break outer
			}
		}
	}

	newRow := max(0, min(2, curRow+dRow))
	newCol := max(0, min(3, curCol+dCol))
	if target := grid[newRow][newCol]; target != -1 {
		s.ActivePanel = target
	}
}

// toSynthUpdated wraps a panel command, converting any synth component update
// message (OscillatorUpdated, EnvelopeUpdated, etc.) into a SynthUpdated that
// carries the full current synth state.
func (s *SynthScreen) toSynthUpdated(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		inner := cmd()
		switch inner.(type) {
		case OscillatorUpdated, EnvelopeUpdated, MixerUpdated, FilterUpdated, LFOUpdated:
			return ui.SynthUpdated{Synth: s.GetSynth()}
		}
		return inner
	}
}

// Accessors for synth component children, used by main for audio playback and
// cross-screen state synchronisation.
// Panel order: 0=Osc1, 1=Env1, 2=LFO1, 3=Osc2, 4=Env2, 5=LFO2, 6=Osc3, 7=Env3, 8=LFO3, 9=Mixer, 10=Filter
func (s *SynthScreen) Osc1() *OscillatorModel  { return s.panels[0].Child.(*OscillatorModel) }
func (s *SynthScreen) Env1() *EnvelopeModel    { return s.panels[1].Child.(*EnvelopeModel) }
func (s *SynthScreen) LFO1() *LFOModel         { return s.panels[2].Child.(*LFOModel) }
func (s *SynthScreen) Osc2() *OscillatorModel  { return s.panels[3].Child.(*OscillatorModel) }
func (s *SynthScreen) Env2() *EnvelopeModel    { return s.panels[4].Child.(*EnvelopeModel) }
func (s *SynthScreen) LFO2() *LFOModel         { return s.panels[5].Child.(*LFOModel) }
func (s *SynthScreen) Osc3() *OscillatorModel  { return s.panels[6].Child.(*OscillatorModel) }
func (s *SynthScreen) Env3() *EnvelopeModel    { return s.panels[7].Child.(*EnvelopeModel) }
func (s *SynthScreen) LFO3() *LFOModel         { return s.panels[8].Child.(*LFOModel) }
func (s *SynthScreen) GetMixer() *Mixer        { return s.panels[9].Child.(*Mixer) }
func (s *SynthScreen) GetFilter() *FilterModel { return s.panels[10].Child.(*FilterModel) }

// ApplyTrackChange updates all panels to reflect the settings of the newly
// selected track.
func (s *SynthScreen) ApplyTrackChange(msg ui.TrackChanged) {
	synth := msg.Synth
	s.Osc1().Oscillator = normalizeOscillator(synth.Oscillator1)
	s.Env1().Envelope = synth.Envelope1
	s.Osc2().Oscillator = normalizeOscillator(synth.Oscillator2)
	s.Env2().Envelope = synth.Envelope2
	s.Osc3().Oscillator = normalizeOscillator(synth.Oscillator3)
	s.Env3().Envelope = synth.Envelope3
	s.LFO3().LFO = synth.LFO3
	s.GetMixer().SetMixer(synth.Mixer)
	s.GetFilter().Filter = synth.Filter
	s.GetFilter().FilterEnvelope = synth.FilterEnvelope
	s.GetFilter().SyncBars()
	s.LFO1().LFO = synth.LFO1
	s.LFO2().LFO = synth.LFO2
}

// Title returns the tab label for the SynthScreen.
func (s *SynthScreen) Title() string { return "Synth" }

// View renders the synth panels grouped by voice:
//
//	Voice 1: Osc1  Env1  LFO1  │ Mixer
//	Voice 2: Osc2  Env2  LFO2  │ Filter
//	Voice 3: Osc3  Env3  LFO3  │
func (s *SynthScreen) View() string {
	render := func(idx int) string {
		p := s.panels[idx]
		return ui.RenderPanel(p.Title, p.Color, p.Child.View(), idx == s.ActivePanel)
	}

	voice1 := s.renderVoice(common.ColorAccentPrimary, 0, 1, 2)
	voice2 := s.renderVoice(common.ColorAccentEnvelope, 3, 4, 5)
	voice3 := s.renderVoice(common.ColorAccentPlay, 6, 7, 8)
	left := lipgloss.JoinVertical(lipgloss.Left, voice1, "", voice2, "", voice3)

	right := lipgloss.JoinVertical(lipgloss.Left, render(9), render(10))

	spacer := "  "
	return lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
}

// renderVoice renders a voice group with a colored horizontal rule above
// and a left-edge ▌ strip in the voice color beside the panels.
func (s *SynthScreen) renderVoice(clr color.Color, panelIndices ...int) string {
	// Render all panels in the row with equalised content heights.
	contents := make([]string, len(panelIndices))
	maxH := 0
	for i, idx := range panelIndices {
		contents[i] = s.panels[idx].Child.View()
		if h := lipgloss.Height(contents[i]); h > maxH {
			maxH = h
		}
	}
	parts := make([]string, len(panelIndices))
	for i, idx := range panelIndices {
		p := s.panels[idx]
		content := lipgloss.NewStyle().Height(maxH).Render(contents[i])
		parts[i] = ui.RenderPanel(p.Title, p.Color, content, idx == s.ActivePanel)
	}
	panels := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	// Left-edge ▌ strip, one character wide, same height as the panels.
	h := lipgloss.Height(panels)
	stripLines := make([]string, h)
	for i := range stripLines {
		stripLines[i] = "▌"
	}
	strip := lipgloss.NewStyle().Foreground(clr).Render(strings.Join(stripLines, "\n"))
	withStrip := lipgloss.JoinHorizontal(lipgloss.Top, strip, panels)

	// Horizontal rule above, spanning the full row width.
	w := lipgloss.Width(withStrip)
	rule := lipgloss.NewStyle().Foreground(clr).Render(strings.Repeat("─", w))

	return lipgloss.JoinVertical(lipgloss.Left, rule, withStrip)
}

// Help returns screen-specific keyboard shortcut sections for the help dialog.
func (s *SynthScreen) Help() []ui.HelpSection {
	return []ui.HelpSection{
		{
			Title: "Synth",
			Entries: []ui.HelpEntry{
				{Key: "↑↓", Desc: "Select parameter"},
				{Key: "←→", Desc: "Adjust parameter value"},
				{Key: "Shift+←→", Desc: "Large adjustment"},
				{Key: "", Desc: ""},
				{Key: "b / B", Desc: "Open patch bank"},
				{Key: "Z-M / Q-U + black keys", Desc: "Preview note"},
			},
		},
	}
}

// activeOscillator returns the active panel as an *OscillatorModel if it is one,
// otherwise nil.
func (s *SynthScreen) activeOscillator() *OscillatorModel {
	child := s.panels[s.ActivePanel].Child
	osc, ok := child.(*OscillatorModel)
	if !ok {
		return nil
	}
	return osc
}

// GetSynth builds and returns an audio.Synth from the current panel state.
func (s *SynthScreen) GetSynth() *audio.Synth {
	synth := audio.NewSynth(s.Osc1().Oscillator, s.Env1().Envelope, s.Osc2().Oscillator, s.Env2().Envelope, s.Osc3().Oscillator, s.Env3().Envelope, s.GetMixer().Mixer, s.GetFilter().Filter, s.LFO1().LFO, s.LFO2().LFO, s.LFO3().LFO)
	synth.FilterEnvelope = s.GetFilter().FilterEnvelope
	return synth
}
