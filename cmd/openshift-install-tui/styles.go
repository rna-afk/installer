package main

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// Extra top padding leaves room for a decorative header inside the panel.
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			PaddingTop(3)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	faintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	wheelSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("#333333"))

	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	blinkStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226"))

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("111"))

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))
)
