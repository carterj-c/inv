package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/model"
)

type clientViewState int

const (
	clientSelect clientViewState = iota
	clientAddName
	clientAddAddress
	clientAddTerms
	clientAddCurrency
)

type ClientModel struct {
	clients  []string // slugs
	cursor   int
	state    clientViewState
	forceNew bool // true when no clients exist yet
	newName  string
	newAddr  string
	newTerms string
	newCur   string
	input    TextInput
}

func NewClientModel(clients []string, forceNew bool) ClientModel {
	m := ClientModel{
		clients:  clients,
		forceNew: forceNew,
		newTerms: "Net 30",
		newCur:   "CAD",
	}
	if forceNew {
		m.state = clientAddName
	}
	return m
}

func (m ClientModel) Init() tea.Cmd {
	return nil
}

func (m ClientModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()

		if m.state != clientSelect {
			return m.handleAddClient(msg)
		}

		switch key {
		case "esc":
			if m.forceNew {
				return m, nil
			}
			return m, func() tea.Msg { return switchToListMsg{} }

		case "j", "down":
			if m.cursor < len(m.clients) {
				m.cursor++
			}

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "enter":
			if m.cursor < len(m.clients) {
				slug := m.clients[m.cursor]
				return m, func() tea.Msg { return clientSelectedMsg{slug: slug} }
			}
			m.state = clientAddName
			m.input = NewTextInput("")
		}
	}

	return m, nil
}

func (m ClientModel) handleAddClient(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.forceNew {
			return m, nil
		}
		m.state = clientSelect
		return m, nil

	case "enter":
		switch m.state {
		case clientAddName:
			m.newName = m.input.Value
			if m.newName == "" {
				return m, nil
			}
			m.input = NewTextInput("")
			m.state = clientAddAddress

		case clientAddAddress:
			m.newAddr = m.input.Value
			m.input = NewTextInput(m.newTerms)
			m.state = clientAddTerms

		case clientAddTerms:
			m.newTerms = m.input.Value
			m.input = NewTextInput(m.newCur)
			m.state = clientAddCurrency

		case clientAddCurrency:
			m.newCur = m.input.Value
			if m.newCur == "" {
				m.newCur = "CAD"
			}
			slug := config.Slugify(m.newName)
			cc := model.ClientConfig{
				Name:         m.newName,
				Address:      m.newAddr,
				PaymentTerms: m.newTerms,
				Currency:     m.newCur,
			}
			if err := config.SaveClient(slug, cc); err != nil {
				return m, func() tea.Msg {
					return statusMsg{text: fmt.Sprintf("Error saving client: %v", err)}
				}
			}
			return m, func() tea.Msg { return clientSelectedMsg{slug: slug} }
		}
		return m, nil

	default:
		m.input.Update(msg)
	}
	return m, nil
}

func (m ClientModel) View() tea.View {
	var b strings.Builder

	if m.state != clientSelect {
		return m.viewAddClient()
	}

	b.WriteString(headerStyle.Render(" Select Client"))
	b.WriteString("\n\n")

	for i, slug := range m.clients {
		cc, _ := config.LoadClient(slug)
		name := cc.Name
		if name == "" {
			name = slug
		}

		line := fmt.Sprintf("   %s", name)
		if i == m.cursor {
			line = selectedStyle.Render(" > " + name)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	addLine := "   + Add new client..."
	if m.cursor == len(m.clients) {
		addLine = selectedStyle.Render(" > + Add new client...")
	}
	b.WriteString(addLine)
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render(" [j/k] navigate  [Enter] select  [Esc] back"))

	return tea.NewView(b.String())
}

func (m ClientModel) viewAddClient() tea.View {
	var b strings.Builder

	b.WriteString(headerStyle.Render(" New Client"))
	b.WriteString("\n\n")

	fields := []struct {
		label string
		value string
		state clientViewState
	}{
		{"Client name:", m.newName, clientAddName},
		{"Address:", m.newAddr, clientAddAddress},
		{"Payment terms:", m.newTerms, clientAddTerms},
		{"Currency:", m.newCur, clientAddCurrency},
	}

	for _, f := range fields {
		l := labelStyle.Render(f.label)
		v := f.value
		if f.state == m.state {
			v = focusedStyle.Render(m.input.Render())
		} else if f.state > m.state {
			v = mutedStyle.Render("(pending)")
		}
		b.WriteString(fmt.Sprintf(" %s %s\n", l, v))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(" [Enter] next  [Esc] cancel"))

	return tea.NewView(b.String())
}
