package admin

import "charm.land/lipgloss/v2"

// Palette orange/ambre : distingue l'interface admin du client (violet)
var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1A1A1A")).Background(lipgloss.Color("#FFA500")).Padding(0, 1).MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA500"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
	appStyle      = lipgloss.NewStyle().Padding(1, 2)
)
