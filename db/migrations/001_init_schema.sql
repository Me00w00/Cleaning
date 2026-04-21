PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

DROP TABLE IF EXISTS order_status_history;
DROP TABLE IF EXISTS staff_unavailability;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS service_catalog;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS addresses;

DROP TABLE IF EXISTS price_list;
DROP TABLE IF EXISTS client;
DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS address;

CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'manager', 'staff', 'client')),
    full_name TEXT NOT NULL,
    phone TEXT NOT NULL,
    email TEXT,
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
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

CREATE TABLE service_catalog (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    base_price INTEGER NOT NULL DEFAULT 0 CHECK (base_price >= 0),
    price_per_square_meter INTEGER CHECK (price_per_square_meter IS NULL OR price_per_square_meter >= 0),
    price_per_window INTEGER CHECK (price_per_window IS NULL OR price_per_window >= 0),
    is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
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
    square INTEGER NOT NULL CHECK (square >= 0),
    window_count INTEGER NOT NULL DEFAULT 0 CHECK (window_count >= 0),
    status TEXT NOT NULL CHECK (status IN (
        'new',
        'assigned_manager',
        'assigned_staff',
        'staff_confirmed',
        'in_progress',
        'completed',
        'closed',
        'cancelled'
    )),
    payment_status TEXT NOT NULL DEFAULT 'unpaid' CHECK (payment_status IN ('unpaid', 'paid')),
    price_total INTEGER NOT NULL DEFAULT 0 CHECK (price_total >= 0),
    cancel_reason TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (manager_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (staff_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (address_id) REFERENCES addresses(id) ON DELETE RESTRICT
);

CREATE TABLE order_status_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id INTEGER NOT NULL,
    old_status TEXT,
    new_status TEXT NOT NULL CHECK (new_status IN (
        'new',
        'assigned_manager',
        'assigned_staff',
        'staff_confirmed',
        'in_progress',
        'completed',
        'closed',
        'cancelled'
    )),
    changed_by_user_id INTEGER NOT NULL,
    changed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment TEXT,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (changed_by_user_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE TABLE staff_unavailability (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    staff_id INTEGER NOT NULL,
    date_from TEXT NOT NULL,
    date_to TEXT NOT NULL,
    reason TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (date(date_from) <= date(date_to)),
    FOREIGN KEY (staff_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE idempotency_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scope TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    response_ref TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (scope, idempotency_key)
);

CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    actor_user_id INTEGER,
    payload_json TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_is_active ON users(is_active);

CREATE INDEX idx_orders_client_id ON orders(client_id);
CREATE INDEX idx_orders_manager_id ON orders(manager_id);
CREATE INDEX idx_orders_staff_id ON orders(staff_id);
CREATE INDEX idx_orders_address_id ON orders(address_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);
CREATE INDEX idx_orders_scheduled_date ON orders(scheduled_date);
CREATE INDEX idx_orders_manager_status ON orders(manager_id, status);
CREATE INDEX idx_orders_staff_status ON orders(staff_id, status);

CREATE INDEX idx_order_status_history_order_id ON order_status_history(order_id);
CREATE INDEX idx_order_status_history_changed_at ON order_status_history(changed_at);

CREATE INDEX idx_staff_unavailability_staff_id ON staff_unavailability(staff_id);
CREATE INDEX idx_staff_unavailability_dates ON staff_unavailability(staff_id, date_from, date_to);

CREATE INDEX idx_idempotency_keys_created_at ON idempotency_keys(created_at);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_actor ON audit_log(actor_user_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

CREATE TRIGGER trg_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE users
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

CREATE TRIGGER trg_service_catalog_updated_at
AFTER UPDATE ON service_catalog
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE service_catalog
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

CREATE TRIGGER trg_orders_updated_at
AFTER UPDATE ON orders
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE orders
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

CREATE TRIGGER trg_orders_validate_manager_assignment
BEFORE UPDATE OF manager_id ON orders
FOR EACH ROW
WHEN NEW.manager_id IS NOT NULL AND OLD.manager_id IS NULL AND OLD.status <> 'new'
BEGIN
    SELECT RAISE(ABORT, 'manager can only be assigned when order status is new');
END;

CREATE TRIGGER trg_orders_validate_staff_assignment
BEFORE UPDATE OF staff_id ON orders
FOR EACH ROW
WHEN NEW.staff_id IS NOT NULL AND OLD.staff_id IS NULL AND OLD.status <> 'assigned_manager'
BEGIN
    SELECT RAISE(ABORT, 'staff can only be assigned when order status is assigned_manager');
END;

CREATE TRIGGER trg_staff_unavailability_validate_role_insert
BEFORE INSERT ON staff_unavailability
FOR EACH ROW
WHEN (SELECT role FROM users WHERE id = NEW.staff_id) <> 'staff'
BEGIN
    SELECT RAISE(ABORT, 'staff_unavailability can only reference users with role staff');
END;

CREATE TRIGGER trg_staff_unavailability_validate_role_update
BEFORE UPDATE OF staff_id ON staff_unavailability
FOR EACH ROW
WHEN (SELECT role FROM users WHERE id = NEW.staff_id) <> 'staff'
BEGIN
    SELECT RAISE(ABORT, 'staff_unavailability can only reference users with role staff');
END;

CREATE TRIGGER trg_orders_validate_client_role_insert
BEFORE INSERT ON orders
FOR EACH ROW
WHEN (SELECT role FROM users WHERE id = NEW.client_id) <> 'client'
BEGIN
    SELECT RAISE(ABORT, 'client_id must reference a user with role client');
END;

CREATE TRIGGER trg_orders_validate_client_role_update
BEFORE UPDATE OF client_id ON orders
FOR EACH ROW
WHEN (SELECT role FROM users WHERE id = NEW.client_id) <> 'client'
BEGIN
    SELECT RAISE(ABORT, 'client_id must reference a user with role client');
END;

CREATE TRIGGER trg_orders_validate_manager_role
BEFORE UPDATE OF manager_id ON orders
FOR EACH ROW
WHEN NEW.manager_id IS NOT NULL
 AND (SELECT role FROM users WHERE id = NEW.manager_id) <> 'manager'
BEGIN
    SELECT RAISE(ABORT, 'manager_id must reference a user with role manager');
END;

CREATE TRIGGER trg_orders_validate_staff_role
BEFORE UPDATE OF staff_id ON orders
FOR EACH ROW
WHEN NEW.staff_id IS NOT NULL
 AND (SELECT role FROM users WHERE id = NEW.staff_id) <> 'staff'
BEGIN
    SELECT RAISE(ABORT, 'staff_id must reference a user with role staff');
END;

CREATE TRIGGER trg_orders_validate_status_transition
BEFORE UPDATE OF status ON orders
FOR EACH ROW
WHEN NOT (
    OLD.status = NEW.status OR
    (OLD.status = 'new' AND NEW.status IN ('assigned_manager', 'cancelled')) OR
    (OLD.status = 'assigned_manager' AND NEW.status IN ('assigned_staff', 'cancelled')) OR
    (OLD.status = 'assigned_staff' AND NEW.status IN ('staff_confirmed', 'assigned_manager', 'cancelled')) OR
    (OLD.status = 'staff_confirmed' AND NEW.status IN ('in_progress', 'cancelled')) OR
    (OLD.status = 'in_progress' AND NEW.status IN ('completed', 'cancelled')) OR
    (OLD.status = 'completed' AND NEW.status IN ('closed', 'cancelled'))
)
BEGIN
    SELECT RAISE(ABORT, 'invalid order status transition');
END;

INSERT INTO service_catalog (code, name, base_price, price_per_square_meter, price_per_window, is_active)
VALUES
    ('basic_cleaning', 'Базовая уборка', 0, 120, 250, 1),
    ('general_cleaning', 'Генеральная уборка', 1000, 180, 300, 1),
    ('window_cleaning', 'Мойка окон', 0, NULL, 400, 1);

COMMIT;
PRAGMA foreign_keys = ON;

-- Примечание:
-- Пользователь admin/admin должен создаваться в bootstrap-логике приложения,
-- так как password_hash должен быть сформирован кодом приложения безопасным алгоритмом.
