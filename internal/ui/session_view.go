package ui

import (
	"fmt"
	"strings"

	"github.com/adzin/port-forward-cli/internal/db"
	"github.com/adzin/port-forward-cli/internal/tunnel"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SessionViewModel struct {
	server  *db.Server
	ports   []db.Port
	manager *tunnel.Manager
	statuses map[int64]tunnel.TunnelUpdate
	width   int
	height  int
}

type TunnelUpdateMsg tunnel.TunnelUpdate

func NewSessionView(server *db.Server, ports []db.Port, mgr *tunnel.Manager) SessionViewModel {
	return SessionViewModel{
		server:   server,
		ports:    ports,
		manager:  mgr,
		statuses: make(map[int64]tunnel.TunnelUpdate),
	}
}

func (m SessionViewModel) Init() tea.Cmd {
	return waitForUpdate(m.manager)
}

func waitForUpdate(mgr *tunnel.Manager) tea.Cmd {
	return func() tea.Msg {
		update := <-mgr.Updates
		return TunnelUpdateMsg(update)
	}
}

func (m SessionViewModel) Update(msg tea.Msg) (SessionViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case TunnelUpdateMsg:
		m.statuses[msg.PortID] = tunnel.TunnelUpdate(msg)
		return m, waitForUpdate(m.manager)
	}
	return m, nil
}

func (m SessionViewModel) View() string {
	var b strings.Builder

	b.WriteString(StyleTitle.Render("  Active Tunnels"))
	b.WriteString("\n")
	b.WriteString(StyleSubtitle.Render(fmt.Sprintf("  Server: %s (%s@%s:%d)", m.server.Name, m.server.User, m.server.Host, m.server.SSHPort)))
	b.WriteString("\n\n")

	// Table header
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		StyleLabel.Width(22).Render(" Port Label"),
		StyleLabel.Width(20).Render(" Local Bind"),
		StyleLabel.Width(26).Render(" Remote Target"),
		StyleLabel.Width(14).Render(" Status"),
	)
	b.WriteString(StyleCard.Render(header))
	b.WriteString("\n")

	for _, p := range m.ports {
		statusBadge := StyleBadgeConnecting.Render(" connecting… ")
		if u, ok := m.statuses[p.ID]; ok {
			switch u.Status {
			case tunnel.StatusActive:
				statusBadge = StyleBadgeActive.Render("  active  ")
			case tunnel.StatusError:
				msg := u.ErrMsg
				if len(msg) > 20 {
					msg = msg[:20] + "…"
				}
				statusBadge = StyleBadgeError.Render(" ✗ " + msg + " ")
			case tunnel.StatusStopped:
				statusBadge = StyleBadgeStopped.Render(" stopped ")
			}
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			StyleValue.Width(22).Render(" "+p.Label),
			StyleValue.Width(20).Render(fmt.Sprintf(" 127.0.0.1:%d", p.LocalPort)),
			StyleValue.Width(26).Render(fmt.Sprintf(" %s:%d", p.RemoteHost, p.RemotePort)),
			statusBadge,
		)
		b.WriteString(StyleRow.Render(row))
		b.WriteString("\n")
	}

	b.WriteString(StyleHelp.Render(" q / ctrl+c:stop all tunnels & go back"))

	return b.String()
}
