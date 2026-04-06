package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/tetrackt/tetrackt/audio"
)

// SynthScreen is the synthesizer editor screen containing the five synth panels.
// It owns panel navigation state (which panel is active) that was previously
// held as InputMode in main.go.
type SynthScreen struct {
	panels      []Panel
	ActivePanel int // 0=Osc1, 1=Env1, 2=Osc2, 3=Env2, 4=Mixer
}

// NewSynthScreen creates a new SynthScreen wrapping the given panels.
func NewSynthScreen(panels []Panel) *SynthScreen {
	return &SynthScreen{
		panels:      panels,
		ActivePanel: 0,
	}
}

// Update handles keyboard navigation between panels and forwards all other key
// events to the active panel's Update method.
func (s *SynthScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			s.ActivePanel = (s.ActivePanel + 1) % len(s.panels)
			return s, nil
		case "shift+tab":
			s.ActivePanel = (s.ActivePanel - 1 + len(s.panels)) % len(s.panels)
			return s, nil
		}

		// Forward to active panel
		var cmd tea.Cmd
		s.panels[s.ActivePanel], cmd = s.panels[s.ActivePanel].Update(msg)
		return s, cmd
	}

	return s, nil
}

// Accessors for synth component children, used by main for audio playback and
// cross-screen state synchronisation.
func (s *SynthScreen) Osc1() *OscillatorModel  { return s.panels[0].Child.(*OscillatorModel) }
func (s *SynthScreen) Env1() *EnvelopeModel    { return s.panels[1].Child.(*EnvelopeModel) }
func (s *SynthScreen) Osc2() *OscillatorModel  { return s.panels[2].Child.(*OscillatorModel) }
func (s *SynthScreen) Env2() *EnvelopeModel    { return s.panels[3].Child.(*EnvelopeModel) }
func (s *SynthScreen) GetMixer() *Mixer        { return s.panels[4].Child.(*Mixer) }
func (s *SynthScreen) GetFilter() *FilterModel { return s.panels[5].Child.(*FilterModel) }
func (s *SynthScreen) LFO1() *LFOModel         { return s.panels[6].Child.(*LFOModel) }
func (s *SynthScreen) LFO2() *LFOModel         { return s.panels[7].Child.(*LFOModel) }

// ApplyTrackChange updates all panels to reflect the settings of the newly
// selected track.
func (s *SynthScreen) ApplyTrackChange(msg TrackChanged) {
	s.Osc1().Oscillator = msg.Oscillator1
	s.Env1().Envelope = msg.Envelope1
	s.Osc2().Oscillator = msg.Oscillator2
	s.Env2().Envelope = msg.Envelope2
	s.GetMixer().SetMixer(msg.Mixer)
	s.GetFilter().Filter = msg.Filter
	s.GetFilter().SyncBars()
	s.LFO1().LFO = msg.LFO1
	s.LFO1().Dest = msg.LFO1Dest
	s.LFO2().LFO = msg.LFO2
	s.LFO2().Dest = msg.LFO2Dest
}

// Title returns the tab label for the SynthScreen.
func (s *SynthScreen) Title() string { return "Synth" }

// View renders the synth panels in a 2x2 + right-column layout:
//
//	Row 1: Osc1  Env1  │ Mixer  │ LFO1
//	Row 2: Osc2  Env2  │ Filter │ LFO2
func (s *SynthScreen) View() string {
	render := func(idx int) string {
		p := s.panels[idx]
		return RenderPanel(p.Title, p.Color, p.Child.View(), idx == s.ActivePanel)
	}

	row1 := lipgloss.JoinHorizontal(lipgloss.Top, render(0), render(1))
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, render(2), render(3))
	left := lipgloss.JoinVertical(lipgloss.Left, row1, row2)

	mid := lipgloss.JoinVertical(lipgloss.Left, render(4), render(5))
	right := lipgloss.JoinVertical(lipgloss.Left, render(6), render(7))

	return lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
}

// Footer returns the help text shown in the footer bar on the Synth screen.
func (s *SynthScreen) Footer() string {
	return "Tab/Shift+Tab: Switch panel | ↑↓: Select | ←→: Adjust | +/-: Octave | I: Instruments | p: Play/Pause | P: Loop | S: Save | L: Load | T: Tracker | Q: Quit"
}

// GetActiveSynthParams returns the oscillator/envelope/mixer/filter settings for audio playback.
func (s *SynthScreen) GetActiveSynthParams() (audio.Oscillator, audio.Envelope, audio.Oscillator, audio.Envelope, audio.Mixer, audio.Filter) {
	return s.Osc1().Oscillator, s.Env1().Envelope, s.Osc2().Oscillator, s.Env2().Envelope, s.GetMixer().Mixer, s.GetFilter().Filter
}

// GetActiveLFOs returns the current LFO settings for both voices.
func (s *SynthScreen) GetActiveLFOs() (audio.LFO, audio.ModDest, audio.LFO, audio.ModDest) {
	return s.LFO1().LFO, s.LFO1().Dest, s.LFO2().LFO, s.LFO2().Dest
}
