package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/adzin/port-forward-cli/internal/db"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type authMode int

const (
	authPassword authMode = iota
	authKey
)

type ServerFormModel struct {
	inputs    []textinput.Model
	authMode  authMode
	focused   int
	editID    int64 // 0 = new
	err       string
	submitted bool
}

const (
	fieldName = iota
	fieldHost
	fieldSSHPort
	fieldUser
	fieldAuth // toggle row
	fieldPassword
	fieldKeyPath
	fieldKeyPassphrase
	fieldCount
)

func NewServerForm(existing *db.Server) ServerFormModel {
	inputs := make([]textinput.Model, fieldCount)

	mkInput := func(placeholder string, limit int) textinput.Model {
		t := textinput.New()
		t.Placeholder = placeholder
		t.CharLimit = limit
		t.Width = 40
		return t
	}

	inputs[fieldName] = mkInput("My Production Server", 64)
	inputs[fieldHost] = mkInput("192.168.1.100 or hostname", 128)
	inputs[fieldSSHPort] = mkInput("22", 5)
	inputs[fieldUser] = mkInput("ubuntu", 64)
	inputs[fieldAuth] = mkInput("", 0) // unused — toggle only
	inputs[fieldPassword] = mkInput("SSH password", 128)
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].EchoCharacter = '•'
	inputs[fieldKeyPath] = mkInput("~/.ssh/id_rsa", 256)
	inputs[fieldKeyPassphrase] = mkInput("key passphrase (optional)", 128)
	inputs[fieldKeyPassphrase].EchoMode = textinput.EchoPassword
	inputs[fieldKeyPassphrase].EchoCharacter = '•'

	m := ServerFormModel{inputs: inputs, authMode: authPassword, focused: fieldName}

	if existing != nil {
		m.editID = existing.ID
		inputs[fieldName].SetValue(existing.Name)
		inputs[fieldHost].SetValue(existing.Host)
		inputs[fieldSSHPort].SetValue(strconv.Itoa(existing.SSHPort))
		inputs[fieldUser].SetValue(existing.User)
		inputs[fieldPassword].SetValue(existing.Password)
		inputs[fieldKeyPath].SetValue(existing.KeyPath)
		inputs[fieldKeyPassphrase].SetValue(existing.KeyPassphrase)
		if existing.AuthType == "key" {
			m.authMode = authKey
		}
	} else {
		inputs[fieldSSHPort].SetValue("22")
	}

	inputs[m.focused].Focus()
	return m
}

func (m ServerFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ServerFormModel) Update(msg tea.Msg) (ServerFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			m.moveFocus(1)
			return m, nil
		case "shift+tab", "up":
			m.moveFocus(-1)
			return m, nil
		case " ", "enter":
			if m.focused == fieldAuth {
				if m.authMode == authPassword {
					m.authMode = authKey
				} else {
					m.authMode = authPassword
				}
				return m, nil
			}
		}
	}

	// Forward to focused input
	if m.focused != fieldAuth {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *ServerFormModel) moveFocus(dir int) {
	m.inputs[m.focused].Blur()
	m.focused = (m.focused + dir + fieldCount) % fieldCount
	// Skip irrelevant auth fields
	if m.authMode == authPassword && (m.focused == fieldKeyPath || m.focused == fieldKeyPassphrase) {
		m.focused = (m.focused + dir + fieldCount) % fieldCount
	}
	if m.authMode == authKey && m.focused == fieldPassword {
		m.focused = (m.focused + dir + fieldCount) % fieldCount
	}
	m.inputs[m.focused].Focus()
}

func (m ServerFormModel) Validate() (db.Server, error) {
	name := strings.TrimSpace(m.inputs[fieldName].Value())
	host := strings.TrimSpace(m.inputs[fieldHost].Value())
	portStr := strings.TrimSpace(m.inputs[fieldSSHPort].Value())
	user := strings.TrimSpace(m.inputs[fieldUser].Value())

	if name == "" {
		return db.Server{}, fmt.Errorf("name is required")
	}
	if host == "" {
		return db.Server{}, fmt.Errorf("host is required")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return db.Server{}, fmt.Errorf("invalid SSH port")
	}
	if user == "" {
		return db.Server{}, fmt.Errorf("user is required")
	}

	s := db.Server{
		ID:      m.editID,
		Name:    name,
		Host:    host,
		SSHPort: port,
		User:    user,
	}

	if m.authMode == authPassword {
		s.AuthType = "password"
		s.Password = m.inputs[fieldPassword].Value()
		if s.Password == "" {
			return db.Server{}, fmt.Errorf("password is required")
		}
	} else {
		s.AuthType = "key"
		s.KeyPath = strings.TrimSpace(m.inputs[fieldKeyPath].Value())
		if s.KeyPath == "" {
			return db.Server{}, fmt.Errorf("key path is required")
		}
		s.KeyPassphrase = m.inputs[fieldKeyPassphrase].Value()
	}

	return s, nil
}

func (m ServerFormModel) View() string {
	var b strings.Builder

	title := "  Add New Server"
	if m.editID != 0 {
		title = "  Edit Server"
	}
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n\n")

	renderField := func(label string, idx int, val string) string {
		style := StyleFieldInactive
		if m.focused == idx {
			style = StyleFieldActive
		}
		var content string
		if idx == fieldAuth {
			// Render toggle
			pw := StyleMuted.Render("[ password ]")
			key := StyleMuted.Render("[ key file ]")
			if m.authMode == authPassword {
				pw = StyleChecked.Render("● password ")
			} else {
				key = StyleChecked.Render("● key file ")
			}
			content = lipgloss.JoinHorizontal(lipgloss.Center, pw, "  ", key)
		} else {
			content = m.inputs[idx].View()
		}
		fieldStr := style.Render(content)
		return fmt.Sprintf("  %s\n  %s\n", StyleLabel.Render(label), fieldStr)
	}

	b.WriteString(renderField("Name", fieldName, ""))
	b.WriteString(renderField("Host / IP", fieldHost, ""))
	b.WriteString(renderField("SSH Port", fieldSSHPort, ""))
	b.WriteString(renderField("Username", fieldUser, ""))
	b.WriteString(renderField("Auth Method  (space/enter to toggle)", fieldAuth, ""))

	if m.authMode == authPassword {
		b.WriteString(renderField("Password", fieldPassword, ""))
	} else {
		b.WriteString(renderField("Private Key Path", fieldKeyPath, ""))
		b.WriteString(renderField("Key Passphrase", fieldKeyPassphrase, ""))
	}

	if m.err != "" {
		b.WriteString("\n  " + StyleError.Render("✗ "+m.err) + "\n")
	}

	b.WriteString(StyleHelp.Render(" tab/↑↓:navigate  ctrl+s:save  esc:cancel"))
	return b.String()
}

func (m *ServerFormModel) SetError(e string) { m.err = e }
