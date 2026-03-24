package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/carter/inv/internal/model"
	"github.com/carter/inv/internal/money"
	"github.com/carter/inv/internal/store"
)

type editorField int

const (
	fieldDate editorField = iota
	fieldPaymentTerms
	fieldLineItems
)

type lineItemField int

const (
	liFieldDesc lineItemField = iota
	liFieldQty
	liFieldRate
)

type EditorModel struct {
	invoice     model.Invoice
	isNew       bool
	clientCfg   model.ClientConfig
	globals     model.GlobalConfig
	storeRef    model.InvoiceStore
	currency    string
	focusField  editorField
	lineIdx     int // selected line item
	editing     bool
	editingLine bool
	liField     lineItemField
	input       TextInput
	err         string
}

func NewEditorModel(inv *model.Invoice, s model.InvoiceStore, slug string, cc model.ClientConfig, g model.GlobalConfig) EditorModel {
	cur := cc.Currency
	if cur == "" {
		cur = g.Settings.DefaultCurrency
	}
	if cur == "" {
		cur = "CAD"
	}

	m := EditorModel{
		clientCfg: cc,
		globals:   g,
		storeRef:  s,
		currency:  cur,
	}

	if inv == nil {
		year := time.Now().Format("2006")
		id := store.NextInvoiceNumber(&m.storeRef, year)
		m.invoice = model.Invoice{
			ID:           id,
			ClientSlug:   slug,
			Date:         time.Now().Format("2006-01-02"),
			PaymentTerms: cc.PaymentTerms,
			Status:       "draft",
		}
		if len(cc.LastLineItems) > 0 {
			m.invoice.LineItems = make([]model.LineItem, len(cc.LastLineItems))
			copy(m.invoice.LineItems, cc.LastLineItems)
		}
		m.isNew = true
	} else {
		m.invoice = *inv
		m.isNew = false
	}

	return m
}

func (m EditorModel) Init() tea.Cmd {
	return nil
}

func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorFinishedMsg:
		if msg.err == nil && m.lineIdx < len(m.invoice.LineItems) {
			if data, err := os.ReadFile(editorTmpFile()); err == nil {
				m.invoice.LineItems[m.lineIdx].Description = strings.TrimRight(string(data), "\n")
			}
			os.Remove(editorTmpFile())
		}
		return m, nil

	case tea.KeyPressMsg:
		key := msg.String()

		if key == "esc" {
			if m.editingLine {
				m.editingLine = false
				m.editing = false
				return m, nil
			}
			if m.editing {
				m.editing = false
				return m, nil
			}
			return m, func() tea.Msg { return switchToListMsg{} }
		}

		if m.editing {
			return m.handleEditing(msg)
		}

		if m.editingLine {
			return m.handleLineEditing(msg)
		}

		switch key {
		case "enter":
			if len(m.invoice.LineItems) == 0 {
				m.err = "Add at least one line item"
				return m, nil
			}
			return m, func() tea.Msg {
				return invoiceSavedMsg{invoice: m.invoice, isNew: m.isNew}
			}

		case "tab":
			m.focusField = (m.focusField + 1) % 3
			if m.focusField == fieldLineItems && len(m.invoice.LineItems) == 0 {
				m.focusField = fieldDate
			}

		case "shift+tab":
			if m.focusField == 0 {
				m.focusField = fieldLineItems
			} else {
				m.focusField--
			}

		case "e":
			if m.focusField == fieldDate || m.focusField == fieldPaymentTerms {
				m.editing = true
				if m.focusField == fieldDate {
					m.input = NewTextInput(m.invoice.Date)
				} else {
					m.input = NewTextInput(m.invoice.PaymentTerms)
				}
			} else if m.focusField == fieldLineItems && len(m.invoice.LineItems) > 0 {
				m.editingLine = true
				m.liField = liFieldDesc
				m.input = NewTextInput(m.invoice.LineItems[m.lineIdx].Description)
			}

		case "a":
			m.invoice.LineItems = append(m.invoice.LineItems, model.LineItem{
				Description: "",
				Quantity:    1,
				Rate:        0,
			})
			m.lineIdx = len(m.invoice.LineItems) - 1
			m.focusField = fieldLineItems
			m.editingLine = true
			m.liField = liFieldDesc
			m.input = NewTextInput("")

		case "d":
			if m.focusField == fieldLineItems && len(m.invoice.LineItems) > 0 {
				m.invoice.LineItems = append(
					m.invoice.LineItems[:m.lineIdx],
					m.invoice.LineItems[m.lineIdx+1:]...,
				)
				if m.lineIdx >= len(m.invoice.LineItems) && m.lineIdx > 0 {
					m.lineIdx--
				}
			}

		case "j", "down":
			if m.focusField == fieldLineItems && m.lineIdx < len(m.invoice.LineItems)-1 {
				m.lineIdx++
			}

		case "k", "up":
			if m.focusField == fieldLineItems && m.lineIdx > 0 {
				m.lineIdx--
			}

		case "ctrl+e":
			if m.focusField == fieldLineItems && m.lineIdx < len(m.invoice.LineItems) {
				return m, m.openEditor()
			}
		}
	}

	return m, nil
}

func (m EditorModel) handleEditing(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.focusField == fieldDate {
			m.invoice.Date = m.input.Value
		} else {
			m.invoice.PaymentTerms = m.input.Value
		}
		m.editing = false
	default:
		m.input.Update(msg)
	}
	return m, nil
}

func (m EditorModel) handleLineEditing(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.applyLineInput()
		m.editingLine = false
		m.editing = false
	case "tab":
		m.applyLineInput()
		m.liField = (m.liField + 1) % 3
		m.loadLineInput()
	case "shift+tab":
		m.applyLineInput()
		if m.liField == 0 {
			m.liField = liFieldRate
		} else {
			m.liField--
		}
		m.loadLineInput()
	default:
		m.input.Update(msg)
	}
	return m, nil
}

func (m *EditorModel) applyLineInput() {
	if m.lineIdx >= len(m.invoice.LineItems) {
		return
	}
	li := &m.invoice.LineItems[m.lineIdx]
	switch m.liField {
	case liFieldDesc:
		li.Description = m.input.Value
	case liFieldQty:
		if v, err := strconv.ParseFloat(m.input.Value, 64); err == nil && v > 0 {
			li.Quantity = v
		}
	case liFieldRate:
		if v, err := strconv.ParseFloat(m.input.Value, 64); err == nil && v >= 0 {
			li.Rate = int(v*100 + 0.5)
		}
	}
}

func (m *EditorModel) loadLineInput() {
	if m.lineIdx >= len(m.invoice.LineItems) {
		return
	}
	li := m.invoice.LineItems[m.lineIdx]
	switch m.liField {
	case liFieldDesc:
		m.input = NewTextInput(li.Description)
	case liFieldQty:
		m.input = NewTextInput(money.FormatQuantity(li.Quantity))
	case liFieldRate:
		m.input = NewTextInput(fmt.Sprintf("%.2f", float64(li.Rate)/100))
	}
}

func (m EditorModel) View() tea.View {
	var b strings.Builder
	cur := m.currency

	title := "Edit Invoice"
	if m.isNew {
		title = "New Invoice"
	}
	b.WriteString(headerStyle.Render(fmt.Sprintf(" %s                                   [client: %s]",
		title, m.clientCfg.Name)))
	b.WriteString("\n\n")

	// Invoice fields
	b.WriteString(m.renderField(" Invoice #:", m.invoice.ID+" (auto)", fieldDate, false))
	b.WriteString(m.renderField(" Date:", m.invoice.Date, fieldDate, true))
	b.WriteString(m.renderField(" Payment Terms:", m.invoice.PaymentTerms, fieldPaymentTerms, true))

	b.WriteString(separatorStyle.Render(" " + strings.Repeat("─", 70)))
	b.WriteString("\n\n")

	// Line items header
	liHeader := fmt.Sprintf("  %-3s %-30s %6s %10s %10s", "#", "Description", "Qty", "Rate", "Total")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(liHeader))
	b.WriteString("\n")

	for i, li := range m.invoice.LineItems {
		desc := li.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}

		line := fmt.Sprintf("  %-3d %-30s %6s %10s %10s",
			i+1,
			desc,
			money.FormatQuantity(li.Quantity),
			money.FormatRate(li.Rate, cur),
			money.FormatCents(li.LineTotal(), cur),
		)

		if m.focusField == fieldLineItems && i == m.lineIdx {
			if m.editingLine {
				line = m.renderEditingLine(i, li)
			} else {
				line = selectedStyle.Render(line)
			}
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Totals
	subtotal := m.invoice.Subtotal()
	if len(m.clientCfg.Tax) > 0 {
		b.WriteString(fmt.Sprintf("%53s %10s\n",
			"Subtotal:", money.FormatCents(subtotal, cur)))
		for _, tax := range m.clientCfg.Tax {
			taxAmt := model.TaxAmount(subtotal, tax.Rate)
			b.WriteString(fmt.Sprintf("%53s %10s\n",
				fmt.Sprintf("%s (%.4g%%):", tax.Name, tax.Rate*100),
				money.FormatCents(taxAmt, cur)))
		}
	}
	total := m.invoice.Total(m.clientCfg.Tax)
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("%53s %10s", "Total:", money.FormatCents(total, cur))))
	b.WriteString("\n\n")

	// Help
	if m.editingLine {
		b.WriteString(helpStyle.Render(" [Tab] next field  [Enter] done editing  [Esc] cancel"))
	} else if m.editing {
		b.WriteString(helpStyle.Render(" [Enter] confirm  [Esc] cancel"))
	} else {
		b.WriteString(helpStyle.Render(" [a]dd line  [e]dit  [d] remove line  [Ctrl+e] $EDITOR  [Enter] save  [Esc] cancel"))
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(dangerStyle.Render(" " + m.err))
	}

	return tea.NewView(b.String())
}

func (m EditorModel) renderField(label, value string, field editorField, editable bool) string {
	l := labelStyle.Render(label)
	v := value
	if m.editing && m.focusField == field && editable {
		v = focusedStyle.Render(m.input.Render())
	} else if m.focusField == field && editable {
		v = focusedStyle.Render(v)
	}
	return fmt.Sprintf(" %s %s\n", l, v)
}

func (m EditorModel) renderEditingLine(idx int, li model.LineItem) string {
	cur := m.currency
	desc := li.Description
	qty := money.FormatQuantity(li.Quantity)
	rate := fmt.Sprintf("%.2f", float64(li.Rate)/100)

	switch m.liField {
	case liFieldDesc:
		desc = focusedStyle.Render(m.input.Render())
	case liFieldQty:
		qty = focusedStyle.Render(m.input.Render())
	case liFieldRate:
		rate = focusedStyle.Render(m.input.Render())
	}

	if m.liField != liFieldDesc && len(desc) > 30 {
		desc = desc[:27] + "..."
	}

	total := money.FormatCents(li.LineTotal(), cur)

	return fmt.Sprintf("  %-3d %-30s %6s %10s %10s", idx+1, desc, qty, rate, total)
}

func (m EditorModel) openEditor() tea.Cmd {
	if m.lineIdx >= len(m.invoice.LineItems) {
		return nil
	}

	tmpFile := editorTmpFile()
	content := m.invoice.LineItems[m.lineIdx].Description
	os.WriteFile(tmpFile, []byte(content), 0600)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, tmpFile)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

func editorTmpFile() string {
	return fmt.Sprintf("%s/inv-edit.tmp", os.TempDir())
}
