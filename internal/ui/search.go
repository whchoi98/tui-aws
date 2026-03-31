package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

type SearchModel struct {
	Query  string
	Active bool
}

func (s *SearchModel) Insert(char rune) {
	s.Query += string(char)
}

func (s *SearchModel) Backspace() {
	if len(s.Query) > 0 {
		s.Query = s.Query[:len(s.Query)-1]
	}
}

func (s *SearchModel) Clear() {
	s.Query = ""
	s.Active = false
}

func (s *SearchModel) Render(width int) string {
	if !s.Active {
		return ""
	}
	prompt := SearchPromptStyle.Render(" /")
	return lipgloss.NewStyle().Width(width).Render(
		fmt.Sprintf("%s %s█", prompt, s.Query),
	)
}
