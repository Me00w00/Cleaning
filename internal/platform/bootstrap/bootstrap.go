package bootstrap

import (
	"database/sql"
	"fmt"

	"project_cleaning/internal/platform/auth"
	"project_cleaning/internal/platform/config"
	"project_cleaning/internal/platform/logging"
	"project_cleaning/internal/platform/migrations"
	sqliterepo "project_cleaning/internal/repository/sqlite"
	"project_cleaning/internal/ui/fyneapp"
)

// Run wires the application dependencies and starts the desktop UI.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := logging.New(cfg.LogFilePath)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() {
		_ = logger.Close()
	}()

	logger.Info("starting application", "db_path", cfg.DBPath, "app_name", cfg.AppName)

	db, err := sqliterepo.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := migrations.Run(db, cfg.Migrations, logger); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	if err := ensureDefaultAdmin(db, logger); err != nil {
		return fmt.Errorf("ensure default admin: %w", err)
	}

	app := fyneapp.New(cfg, logger, db)
	app.Run()

	return nil
}

func ensureDefaultAdmin(db *sql.DB, logger *logging.Logger) error {
	const selectAdminQuery = `
		SELECT COUNT(1)
		FROM users
		WHERE login = ?`

	var count int
	if err := db.QueryRow(selectAdminQuery, "admin").Scan(&count); err != nil {
		return fmt.Errorf("check admin existence: %w", err)
	}

	if count > 0 {
		logger.Info("default admin already exists")
		return nil
	}

	passwordHash, err := auth.HashPassword("admin")
	if err != nil {
		return fmt.Errorf("hash default admin password: %w", err)
	}

	const insertAdminQuery = `
		INSERT INTO users (login, password_hash, role, full_name, phone, is_active)
		VALUES (?, ?, 'admin', ?, ?, 1)`

	if _, err := db.Exec(insertAdminQuery, "admin", passwordHash, "System Administrator", "+0000000000"); err != nil {
		return fmt.Errorf("insert default admin: %w", err)
	}

	logger.Info("default admin created", "login", "admin")
	return nil
}
