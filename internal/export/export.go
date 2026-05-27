package export

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/adzin/port-forward-cli/internal/crypto"
	"github.com/adzin/port-forward-cli/internal/db"
)

type ExportPort struct {
	Label      string `json:"label"`
	LocalPort  int    `json:"local_port"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
}

type ExportServer struct {
	Name          string       `json:"name"`
	Host          string       `json:"host"`
	SSHPort       int          `json:"ssh_port"`
	User          string       `json:"user"`
	AuthType      string       `json:"auth_type"`
	Password      string       `json:"password,omitempty"`
	KeyPath       string       `json:"key_path,omitempty"`
	KeyPassphrase string       `json:"key_passphrase,omitempty"`
	Ports         []ExportPort `json:"ports"`
}

// ServerCredentials holds the sensitive fields that get encrypted.
type ServerCredentials struct {
	Password      string `json:"password,omitempty"`
	KeyPath       string `json:"key_path,omitempty"`
	KeyPassphrase string `json:"key_passphrase,omitempty"`
}

type ExportData struct {
	Version     int                      `json:"version"`
	ExportedAt  string                   `json:"exported_at"`
	Encrypted   bool                     `json:"encrypted"`
	Credentials string                   `json:"credentials,omitempty"`
	Salt        string                   `json:"salt,omitempty"`
	Servers     []ExportServer           `json:"servers"`
}

func FromDBServers(servers []db.Server, portsByServer map[int64][]db.Port) ExportData {
	now := time.Now().Format(time.RFC3339)
	exportServers := make([]ExportServer, 0, len(servers))
	for _, s := range servers {
		ports := portsByServer[s.ID]
		exportPorts := make([]ExportPort, 0, len(ports))
		for _, p := range ports {
			exportPorts = append(exportPorts, ExportPort{
				Label:      p.Label,
				LocalPort:  p.LocalPort,
				RemoteHost: p.RemoteHost,
				RemotePort: p.RemotePort,
			})
		}
		es := ExportServer{
			Name:          s.Name,
			Host:          s.Host,
			SSHPort:       s.SSHPort,
			User:          s.User,
			AuthType:      s.AuthType,
			Password:      s.Password,
			KeyPath:       s.KeyPath,
			KeyPassphrase: s.KeyPassphrase,
			Ports:         exportPorts,
		}
		exportServers = append(exportServers, es)
	}
	return ExportData{
		Version:    2,
		ExportedAt: now,
		Servers:    exportServers,
	}
}

func (e *ExportData) EncryptCredentials(password string) error {
	creds := make(map[string]ServerCredentials, len(e.Servers))
	for i := range e.Servers {
		s := &e.Servers[i]
		key := fmt.Sprintf("%d", i)
		creds[key] = ServerCredentials{
			Password:      s.Password,
			KeyPath:       s.KeyPath,
			KeyPassphrase: s.KeyPassphrase,
		}
		s.Password = ""
		s.KeyPath = ""
		s.KeyPassphrase = ""
	}
	plainJSON, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	cipherB64, saltHex, err := crypto.Encrypt(string(plainJSON), password)
	if err != nil {
		return err
	}
	e.Encrypted = true
	e.Credentials = cipherB64
	e.Salt = saltHex
	return nil
}

func (e *ExportData) DecryptCredentials(password string) error {
	if !e.Encrypted {
		return nil
	}
	plainJSON, err := crypto.Decrypt(e.Credentials, password, e.Salt)
	if err != nil {
		return fmt.Errorf("decrypt credentials: %w", err)
	}
	var creds map[string]ServerCredentials
	if err := json.Unmarshal([]byte(plainJSON), &creds); err != nil {
		return fmt.Errorf("invalid credentials data: %w", err)
	}
	for i := range e.Servers {
		key := fmt.Sprintf("%d", i)
		if entry, ok := creds[key]; ok {
			e.Servers[i].Password = entry.Password
			e.Servers[i].KeyPath = entry.KeyPath
			e.Servers[i].KeyPassphrase = entry.KeyPassphrase
		}
	}
	return nil
}

func Marshal(data ExportData) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

func WriteExportFile(filepath string, data ExportData) error {
	bytes, err := Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, bytes, 0644)
}

func ImportFromFile(filepath string) (*ExportData, error) {
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return ImportFromBytes(bytes)
}

func ImportFromBytes(data []byte) (*ExportData, error) {
	var export ExportData
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, err
	}
	return &export, nil
}

func (e *ExportData) HasServers() bool {
	return len(e.Servers) > 0
}

func (e *ExportData) TotalPorts() int {
	n := 0
	for _, s := range e.Servers {
		n += len(s.Ports)
	}
	return n
}
