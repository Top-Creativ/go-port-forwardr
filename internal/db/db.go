package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() error {
	dir, err := configDir()
	if err != nil {
		return fmt.Errorf("config dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	dbPath := filepath.Join(dir, "data.db")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	DB = db
	return migrate()
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "port-forward-cli"), nil
}

func migrate() error {
	_, err := DB.Exec(`
	CREATE TABLE IF NOT EXISTS servers (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		name          TEXT    NOT NULL,
		host          TEXT    NOT NULL,
		ssh_port      INTEGER NOT NULL DEFAULT 22,
		user          TEXT    NOT NULL,
		auth_type     TEXT    NOT NULL DEFAULT 'password',
		password      TEXT    NOT NULL DEFAULT '',
		key_path      TEXT    NOT NULL DEFAULT '',
		key_passphrase TEXT   NOT NULL DEFAULT '',
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ports (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id    INTEGER NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
		label        TEXT    NOT NULL,
		local_port   INTEGER NOT NULL,
		remote_host  TEXT    NOT NULL DEFAULT 'localhost',
		remote_port  INTEGER NOT NULL,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`)
	return err
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
