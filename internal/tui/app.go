package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/carter/inv/internal/config"
	"github.com/carter/inv/internal/git"
	"github.com/carter/inv/internal/model"
	"github.com/carter/inv/internal/pdfsync"
	"github.com/carter/inv/internal/store"
)

type viewState int

const (
	viewSetup viewState = iota
	viewList
	viewEditor
	viewClient
)

// Messages for view transitions
type switchToListMsg struct{}
type switchToEditorMsg struct {
	invoice *model.Invoice // nil for new invoice
}
type switchToClientMsg struct{}
type switchToSetupMsg struct{}
type invoiceSavedMsg struct {
	invoice model.Invoice
	isNew   bool
}
type invoiceDeletedMsg struct{ id string }
type invoiceSentMsg struct{ id string }
type clientSelectedMsg struct{ slug string }
type setupCompleteMsg struct {
	global model.GlobalConfig
	client model.ClientConfig
	slug   string
}
type editorFinishedMsg struct{ err error }
type statusMsg struct{ text string }
type gitDoneMsg struct{ err error }
type gitHubSetupDoneMsg struct {
	ok  bool
	err error
}
type pdfExportedMsg struct {
	invoiceID   string
	exportPath  string
	archivePath string
	err         error
}
type pdfSyncDoneMsg struct {
	result pdfsync.Result
	err    error
}

type AppModel struct {
	state    viewState
	list     ListModel
	editor   EditorModel
	client   ClientModel
	setup    SetupModel
	modal    *ModalModel
	showHelp bool

	globals   model.GlobalConfig
	active    string // active client slug
	clientCfg model.ClientConfig
	store     model.InvoiceStore

	width  int
	height int
	status string // status bar message
}

func NewApp() AppModel {
	m := AppModel{}

	if !config.Exists() {
		m.state = viewSetup
		m.setup = NewSetupModel()
		return m
	}

	// Load existing config
	g, err := config.LoadGlobal()
	if err != nil {
		m.status = fmt.Sprintf("Error loading config: %v", err)
		m.state = viewSetup
		m.setup = NewSetupModel()
		return m
	}
	m.globals = g

	s, err := store.Load()
	if err != nil {
		m.status = fmt.Sprintf("Error loading invoices: %v", err)
	}
	m.store = s

	// Load active client
	m.active = g.Settings.ActiveClient
	clients, _ := config.ListClients()
	if m.active == "" && len(clients) > 0 {
		m.active = clients[0]
	}

	if m.active != "" {
		cc, err := config.LoadClient(m.active)
		if err != nil {
			m.status = fmt.Sprintf("Error loading client %s: %v", m.active, err)
		}
		m.clientCfg = cc
	}

	if m.active == "" {
		m.state = viewClient
		m.client = NewClientModel(nil, true)
	} else {
		m.state = viewList
		m.list = NewListModel(m.store, m.active, m.clientCfg, m.globals)
	}

	return m
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		func() tea.Msg { return tea.RequestWindowSize() },
	}

	// Git pull on launch (async, best-effort)
	dir := config.Dir()
	if git.IsRepo(dir) {
		cmds = append(cmds, func() tea.Msg {
			err := git.Pull(dir)
			return gitDoneMsg{err}
		})
	}

	if m.state == viewSetup {
		cmds = append(cmds, m.setup.Init())
	} else if !git.IsRepo(dir) {
		cmds = append(cmds, syncPDFsAsync(m.globals))
	}

	return tea.Batch(cmds...)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case gitDoneMsg:
		if msg.err != nil {
			m.status = "git sync: offline"
		}
		// Reload store after pull
		if s, err := store.Load(); err == nil {
			m.store = s
			if m.state == viewList {
				m.list = NewListModel(m.store, m.active, m.clientCfg, m.globals)
			}
		}
		return m, syncPDFsAsync(m.globals)

	case gitHubSetupDoneMsg:
		if msg.ok {
			m.status = "GitHub backup repo created via gh"
		} else if msg.err != nil {
			m.status = fmt.Sprintf("GitHub backup not configured: %v", msg.err)
		}
		return m, nil

	case pdfExportedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("PDF error: %v", msg.err)
			return m, nil
		}

		if msg.archivePath != "" && msg.archivePath != msg.exportPath {
			m.status = fmt.Sprintf("PDF exported: %s | tracked copy: %s", msg.exportPath, msg.archivePath)
		} else {
			m.status = fmt.Sprintf("PDF exported: %s", msg.exportPath)
		}

		if msg.archivePath != "" {
			return m, gitCommitAsync(config.Dir(), fmt.Sprintf("archive pdf for invoice %s", msg.invoiceID))
		}
		return m, nil

	case pdfSyncDoneMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("PDF sync error: %v", msg.err)
			return m, nil
		}
		if msg.result.ExportUpdated == 0 && msg.result.ArchiveUpdated == 0 {
			return m, nil
		}

		m.status = fmt.Sprintf(
			"PDF sync: %d export updated, %d archived updated",
			msg.result.ExportUpdated,
			msg.result.ArchiveUpdated,
		)

		if msg.result.ArchiveUpdated > 0 && git.IsRepo(config.Dir()) {
			return m, gitCommitAsync(config.Dir(), "sync pdf archive")
		}
		return m, nil

	case statusMsg:
		m.status = msg.text
		return m, nil

	case setupCompleteMsg:
		m.globals = msg.global
		m.clientCfg = msg.client
		m.active = msg.slug

		if err := config.EnsureDirs(); err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		if err := config.SaveGlobal(m.globals); err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		if err := config.SaveClient(m.active, m.clientCfg); err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}

		// Init git
		dir := config.Dir()
		if !git.IsRepo(dir) {
			git.Init(dir)
		}
		gitCommit(dir, "initial setup")

		m.store, _ = store.Load()
		m.state = viewList
		m.list = NewListModel(m.store, m.active, m.clientCfg, m.globals)
		return m, setupGitHubAsync(dir)

	case switchToEditorMsg:
		m.state = viewEditor
		m.editor = NewEditorModel(msg.invoice, m.store, m.active, m.clientCfg, m.globals)
		return m, m.editor.Init()

	case switchToListMsg:
		m.state = viewList
		m.list = NewListModel(m.store, m.active, m.clientCfg, m.globals)
		return m, nil

	case switchToClientMsg:
		m.state = viewClient
		clients, _ := config.ListClients()
		m.client = NewClientModel(clients, false)
		return m, nil

	case clientSelectedMsg:
		m.active = msg.slug
		cc, err := config.LoadClient(m.active)
		if err != nil {
			m.status = fmt.Sprintf("Error loading client: %v", err)
			return m, nil
		}
		m.clientCfg = cc

		// Persist active client
		m.globals.Settings.ActiveClient = m.active
		config.SaveGlobal(m.globals)

		m.state = viewList
		m.list = NewListModel(m.store, m.active, m.clientCfg, m.globals)
		return m, nil

	case invoiceSavedMsg:
		if msg.isNew {
			store.AddInvoice(&m.store, msg.invoice)
		} else {
			store.UpdateInvoice(&m.store, msg.invoice)
		}

		// Update client's last_line_items
		m.clientCfg.LastLineItems = msg.invoice.LineItems
		config.SaveClient(m.active, m.clientCfg)

		if err := store.Save(m.store); err != nil {
			m.status = fmt.Sprintf("Error saving: %v", err)
			return m, nil
		}

		action := "update"
		if msg.isNew {
			action = "create"
		}
		m.status = fmt.Sprintf("Invoice %s saved", msg.invoice.ID)

		m.state = viewList
		m.list = NewListModel(m.store, m.active, m.clientCfg, m.globals)

		return m, gitCommitAsync(
			config.Dir(),
			fmt.Sprintf("%s invoice %s for %s", action, msg.invoice.ID, m.active),
		)

	case invoiceDeletedMsg:
		if err := store.DeleteInvoice(&m.store, msg.id); err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		store.Save(m.store)
		m.modal = nil
		m.status = fmt.Sprintf("Invoice %s deleted", msg.id)
		m.list = NewListModel(m.store, m.active, m.clientCfg, m.globals)

		return m, gitCommitAsync(
			config.Dir(),
			fmt.Sprintf("delete invoice %s for %s", msg.id, m.active),
		)

	case invoiceSentMsg:
		if err := store.MarkSent(&m.store, msg.id); err != nil {
			m.status = fmt.Sprintf("Error: %v", err)
			return m, nil
		}
		store.Save(m.store)
		m.modal = nil
		m.status = fmt.Sprintf("Invoice %s marked as sent", msg.id)
		m.list = NewListModel(m.store, m.active, m.clientCfg, m.globals)

		return m, gitCommitAsync(
			config.Dir(),
			fmt.Sprintf("mark invoice %s as sent", msg.id),
		)
	}

	// Handle help overlay
	if m.showHelp {
		if msg, ok := msg.(tea.KeyPressMsg); ok {
			_ = msg
			m.showHelp = false
			return m, nil
		}
		return m, nil
	}

	// Toggle help from list view
	if msg, ok := msg.(tea.KeyPressMsg); ok && msg.String() == "?" && m.state == viewList {
		m.showHelp = true
		return m, nil
	}

	// Handle modal first if active
	if m.modal != nil {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			newModal, cmd := m.modal.Update(msg)
			if newModal == nil {
				m.modal = nil
				return m, cmd
			}
			m.modal = newModal
			return m, cmd
		}
		return m, nil
	}

	// Delegate to active view
	var cmd tea.Cmd
	switch m.state {
	case viewSetup:
		var newSetup tea.Model
		newSetup, cmd = m.setup.Update(msg)
		m.setup = newSetup.(SetupModel)

	case viewList:
		var result tea.Model
		result, cmd = m.list.Update(msg)
		m.list = result.(ListModel)

		// Check if list wants to show a modal
		if m.list.pendingModal != nil {
			m.modal = m.list.pendingModal
			m.list.pendingModal = nil
		}

	case viewEditor:
		var result tea.Model
		result, cmd = m.editor.Update(msg)
		m.editor = result.(EditorModel)

	case viewClient:
		var result tea.Model
		result, cmd = m.client.Update(msg)
		m.client = result.(ClientModel)
	}

	return m, cmd
}

func (m AppModel) View() tea.View {
	var content string

	switch m.state {
	case viewSetup:
		content = m.setup.View().Content
	case viewList:
		content = m.list.View().Content
	case viewEditor:
		content = m.editor.View().Content
	case viewClient:
		content = m.client.View().Content
	}

	if m.modal != nil {
		content = m.modal.Overlay(content, m.width, m.height)
	}

	if m.showHelp {
		content = RenderHelp(content, m.width, m.height)
	}

	if m.status != "" {
		content += "\n" + mutedStyle.Render(m.status)
	}

	return tea.NewView(docStyle.Width(m.width - 4).Render(content))
}

func gitCommit(dir, message string) {
	git.CommitAll(dir, message)
	if git.HasRemote(dir) {
		git.Push(dir)
	}
}

func gitCommitAsync(dir, message string) tea.Cmd {
	return func() tea.Msg {
		gitCommit(dir, message)
		return gitDoneMsg{}
	}
}

func setupGitHubAsync(dir string) tea.Cmd {
	return func() tea.Msg {
		if !git.IsRepo(dir) || git.HasRemote(dir) || !git.HasGH() {
			return gitHubSetupDoneMsg{}
		}

		err := git.SetupGitHub(dir)
		if err != nil {
			return gitHubSetupDoneMsg{err: err}
		}
		return gitHubSetupDoneMsg{ok: true}
	}
}

func syncPDFsAsync(global model.GlobalConfig) tea.Cmd {
	return func() tea.Msg {
		result, err := pdfsync.Sync(global)
		return pdfSyncDoneMsg{result: result, err: err}
	}
}
