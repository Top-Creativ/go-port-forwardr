package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/adzin/port-forward-cli/internal/db"
	"github.com/adzin/port-forward-cli/internal/export"
	tea "github.com/charmbracelet/bubbletea"
)

type ExportSelectModel struct {
	servers   []db.Server
	checked   map[int64]bool
	cursor    int
	result    string
	err       string
	exporting bool
}

type ExportDoneMsg struct {
	Path       string
	ServerCount int
}

func NewExportSelect(servers []db.Server) ExportSelectModel {
	checked := make(map[int64]bool, len(servers))
	for _, s := range servers {
		checked[s.ID] = true
	}
	return ExportSelectModel{
		servers: servers,
		checked: checked,
	}
}

func (m ExportSelectModel) Init() tea.Cmd {
	return nil
}

func (m ExportSelectModel) Update(msg tea.Msg) (ExportSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// no-op, sizes not used
	case errMsg:
		m.err = msg.err.Error()
		m.exporting = false
	case ExportDoneMsg:
		m.result = fmt.Sprintf("Exported %d server(s) to %s", msg.ServerCount, msg.Path)
		m.exporting = false
	}
	return m, nil
}

func (m ExportSelectModel) View() string {
	var b strings.Builder
	b.WriteString(StyleTitle.Render("  Export Servers"))
	b.WriteString("\n")
	b.WriteString(StyleSubtitle.Render(" Select servers to export  |  space:toggle  t:toggle all  ctrl+s:save  esc:cancel"))
	b.WriteString("\n\n")

	if m.result != "" {
		b.WriteString(StyleCard.Render(StyleSuccess.Render(m.result)))
		b.WriteString("\n")
	}

	if m.err != "" {
		b.WriteString(StyleError.Render("Error: " + m.err))
		b.WriteString("\n")
	}

	if len(m.servers) == 0 {
		b.WriteString(StyleMuted.Render("  No servers to export."))
		b.WriteString("\n")
	} else {
		for i, s := range m.servers {
			check := StyleUnchecked.Render("☐")
			if m.checked[s.ID] {
				check = StyleChecked.Render("☑")
			}

			authBadge := StyleMuted.Render("[pw]")
			if s.AuthType == "key" {
				authBadge = StyleAccent.Render("[key]")
			}

			row := fmt.Sprintf("%s %s  %-20s  %s@%s:%d",
				check,
				authBadge,
				s.Name,
				s.User,
				s.Host,
				s.SSHPort,
			)

			if i == m.cursor {
				b.WriteString(StyleSelected.Render(fmt.Sprintf("▶ %s", row)))
			} else {
				b.WriteString(StyleRow.Render(fmt.Sprintf("  %s", row)))
			}
			b.WriteString("\n")
		}
	}

	if m.exporting {
		b.WriteString("\n")
		b.WriteString(StyleMuted.Render("  Exporting..."))
		b.WriteString("\n")
	}

	checkedCount := 0
	for _, v := range m.checked {
		if v {
			checkedCount++
		}
	}
	if checkedCount > 0 && !m.exporting && m.result == "" {
		b.WriteString("\n")
		b.WriteString(StyleSuccess.Render(fmt.Sprintf("  %d server(s) selected — press ctrl+s to export", checkedCount)))
	}

	b.WriteString(StyleHelp.Render(" space:toggle  t:toggle all  ctrl+s:export  esc:back"))
	return b.String()
}

func (m ExportSelectModel) SelectedServerIDs() []int64 {
	var ids []int64
	for _, s := range m.servers {
		if m.checked[s.ID] {
			ids = append(ids, s.ID)
		}
	}
	return ids
}

func (m *ExportSelectModel) ToggleCurrent() {
	if len(m.servers) == 0 {
		return
	}
	id := m.servers[m.cursor].ID
	m.checked[id] = !m.checked[id]
}

func (m *ExportSelectModel) ToggleAll() {
	if len(m.servers) == 0 {
		return
	}
	allChecked := true
	for _, s := range m.servers {
		if !m.checked[s.ID] {
			allChecked = false
			break
		}
	}
	for _, s := range m.servers {
		m.checked[s.ID] = !allChecked
	}
}

func (m *ExportSelectModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *ExportSelectModel) MoveDown() {
	if m.cursor < len(m.servers)-1 {
		m.cursor++
	}
}

func DoExport(ids []int64) tea.Cmd {
	return func() tea.Msg {
		servers, err := db.GetServersByIDs(ids)
		if err != nil {
			return errMsg{err}
		}

		portsByServer := make(map[int64][]db.Port, len(servers))
		for _, s := range servers {
			ports, err := db.ListPorts(s.ID)
			if err != nil {
				return errMsg{err}
			}
			portsByServer[s.ID] = ports
		}

		data := export.FromDBServers(servers, portsByServer)

		ts := time.Now().Format("20060102-150405")
		filename := filepath.Join(".", fmt.Sprintf("port-forward-export-%s.pfexport", ts))

		if err := export.WriteExportFile(filename, data); err != nil {
			return errMsg{err}
		}

		return ExportDoneMsg{Path: filename, ServerCount: len(servers)}
	}
}
