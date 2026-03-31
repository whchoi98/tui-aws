package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"tui-aws/internal/ui/shared"
)

// PlaceholderTab is a stub tab that shows "Coming soon" for unimplemented tabs.
type PlaceholderTab struct {
	label string
}

// NewPlaceholderTab creates a PlaceholderTab with the given label.
func NewPlaceholderTab(label string) *PlaceholderTab {
	return &PlaceholderTab{label: label}
}

func (p *PlaceholderTab) Init(_ *shared.SharedState) tea.Cmd {
	return nil
}

func (p *PlaceholderTab) Update(_ tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
	return p, nil
}

func (p *PlaceholderTab) View(s *shared.SharedState) string {
	msg := fmt.Sprintf("%s — Coming soon", p.label)
	return lipgloss.NewStyle().
		Width(s.Width).
		Padding(4, 4).
		Foreground(lipgloss.Color("#928374")).
		Render(msg)
}

func (p *PlaceholderTab) ShortHelp() string {
	return ""
}
