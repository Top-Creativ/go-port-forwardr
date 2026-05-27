package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/adzin/port-forward-cli/internal/db"
	"github.com/adzin/port-forward-cli/internal/export"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type exportState int

const (
	exportSelect exportState = iota
	exportPassword
)

type ExportSelectModel struct {
	state       exportState
	servers     []db.Server
	checked     map[int64]bool
	cursor      int
	result      string
	err         string
	exporting   bool
	hasCreds    bool
	password1   textinput.Model
	password2   textinput.Model
	passFocused int
	selectedIDs []int64
}

type ExportDoneMsg struct {
	Path        string
	ServerCount int
}

func NewExportSelect(servers []db.Server) ExportSelectModel {
	checked := make(map[int64]bool, len(servers))
	for _, s := range servers {
		checked[s.ID] = true
	}

	hasCreds := false
	for _, s := range servers {
		if s.Password != "" || s.KeyPath != "" || s.AuthType == "password" || s.AuthType == "key" {
			hasCreds = true
			break
		}
	}

	return ExportSelectModel{
		state:    exportSelect,
		servers:  servers,
		checked:  checked,
		hasCreds: hasCreds,
	}
}

func newExportPasswordInputs() (textinput.Model, textinput.Model) {
	p1 := textinput.New()
	p1.Placeholder = "Enter master password"
	p1.EchoMode = textinput.EchoPassword
	p1.EchoCharacter = '*'
	p1.CharLimit = 128
	p1.Width = 40
	p1.Focus()

	p2 := textinput.New()
	p2.Placeholder = "Confirm master password"
	p2.EchoMode = textinput.EchoPassword
	p2.EchoCharacter = '*'
	p2.CharLimit = 128
	p2.Width = 40

	return p1, p2
}

func (m ExportSelectModel) Init() tea.Cmd {
	return nil
}

func (m ExportSelectModel) Update(msg tea.Msg) (ExportSelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:

	case errMsg:
		m.err = msg.err.Error()
		m.exporting = false
	case ExportDoneMsg:
		m.result = fmt.Sprintf("Exported %d server(s) to %s", msg.ServerCount, msg.Path)
		m.exporting = false
		m.state = exportSelect
	case exportPasswordAccepted:
		m.exporting = true
		return m, DoExportEncrypted(m.selectedIDs, msg.password)
	}

	if m.state == exportPassword {
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.password1, cmd = m.password1.Update(msg)
		cmds = append(cmds, cmd)
		m.password2, cmd = m.password2.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m ExportSelectModel) View() string {
	var b strings.Builder

	if m.state == exportPassword {
		b.WriteString(StyleTitle.Render("  Export — Master Password"))
		b.WriteString("\n")
		b.WriteString(StyleSubtitle.Render(" Set a master password to encrypt credentials (optional)"))
		b.WriteString("\n\n")

		style1 := StyleFieldInactive
		style2 := StyleFieldInactive
		if m.passFocused == 0 {
			style1 = StyleFieldActive
		} else {
			style2 = StyleFieldActive
		}

		b.WriteString(fmt.Sprintf("  %s\n  %s\n\n", StyleLabel.Render("Password"), style1.Render(m.password1.View())))
		b.WriteString(fmt.Sprintf("  %s\n  %s\n\n", StyleLabel.Render("Confirm Password"), style2.Render(m.password2.View())))

		b.WriteString(StyleMuted.Render("  Tip: leave both empty to export without encryption"))
		b.WriteString("\n")

		if m.err != "" {
			b.WriteString("\n  " + StyleError.Render("✗ "+m.err) + "\n")
		}

		if m.exporting {
			b.WriteString("\n" + StyleMuted.Render("  Exporting..."))
			b.WriteString("\n")
		}

		b.WriteString(StyleHelp.Render(" tab/↑↓:switch field  ctrl+s:export  esc:back to select"))
		return b.String()
	}

	b.WriteString(StyleTitle.Render("  Export Servers"))
	b.WriteString("\n")
	b.WriteString(StyleSubtitle.Render(" Select servers to export  |  space:toggle  t:toggle all  ctrl+s:continue  esc:cancel"))
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
		b.WriteString(StyleSuccess.Render(fmt.Sprintf("  %d server(s) selected — press ctrl+s to continue", checkedCount)))
	}

	b.WriteString(StyleHelp.Render(" space:toggle  t:toggle all  ctrl+s:continue  esc:back"))
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

func (m ExportSelectModel) Password() (string, string) {
	return m.password1.Value(), m.password2.Value()
}

type exportPasswordAccepted struct {
	password string
}

func DoExportEncrypted(ids []int64, password string) tea.Cmd {
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

		if password != "" {
			if err := data.EncryptCredentials(password); err != nil {
				return errMsg{err}
			}
		}

		ts := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("port-forward-export-%s.pfexport", ts)

		if err := export.WriteExportFile(filename, data); err != nil {
			return errMsg{err}
		}

		return ExportDoneMsg{Path: filename, ServerCount: len(servers)}
	}
}
