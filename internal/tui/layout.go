package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const mainContentWidth = 71

func justifyLine(left, right string, width int) string {
	leftWidth := ansi.StringWidth(left)
	rightWidth := ansi.StringWidth(right)

	if leftWidth+rightWidth >= width {
		maxRightWidth := width - leftWidth - 1
		if maxRightWidth < 0 {
			maxRightWidth = 0
		}
		right = ansi.Truncate(right, maxRightWidth, "...")
		rightWidth = ansi.StringWidth(right)
	}

	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + right
}
