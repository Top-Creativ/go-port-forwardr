package db

import (
	"database/sql"
	"fmt"
)

func FindServerByName(name string) (*Server, error) {
	var s Server
	err := DB.QueryRow(
		`SELECT id, name, host, ssh_port, user, auth_type, password, key_path, key_passphrase, created_at FROM servers WHERE name=?`,
		name,
	).Scan(&s.ID, &s.Name, &s.Host, &s.SSHPort, &s.User, &s.AuthType, &s.Password, &s.KeyPath, &s.KeyPassphrase, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetServersByIDs(ids []int64) ([]Server, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := DB.Query(
		`SELECT id, name, host, ssh_port, user, auth_type, password, key_path, key_passphrase, created_at FROM servers WHERE id IN (`+placeholders(len(ids))+`) ORDER BY id`,
		toAny(ids)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var servers []Server
	for rows.Next() {
		var s Server
		if err := rows.Scan(&s.ID, &s.Name, &s.Host, &s.SSHPort, &s.User, &s.AuthType, &s.Password, &s.KeyPath, &s.KeyPassphrase, &s.CreatedAt); err != nil {
			return nil, err
		}
		servers = append(servers, s)
	}
	return servers, nil
}

func BulkCreateServer(s *Server, ports []Port) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO servers (name, host, ssh_port, user, auth_type, password, key_path, key_passphrase) VALUES (?,?,?,?,?,?,?,?)`,
		s.Name, s.Host, s.SSHPort, s.User, s.AuthType, s.Password, s.KeyPath, s.KeyPassphrase,
	)
	if err != nil {
		return fmt.Errorf("insert server: %w", err)
	}
	serverID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}

	for _, p := range ports {
		_, err := tx.Exec(
			`INSERT INTO ports (server_id, label, local_port, remote_host, remote_port) VALUES (?,?,?,?,?)`,
			serverID, p.Label, p.LocalPort, p.RemoteHost, p.RemotePort,
		)
		if err != nil {
			return fmt.Errorf("insert port: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	s.ID = serverID
	return nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, 0, n*2-1)
	for i := 0; i < n; i++ {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '?')
	}
	return string(b)
}

func toAny(ids []int64) []interface{} {
	out := make([]interface{}, len(ids))
	for i, v := range ids {
		out[i] = v
	}
	return out
}
