package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/adzin/port-forward-cli/internal/db"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type PortSelectorModel struct {
	serverID int64
	ports    []db.Port
	checked  map[int64]bool
	cursor   int
	addMode  bool
	delMode  bool
	addForm  portAddForm
	err      string
	width    int
	height   int
}

type PortsLoadedMsg struct{ Ports []db.Port }
type PortAddedMsg struct{}
type PortDeletedMsg struct{}

type portAddForm struct {
	inputs  []textinput.Model
	focused int
}

const (
	pfLabel = iota
	pfLocalPort
	pfRemoteHost
	pfRemotePort
	pfCount
)

func newPortAddForm() portAddForm {
	mkInput := func(ph string, limit int) textinput.Model {
		t := textinput.New()
		t.Placeholder = ph
		t.CharLimit = limit
		t.Width = 30
		return t
	}
	inputs := []textinput.Model{
		mkInput("Database 5432", 64),
		mkInput("5432", 5),
		mkInput("localhost", 128),
		mkInput("5432", 5),
	}
	inputs[0].Focus()
	return portAddForm{inputs: inputs}
}

func (f *portAddForm) Update(msg tea.Msg) (portAddForm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			f.inputs[f.focused].Blur()
			f.focused = (f.focused + 1) % pfCount
			f.inputs[f.focused].Focus()
			return *f, nil
		case "shift+tab", "up":
			f.inputs[f.focused].Blur()
			f.focused = (f.focused - 1 + pfCount) % pfCount
			f.inputs[f.focused].Focus()
			return *f, nil
		}
	}
	var cmd tea.Cmd
	f.inputs[f.focused], cmd = f.inputs[f.focused].Update(msg)
	return *f, cmd
}

func (f portAddForm) Validate() (db.Port, error) {
	label := strings.TrimSpace(f.inputs[pfLabel].Value())
	lp, err1 := strconv.Atoi(strings.TrimSpace(f.inputs[pfLocalPort].Value()))
	rh := strings.TrimSpace(f.inputs[pfRemoteHost].Value())
	rp, err2 := strconv.Atoi(strings.TrimSpace(f.inputs[pfRemotePort].Value()))
	if label == "" {
		return db.Port{}, fmt.Errorf("label required")
	}
	if err1 != nil || lp < 1 || lp > 65535 {
		return db.Port{}, fmt.Errorf("invalid local port")
	}
	if rh == "" {
		return db.Port{}, fmt.Errorf("remote host required")
	}
	if err2 != nil || rp < 1 || rp > 65535 {
		return db.Port{}, fmt.Errorf("invalid remote port")
	}
	return db.Port{Label: label, LocalPort: lp, RemoteHost: rh, RemotePort: rp}, nil
}

func NewPortSelector(serverID int64) PortSelectorModel {
	return PortSelectorModel{serverID: serverID, checked: make(map[int64]bool)}
}

func loadPorts(serverID int64) tea.Cmd {
	return func() tea.Msg {
		ports, err := db.ListPorts(serverID)
		if err != nil {
			return errMsg{err}
		}
		return PortsLoadedMsg{ports}
	}
}

func (m PortSelectorModel) Init() tea.Cmd {
	return loadPorts(m.serverID)
}

func (m PortSelectorModel) Update(msg tea.Msg) (PortSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case PortsLoadedMsg:
		m.ports = msg.Ports
		if m.cursor >= len(m.ports) {
			m.cursor = max(0, len(m.ports)-1)
		}
	case PortAddedMsg:
		m.addMode = false
		m.err = ""
		return m, loadPorts(m.serverID)
	case PortDeletedMsg:
		m.delMode = false
		m.err = ""
		return m, loadPorts(m.serverID)
	case errMsg:
		m.err = msg.err.Error()
	}

	if m.addMode {
		var cmd tea.Cmd
		m.addForm, cmd = m.addForm.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m PortSelectorModel) View() string {
	var b strings.Builder
	b.WriteString(StyleTitle.Render("  Port Forwarding"))
	b.WriteString("\n")
	b.WriteString(StyleSubtitle.Render(fmt.Sprintf("  Select ports to forward for server (id=%d)", m.serverID)))
	b.WriteString("\n\n")

	if m.addMode {
		b.WriteString(StyleCard.Render(m.renderAddForm()))
	} else {
		if len(m.ports) == 0 {
			b.WriteString(StyleMuted.Render("  No ports configured. Press ") + StyleAccent.Render("a") + StyleMuted.Render(" to add one."))
			b.WriteString("\n")
		} else {
			for i, p := range m.ports {
				check := StyleUnchecked.Render("☐")
				if m.checked[p.ID] {
					check = StyleChecked.Render("☑")
				}

				row := fmt.Sprintf("%s  %-20s  127.0.0.1:%-5d → %s:%d",
					check, p.Label, p.LocalPort, p.RemoteHost, p.RemotePort)

				if i == m.cursor {
					b.WriteString(StyleSelected.Render(fmt.Sprintf("▶ %s", row)))
				} else {
					b.WriteString(StyleRow.Render(fmt.Sprintf("  %s", row)))
				}
				b.WriteString("\n")
			}
		}

		if m.delMode && len(m.ports) > 0 {
			b.WriteString("\n")
			b.WriteString(StyleError.Render(fmt.Sprintf("  ⚠  Delete %q? y to confirm, n to cancel.", m.ports[m.cursor].Label)))
			b.WriteString("\n")
		}
	}

	if m.err != "" {
		b.WriteString("\n  " + StyleError.Render("✗ "+m.err) + "\n")
	}

	checkedCount := 0
	for _, v := range m.checked {
		if v {
			checkedCount++
		}
	}

	help := " ↑↓:navigate  space:toggle  a:add  d:delete  enter:start tunnels  esc:back"
	if m.addMode {
		help = " tab:next field  ctrl+s:save port  esc:cancel"
	}
	b.WriteString(StyleHelp.Render(help))

	if checkedCount > 0 && !m.addMode {
		b.WriteString("\n")
		b.WriteString(StyleSuccess.Render(fmt.Sprintf("  %d port(s) selected — press enter to start forwarding", checkedCount)))
	}

	return b.String()
}

func (m PortSelectorModel) renderAddForm() string {
	var b strings.Builder
	b.WriteString(StyleLabel.Render("Add New Port Rule") + "\n\n")

	fields := []struct {
		label string
		idx   int
	}{
		{"Label", pfLabel},
		{"Local Port (bind on your machine)", pfLocalPort},
		{"Remote Host", pfRemoteHost},
		{"Remote Port (on the server)", pfRemotePort},
	}

	for _, f := range fields {
		style := StyleFieldInactive
		if m.addForm.focused == f.idx {
			style = StyleFieldActive
		}
		b.WriteString(fmt.Sprintf("  %s\n  %s\n", StyleLabel.Render(f.label), style.Render(m.addForm.inputs[f.idx].View())))
	}
	return b.String()
}

func (m PortSelectorModel) SelectedPorts() []db.Port {
	var out []db.Port
	for _, p := range m.ports {
		if m.checked[p.ID] {
			out = append(out, p)
		}
	}
	return out
}

func (m *PortSelectorModel) ToggleCurrent() {
	if len(m.ports) == 0 {
		return
	}
	id := m.ports[m.cursor].ID
	m.checked[id] = !m.checked[id]
}

func (m *PortSelectorModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *PortSelectorModel) MoveDown() {
	if m.cursor < len(m.ports)-1 {
		m.cursor++
	}
}

func (m *PortSelectorModel) EnterAddMode() {
	m.addMode = true
	m.addForm = newPortAddForm()
}

func (m *PortSelectorModel) ExitAddMode() {
	m.addMode = false
	m.err = ""
}

func (m *PortSelectorModel) SetDelMode(on bool) {
	m.delMode = on
}

func (m *PortSelectorModel) IsAddMode() bool { return m.addMode }
func (m *PortSelectorModel) IsDelMode() bool { return m.delMode }

func (m PortSelectorModel) AddFormValidate() (db.Port, error) { return m.addForm.Validate() }

func (m PortSelectorModel) CurrentPort() *db.Port {
	if len(m.ports) == 0 || m.cursor >= len(m.ports) {
		return nil
	}
	return &m.ports[m.cursor]
}
