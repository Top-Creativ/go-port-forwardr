package ui

import (
	"fmt"
	"strings"

	"github.com/adzin/port-forward-cli/internal/db"
	"github.com/adzin/port-forward-cli/internal/export"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type importState int

const (
	importPath importState = iota
	importPreview
)

type importServerItem struct {
	server   export.ExportServer
	checked  bool
	conflict bool
}

type ImportViewModel struct {
	state      importState
	pathInput  textinput.Model
	servers    []importServerItem
	cursor     int
	err        string
	result     string
	loading    bool
}

type ImportDoneMsg struct{}

func NewImportModel() ImportViewModel {
	ti := textinput.New()
	ti.Placeholder = "./port-forward-export-20260101-120000.pfexport"
	ti.CharLimit = 256
	ti.Width = 50
	ti.Focus()
	return ImportViewModel{
		state:     importPath,
		pathInput: ti,
	}
}

func (m ImportViewModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ImportViewModel) Update(msg tea.Msg) (ImportViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// no-op

	case errMsg:
		m.err = msg.err.Error()
		m.loading = false

	case ImportDoneMsg:
		m.result = "Import completed successfully."
		m.loading = false
	}

	if m.state == importPath {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ImportViewModel) View() string {
	var b strings.Builder

	if m.state == importPath {
		b.WriteString(StyleTitle.Render("  Import Servers"))
		b.WriteString("\n")
		b.WriteString(StyleSubtitle.Render(" Enter the path to a .pfexport file"))
		b.WriteString("\n\n")

		style := StyleFieldActive
		b.WriteString(fmt.Sprintf("  %s\n  %s\n", StyleLabel.Render("File path"), style.Render(m.pathInput.View())))

		if m.err != "" {
			b.WriteString("\n  " + StyleError.Render("✗ "+m.err) + "\n")
		}

		if m.loading {
			b.WriteString("\n  " + StyleMuted.Render("Loading...") + "\n")
		}

		b.WriteString(StyleHelp.Render(" enter/ctrl+s:load file  esc:cancel"))
		return b.String()
	}

	b.WriteString(StyleTitle.Render("  Import Preview"))
	b.WriteString("\n")
	b.WriteString(StyleSubtitle.Render(" Review servers to import  |  space:toggle  ctrl+s:import  esc:cancel"))
	b.WriteString("\n\n")

	if m.result != "" {
		b.WriteString(StyleCard.Render(StyleSuccess.Render(m.result)))
		b.WriteString("\n")
	}

	if m.err != "" {
		b.WriteString(StyleError.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	if m.loading {
		b.WriteString("\n  " + StyleMuted.Render("Importing...") + "\n")
	}

	if len(m.servers) == 0 {
		b.WriteString(StyleMuted.Render("  No servers found in export file."))
		b.WriteString("\n")
	} else {
		for i, item := range m.servers {
			check := StyleUnchecked.Render("☐")
			if item.checked {
				check = StyleChecked.Render("☑")
			}

			conflictBadge := ""
			if item.conflict {
				conflictBadge = " " + StyleError.Render("[duplicate → (imported)]")
			}

			authBadge := StyleMuted.Render("[pw]")
			if item.server.AuthType == "key" {
				authBadge = StyleAccent.Render("[key]")
			}

			row := fmt.Sprintf("%s %s  %-20s  %s@%s:%d  %d port(s)%s",
				check,
				authBadge,
				item.server.Name,
				item.server.User,
				item.server.Host,
				item.server.SSHPort,
				len(item.server.Ports),
				conflictBadge,
			)

			if i == m.cursor {
				b.WriteString(StyleSelected.Render(fmt.Sprintf("▶ %s", row)))
			} else {
				b.WriteString(StyleRow.Render(fmt.Sprintf("  %s", row)))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(StyleHelp.Render(" space:toggle  ctrl+s:import  esc:cancel"))
	return b.String()
}

func (m ImportViewModel) FilePath() string {
	return strings.TrimSpace(m.pathInput.Value())
}

func (m *ImportViewModel) SetServers(items []importServerItem) {
	m.servers = items
}

func (m *ImportViewModel) ToggleCurrent() {
	if len(m.servers) == 0 {
		return
	}
	m.servers[m.cursor].checked = !m.servers[m.cursor].checked
}

func (m *ImportViewModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *ImportViewModel) MoveDown() {
	if m.cursor < len(m.servers)-1 {
		m.cursor++
	}
}

func (m ImportViewModel) SelectedServers() []export.ExportServer {
	var out []export.ExportServer
	for _, s := range m.servers {
		if s.checked {
			out = append(out, s.server)
		}
	}
	return out
}

func LoadImportFile(path string) tea.Cmd {
	return func() tea.Msg {
		data, err := export.ImportFromFile(path)
		if err != nil {
			return errMsg{err}
		}
		if !data.HasServers() {
			return errMsg{fmt.Errorf("no servers found in export file")}
		}

		items := make([]importServerItem, len(data.Servers))
		for i, s := range data.Servers {
			existing, err := db.FindServerByName(s.Name)
			conflict := err == nil && existing != nil
			name := s.Name
			if conflict {
				name = fmt.Sprintf("%s (imported)", s.Name)
				items[i].server = s
				items[i].server.Name = name
			} else {
				items[i].server = s
			}
			items[i].checked = true
			items[i].conflict = conflict
		}

		return importPreviewReady{items: items}
	}
}

type importPreviewReady struct {
	items []importServerItem
}

func DoImport(servers []export.ExportServer) tea.Cmd {
	return func() tea.Msg {
		for _, s := range servers {
			dbServer := &db.Server{
				Name:          s.Name,
				Host:          s.Host,
				SSHPort:       s.SSHPort,
				User:          s.User,
				AuthType:      s.AuthType,
				Password:      s.Password,
				KeyPath:       s.KeyPath,
				KeyPassphrase: s.KeyPassphrase,
			}

			ports := make([]db.Port, len(s.Ports))
			for j, p := range s.Ports {
				ports[j] = db.Port{
					Label:      p.Label,
					LocalPort:  p.LocalPort,
					RemoteHost: p.RemoteHost,
					RemotePort: p.RemotePort,
				}
			}

			if err := db.BulkCreateServer(dbServer, ports); err != nil {
				return errMsg{err}
			}
		}
		return ImportDoneMsg{}
	}
}
