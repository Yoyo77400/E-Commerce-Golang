package client

import "charm.land/lipgloss/v2"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EE6FF8"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
	appStyle      = lipgloss.NewStyle().Padding(1, 2)
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7D56F4")).Padding(1, 2)
)
