package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/model"
	"github.com/carter/inv/internal/money"
	"github.com/carter/inv/internal/pdf"
	"github.com/carter/inv/internal/store"
)

type ListModel struct {
	invoices     []model.Invoice
	cursor       int
	clientSlug   string
	clientCfg    model.ClientConfig
	globals      model.GlobalConfig
	pendingModal *ModalModel
}

func NewListModel(s model.InvoiceStore, slug string, cc model.ClientConfig, g model.GlobalConfig) ListModel {
	return ListModel{
		invoices:   store.InvoicesForClient(s, slug),
		clientSlug: slug,
		clientCfg:  cc,
		globals:    g,
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, func() tea.Msg { return tea.Quit() }

		case "j", "down":
			if m.cursor < len(m.invoices)-1 {
				m.cursor++
			}

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "n":
			return m, func() tea.Msg {
				return switchToEditorMsg{invoice: nil}
			}

		case "e", "enter":
			if inv := m.selected(); inv != nil {
				if inv.IsSent() {
					return m, func() tea.Msg {
						return statusMsg{text: "This invoice is locked (sent)."}
					}
				}
				inv := *inv
				return m, func() tea.Msg {
					return switchToEditorMsg{invoice: &inv}
				}
			}

		case "d":
			if inv := m.selected(); inv != nil {
				if inv.IsSent() {
					return m, func() tea.Msg {
						return statusMsg{text: "Cannot delete a sent invoice."}
					}
				}
				modal := NewDeleteModal(inv.ID)
				m.pendingModal = &modal
			}

		case "s":
			if inv := m.selected(); inv != nil {
				if inv.IsSent() {
					return m, func() tea.Msg {
						return statusMsg{text: "Invoice is already sent."}
					}
				}
				modal := NewSendModal(inv.ID)
				m.pendingModal = &modal
			}

		case "p":
			if inv := m.selected(); inv != nil {
				return m, m.exportPDF(*inv)
			}

		case "c":
			return m, func() tea.Msg {
				return switchToClientMsg{}
			}

		case "?":
			// TODO: help overlay
		}
	}

	return m, nil
}

func (m ListModel) View() tea.View {
	var b strings.Builder

	// Header
	cur := m.clientCfg.Currency
	if cur == "" {
		cur = m.globals.Settings.DefaultCurrency
	}
	if cur == "" {
		cur = "CAD"
	}
	header := justifyLine(
		" inv",
		fmt.Sprintf("[client: %s | %s]", m.clientCfg.Name, cur),
		mainContentWidth,
	)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	// Column headers
	colHeader := fmt.Sprintf(" %-10s %-8s %-12s %-26s %10s",
		"#", "Status", "Date", "Description", "Total")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(colHeader))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(" " + strings.Repeat("─", mainContentWidth-1)))
	b.WriteString("\n")

	if len(m.invoices) == 0 {
		b.WriteString(mutedStyle.Render("  No invoices yet. Press [n] to create one."))
		b.WriteString("\n")
	}

	for i, inv := range m.invoices {
		status := statusDraftStyle.Render("draft  ")
		if inv.IsSent() {
			status = statusSentStyle.Render("✓ sent ")
		}

		desc := store.FirstLineDescription(inv)
		if len(desc) > 26 {
			desc = desc[:23] + "..."
		}

		total := inv.Total(m.clientCfg.Tax)
		totalStr := money.FormatCents(total, cur)

		line := fmt.Sprintf(" %-10s %s %-12s %-26s %10s",
			inv.ID, status, inv.Date, desc, totalStr)

		if i == m.cursor {
			if inv.IsSent() {
				line = selectedStyle.Copy().Foreground(colorMuted).Render(line)
			} else {
				line = selectedStyle.Render(line)
			}
		} else if inv.IsSent() {
			line = sentStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(" [n]ew  [e]dit  [d]elete  [p]df  [s]end  [c]lient  [?]help  [q]uit"))

	return tea.NewView(b.String())
}

func (m ListModel) selected() *model.Invoice {
	if m.cursor >= 0 && m.cursor < len(m.invoices) {
		return &m.invoices[m.cursor]
	}
	return nil
}

func (m ListModel) exportPDF(inv model.Invoice) tea.Cmd {
	return func() tea.Msg {
		exportPath, archivePath, err := pdf.Generate(inv, m.globals, m.clientCfg)
		if err != nil {
			return pdfExportedMsg{invoiceID: inv.ID, err: err}
		}

		// Try to open the PDF
		if cmd, err := exec.LookPath("xdg-open"); err == nil {
			exec.Command(cmd, exportPath).Start()
		}

		return pdfExportedMsg{
			invoiceID:   inv.ID,
			exportPath:  exportPath,
			archivePath: archivePath,
		}
	}
}

// ExportDir returns the resolved export directory.
func ExportDir(g model.GlobalConfig) string {
	dir := g.Settings.ExportDir
	if dir == "" {
		dir = "~/invoices"
	}
	return config.ExpandPath(dir)
}
