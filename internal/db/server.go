package db

import "time"

type Server struct {
	ID            int64
	Name          string
	Host          string
	SSHPort       int
	User          string
	AuthType      string // "password" | "key"
	Password      string
	KeyPath       string
	KeyPassphrase string
	CreatedAt     time.Time
}

func ListServers() ([]Server, error) {
	rows, err := DB.Query(`SELECT id, name, host, ssh_port, user, auth_type, password, key_path, key_passphrase, created_at FROM servers ORDER BY id`)
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

func GetServer(id int64) (*Server, error) {
	var s Server
	err := DB.QueryRow(`SELECT id, name, host, ssh_port, user, auth_type, password, key_path, key_passphrase, created_at FROM servers WHERE id=?`, id).
		Scan(&s.ID, &s.Name, &s.Host, &s.SSHPort, &s.User, &s.AuthType, &s.Password, &s.KeyPath, &s.KeyPassphrase, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func CreateServer(s *Server) error {
	res, err := DB.Exec(
		`INSERT INTO servers (name, host, ssh_port, user, auth_type, password, key_path, key_passphrase) VALUES (?,?,?,?,?,?,?,?)`,
		s.Name, s.Host, s.SSHPort, s.User, s.AuthType, s.Password, s.KeyPath, s.KeyPassphrase,
	)
	if err != nil {
		return err
	}
	s.ID, _ = res.LastInsertId()
	return nil
}

func UpdateServer(s *Server) error {
	_, err := DB.Exec(
		`UPDATE servers SET name=?, host=?, ssh_port=?, user=?, auth_type=?, password=?, key_path=?, key_passphrase=? WHERE id=?`,
		s.Name, s.Host, s.SSHPort, s.User, s.AuthType, s.Password, s.KeyPath, s.KeyPassphrase, s.ID,
	)
	return err
}

func DeleteServer(id int64) error {
	_, err := DB.Exec(`DELETE FROM servers WHERE id=?`, id)
	return err
}
