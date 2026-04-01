package shared

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Column defines a table column with a key, display title, and width.
type Column struct {
	Key   string
	Title string
	Width int
}

// ExpandNameColumn adjusts the "name" column width to fill remaining terminal space.
// It calculates the total fixed column widths, subtracts from the available width,
// and assigns the remaining space to the "name" column (minimum 20, maximum 60).
func ExpandNameColumn(columns []Column, termWidth int) []Column {
	result := make([]Column, len(columns))
	copy(result, columns)

	fixedWidth := 0
	nameIdx := -1
	for i, col := range result {
		if col.Key == "name" {
			nameIdx = i
		} else {
			fixedWidth += col.Width + 1 // +1 for space separator
		}
	}
	if nameIdx < 0 {
		return result
	}

	available := termWidth - fixedWidth - 1 // -1 for name column's separator
	if available < 20 {
		available = 20
	}
	if available > 60 {
		available = 60
	}
	result[nameIdx].Width = available
	return result
}

// RenderRow renders a single table row from column definitions,
// a text function, and an optional style function.
func RenderRow(columns []Column, getText func(Column) string, styleFn func(Column) lipgloss.Style) string {
	var parts []string
	for _, col := range columns {
		val := getText(col)
		// Truncate using terminal cell width (handles wide chars like stars, bullets, hangul)
		w := lipgloss.Width(val)
		if w > col.Width {
			val = ansi.Truncate(val, col.Width-1, "...")
			w = lipgloss.Width(val)
		}
		// Pad with spaces to fill column width (cell-width aware)
		if pad := col.Width - w; pad > 0 {
			val += strings.Repeat(" ", pad)
		}
		// Apply style after truncation/padding so ANSI codes don't break layout
		if styleFn != nil {
			if style := styleFn(col); style.GetForeground() != nil {
				val = style.Render(val)
			}
		}
		parts = append(parts, val)
	}
	return strings.Join(parts, " ")
}
