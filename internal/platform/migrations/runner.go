package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"project_cleaning/internal/platform/logging"
)

func Run(db *sql.DB, migrationsDir string, logger *logging.Logger) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}

	sort.Strings(files)

	for _, migrationPath := range files {
		version := filepath.Base(migrationPath)

		applied, err := isApplied(db, version)
		if err != nil {
			return err
		}

		if applied {
			logger.Info("migration already applied", "version", version)
			continue
		}

		content, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("execute migration %s: %w", version, err)
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			return fmt.Errorf("persist migration %s: %w", version, err)
		}

		logger.Info("migration applied", "version", version)
	}

	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	return nil
}

func isApplied(db *sql.DB, version string) (bool, error) {
	var count int

	if err := db.QueryRow(
		`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`,
		version,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}

	return count > 0, nil
}
