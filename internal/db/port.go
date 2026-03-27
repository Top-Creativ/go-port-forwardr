package db

import "time"

type Port struct {
	ID         int64
	ServerID   int64
	Label      string
	LocalPort  int
	RemoteHost string
	RemotePort int
	CreatedAt  time.Time
}

func ListPorts(serverID int64) ([]Port, error) {
	rows, err := DB.Query(`SELECT id, server_id, label, local_port, remote_host, remote_port, created_at FROM ports WHERE server_id=? ORDER BY id`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ports []Port
	for rows.Next() {
		var p Port
		if err := rows.Scan(&p.ID, &p.ServerID, &p.Label, &p.LocalPort, &p.RemoteHost, &p.RemotePort, &p.CreatedAt); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}
	return ports, nil
}

func CreatePort(p *Port) error {
	res, err := DB.Exec(
		`INSERT INTO ports (server_id, label, local_port, remote_host, remote_port) VALUES (?,?,?,?,?)`,
		p.ServerID, p.Label, p.LocalPort, p.RemoteHost, p.RemotePort,
	)
	if err != nil {
		return err
	}
	p.ID, _ = res.LastInsertId()
	return nil
}

func UpdatePort(p *Port) error {
	_, err := DB.Exec(
		`UPDATE ports SET label=?, local_port=?, remote_host=?, remote_port=? WHERE id=?`,
		p.Label, p.LocalPort, p.RemoteHost, p.RemotePort, p.ID,
	)
	return err
}

func DeletePort(id int64) error {
	_, err := DB.Exec(`DELETE FROM ports WHERE id=?`, id)
	return err
}
