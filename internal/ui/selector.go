package ui

import (
	"fmt"
	"strings"
)

type SelectorModel struct {
	Title  string
	Items  []string
	Cursor int
	Active bool
}

func NewSelector(title string, items []string, current string) SelectorModel {
	cursor := 0
	for i, item := range items {
		if item == current {
			cursor = i
			break
		}
	}
	return SelectorModel{
		Title:  title,
		Items:  items,
		Cursor: cursor,
	}
}

func (s *SelectorModel) MoveUp() {
	if s.Cursor > 0 {
		s.Cursor--
	}
}

func (s *SelectorModel) MoveDown() {
	if s.Cursor < len(s.Items)-1 {
		s.Cursor++
	}
}

func (s *SelectorModel) Selected() string {
	if s.Cursor < len(s.Items) {
		return s.Items[s.Cursor]
	}
	return ""
}

func (s *SelectorModel) Render(width int) string {
	if !s.Active {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s\n", s.Title))
	b.WriteString("  ─────────────────────────\n")

	maxVisible := 15
	start := 0
	if s.Cursor >= maxVisible {
		start = s.Cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(s.Items) {
		end = len(s.Items)
	}

	if start > 0 {
		b.WriteString("    ↑ more\n")
	}
	for i := start; i < end; i++ {
		cursor := "  "
		if i == s.Cursor {
			cursor = "▸ "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, s.Items[i]))
	}
	if end < len(s.Items) {
		b.WriteString("    ↓ more\n")
	}
	b.WriteString("\n  Enter: select  Esc: cancel")

	return OverlayStyle.Render(b.String())
}
