package ui

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Panel is a layout container that wraps a Component inside a titled, bordered panel.
type Panel struct {
	Title  string
	Color  color.Color
	Child  Component
	active bool
}

// NewPanel creates a Panel wrapping the given child component.
func NewPanel(title string, clr color.Color, child Component) Panel {
	return Panel{Title: title, Color: clr, Child: child}
}

func (p Panel) Init() tea.Cmd {
	return p.Child.Init()
}

// Update forwards the message to the child component and returns the updated Panel.
func (p Panel) Update(msg tea.Msg) (Panel, tea.Cmd) {
	child, cmd := p.Child.Update(msg)
	p.Child = child
	return p, cmd
}

func (p Panel) View() string {
	return RenderPanel(p.Title, p.Color, p.Child.View(), p.active)
}

// Focus marks the panel as active (highlighted border).
func (p Panel) Focus() Panel {
	p.active = true
	return p
}

// Blur marks the panel as inactive.
func (p Panel) Blur() Panel {
	p.active = false
	return p
}

// SetActive sets the active state and returns the updated Panel.
func (p Panel) SetActive(active bool) Panel {
	p.active = active
	return p
}

// RenderPanel renders content inside a rounded border with an optional title
// embedded in the top border line using the lipgloss v2 compositor.
// titleColor is the foreground color for the title text.
// active controls whether the active or inactive border color is used.
func RenderPanel(title string, titleColor color.Color, content string, active bool) string {
	borderColor := ColorBorder
	if active {
		borderColor = ColorBorderActive
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 2)

	bordered := borderStyle.Render(content)

	if title == "" {
		return bordered
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Background(ColorSurface)
	titleRendered := titleStyle.Render(" " + title + " ")

	w := lipgloss.Width(bordered)
	h := lipgloss.Height(bordered)

	bgLayer := lipgloss.NewLayer(bordered).Z(0)
	titleLayer := lipgloss.NewLayer(titleRendered).X(2).Y(0).Z(1)

	return lipgloss.NewCanvas(w, h).
		Compose(lipgloss.NewCompositor(bgLayer, titleLayer)).
		Render()
}
