package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	availabilitydomain "project_cleaning/internal/domain/availability"
	orderdomain "project_cleaning/internal/domain/order"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) ListServices(ctx context.Context) ([]orderdomain.ServiceCatalogItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, name, base_price, COALESCE(price_per_square_meter, 0), COALESCE(price_per_window, 0), is_active
		FROM service_catalog
		WHERE is_active = 1
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query services: %w", err)
	}
	defer rows.Close()

	items := make([]orderdomain.ServiceCatalogItem, 0)
	for rows.Next() {
		var item orderdomain.ServiceCatalogItem
		var active int
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.BasePrice, &item.PricePerSquareMeter, &item.PricePerWindow, &active); err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}
		item.IsActive = active == 1
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate services: %w", err)
	}

	return items, nil
}

func (r *OrderRepository) GetServiceByCode(ctx context.Context, code string) (orderdomain.ServiceCatalogItem, error) {
	var item orderdomain.ServiceCatalogItem
	var active int
	if err := r.db.QueryRowContext(ctx, `
		SELECT id, code, name, base_price, COALESCE(price_per_square_meter, 0), COALESCE(price_per_window, 0), is_active
		FROM service_catalog
		WHERE code = ? AND is_active = 1`, code,
	).Scan(&item.ID, &item.Code, &item.Name, &item.BasePrice, &item.PricePerSquareMeter, &item.PricePerWindow, &active); err != nil {
		return orderdomain.ServiceCatalogItem{}, fmt.Errorf("query service by code: %w", err)
	}
	item.IsActive = active == 1
	return item, nil
}

func (r *OrderRepository) CreateAddress(ctx context.Context, address orderdomain.Address) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO addresses (city, street, house, floor, flat, entrance, comment)
		VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))`,
		address.City,
		address.Street,
		address.House,
		address.Floor,
		address.Flat,
		address.Entrance,
		address.Comment,
	)
	if err != nil {
		return 0, fmt.Errorf("insert address: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("address last insert id: %w", err)
	}
	return id, nil
}

func (r *OrderRepository) UpdateAddress(ctx context.Context, address orderdomain.Address) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE addresses
		SET city = ?, street = ?, house = ?, floor = NULLIF(?, ''), flat = NULLIF(?, ''), entrance = NULLIF(?, ''), comment = NULLIF(?, '')
		WHERE id = ?`,
		address.City,
		address.Street,
		address.House,
		address.Floor,
		address.Flat,
		address.Entrance,
		address.Comment,
		address.ID,
	)
	if err != nil {
		return fmt.Errorf("update address: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("address not found")
	}
	return nil
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order orderdomain.Order) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO orders (
			client_id, manager_id, staff_id, address_id, scheduled_date, scheduled_time_from, scheduled_time_to,
			service_type, details, square, window_count, status, payment_status, price_total, cancel_reason
		) VALUES (?, NULL, NULL, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, ?, ?, ?, ?, NULLIF(?, ''))`,
		order.ClientID,
		order.Address.ID,
		order.ScheduledDate.Format("2006-01-02"),
		order.ScheduledTimeFrom,
		order.ScheduledTimeTo,
		order.ServiceType,
		order.Details,
		order.Square,
		order.WindowCount,
		string(order.Status),
		string(order.PaymentStatus),
		order.PriceTotal,
		order.CancelReason,
	)
	if err != nil {
		return 0, fmt.Errorf("insert order: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("order last insert id: %w", err)
	}
	return id, nil
}

func (r *OrderRepository) UpdateOrderForClient(ctx context.Context, order orderdomain.Order) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET address_id = ?, scheduled_date = ?, scheduled_time_from = NULLIF(?, ''), scheduled_time_to = NULLIF(?, ''),
			service_type = ?, details = NULLIF(?, ''), square = ?, window_count = ?, price_total = ?
		WHERE id = ? AND client_id = ? AND status = 'new'`,
		order.Address.ID,
		order.ScheduledDate.Format("2006-01-02"),
		order.ScheduledTimeFrom,
		order.ScheduledTimeTo,
		order.ServiceType,
		order.Details,
		order.Square,
		order.WindowCount,
		order.PriceTotal,
		order.ID,
		order.ClientID,
	)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	return ensureRowsAffected(result, "order not updated")
}

func (r *OrderRepository) CancelOrderForClient(ctx context.Context, orderID, clientID int64, cancelReason string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'cancelled', cancel_reason = NULLIF(?, '')
		WHERE id = ? AND client_id = ? AND status = 'new'`,
		cancelReason,
		orderID,
		clientID,
	)
	if err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}
	return ensureRowsAffected(result, "order not cancelled")
}

func (r *OrderRepository) DeleteHistoricalForClient(ctx context.Context, orderID, clientID int64) error {
	return r.deleteHistoricalOrder(ctx, `
		SELECT address_id FROM orders
		WHERE id = ? AND client_id = ? AND status IN ('completed', 'closed', 'cancelled')`,
		`DELETE FROM orders WHERE id = ? AND client_id = ? AND status IN ('completed', 'closed', 'cancelled')`,
		"historical client order not deleted",
		orderID, clientID,
	)
}

func (r *OrderRepository) DeleteHistoricalForManager(ctx context.Context, orderID int64) error {
	return r.deleteHistoricalOrder(ctx, `
		SELECT address_id FROM orders
		WHERE id = ? AND status IN ('completed', 'closed', 'cancelled')`,
		`DELETE FROM orders WHERE id = ? AND status IN ('completed', 'closed', 'cancelled')`,
		"historical manager order not deleted",
		orderID,
	)
}

func (r *OrderRepository) deleteHistoricalOrder(ctx context.Context, selectQuery, deleteQuery, notFoundMessage string, args ...any) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete historical order tx: %w", err)
	}
	defer rollbackTx(tx)

	addressID, err := queryAddressID(ctx, tx, selectQuery, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s", notFoundMessage)
		}
		return fmt.Errorf("query historical order address: %w", err)
	}

	result, err := tx.ExecContext(ctx, deleteQuery, args...)
	if err != nil {
		return fmt.Errorf("delete historical order: %w", err)
	}
	if err := ensureRowsAffected(result, notFoundMessage); err != nil {
		return err
	}
	if err := deleteOrphanAddressTx(ctx, tx, addressID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete historical order: %w", err)
	}
	return nil
}

func queryAddressID(ctx context.Context, tx *sql.Tx, query string, args ...any) (int64, error) {
	var addressID int64
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&addressID); err != nil {
		return 0, err
	}
	return addressID, nil
}

func deleteOrphanAddressTx(ctx context.Context, tx *sql.Tx, addressID int64) error {
	var usedCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM orders WHERE address_id = ?`, addressID).Scan(&usedCount); err != nil {
		return fmt.Errorf("query address usage: %w", err)
	}
	if usedCount > 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM addresses WHERE id = ?`, addressID); err != nil {
		return fmt.Errorf("delete orphan address: %w", err)
	}
	return nil
}

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func (r *OrderRepository) AssignManager(ctx context.Context, orderID, managerID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET manager_id = ?, status = 'assigned_manager'
		WHERE id = ? AND status = 'new'`, managerID, orderID)
	if err != nil {
		return fmt.Errorf("assign manager: %w", err)
	}
	return ensureRowsAffected(result, "order not assigned to manager")
}

func (r *OrderRepository) AssignStaff(ctx context.Context, orderID, managerID, staffID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET staff_id = ?, status = 'assigned_staff'
		WHERE id = ? AND manager_id = ? AND status = 'assigned_manager'`, staffID, orderID, managerID)
	if err != nil {
		return fmt.Errorf("assign staff: %w", err)
	}
	return ensureRowsAffected(result, "order not assigned to staff")
}

func (r *OrderRepository) ConfirmPayment(ctx context.Context, orderID, managerID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET payment_status = 'paid'
		WHERE id = ? AND (manager_id = ? OR status = 'new') AND payment_status = 'unpaid'`, orderID, managerID)
	if err != nil {
		return fmt.Errorf("confirm payment: %w", err)
	}
	return ensureRowsAffected(result, "payment not confirmed")
}

func (r *OrderRepository) CloseOrder(ctx context.Context, orderID, managerID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'closed'
		WHERE id = ? AND manager_id = ? AND status = 'completed'`, orderID, managerID)
	if err != nil {
		return fmt.Errorf("close order: %w", err)
	}
	return ensureRowsAffected(result, "order not closed")
}

func (r *OrderRepository) StaffAcceptOrder(ctx context.Context, orderID, staffID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'staff_confirmed'
		WHERE id = ? AND staff_id = ? AND status = 'assigned_staff'`, orderID, staffID)
	if err != nil {
		return fmt.Errorf("staff accept order: %w", err)
	}
	return ensureRowsAffected(result, "order not accepted by staff")
}

func (r *OrderRepository) StaffDeclineOrder(ctx context.Context, orderID, staffID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET staff_id = NULL, status = 'assigned_manager'
		WHERE id = ? AND staff_id = ? AND status = 'assigned_staff'`, orderID, staffID)
	if err != nil {
		return fmt.Errorf("staff decline order: %w", err)
	}
	return ensureRowsAffected(result, "order not declined by staff")
}

func (r *OrderRepository) StaffStartOrder(ctx context.Context, orderID, staffID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'in_progress'
		WHERE id = ? AND staff_id = ? AND status = 'staff_confirmed'`, orderID, staffID)
	if err != nil {
		return fmt.Errorf("staff start order: %w", err)
	}
	return ensureRowsAffected(result, "order not started by staff")
}

func (r *OrderRepository) StaffCompleteOrder(ctx context.Context, orderID, staffID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = 'completed'
		WHERE id = ? AND staff_id = ? AND status = 'in_progress'`, orderID, staffID)
	if err != nil {
		return fmt.Errorf("staff complete order: %w", err)
	}
	return ensureRowsAffected(result, "order not completed by staff")
}

func (r *OrderRepository) CreateStatusHistory(ctx context.Context, orderID, actorID int64, newStatus orderdomain.Status, comment string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO order_status_history (order_id, old_status, new_status, changed_by_user_id, comment)
		VALUES (
			?,
			(SELECT h.new_status FROM order_status_history h WHERE h.order_id = ? ORDER BY h.id DESC LIMIT 1),
			?,
			?,
			NULLIF(?, '')
		)`,
		orderID,
		orderID,
		string(newStatus),
		actorID,
		comment,
	)
	if err != nil {
		return fmt.Errorf("insert order status history: %w", err)
	}
	return nil
}

func (r *OrderRepository) ListStatusHistory(ctx context.Context, orderID int64) ([]orderdomain.StatusHistoryEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT h.id, h.order_id, COALESCE(h.old_status, ''), h.new_status, h.changed_by_user_id, COALESCE(u.full_name, ''), h.changed_at, COALESCE(h.comment, '')
		FROM order_status_history h
		LEFT JOIN users u ON u.id = h.changed_by_user_id
		WHERE h.order_id = ?
		ORDER BY h.changed_at DESC, h.id DESC`, orderID)
	if err != nil {
		return nil, fmt.Errorf("query order history: %w", err)
	}
	defer rows.Close()

	items := make([]orderdomain.StatusHistoryEntry, 0)
	for rows.Next() {
		var item orderdomain.StatusHistoryEntry
		var oldStatus string
		var newStatus string
		var changedAt string
		if err := rows.Scan(&item.ID, &item.OrderID, &oldStatus, &newStatus, &item.ChangedByUserID, &item.ChangedByName, &changedAt, &item.Comment); err != nil {
			return nil, fmt.Errorf("scan order history: %w", err)
		}
		parsedChangedAt, err := parseSQLiteTime(changedAt)
		if err != nil {
			return nil, fmt.Errorf("parse changed_at: %w", err)
		}
		item.OldStatus = orderdomain.Status(oldStatus)
		item.NewStatus = orderdomain.Status(newStatus)
		item.ChangedAt = parsedChangedAt
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order history: %w", err)
	}
	return items, nil
}

func (r *OrderRepository) ListByClient(ctx context.Context, clientID int64) ([]orderdomain.Order, error) {
	rows, err := r.db.QueryContext(ctx, baseOrderSelect+`
		WHERE o.client_id = ?
		ORDER BY o.created_at DESC, o.id DESC`, clientID)
	if err != nil {
		return nil, fmt.Errorf("query client orders: %w", err)
	}
	defer rows.Close()
	return scanOrders(rows)
}

func (r *OrderRepository) ListForManager(ctx context.Context) ([]orderdomain.Order, error) {
	rows, err := r.db.QueryContext(ctx, baseOrderSelect+`
		ORDER BY o.created_at DESC, o.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("query manager orders: %w", err)
	}
	defer rows.Close()
	return scanOrders(rows)
}

func (r *OrderRepository) ListForStaff(ctx context.Context, staffID int64) ([]orderdomain.Order, error) {
	rows, err := r.db.QueryContext(ctx, baseOrderSelect+`
		WHERE o.staff_id = ?
		ORDER BY o.scheduled_date DESC, o.id DESC`, staffID)
	if err != nil {
		return nil, fmt.Errorf("query staff orders: %w", err)
	}
	defer rows.Close()
	return scanOrders(rows)
}

func (r *OrderRepository) FindStaffUnavailability(ctx context.Context, staffID int64, scheduledStart, scheduledEnd time.Time) (availabilitydomain.Period, bool, error) {
	var period availabilitydomain.Period
	var startsAt string
	var endsAt string
	query := `
		SELECT id, staff_id, COALESCE(starts_at, date(date_from) || ' 00:00'), COALESCE(ends_at, date(date_to) || ' 23:59'), COALESCE(reason, '')
		FROM staff_unavailability
		WHERE staff_id = ?
		  AND COALESCE(starts_at, date(date_from) || ' 00:00') < ?
		  AND COALESCE(ends_at, date(date_to) || ' 23:59') > ?
		ORDER BY COALESCE(starts_at, date(date_from) || ' 00:00') ASC
		LIMIT 1`
	err := r.db.QueryRowContext(
		ctx,
		query,
		staffID,
		scheduledEnd.Format("2006-01-02 15:04"),
		scheduledStart.Format("2006-01-02 15:04"),
	).Scan(&period.ID, &period.StaffID, &startsAt, &endsAt, &period.Reason)
	if errors.Is(err, sql.ErrNoRows) {
		return availabilitydomain.Period{}, false, nil
	}
	if err != nil {
		return availabilitydomain.Period{}, false, fmt.Errorf("query staff availability: %w", err)
	}
	parsedStart, err := time.Parse("2006-01-02 15:04", startsAt)
	if err != nil {
		return availabilitydomain.Period{}, false, fmt.Errorf("parse starts_at: %w", err)
	}
	parsedEnd, err := time.Parse("2006-01-02 15:04", endsAt)
	if err != nil {
		return availabilitydomain.Period{}, false, fmt.Errorf("parse ends_at: %w", err)
	}
	period.StartsAt = parsedStart
	period.EndsAt = parsedEnd
	return period, true, nil
}

func (r *OrderRepository) ListStaffUnavailability(ctx context.Context, staffID int64) ([]availabilitydomain.Period, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, staff_id, COALESCE(starts_at, date(date_from) || ' 00:00'), COALESCE(ends_at, date(date_to) || ' 23:59'), COALESCE(reason, ''), created_at
		FROM staff_unavailability
		WHERE staff_id = ?
		ORDER BY COALESCE(starts_at, date(date_from) || ' 00:00') DESC, id DESC`, staffID)
	if err != nil {
		return nil, fmt.Errorf("query staff unavailability: %w", err)
	}
	defer rows.Close()

	items := make([]availabilitydomain.Period, 0)
	for rows.Next() {
		var period availabilitydomain.Period
		var startsAt string
		var endsAt string
		var createdAt string
		if err := rows.Scan(&period.ID, &period.StaffID, &startsAt, &endsAt, &period.Reason, &createdAt); err != nil {
			return nil, fmt.Errorf("scan staff unavailability: %w", err)
		}
		parsedStart, err := time.Parse("2006-01-02 15:04", startsAt)
		if err != nil {
			return nil, fmt.Errorf("parse starts_at: %w", err)
		}
		parsedEnd, err := time.Parse("2006-01-02 15:04", endsAt)
		if err != nil {
			return nil, fmt.Errorf("parse ends_at: %w", err)
		}
		parsedCreatedAt, err := parseSQLiteTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		period.StartsAt = parsedStart
		period.EndsAt = parsedEnd
		period.CreatedAt = parsedCreatedAt
		items = append(items, period)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate staff unavailability: %w", err)
	}
	return items, nil
}

func (r *OrderRepository) HasAvailabilityOverlap(ctx context.Context, staffID int64, startsAt, endsAt time.Time) (bool, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM staff_unavailability
		WHERE staff_id = ?
		  AND COALESCE(starts_at, date(date_from) || ' 00:00') < ?
		  AND COALESCE(ends_at, date(date_to) || ' 23:59') > ?`,
		staffID,
		endsAt.Format("2006-01-02 15:04"),
		startsAt.Format("2006-01-02 15:04"),
	).Scan(&count); err != nil {
		return false, fmt.Errorf("query availability overlap: %w", err)
	}
	return count > 0, nil
}

func (r *OrderRepository) CreateStaffUnavailability(ctx context.Context, period availabilitydomain.Period) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO staff_unavailability (staff_id, date_from, date_to, starts_at, ends_at, reason)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''))`,
		period.StaffID,
		period.StartsAt.Format("2006-01-02"),
		period.EndsAt.Format("2006-01-02"),
		period.StartsAt.Format("2006-01-02 15:04"),
		period.EndsAt.Format("2006-01-02 15:04"),
		period.Reason,
	)
	if err != nil {
		return fmt.Errorf("insert staff unavailability: %w", err)
	}
	return nil
}

const baseOrderSelect = `
	SELECT
		o.id,
		o.client_id,
		COALESCE(o.manager_id, 0),
		COALESCE(o.staff_id, 0),
		COALESCE(c.id, 0),
		COALESCE(c.full_name, ''),
		COALESCE(c.phone, ''),
		COALESCE(c.email, ''),
		COALESCE(m.id, 0),
		COALESCE(m.full_name, ''),
		COALESCE(m.phone, ''),
		COALESCE(m.email, ''),
		COALESCE(s.id, 0),
		COALESCE(s.full_name, ''),
		COALESCE(s.phone, ''),
		COALESCE(s.email, ''),
		a.id,
		a.city,
		a.street,
		a.house,
		COALESCE(a.floor, ''),
		COALESCE(a.flat, ''),
		COALESCE(a.entrance, ''),
		COALESCE(a.comment, ''),
		o.scheduled_date,
		COALESCE(o.scheduled_time_from, ''),
		COALESCE(o.scheduled_time_to, ''),
		o.service_type,
		COALESCE(o.details, ''),
		o.square,
		o.window_count,
		o.status,
		o.payment_status,
		o.price_total,
		COALESCE(o.cancel_reason, ''),
		o.created_at,
		o.updated_at
	FROM orders o
	JOIN addresses a ON a.id = o.address_id
	LEFT JOIN users c ON c.id = o.client_id
	LEFT JOIN users m ON m.id = o.manager_id
	LEFT JOIN users s ON s.id = o.staff_id`

func scanOrders(rows *sql.Rows) ([]orderdomain.Order, error) {
	orders := make([]orderdomain.Order, 0)
	for rows.Next() {
		var order orderdomain.Order
		var scheduledDate string
		var createdAt string
		var updatedAt string
		var status string
		var paymentStatus string
		if err := rows.Scan(
			&order.ID,
			&order.ClientID,
			&order.ManagerID,
			&order.StaffID,
			&order.Client.ID,
			&order.Client.FullName,
			&order.Client.Phone,
			&order.Client.Email,
			&order.Manager.ID,
			&order.Manager.FullName,
			&order.Manager.Phone,
			&order.Manager.Email,
			&order.Staff.ID,
			&order.Staff.FullName,
			&order.Staff.Phone,
			&order.Staff.Email,
			&order.Address.ID,
			&order.Address.City,
			&order.Address.Street,
			&order.Address.House,
			&order.Address.Floor,
			&order.Address.Flat,
			&order.Address.Entrance,
			&order.Address.Comment,
			&scheduledDate,
			&order.ScheduledTimeFrom,
			&order.ScheduledTimeTo,
			&order.ServiceType,
			&order.Details,
			&order.Square,
			&order.WindowCount,
			&status,
			&paymentStatus,
			&order.PriceTotal,
			&order.CancelReason,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		parsedScheduledDate, err := time.Parse("2006-01-02", scheduledDate)
		if err != nil {
			return nil, fmt.Errorf("parse scheduled_date: %w", err)
		}
		parsedCreatedAt, err := parseSQLiteTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		parsedUpdatedAt, err := parseSQLiteTime(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse updated_at: %w", err)
		}
		order.ScheduledDate = parsedScheduledDate
		order.CreatedAt = parsedCreatedAt
		order.UpdatedAt = parsedUpdatedAt
		order.Status = orderdomain.Status(status)
		order.PaymentStatus = orderdomain.PaymentStatus(paymentStatus)
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}
	return orders, nil
}

func ensureRowsAffected(result sql.Result, message string) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%s", message)
	}
	return nil
}
