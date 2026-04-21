package sqlite

import (
	"path/filepath"
	"testing"
)

func TestScanUserParsesSQLiteTimestamps(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "repo.sqlite"))
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
			email TEXT,
			is_active INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO users (login, password_hash, role, full_name, phone, email, is_active, created_at, updated_at)
		VALUES ('admin', 'hash', 'admin', 'Admin', '+100', 'admin@example.com', 1, '2026-03-28 14:10:15', '2026-03-28 14:10:15')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	repo := NewUserRepository(db)
	user, err := repo.FindByLogin(t.Context(), "admin")
	if err != nil {
		t.Fatalf("find by login: %v", err)
	}

	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed timestamps, got zero values")
	}
	if user.Login != "admin" {
		t.Fatalf("expected admin login, got %s", user.Login)
	}
}
