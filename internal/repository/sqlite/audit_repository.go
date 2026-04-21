package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	auditdomain "project_cleaning/internal/domain/audit"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, entry auditdomain.Entry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_log (entity_type, entity_id, action, actor_user_id, payload_json)
		VALUES (?, ?, ?, NULLIF(?, 0), NULLIF(?, ''))`,
		entry.EntityType,
		entry.EntityID,
		entry.Action,
		entry.ActorUserID,
		entry.PayloadJSON,
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *AuditRepository) ListRecent(ctx context.Context, limit int) ([]auditdomain.Entry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.entity_type, a.entity_id, a.action, COALESCE(a.actor_user_id, 0), COALESCE(u.full_name, ''), COALESCE(a.payload_json, ''), a.created_at
		FROM audit_log a
		LEFT JOIN users u ON u.id = a.actor_user_id
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	items := make([]auditdomain.Entry, 0)
	for rows.Next() {
		var entry auditdomain.Entry
		var createdAt string
		if err := rows.Scan(&entry.ID, &entry.EntityType, &entry.EntityID, &entry.Action, &entry.ActorUserID, &entry.ActorName, &entry.PayloadJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		parsedCreatedAt, err := parseSQLiteTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse audit created_at: %w", err)
		}
		entry.CreatedAt = parsedCreatedAt
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit log: %w", err)
	}
	return items, nil
}
