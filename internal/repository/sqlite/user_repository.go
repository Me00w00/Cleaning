package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	userdomain "project_cleaning/internal/domain/user"
)

var ErrUserNotFound = errors.New("пользователь не найден")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByLogin(ctx context.Context, login string) (userdomain.User, error) {
	const query = `
		SELECT id, login, password_hash, role, full_name, phone, COALESCE(email, ''), is_active, created_at, updated_at
		FROM users
		WHERE lower(login) = lower(?)`

	return r.scanOne(ctx, query, strings.TrimSpace(login))
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (userdomain.User, error) {
	const query = `
		SELECT id, login, password_hash, role, full_name, phone, COALESCE(email, ''), is_active, created_at, updated_at
		FROM users
		WHERE id = ?`

	return r.scanOne(ctx, query, id)
}

func (r *UserRepository) List(ctx context.Context) ([]userdomain.User, error) {
	const query = `
		SELECT id, login, password_hash, role, full_name, phone, COALESCE(email, ''), is_active, created_at, updated_at
		FROM users
		ORDER BY role, login`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := make([]userdomain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func (r *UserRepository) Create(ctx context.Context, user userdomain.User) (int64, error) {
	const query = `
		INSERT INTO users (login, password_hash, role, full_name, phone, email, is_active)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?)`

	result, err := r.db.ExecContext(ctx, query,
		user.Login,
		user.PasswordHash,
		string(user.Role),
		user.FullName,
		user.Phone,
		user.Email,
		boolToInt(user.IsActive),
	)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}

	return id, nil
}

func (r *UserRepository) Update(ctx context.Context, user userdomain.User) error {
	const query = `
		UPDATE users
		SET full_name = ?, phone = ?, email = NULLIF(?, ''), role = ?, is_active = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		user.FullName,
		user.Phone,
		user.Email,
		string(user.Role),
		boolToInt(user.IsActive),
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) scanOne(ctx context.Context, query string, arg any) (userdomain.User, error) {
	row := r.db.QueryRowContext(ctx, query, arg)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return userdomain.User{}, ErrUserNotFound
	}
	if err != nil {
		return userdomain.User{}, err
	}

	return user, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner rowScanner) (userdomain.User, error) {
	var user userdomain.User
	var isActive int
	var role string
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&role,
		&user.FullName,
		&user.Phone,
		&user.Email,
		&isActive,
		&createdAt,
		&updatedAt,
	); err != nil {
		return userdomain.User{}, err
	}

	parsedCreatedAt, err := parseSQLiteTime(createdAt)
	if err != nil {
		return userdomain.User{}, fmt.Errorf("parse created_at: %w", err)
	}

	parsedUpdatedAt, err := parseSQLiteTime(updatedAt)
	if err != nil {
		return userdomain.User{}, fmt.Errorf("parse updated_at: %w", err)
	}

	user.Role = userdomain.Role(role)
	user.IsActive = isActive == 1
	user.CreatedAt = parsedCreatedAt
	user.UpdatedAt = parsedUpdatedAt
	return user, nil
}

func parseSQLiteTime(value string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported sqlite datetime format: %s", value)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}

	return 0
}
