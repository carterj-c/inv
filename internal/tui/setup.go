package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/model"
)

type setupStep int

const (
	setupName setupStep = iota
	setupAddress
	setupExportDir
	setupCurrency
	setupClientName
	setupClientAddr
	setupClientTerms
	setupClientCurrency
)

type SetupModel struct {
	step   setupStep
	input  TextInput
	values map[setupStep]string
}

func NewSetupModel() SetupModel {
	return SetupModel{
		values: map[setupStep]string{
			setupExportDir:      "~/invoices",
			setupCurrency:       "CAD",
			setupClientTerms:    "Net 30",
			setupClientCurrency: "CAD",
		},
	}
}

func (m SetupModel) Init() tea.Cmd {
	return nil
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, func() tea.Msg { return tea.Quit() }

		case "enter":
			val := m.input.Value
			if val == "" {
				val = m.values[m.step]
			}
			m.values[m.step] = val

			if m.step == setupClientCurrency {
				return m, m.complete()
			}

			m.step++
			def := m.values[m.step]
			m.input = NewTextInput(def)

		default:
			m.input.Update(msg)
		}
	}
	return m, nil
}

func (m SetupModel) View() tea.View {
	var b strings.Builder

	b.WriteString(headerStyle.Render(" inv — First Time Setup"))
	b.WriteString("\n\n")

	steps := []struct {
		step  setupStep
		label string
	}{
		{setupName, "Your name / business name:"},
		{setupAddress, "Your address:"},
		{setupExportDir, "PDF export directory:"},
		{setupCurrency, "Default currency:"},
		{setupClientName, "First client name:"},
		{setupClientAddr, "Client address:"},
		{setupClientTerms, "Payment terms:"},
		{setupClientCurrency, "Client currency:"},
	}

	for _, s := range steps {
		l := labelStyle.Width(28).Render(s.label)
		if s.step < m.step {
			b.WriteString(fmt.Sprintf(" %s %s\n", l, m.values[s.step]))
		} else if s.step == m.step {
			b.WriteString(fmt.Sprintf(" %s %s\n", l, focusedStyle.Render(m.input.Render())))
		} else {
			b.WriteString(fmt.Sprintf(" %s %s\n", l, mutedStyle.Render("")))
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(" [Enter] next  [Ctrl+c] quit"))

	return tea.NewView(b.String())
}

func (m SetupModel) complete() tea.Cmd {
	return func() tea.Msg {
		g := model.GlobalConfig{
			User: model.UserConfig{
				Name:    m.values[setupName],
				Address: m.values[setupAddress],
			},
			Settings: model.SettingsConfig{
				ExportDir:       m.values[setupExportDir],
				DefaultCurrency: m.values[setupCurrency],
			},
		}

		clientName := m.values[setupClientName]
		slug := config.Slugify(clientName)
		g.Settings.ActiveClient = slug

		cc := model.ClientConfig{
			Name:         clientName,
			Address:      m.values[setupClientAddr],
			PaymentTerms: m.values[setupClientTerms],
			Currency:     m.values[setupClientCurrency],
		}

		return setupCompleteMsg{global: g, client: cc, slug: slug}
	}
}
