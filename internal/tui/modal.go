package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type modalKind int

const (
	modalDelete modalKind = iota
	modalSend
)

type ModalModel struct {
	kind      modalKind
	invoiceID string
}

func NewDeleteModal(id string) ModalModel {
	return ModalModel{kind: modalDelete, invoiceID: id}
}

func NewSendModal(id string) ModalModel {
	return ModalModel{kind: modalSend, invoiceID: id}
}

func (m *ModalModel) Update(msg tea.KeyPressMsg) (*ModalModel, tea.Cmd) {
	switch msg.String() {
	case "y":
		if m.kind == modalDelete {
			id := m.invoiceID
			return nil, func() tea.Msg { return invoiceDeletedMsg{id: id} }
		}
		id := m.invoiceID
		return nil, func() tea.Msg { return invoiceSentMsg{id: id} }
	case "n", "esc":
		return nil, nil
	}
	return m, nil
}

func (m ModalModel) Overlay(background string, width, height int) string {
	var title, body string
	if m.kind == modalDelete {
		title = dangerStyle.Render(fmt.Sprintf("Delete invoice %s?", m.invoiceID))
		body = "This will permanently remove the invoice.\nThe invoice number will not be reused."
	} else {
		title = titleStyle.Render(fmt.Sprintf("Mark invoice %s as sent?", m.invoiceID))
		body = "This will lock the invoice permanently.\nIt cannot be edited or deleted after."
	}

	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		title,
		body,
		mutedStyle.Render("[y] Yes    [n] Cancel"),
	)

	modal := modalStyle.Render(content)

	// Center the modal over the background
	bgLines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")

	startRow := (height - len(modalLines)) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (width - lipgloss.Width(modal)) / 2
	if startCol < 0 {
		startCol = 0
	}

	pad := strings.Repeat(" ", startCol)

	// Overlay modal lines onto background
	for i, mLine := range modalLines {
		row := startRow + i
		if row < len(bgLines) {
			bgLines[row] = pad + mLine
		}
	}

	return strings.Join(bgLines, "\n")
}
