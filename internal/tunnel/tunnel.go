package tunnel

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

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

const (
	keepaliveInterval = 30 * time.Second
	reconnectDelay    = 5 * time.Second
	dialTimeout       = 15 * time.Second
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

// runTunnel runs a persistent tunnel with automatic reconnection.
// It loops forever until ti.cancel is closed.
func (m *Manager) runTunnel(server *db.Server, ti *TunnelInfo) {
	for {
		// Check for cancellation before each attempt.
		select {
		case <-ti.cancel:
			m.setStatus(ti, StatusStopped, "")
			return
		default:
		}

		m.setStatus(ti, StatusConnecting, "")

		client, err := dialSSH(server)
		if err != nil {
			m.setStatus(ti, StatusError, err.Error())
			select {
			case <-ti.cancel:
				m.setStatus(ti, StatusStopped, "")
				return
			case <-time.After(reconnectDelay):
				continue
			}
		}

		addr := fmt.Sprintf("127.0.0.1:%d", ti.Port.LocalPort)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			client.Close()
			m.setStatus(ti, StatusError, fmt.Sprintf("listen %s: %v", addr, err))
			select {
			case <-ti.cancel:
				m.setStatus(ti, StatusStopped, "")
				return
			case <-time.After(reconnectDelay):
				continue
			}
		}

		m.mu.Lock()
		ti.listener = ln
		m.mu.Unlock()

		m.setStatus(ti, StatusActive, "")

		// Keepalive goroutine: sends periodic pings to detect silent SSH drops.
		// If the keepalive times out or errors we close the listener, which
		// unblocks Accept and triggers a reconnect.
		go func(c *ssh.Client, l net.Listener) {
			ticker := time.NewTicker(keepaliveInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ti.cancel:
					return
				case <-ticker.C:
					if err := sendKeepAliveWithTimeout(c, 10*time.Second); err != nil {
						l.Close()
						c.Close()
						return
					}
				}
			}
		}(client, ln)

		// Cancel watcher: closes this iteration's resources when stopped.
		go func(c *ssh.Client, l net.Listener) {
			<-ti.cancel
			l.Close()
			c.Close()
		}(client, ln)

		// Accept loop: runs until the listener is closed (by keepalive failure,
		// cancel, or an external error).
		for {
			local, err := ln.Accept()
			if err != nil {
				select {
				case <-ti.cancel:
					m.setStatus(ti, StatusStopped, "")
					return
				default:
					m.setStatus(ti, StatusError, "connection lost, reconnecting…")
				}
				break
			}
			go m.handleConn(client, local, ti.Port)
		}

		client.Close()
		ln.Close()

		// Wait before reconnecting, but respect cancellation.
		select {
		case <-ti.cancel:
			m.setStatus(ti, StatusStopped, "")
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func (m *Manager) handleConn(client *ssh.Client, local net.Conn, port db.Port) {
	defer local.Close()
	remote, err := client.Dial("tcp", fmt.Sprintf("%s:%d", port.RemoteHost, port.RemotePort))
	if err != nil {
		return
	}
	defer remote.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(remote, local)
	}()
	go func() {
		defer wg.Done()
		io.Copy(local, remote)
	}()
	wg.Wait()
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
		Timeout:         dialTimeout,
	}

	addr := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.SSHPort))

	// Dial TCP manually so we can enable OS-level TCP keepalive.
	tcpConn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, err
	}
	if tcp, ok := tcpConn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(keepaliveInterval)
	}

	c, chans, reqs, err := ssh.NewClientConn(tcpConn, addr, cfg)
	if err != nil {
		tcpConn.Close()
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func sendKeepAliveWithTimeout(client *ssh.Client, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("keepalive timeout")
	}
}
