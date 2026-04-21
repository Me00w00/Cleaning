package bootstrap

import (
	"path/filepath"
	"testing"

	"project_cleaning/internal/platform/logging"
	sqliterepo "project_cleaning/internal/repository/sqlite"
)

func TestEnsureDefaultAdminCreatesSingleUser(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.sqlite")

	db, err := sqliterepo.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			login TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			full_name TEXT NOT NULL,
			phone TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		t.Fatalf("create users table: %v", err)
	}

	logger, err := logging.New(filepath.Join(tempDir, "app.log"))
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer logger.Close()

	if err := ensureDefaultAdmin(db, logger); err != nil {
		t.Fatalf("first ensure default admin: %v", err)
	}

	if err := ensureDefaultAdmin(db, logger); err != nil {
		t.Fatalf("second ensure default admin: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users WHERE login = ?`, "admin").Scan(&count); err != nil {
		t.Fatalf("count admin users: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 admin user, got %d", count)
	}
}
