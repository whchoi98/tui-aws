package shared

import "charm.land/lipgloss/v2"

// RenderOverlay wraps content in the standard overlay style (rounded border).
func RenderOverlay(content string) string {
	return OverlayStyle.Render(content)
}

// PlaceOverlay centers an overlay string using lipgloss.Place.
func PlaceOverlay(width int, overlay string) string {
	return lipgloss.Place(width, 0, lipgloss.Center, lipgloss.Center, overlay)
}
