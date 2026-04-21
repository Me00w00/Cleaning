package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrIdempotencyConflict = errors.New("конфликт ключа идемпотентности")

type IdempotencyRepository struct {
	db *sql.DB
}

func NewIdempotencyRepository(db *sql.DB) *IdempotencyRepository {
	return &IdempotencyRepository{db: db}
}

func (r *IdempotencyRepository) Begin(ctx context.Context, scope, key, requestHash string) (bool, string, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO idempotency_keys (scope, idempotency_key, request_hash)
		VALUES (?, ?, ?)`, scope, key, requestHash)
	if err != nil {
		return false, "", fmt.Errorf("reserve idempotency key: %w", err)
	}

	var storedHash string
	var responseRef string
	if err := r.db.QueryRowContext(ctx, `
		SELECT request_hash, COALESCE(response_ref, '')
		FROM idempotency_keys
		WHERE scope = ? AND idempotency_key = ?`, scope, key,
	).Scan(&storedHash, &responseRef); err != nil {
		return false, "", fmt.Errorf("query idempotency key: %w", err)
	}
	if storedHash != requestHash {
		return false, "", ErrIdempotencyConflict
	}
	if responseRef != "" {
		return true, responseRef, nil
	}
	return false, "", nil
}

func (r *IdempotencyRepository) Complete(ctx context.Context, scope, key, responseRef string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE idempotency_keys
		SET response_ref = ?
		WHERE scope = ? AND idempotency_key = ?`, responseRef, scope, key)
	if err != nil {
		return fmt.Errorf("update idempotency key: %w", err)
	}
	return nil
}
