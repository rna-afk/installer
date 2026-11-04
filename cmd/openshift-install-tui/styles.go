package main

import "github.com/charmbracelet/lipgloss"

// maxBoxWidth and maxBoxHeight cap the bordered panel so it reads like a
// compact window rather than filling the whole terminal.
const (
	appBackground       = "#000000"
	panelBackground     = "#1a1a1a"
	selectionBackground = "#4a4a4a"

	maxBoxWidth  = 90
	maxBoxHeight = 26

	// maxLogBoxWidth and maxLogBoxHeight give the installer log views a larger
	// budget so more output is readable at once.
	maxLogBoxWidth  = 140
	maxLogBoxHeight = 48

	// commandPanelHeight keeps the command picker as small as its three
	// visible options (excluding the border).
	commandPanelHeight = 8

	// headerGap is the number of blank rows between the header and the panel.
	headerGap = 2

	// bottomMargin is how many rows stay clear below the panel block.
	bottomMargin = 7

	// topMargin is the number of blank rows above the title on the full-bleed
	// install-cluster screens.
	topMargin = 1
)

var (
	appStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(appBackground))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	// bannerStyle paints the block-letter wordmark in the OpenShift red.
	bannerStyle = titleStyle

	// promptStyle paints the command question on the same black canvas as the
	// rest of the header so no other background shows through.
	promptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color(appBackground))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	// boxStyle is a grey panel with a single red rule down the left edge.
	boxStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(panelBackground)).
			Border(lipgloss.Border{Left: "┃"}, false, false, false, true).
			BorderLeftForeground(lipgloss.Color("196")).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	faintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

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
			Foreground(lipgloss.Color("196"))

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))
)
