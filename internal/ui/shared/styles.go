package shared

import "charm.land/lipgloss/v2"

var (
	// Status bar (top)
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3c3836")).
			Foreground(lipgloss.Color("#ebdbb2")).
			Padding(0, 1)

	StatusKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fabd2f")).
			Bold(true)

	// Help bar (bottom)
	HelpBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3c3836")).
			Foreground(lipgloss.Color("#a89984")).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#83a598")).
			Bold(true)

	// Tab bar
	TabBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3c3836")).
			Foreground(lipgloss.Color("#a89984")).
			Padding(0, 1)

	TabActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ebdbb2")).
			Bold(true)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#928374"))

	// Table
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#83a598")).
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#504945"))

	TableSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#504945")).
				Foreground(lipgloss.Color("#ebdbb2"))

	// State colors
	StateRunning    = lipgloss.NewStyle().Foreground(lipgloss.Color("#b8bb26"))
	StateStopped    = lipgloss.NewStyle().Foreground(lipgloss.Color("#fb4934"))
	StatePending    = lipgloss.NewStyle().Foreground(lipgloss.Color("#fabd2f"))
	StateStopping   = lipgloss.NewStyle().Foreground(lipgloss.Color("#fe8019"))
	StateTerminated = lipgloss.NewStyle().Foreground(lipgloss.Color("#928374"))

	// Favorites & history markers
	FavoriteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#fabd2f"))
	RecentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#83a598"))

	// Overlay
	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#83a598")).
			Padding(1, 2)

	// Error
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fb4934")).
			Bold(true)

	// Search
	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fabd2f")).
				Bold(true)
)

// StateStyle returns the lipgloss style for a given instance state.
func StateStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return StateRunning
	case "stopped":
		return StateStopped
	case "pending":
		return StatePending
	case "stopping":
		return StateStopping
	case "terminated":
		return StateTerminated
	default:
		return lipgloss.NewStyle()
	}
}
