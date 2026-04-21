package migrations

import (
	"path/filepath"
	"testing"

	"project_cleaning/internal/platform/logging"
	sqliterepo "project_cleaning/internal/repository/sqlite"
)

func TestRunAppliesSchemaOnce(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.sqlite")
	migrationsDir, err := filepath.Abs(filepath.Join("..", "..", "..", "db", "migrations"))
	if err != nil {
		t.Fatalf("resolve migrations dir: %v", err)
	}

	db, err := sqliterepo.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	logger, err := logging.New(filepath.Join(tempDir, "app.log"))
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer logger.Close()

	if err := Run(db, migrationsDir, logger); err != nil {
		t.Fatalf("first run: %v", err)
	}

	if err := Run(db, migrationsDir, logger); err != nil {
		t.Fatalf("second run: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected migration count 2, got %d", count)
	}

	if err := db.QueryRow(`SELECT COUNT(1) FROM service_catalog`).Scan(&count); err != nil {
		t.Fatalf("query service_catalog: %v", err)
	}

	if count != 3 {
		t.Fatalf("expected 3 seeded services, got %d", count)
	}
}
