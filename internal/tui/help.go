package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"

	"go.withmatt.com/inbox/internal/config"
)

func newHelpModel(theme config.Theme) help.Model {
	m := help.New()
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Status.Fg)).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Status.Dim))
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Status.Dim))
	m.Styles.ShortKey = keyStyle
	m.Styles.ShortDesc = descStyle
	m.Styles.ShortSeparator = sepStyle
	m.Styles.FullKey = keyStyle
	m.Styles.FullDesc = descStyle
	m.Styles.FullSeparator = sepStyle
	m.Styles.Ellipsis = sepStyle
	m.ShortSeparator = " • "
	m.FullSeparator = "    "
	m.Ellipsis = "..."
	return m
}
