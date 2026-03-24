package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type HelpModel struct {
	active bool
}

func (m *HelpModel) Toggle() {
	m.active = !m.active
}

func (m HelpModel) View() tea.View {
	if !m.active {
		return tea.NewView("")
	}

	var b strings.Builder

	content := `
 Keybindings

 List View
  j / k         Navigate up/down
  n             New invoice
  e / Enter     Edit selected (drafts only)
  d             Delete selected (drafts only)
  p             Export selected to PDF
  s             Mark selected as sent
  c             Switch client
  ?             Toggle this help
  q             Quit

 Editor View
  Tab / S-Tab   Next/prev field
  a             Add new line item
  e             Edit selected field/line
  d             Remove selected line item
  j / k         Navigate line items
  Ctrl+e        Open $EDITOR for description
  Enter         Save and return to list
  Esc           Cancel and return to list
`

	box := modalStyle.Width(50).Render(content)
	b.WriteString(box)

	return tea.NewView(b.String())
}

func RenderHelp(background string, width, height int) string {
	content := `
 Keybindings

 List View
  j / k         Navigate up/down
  n             New invoice
  e / Enter     Edit selected (drafts only)
  d             Delete selected (drafts only)
  p             Export selected to PDF
  s             Mark selected as sent
  c             Switch client
  ?             Toggle this help
  q             Quit

 Editor View
  Tab / S-Tab   Next/prev field
  a             Add new line item
  e             Edit selected field/line
  d             Remove selected line item
  j / k         Navigate line items
  Ctrl+e        Open $EDITOR for description
  Enter         Save and return
  Esc           Cancel and return
`

	box := modalStyle.Width(50).Render(content)
	boxLines := strings.Split(box, "\n")

	bgLines := strings.Split(background, "\n")
	startRow := (height - len(boxLines)) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (width - 54) / 2
	if startCol < 0 {
		startCol = 0
	}
	pad := strings.Repeat(" ", startCol)

	for i, line := range boxLines {
		row := startRow + i
		if row < len(bgLines) {
			bgLines[row] = pad + line
		}
	}

	return strings.Join(bgLines, "\n")
}
