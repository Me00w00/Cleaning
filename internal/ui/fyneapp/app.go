package fyneapp

import (
	"database/sql"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	auditapp "project_cleaning/internal/app/audit"
	authapp "project_cleaning/internal/app/auth"
	ordersapp "project_cleaning/internal/app/orders"
	staffapp "project_cleaning/internal/app/staff"
	usersapp "project_cleaning/internal/app/users"
	"project_cleaning/internal/platform/config"
	"project_cleaning/internal/platform/logging"
	sqliterepo "project_cleaning/internal/repository/sqlite"
	"project_cleaning/internal/ui/fyneapp/screens"
)

type App struct {
	fyne   fyne.App
	window fyne.Window
	router *screens.Router
}

func New(cfg config.Config, logger *logging.Logger, db *sql.DB) *App {
	fyneApp := app.NewWithID("project_cleaning")
	fyneApp.Settings().SetTheme(newTheme())

	window := fyneApp.NewWindow(cfg.AppName)
	window.Resize(fyne.NewSize(1360, 860))
	window.CenterOnScreen()

	userRepo := sqliterepo.NewUserRepository(db)
	orderRepo := sqliterepo.NewOrderRepository(db)
	auditRepo := sqliterepo.NewAuditRepository(db)
	idempotencyRepo := sqliterepo.NewIdempotencyRepository(db)
	auditService := auditapp.NewService(auditRepo)
	authService := authapp.NewService(userRepo)
	usersService := usersapp.NewService(userRepo, auditService, idempotencyRepo)
	ordersService := ordersapp.NewService(orderRepo, auditService, idempotencyRepo)
	staffService := staffapp.NewService(orderRepo, auditService, idempotencyRepo)
	router := screens.NewRouter(window, cfg, logger, authService, usersService, ordersService, staffService, auditService)
	router.ShowLogin()

	return &App{
		fyne:   fyneApp,
		window: window,
		router: router,
	}
}

func (a *App) Run() {
	a.window.ShowAndRun()
}
