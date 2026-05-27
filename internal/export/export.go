package export

import (
	"encoding/json"
	"os"
	"time"

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
	Password      string       `json:"password"`
	KeyPath       string       `json:"key_path"`
	KeyPassphrase string       `json:"key_passphrase"`
	Ports         []ExportPort `json:"ports"`
}

type ExportData struct {
	Version    int            `json:"version"`
	ExportedAt string         `json:"exported_at"`
	Servers    []ExportServer `json:"servers"`
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
		exportServers = append(exportServers, ExportServer{
			Name:          s.Name,
			Host:          s.Host,
			SSHPort:       s.SSHPort,
			User:          s.User,
			AuthType:      s.AuthType,
			Password:      s.Password,
			KeyPath:       s.KeyPath,
			KeyPassphrase: s.KeyPassphrase,
			Ports:         exportPorts,
		})
	}
	return ExportData{
		Version:    1,
		ExportedAt: now,
		Servers:    exportServers,
	}
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
