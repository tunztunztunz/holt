// Package ui holds acre's shared terminal styling: the single place colors and
// lipgloss styles are defined, so the CLI today and the planned TUI render with
// one consistent palette instead of re-declaring styles per command.
package ui

import "charm.land/lipgloss/v2"

// Palette is acre's terminal color set. Everything visual references these, so a
// theme change happens here, once.
var (
	Accent = lipgloss.Color("212") // pink — emphasis, selection
	Muted  = lipgloss.Color("240") // borders, de-emphasized chrome
	Subtle = lipgloss.Color("244") // secondary text
	Text   = lipgloss.Color("252") // primary text
)

// Table styles used by `acre ls` — reusable by any future tabular view.
var (
	TableHeader = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	TableCell   = lipgloss.NewStyle().Padding(0, 1)
	TableBorder = lipgloss.NewStyle().Foreground(Muted)
)
