package common

import "charm.land/lipgloss/v2"

var (
	StyleSelected = lipgloss.NewStyle().
			Background(ColorGrayDark).
			Foreground(ColorAccentPrimary)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorTextDisabled).
			Padding(1, 1)

	StyleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBackground).
			Background(ColorAccentPrimary).
			Padding(0, 2)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(ColorTextMuted).
				Background(ColorSurface).
				Padding(0, 2)
)

var (
	ColorGrayDarkest = lipgloss.Color("#1c1c1e") // Main application background
	ColorGrayDark    = lipgloss.Color("#242428") // Panel / section backgrounds
	ColorGrayMedium  = lipgloss.Color("#3c3c3c") // Knob bodies, inactive separators
	ColorGray        = lipgloss.Color("#5a5a5a") // Inactive / disabled element fill
	ColorGrayLight   = lipgloss.Color("#8a8a8a") // Secondary labels, inactive track lanes
	ColorGrayLighter = lipgloss.Color("#c8c8c8") // Primary labels, parameter names
)

var (
	ColorCyan   = lipgloss.Color("#00d4e8") // Primary accent — active elements, waveform displays
	ColorOrange = lipgloss.Color("#ff7700") // Envelope curves, active knob rings
	ColorPurple = lipgloss.Color("#8b2fc9") // Modulation / Random section indicators
	ColorGreen  = lipgloss.Color("#00c853") // Function generator, positive modulation
	ColorPink   = lipgloss.Color("#e81e8c") // Keyboard / velocity lane, macro rings
	ColorYellow = lipgloss.Color("#ffb300") // LFO waveform fill, sequencer step accents
	ColorWhite  = lipgloss.Color("#ffffff")
)

var (
	ColorBackground        = ColorGrayDarkest // Root terminal background
	ColorSurface           = ColorGrayDark    // Panel / bordered section background
	ColorBorder            = ColorGrayMedium  // Inactive panel border
	ColorBorderActive      = ColorCyan        // Active / focused panel border
	ColorText              = ColorGrayLighter // Default readable text
	ColorTextMuted         = ColorGrayLight   // Labels, secondary info
	ColorTextDisabled      = ColorGray        // Disabled / empty cells
	ColorAccentPrimary     = ColorCyan        // Selected rows, cursor cells, active modes
	ColorAccentEnvelope    = ColorOrange      // Envelope editor highlights
	ColorAccentOscillator  = ColorCyan        // Oscillator waveform type indicator
	ColorAccentModulation  = ColorPurple      // Modulation / LFO indicators
	ColorAccentPlay        = ColorGreen       // Playback row highlight
	ColorAccentSynthPreset = ColorPink        // Synth preset selection highlight
	ColorAccentWarning     = ColorYellow      // Errors, warnings, unsaved-changes indicator
)
