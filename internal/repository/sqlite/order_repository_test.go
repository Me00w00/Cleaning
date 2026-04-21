package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestDeleteHistoricalForClientRemovesOrderHistoryAndOrphanAddress(t *testing.T) {
	db, repo := newOrderRepositoryTestDB(t)

	clientID := insertOrderRepositoryUser(t, db, "client_delete", "client")
	addressID := insertOrderRepositoryAddress(t, db)
	orderID := insertOrderRepositoryOrder(t, db, clientID, addressID, "closed")
	insertOrderRepositoryHistory(t, db, orderID, clientID, "closed")

	if err := repo.DeleteHistoricalForClient(t.Context(), orderID, clientID); err != nil {
		t.Fatalf("delete historical client order: %v", err)
	}

	assertOrderRepositoryCount(t, db, "SELECT COUNT(1) FROM orders WHERE id = ?", orderID, 0)
	assertOrderRepositoryCount(t, db, "SELECT COUNT(1) FROM order_status_history WHERE order_id = ?", orderID, 0)
	assertOrderRepositoryCount(t, db, "SELECT COUNT(1) FROM addresses WHERE id = ?", addressID, 0)
}

func TestDeleteHistoricalForManagerKeepsSharedAddress(t *testing.T) {
	db, repo := newOrderRepositoryTestDB(t)

	clientID := insertOrderRepositoryUser(t, db, "client_shared", "client")
	addressID := insertOrderRepositoryAddress(t, db)
	deletedOrderID := insertOrderRepositoryOrder(t, db, clientID, addressID, "completed")
	keptOrderID := insertOrderRepositoryOrder(t, db, clientID, addressID, "new")
	insertOrderRepositoryHistory(t, db, deletedOrderID, clientID, "completed")

	if err := repo.DeleteHistoricalForManager(t.Context(), deletedOrderID); err != nil {
		t.Fatalf("delete historical manager order: %v", err)
	}

	assertOrderRepositoryCount(t, db, "SELECT COUNT(1) FROM orders WHERE id = ?", deletedOrderID, 0)
	assertOrderRepositoryCount(t, db, "SELECT COUNT(1) FROM order_status_history WHERE order_id = ?", deletedOrderID, 0)
	assertOrderRepositoryCount(t, db, "SELECT COUNT(1) FROM orders WHERE id = ?", keptOrderID, 1)
	assertOrderRepositoryCount(t, db, "SELECT COUNT(1) FROM addresses WHERE id = ?", addressID, 1)
}

func newOrderRepositoryTestDB(t *testing.T) (*sql.DB, *OrderRepository) {
	t.Helper()

	db, err := Open(filepath.Join(t.TempDir(), "orders.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			login TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			full_name TEXT NOT NULL,
			phone TEXT NOT NULL,
			email TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE addresses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			city TEXT NOT NULL,
			street TEXT NOT NULL,
			house TEXT NOT NULL,
			floor TEXT,
			flat TEXT,
			entrance TEXT,
			comment TEXT
		);

		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			manager_id INTEGER,
			staff_id INTEGER,
			address_id INTEGER NOT NULL,
			scheduled_date TEXT NOT NULL,
			scheduled_time_from TEXT,
			scheduled_time_to TEXT,
			service_type TEXT NOT NULL,
			details TEXT,
			square INTEGER NOT NULL,
			window_count INTEGER NOT NULL,
			status TEXT NOT NULL,
			payment_status TEXT NOT NULL DEFAULT 'unpaid',
			price_total INTEGER NOT NULL,
			cancel_reason TEXT,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE RESTRICT,
			FOREIGN KEY (address_id) REFERENCES addresses(id) ON DELETE RESTRICT
		);

		CREATE TABLE order_status_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL,
			old_status TEXT,
			new_status TEXT NOT NULL,
			changed_by_user_id INTEGER NOT NULL,
			changed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			comment TEXT,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
			FOREIGN KEY (changed_by_user_id) REFERENCES users(id) ON DELETE RESTRICT
		);`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db, NewOrderRepository(db)
}

func insertOrderRepositoryUser(t *testing.T, db *sql.DB, login, role string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO users (login, password_hash, role, full_name, phone)
		VALUES (?, 'hash', ?, 'Test User', '+100')`, login, role)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("user last insert id: %v", err)
	}
	return id
}

func insertOrderRepositoryAddress(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO addresses (city, street, house)
		VALUES ('Moscow', 'Tverskaya', '1')`)
	if err != nil {
		t.Fatalf("insert address: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("address last insert id: %v", err)
	}
	return id
}

func insertOrderRepositoryOrder(t *testing.T, db *sql.DB, clientID, addressID int64, status string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO orders (
			client_id, address_id, scheduled_date, service_type, square, window_count, status, payment_status, price_total
		) VALUES (?, ?, '2026-04-01', 'basic_cleaning', 10, 2, ?, 'paid', 1200)`,
		clientID,
		addressID,
		status,
	)
	if err != nil {
		t.Fatalf("insert order: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("order last insert id: %v", err)
	}
	return id
}

func insertOrderRepositoryHistory(t *testing.T, db *sql.DB, orderID, actorID int64, status string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO order_status_history (order_id, new_status, changed_by_user_id, comment)
		VALUES (?, ?, ?, 'status changed')`,
		orderID,
		status,
		actorID,
	); err != nil {
		t.Fatalf("insert history: %v", err)
	}
}

func assertOrderRepositoryCount(t *testing.T, db *sql.DB, query string, arg int64, want int) {
	t.Helper()

	var got int
	if err := db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected count for %q: got %d want %d", query, got, want)
	}
}
