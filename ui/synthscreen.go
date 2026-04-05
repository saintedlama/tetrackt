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

func (s *SynthScreen) Init() tea.Cmd { return nil }

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
func (s *SynthScreen) Osc1() *OscillatorModel { return s.panels[0].Child.(*OscillatorModel) }
func (s *SynthScreen) Env1() *EnvelopeModel   { return s.panels[1].Child.(*EnvelopeModel) }
func (s *SynthScreen) Osc2() *OscillatorModel { return s.panels[2].Child.(*OscillatorModel) }
func (s *SynthScreen) Env2() *EnvelopeModel   { return s.panels[3].Child.(*EnvelopeModel) }
func (s *SynthScreen) GetMixer() *Mixer       { return s.panels[4].Child.(*Mixer) }

// ApplyTrackChange updates all panels to reflect the settings of the newly
// selected track.
func (s *SynthScreen) ApplyTrackChange(msg TrackChanged) {
	s.Osc1().Oscillator = msg.Oscillator1
	s.Env1().Envelope = msg.Envelope1
	s.Osc2().Oscillator = msg.Oscillator2
	s.Env2().Envelope = msg.Envelope2
	s.GetMixer().SetMixer(msg.Mixer)
}

// Title returns the tab label for the SynthScreen.
func (s *SynthScreen) Title() string { return "Synth" }

// ModeLabel returns the name of the currently active panel for the header bar.
func (s *SynthScreen) ModeLabel() string {
	labels := []string{"OSCILLATOR1", "ENVELOPE1", "OSCILLATOR2", "ENVELOPE2", "MIXER"}
	return labels[s.ActivePanel]
}

// View renders all five synth panels horizontally, taking the full available
// terminal height.
func (s *SynthScreen) View() string {
	panelViews := make([]string, len(s.panels))
	for i, p := range s.panels {
		panelViews[i] = RenderPanel(p.Title, p.Color, p.Child.View(), i == s.ActivePanel)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, panelViews...)
}

// Footer returns the help text shown in the footer bar on the Synth screen.
func (s *SynthScreen) Footer() string {
	return "Tab/Shift+Tab: Switch panel | ↑↓: Select | ←→: Adjust | +/-: Octave | I: Instruments | p: Play/Pause | P: Loop | S: Save | L: Load | T: Tracker | Q: Quit"
}

// GetActiveSynthParams returns the oscillator/envelope/mixer settings for audio playback.
func (s *SynthScreen) GetActiveSynthParams() (audio.Oscillator, audio.Envelope, audio.Oscillator, audio.Envelope, audio.Mixer) {
	return s.Osc1().Oscillator, s.Env1().Envelope, s.Osc2().Oscillator, s.Env2().Envelope, s.GetMixer().Mixer
}
