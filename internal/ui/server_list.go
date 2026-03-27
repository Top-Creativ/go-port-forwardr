package ui

import (
	"fmt"
	"strings"

	"github.com/adzin/port-forward-cli/internal/db"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ServerListModel struct {
	servers  []db.Server
	cursor   int
	err      error
	width    int
	height   int
	delMode  bool
	delMsg   string
}

type ServersLoadedMsg struct{ Servers []db.Server }
type ServerDeletedMsg struct{}

func NewServerList() ServerListModel {
	return ServerListModel{}
}

func (m ServerListModel) Init() tea.Cmd {
	return loadServers
}

func loadServers() tea.Msg {
	servers, err := db.ListServers()
	if err != nil {
		return errMsg{err}
	}
	return ServersLoadedMsg{servers}
}

func (m ServerListModel) Update(msg tea.Msg) (ServerListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case ServersLoadedMsg:
		m.servers = msg.Servers
		if m.cursor >= len(m.servers) {
			m.cursor = max(0, len(m.servers)-1)
		}

	case ServerDeletedMsg:
		m.delMode = false
		m.delMsg = ""
		return m, loadServers

	case errMsg:
		m.err = msg.err
	}
	return m, nil
}

func (m ServerListModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(StyleTitle.Render("  SSH Servers"))
	b.WriteString("\n")
	b.WriteString(StyleSubtitle.Render(" Manage your saved server connections"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(StyleError.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
	}

	if len(m.servers) == 0 {
		b.WriteString(StyleMuted.Render("  No servers yet. Press ") + StyleAccent.Render("n") + StyleMuted.Render(" to add one."))
		b.WriteString("\n")
	} else {
		for i, s := range m.servers {
			authBadge := StyleMuted.Render("[pw]")
			if s.AuthType == "key" {
				authBadge = StyleAccent.Render("[key]")
			}

			row := fmt.Sprintf(" %s  %-20s  %s@%s:%d  %s",
				authBadge,
				s.Name,
				s.User,
				s.Host,
				s.SSHPort,
				StyleMuted.Render(s.CreatedAt.Format("2006-01-02")),
			)

			if i == m.cursor {
				b.WriteString(StyleSelected.Render(fmt.Sprintf("▶ %s", row)))
			} else {
				b.WriteString(StyleRow.Render(fmt.Sprintf("  %s", row)))
			}
			b.WriteString("\n")
		}
	}

	if m.delMode {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("  ⚠  " + m.delMsg))
		b.WriteString("\n")
	}

	// Help
	help := " enter:select  n:new  e:edit  d:delete  q:quit"
	b.WriteString(StyleHelp.Render(help))

	return b.String()
}

func (m *ServerListModel) SetServers(servers []db.Server) {
	m.servers = servers
}

func (m ServerListModel) SelectedServer() *db.Server {
	if len(m.servers) == 0 || m.cursor >= len(m.servers) {
		return nil
	}
	return &m.servers[m.cursor]
}

func (m *ServerListModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *ServerListModel) MoveDown() {
	if m.cursor < len(m.servers)-1 {
		m.cursor++
	}
}

func (m *ServerListModel) SetDelMode(on bool) {
	m.delMode = on
	if on && len(m.servers) > 0 {
		m.delMsg = fmt.Sprintf("Delete %q? Press y to confirm, n to cancel.", m.servers[m.cursor].Name)
	}
}
