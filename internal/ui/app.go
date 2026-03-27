package ui

import (
	"github.com/adzin/port-forward-cli/internal/db"
	"github.com/adzin/port-forward-cli/internal/tunnel"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenServerList screen = iota
	screenServerForm
	screenPortSelector
	screenSession
)

type errMsg struct{ err error }

type AppModel struct {
	screen        screen
	serverList    ServerListModel
	serverForm    ServerFormModel
	portSelector  PortSelectorModel
	sessionView   SessionViewModel
	tunnelManager *tunnel.Manager
	activeServer  *db.Server
	width         int
	height        int
	editMode      bool // distinguish new vs edit in form
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

	// Propagate data messages to active screen
	case ServersLoadedMsg:
		// If we were in the form, go back to list
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

	case errMsg:
		// Surface error in current view
		if m.screen == screenServerForm {
			m.serverForm.SetError(msg.err.Error())
		}
		return m, nil
	}

	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit
	if key == "ctrl+c" {
		if m.screen == screenSession {
			m.tunnelManager.StopAll()
			m.screen = screenPortSelector
			return m, m.portSelector.Init()
		}
		return m, tea.Quit
	}

	switch m.screen {
	//─────────────────────────────────────────────
	case screenServerList:
		// When del-mode is active, handle confirmation keys exclusively
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
			default: // 'n', esc, anything else — cancel
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
		case "enter":
			if s := m.serverList.SelectedServer(); s != nil {
				m.activeServer = s
				m.portSelector = NewPortSelector(s.ID)
				m.screen = screenPortSelector
				return m, m.portSelector.Init()
			}
		}

		// Propagate to server list (handles ServerDeletedMsg etc.)
		var cmd tea.Cmd
		m.serverList, cmd = m.serverList.Update(msg)
		return m, cmd

	//─────────────────────────────────────────────
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
				return loadServers() // triggers reload & screen switch
			}
		default:
			var cmd tea.Cmd
			m.serverForm, cmd = m.serverForm.Update(msg)
			return m, cmd
		}

	//─────────────────────────────────────────────
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
				return m, func() tea.Msg {
					if err := db.CreatePort(&p); err != nil {
						return errMsg{err}
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

	//─────────────────────────────────────────────
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
	}

	return m, nil
}

func (m AppModel) View() string {
	w := m.width
	if w == 0 {
		w = 100
	}

	// Top banner
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
	}
	return ""
}
