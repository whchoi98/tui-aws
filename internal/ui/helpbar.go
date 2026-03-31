package ui

import "fmt"

type ViewState int

const (
	ViewTable ViewState = iota
	ViewSearch
	ViewFilter
	ViewProfileSelect
	ViewRegionSelect
	ViewPortForward
)

func RenderHelpBar(state ViewState, width int) string {
	var keys string
	switch state {
	case ViewSearch:
		keys = helpLine(
			"Enter", "Connect",
			"Esc", "Cancel",
		)
	case ViewFilter, ViewProfileSelect, ViewRegionSelect:
		keys = helpLine(
			"↑↓", "Navigate",
			"Enter", "Select",
			"Esc", "Cancel",
		)
	case ViewPortForward:
		keys = helpLine(
			"Enter", "Start",
			"Esc", "Cancel",
		)
	default:
		keys = helpLine(
			"↑↓", "Navigate",
			"Enter", "Connect",
			"/", "Search",
			"f", "Filter",
			"p", "Profile",
			"r", "Region",
			"s", "Sort",
			"F", "Fav",
			"P", "Port Fwd",
			"R", "Refresh",
			"q", "Quit",
		)
	}
	return HelpBarStyle.Width(width).Render(keys)
}

func helpLine(keyvals ...string) string {
	var s string
	for i := 0; i < len(keyvals)-1; i += 2 {
		if s != "" {
			s += "  "
		}
		s += fmt.Sprintf("%s: %s", HelpKeyStyle.Render(keyvals[i]), keyvals[i+1])
	}
	return " " + s
}
