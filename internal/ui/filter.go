package ui

import (
	"fmt"
	"strings"
)

type FilterModel struct {
	States       []string
	ActiveStates map[string]bool
	Cursor       int
	Active       bool
}

func NewFilterModel() FilterModel {
	return FilterModel{
		States:       []string{"running", "stopped", "pending", "stopping", "terminated"},
		ActiveStates: map[string]bool{},
		Cursor:       0,
	}
}

func (f *FilterModel) Toggle() {
	state := f.States[f.Cursor]
	if f.ActiveStates[state] {
		delete(f.ActiveStates, state)
	} else {
		f.ActiveStates[state] = true
	}
}

func (f *FilterModel) ClearAll() {
	f.ActiveStates = map[string]bool{}
}

func (f *FilterModel) MoveUp() {
	if f.Cursor > 0 {
		f.Cursor--
	}
}

func (f *FilterModel) MoveDown() {
	if f.Cursor < len(f.States)-1 {
		f.Cursor++
	}
}

func (f *FilterModel) Label() string {
	if len(f.ActiveStates) == 0 {
		return "all"
	}
	var active []string
	for _, s := range f.States {
		if f.ActiveStates[s] {
			active = append(active, s)
		}
	}
	return strings.Join(active, ",")
}

func (f *FilterModel) Render(width int) string {
	if !f.Active {
		return ""
	}
	var b strings.Builder
	b.WriteString("  Filter by State\n")
	b.WriteString("  ─────────────────\n")

	for i, state := range f.States {
		cursor := "  "
		if i == f.Cursor {
			cursor = "▸ "
		}
		check := "[ ]"
		if f.ActiveStates[state] {
			check = "[✓]"
		}
		icon := StateStyle(state).Render(fmt.Sprintf("%-12s", state))
		b.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, check, icon))
	}
	b.WriteString("\n  Space: toggle  c: clear all  Esc: close")

	return OverlayStyle.Render(b.String())
}
