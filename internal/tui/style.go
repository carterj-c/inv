package tui

import (
	"charm.land/lipgloss/v2"
)

var (
	// Colors
	colorPrimary   = lipgloss.Color("4")   // blue
	colorMuted     = lipgloss.Color("241") // gray
	colorSuccess   = lipgloss.Color("2")   // green
	colorDanger    = lipgloss.Color("1")   // red
	colorHighlight = lipgloss.Color("6")   // cyan

	// Text styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHighlight)

	sentStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	statusSentStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	statusDraftStyle = lipgloss.NewStyle().
			Foreground(colorPrimary)

	dangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	// Layout
	docStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Help bar at the bottom
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	// Modal
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			Width(46)

	// Input field label
	labelStyle = lipgloss.NewStyle().
			Width(18).
			Foreground(colorMuted)

	// Focused input
	focusedStyle = lipgloss.NewStyle().
			Foreground(colorHighlight)

	// Table separator
	separatorStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)
