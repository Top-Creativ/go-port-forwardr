package ui

import (
	"github.com/adzin/port-forward-cli/internal/db"
	"github.com/adzin/port-forward-cli/internal/tunnel"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenServerList screen = iota
	screenServerForm
	screenPortSelector
	screenSession
	screenExport
	screenImport
)

type errMsg struct{ err error }

type AppModel struct {
	screen        screen
	serverList    ServerListModel
	serverForm    ServerFormModel
	portSelector  PortSelectorModel
	sessionView   SessionViewModel
	exportSelect  ExportSelectModel
	importView    ImportViewModel
	tunnelManager *tunnel.Manager
	activeServer  *db.Server
	width         int
	height        int
	editMode      bool
}

func NewApp() AppModel {
	return AppModel{
		screen:     screenServerList,
		serverList: NewServerList(),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.serverList.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.serverList.Update(msg)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case ServersLoadedMsg:
		if m.screen == screenServerForm {
			m.screen = screenServerList
		}
		var cmd tea.Cmd
		m.serverList, cmd = m.serverList.Update(msg)
		return m, cmd

	case PortsLoadedMsg:
		var cmd tea.Cmd
		m.portSelector, cmd = m.portSelector.Update(msg)
		return m, cmd

	case PortAddedMsg:
		var cmd tea.Cmd
		m.portSelector, cmd = m.portSelector.Update(msg)
		return m, cmd

	case PortDeletedMsg:
		var cmd tea.Cmd
		m.portSelector, cmd = m.portSelector.Update(msg)
		return m, cmd

	case TunnelUpdateMsg:
		var cmd tea.Cmd
		m.sessionView, cmd = m.sessionView.Update(msg)
		return m, cmd

	case ExportDoneMsg:
		var cmd tea.Cmd
		m.exportSelect, cmd = m.exportSelect.Update(msg)
		return m, cmd

	case exportPasswordAccepted:
		var cmd tea.Cmd
		m.exportSelect, cmd = m.exportSelect.Update(msg)
		return m, cmd

	case ImportDoneMsg:
		var cmd tea.Cmd
		m.importView, cmd = m.importView.Update(msg)
		return m, cmd

	case importNeedsPassword:
		m.importView.importData = msg.data
		m.importView.state = importPassword
		m.importView.err = ""
		m.importView.loading = false
		pi := textinput.New()
		pi.Placeholder = "Enter master password"
		pi.EchoMode = textinput.EchoPassword
		pi.EchoCharacter = '*'
		pi.CharLimit = 128
		pi.Width = 40
		pi.Focus()
		m.importView.passInput = pi
		return m, textinput.Blink

	case importReady:
		m.importView.SetServers(BuildImportPreview(msg.data))
		m.importView.state = importPreview
		m.importView.cursor = 0
		m.importView.loading = false
		m.importView.err = ""
		return m, nil

	case importPasswordAccepted:
		m.importView.SetServers(BuildImportPreview(msg.data))
		m.importView.state = importPreview
		m.importView.cursor = 0
		m.importView.loading = false
		m.importView.err = ""
		return m, nil

	case errMsg:
		if m.screen == screenServerForm {
			m.serverForm.SetError(msg.err.Error())
		} else if m.screen == screenExport {
			var cmd tea.Cmd
			m.exportSelect, cmd = m.exportSelect.Update(msg)
			return m, cmd
		} else if m.screen == screenImport {
			var cmd tea.Cmd
			m.importView, cmd = m.importView.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		if m.screen == screenSession {
			m.tunnelManager.StopAll()
			m.screen = screenPortSelector
			return m, m.portSelector.Init()
		}
		return m, tea.Quit
	}

	switch m.screen {
	case screenServerList:
		if m.serverList.delMode {
			switch key {
			case "y":
				s := m.serverList.SelectedServer()
				if s != nil {
					return m, func() tea.Msg {
						if err := db.DeleteServer(s.ID); err != nil {
							return errMsg{err}
						}
						return ServerDeletedMsg{}
					}
				}
				m.serverList.SetDelMode(false)
			default:
				m.serverList.SetDelMode(false)
			}
			return m, nil
		}

		switch key {
		case "q":
			return m, tea.Quit
		case "up", "k":
			m.serverList.MoveUp()
		case "down", "j":
			m.serverList.MoveDown()
		case "n":
			m.editMode = false
			m.serverForm = NewServerForm(nil)
			m.screen = screenServerForm
		case "e":
			if s := m.serverList.SelectedServer(); s != nil {
				m.editMode = true
				m.serverForm = NewServerForm(s)
				m.screen = screenServerForm
			}
		case "d":
			m.serverList.SetDelMode(true)
		case "x":
			if len(m.serverList.servers) == 0 {
				return m, nil
			}
			m.exportSelect = NewExportSelect(m.serverList.servers)
			m.screen = screenExport
			return m, nil
		case "i":
			m.importView = NewImportModel()
			m.screen = screenImport
			return m, m.importView.Init()
		case "enter":
			if s := m.serverList.SelectedServer(); s != nil {
				m.activeServer = s
				m.portSelector = NewPortSelector(s.ID)
				m.screen = screenPortSelector
				return m, m.portSelector.Init()
			}
		}

		var cmd tea.Cmd
		m.serverList, cmd = m.serverList.Update(msg)
		return m, cmd

	case screenServerForm:
		switch key {
		case "esc":
			m.screen = screenServerList
			return m, loadServers
		case "ctrl+s":
			s, err := m.serverForm.Validate()
			if err != nil {
				m.serverForm.SetError(err.Error())
				return m, nil
			}
			return m, func() tea.Msg {
				if s.ID == 0 {
					if err := db.CreateServer(&s); err != nil {
						return errMsg{err}
					}
				} else {
					if err := db.UpdateServer(&s); err != nil {
						return errMsg{err}
					}
				}
				return loadServers()
			}
		default:
			var cmd tea.Cmd
			m.serverForm, cmd = m.serverForm.Update(msg)
			return m, cmd
		}

	case screenPortSelector:
		ps := &m.portSelector

		if ps.IsAddMode() {
			switch key {
			case "esc":
				ps.ExitAddMode()
				return m, nil
			case "ctrl+s":
				p, err := ps.AddFormValidate()
				if err != nil {
					m.portSelector.err = err.Error()
					return m, nil
				}
				p.ServerID = m.activeServer.ID
				p.ID = ps.EditID()
				return m, func() tea.Msg {
					if p.ID == 0 {
						if err := db.CreatePort(&p); err != nil {
							return errMsg{err}
						}
					} else {
						if err := db.UpdatePort(&p); err != nil {
							return errMsg{err}
						}
					}
					return PortAddedMsg{}
				}
			default:
				var cmd tea.Cmd
				m.portSelector, cmd = m.portSelector.Update(msg)
				return m, cmd
			}
		}

		if ps.IsDelMode() {
			switch key {
			case "y":
				p := ps.CurrentPort()
				if p != nil {
					return m, func() tea.Msg {
						if err := db.DeletePort(p.ID); err != nil {
							return errMsg{err}
						}
						return PortDeletedMsg{}
					}
				}
				ps.SetDelMode(false)
			case "n", "esc":
				ps.SetDelMode(false)
			}
			return m, nil
		}

		switch key {
		case "esc":
			m.screen = screenServerList
			return m, loadServers
		case "up", "k":
			ps.MoveUp()
		case "down", "j":
			ps.MoveDown()
		case " ":
			ps.ToggleCurrent()
		case "t":
			ps.ToggleAll()
		case "a":
			ps.EnterAddMode()
		case "e":
			ps.EnterEditMode()
		case "d":
			ps.SetDelMode(true)
		case "enter":
			selected := ps.SelectedPorts()
			if len(selected) == 0 {
				m.portSelector.err = "Select at least one port first (space to toggle, t to toggle all)"
				return m, nil
			}
			m.tunnelManager = tunnel.NewManager()
			m.tunnelManager.Start(m.activeServer, selected)
			m.sessionView = NewSessionView(m.activeServer, selected, m.tunnelManager)
			m.screen = screenSession
			return m, m.sessionView.Init()
		}

	case screenSession:
		switch key {
		case "q":
			m.tunnelManager.StopAll()
			m.screen = screenPortSelector
			return m, m.portSelector.Init()
		}
		var cmd tea.Cmd
		m.sessionView, cmd = m.sessionView.Update(msg)
		return m, cmd

	case screenExport:
		if m.exportSelect.state == exportPassword {
			switch key {
			case "esc":
				m.exportSelect.state = exportSelect
				m.exportSelect.err = ""
				return m, nil
			case "tab", "down":
				if m.exportSelect.passFocused == 0 {
					m.exportSelect.password1.Blur()
					m.exportSelect.passFocused = 1
					m.exportSelect.password2.Focus()
				} else {
					m.exportSelect.password2.Blur()
					m.exportSelect.passFocused = 0
					m.exportSelect.password1.Focus()
				}
				return m, nil
			case "shift+tab", "up":
				if m.exportSelect.passFocused == 1 {
					m.exportSelect.password2.Blur()
					m.exportSelect.passFocused = 0
					m.exportSelect.password1.Focus()
				} else {
					m.exportSelect.password1.Blur()
					m.exportSelect.passFocused = 1
					m.exportSelect.password2.Focus()
				}
				return m, nil
			case "ctrl+s":
				p1, p2 := m.exportSelect.Password()
				if p1 != p2 {
					m.exportSelect.err = "Passwords do not match"
					return m, nil
				}
				m.exportSelect.selectedIDs = m.exportSelect.SelectedServerIDs()
				m.exportSelect.err = ""
				return m, func() tea.Msg {
					return exportPasswordAccepted{password: p1}
				}
			default:
				var cmd tea.Cmd
				m.exportSelect, cmd = m.exportSelect.Update(msg)
				return m, cmd
			}
		}

		switch key {
		case "esc":
			m.screen = screenServerList
			return m, loadServers
		case "up", "k":
			m.exportSelect.MoveUp()
		case "down", "j":
			m.exportSelect.MoveDown()
		case " ":
			m.exportSelect.ToggleCurrent()
		case "t":
			m.exportSelect.ToggleAll()
		case "ctrl+s":
			ids := m.exportSelect.SelectedServerIDs()
			if len(ids) == 0 {
				m.exportSelect.err = "Select at least one server to export"
				return m, nil
			}
			m.exportSelect.selectedIDs = ids
			m.exportSelect.state = exportPassword
			m.exportSelect.err = ""
			m.exportSelect.passFocused = 0
			p1, p2 := newExportPasswordInputs()
			m.exportSelect.password1 = p1
			m.exportSelect.password2 = p2
			return m, textinput.Blink
		}

	case screenImport:
		if m.importView.state == importPath {
			switch key {
			case "esc":
				m.screen = screenServerList
				return m, loadServers
			case "enter", "ctrl+s":
				path := m.importView.FilePath()
				if path == "" {
					m.importView.err = "Please enter a file path"
					return m, nil
				}
				m.importView.loading = true
				m.importView.err = ""
				return m, LoadImportFile(path)
			default:
				var cmd tea.Cmd
				m.importView, cmd = m.importView.Update(msg)
				return m, cmd
			}
		}

		if m.importView.state == importPassword {
			switch key {
			case "esc":
				m.screen = screenServerList
				return m, loadServers
			case "enter", "ctrl+s":
				pw := m.importView.Password()
				if pw == "" {
					m.importView.err = "Password is required to decrypt this file"
					return m, nil
				}
				m.importView.loading = true
				m.importView.err = ""
				return m, DecryptImportFile(m.importView.importData, pw)
			default:
				var cmd tea.Cmd
				m.importView, cmd = m.importView.Update(msg)
				return m, cmd
			}
		}

		switch key {
		case "esc":
			m.screen = screenServerList
			return m, loadServers
		case "up", "k":
			m.importView.MoveUp()
		case "down", "j":
			m.importView.MoveDown()
		case " ":
			m.importView.ToggleCurrent()
		case "ctrl+s":
			selected := m.importView.SelectedServers()
			if len(selected) == 0 {
				m.importView.err = "Select at least one server to import"
				return m, nil
			}
			m.importView.loading = true
			m.importView.err = ""
			m.importView.result = ""
			return m, DoImport(selected)
		}
	}

	return m, nil
}

func (m AppModel) View() string {
	w := m.width
	if w == 0 {
		w = 100
	}

	banner := StyleBanner.Render("⚡  port-forward")
	content := m.activeContent()

	return lipgloss.JoinVertical(lipgloss.Left,
		banner,
		content,
	)
}

func (m AppModel) activeContent() string {
	switch m.screen {
	case screenServerList:
		return m.serverList.View()
	case screenServerForm:
		return m.serverForm.View()
	case screenPortSelector:
		return m.portSelector.View()
	case screenSession:
		return m.sessionView.View()
	case screenExport:
		return m.exportSelect.View()
	case screenImport:
		return m.importView.View()
	}
	return ""
}
