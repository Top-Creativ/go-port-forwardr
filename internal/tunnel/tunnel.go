package tunnel

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/adzin/port-forward-cli/internal/db"
	"golang.org/x/crypto/ssh"
)

type Status string

const (
	StatusConnecting Status = "connecting"
	StatusActive     Status = "active"
	StatusError      Status = "error"
	StatusStopped    Status = "stopped"
)

type TunnelInfo struct {
	Port     db.Port
	Status   Status
	ErrMsg   string
	cancel   chan struct{}
	listener net.Listener
}

type Manager struct {
	mu      sync.Mutex
	tunnels map[int64]*TunnelInfo
	Updates chan TunnelUpdate
}

type TunnelUpdate struct {
	PortID int64
	Status Status
	ErrMsg string
}

func NewManager() *Manager {
	return &Manager{
		tunnels: make(map[int64]*TunnelInfo),
		Updates: make(chan TunnelUpdate, 32),
	}
}

func (m *Manager) Start(server *db.Server, ports []db.Port) {
	for _, p := range ports {
		p := p
		m.mu.Lock()
		if _, exists := m.tunnels[p.ID]; exists {
			m.mu.Unlock()
			continue
		}
		ti := &TunnelInfo{Port: p, Status: StatusConnecting, cancel: make(chan struct{})}
		m.tunnels[p.ID] = ti
		m.mu.Unlock()

		go m.runTunnel(server, ti)
	}
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ti := range m.tunnels {
		close(ti.cancel)
		if ti.listener != nil {
			ti.listener.Close()
		}
	}
	m.tunnels = make(map[int64]*TunnelInfo)
}

func (m *Manager) Stop(portID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ti, ok := m.tunnels[portID]; ok {
		close(ti.cancel)
		if ti.listener != nil {
			ti.listener.Close()
		}
		delete(m.tunnels, portID)
	}
}

func (m *Manager) List() []TunnelInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]TunnelInfo, 0, len(m.tunnels))
	for _, ti := range m.tunnels {
		out = append(out, *ti)
	}
	return out
}

func (m *Manager) runTunnel(server *db.Server, ti *TunnelInfo) {
	client, err := dialSSH(server)
	if err != nil {
		m.setStatus(ti, StatusError, err.Error())
		return
	}
	defer client.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", ti.Port.LocalPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		m.setStatus(ti, StatusError, fmt.Sprintf("listen %s: %v", addr, err))
		return
	}

	m.mu.Lock()
	ti.listener = ln
	m.mu.Unlock()

	m.setStatus(ti, StatusActive, "")

	go func() {
		<-ti.cancel
		ln.Close()
		client.Close()
	}()

	for {
		local, err := ln.Accept()
		if err != nil {
			select {
			case <-ti.cancel:
				m.setStatus(ti, StatusStopped, "")
			default:
				m.setStatus(ti, StatusError, err.Error())
			}
			return
		}
		go m.handleConn(client, local, ti.Port)
	}
}

func (m *Manager) handleConn(client *ssh.Client, local net.Conn, port db.Port) {
	defer local.Close()
	remote, err := client.Dial("tcp", fmt.Sprintf("%s:%d", port.RemoteHost, port.RemotePort))
	if err != nil {
		return
	}
	defer remote.Close()
	done := make(chan struct{}, 2)
	go func() { io.Copy(remote, local); done <- struct{}{} }()
	go func() { io.Copy(local, remote); done <- struct{}{} }()
	<-done
}

func (m *Manager) setStatus(ti *TunnelInfo, s Status, msg string) {
	m.mu.Lock()
	ti.Status = s
	ti.ErrMsg = msg
	m.mu.Unlock()
	m.Updates <- TunnelUpdate{PortID: ti.Port.ID, Status: s, ErrMsg: msg}
}

func dialSSH(server *db.Server) (*ssh.Client, error) {
	var auth []ssh.AuthMethod

	switch server.AuthType {
	case "key":
		keyData, err := os.ReadFile(server.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %q: %w", server.KeyPath, err)
		}
		var signer ssh.Signer
		if server.KeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(server.KeyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	default: // "password"
		auth = append(auth, ssh.Password(server.Password))
	}

	cfg := &ssh.ClientConfig{
		User:            server.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
	}

	addr := fmt.Sprintf("%s:%d", server.Host, server.SSHPort)
	return ssh.Dial("tcp", addr, cfg)
}
