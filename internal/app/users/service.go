package users

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	userdomain "project_cleaning/internal/domain/user"
	platformauth "project_cleaning/internal/platform/auth"
	sqliterepo "project_cleaning/internal/repository/sqlite"
)

var ErrInvalidInput = errors.New("некорректные входные данные")
var ErrLoginExists = errors.New("логин уже существует")

type Repository interface {
	Create(ctx context.Context, user userdomain.User) (int64, error)
	Update(ctx context.Context, user userdomain.User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]userdomain.User, error)
	FindByLogin(ctx context.Context, login string) (userdomain.User, error)
	FindByID(ctx context.Context, id int64) (userdomain.User, error)
}

type Auditor interface {
	Write(ctx context.Context, entityType string, entityID, actorUserID int64, action, payload string) error
}

type IdempotencyStore interface {
	Begin(ctx context.Context, scope, key, requestHash string) (bool, string, error)
	Complete(ctx context.Context, scope, key, responseRef string) error
}

type CreateInput struct {
	Login          string
	Password       string
	Role           userdomain.Role
	FullName       string
	Phone          string
	Email          string
	IdempotencyKey string
}

type UpdateInput struct {
	ID       int64
	FullName string
	Phone    string
	Email    string
	Role     userdomain.Role
	IsActive bool
}

type Service struct {
	repo        Repository
	auditor     Auditor
	idempotency IdempotencyStore
}

func NewService(repo Repository, deps ...any) *Service {
	service := &Service{repo: repo}
	for _, dep := range deps {
		switch v := dep.(type) {
		case Auditor:
			service.auditor = v
		case IdempotencyStore:
			service.idempotency = v
		}
	}
	return service
}

func (s *Service) RegisterClient(ctx context.Context, input CreateInput) (int64, error) {
	input.Role = userdomain.RoleClient
	return s.create(ctx, input, 0)
}

func (s *Service) CreateByAdmin(ctx context.Context, input CreateInput) (int64, error) {
	if input.Role != userdomain.RoleManager && input.Role != userdomain.RoleStaff && input.Role != userdomain.RoleAdmin {
		return 0, ErrInvalidInput
	}

	return s.create(ctx, input, 0)
}

func (s *Service) create(ctx context.Context, input CreateInput, actorID int64) (int64, error) {
	if strings.TrimSpace(input.Login) == "" || strings.TrimSpace(input.Password) == "" || strings.TrimSpace(input.FullName) == "" || strings.TrimSpace(input.Phone) == "" {
		return 0, ErrInvalidInput
	}

	if reused, responseRef, err := s.beginIdempotent(ctx, "user.create", input.IdempotencyKey, hashCreateInput(input)); err != nil {
		return 0, err
	} else if reused {
		return strconv.ParseInt(responseRef, 10, 64)
	}

	if _, err := s.repo.FindByLogin(ctx, input.Login); err == nil {
		return 0, ErrLoginExists
	} else if !errors.Is(err, sqliterepo.ErrUserNotFound) {
		return 0, fmt.Errorf("check login uniqueness: %w", err)
	}

	hash, err := platformauth.HashPassword(input.Password)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	id, err := s.repo.Create(ctx, userdomain.User{
		Login:        strings.TrimSpace(input.Login),
		PasswordHash: hash,
		Role:         input.Role,
		FullName:     strings.TrimSpace(input.FullName),
		Phone:        strings.TrimSpace(input.Phone),
		Email:        strings.TrimSpace(input.Email),
		IsActive:     true,
	})
	if err != nil {
		return 0, err
	}
	if err := s.completeIdempotent(ctx, "user.create", input.IdempotencyKey, id); err != nil {
		return 0, err
	}
	_ = s.writeAudit(ctx, id, actorID, "user_created", fmt.Sprintf("role=%s,login=%s", input.Role, strings.TrimSpace(input.Login)))
	return id, nil
}

func (s *Service) List(ctx context.Context) ([]userdomain.User, error) {
	return s.repo.List(ctx)
}

func (s *Service) FindByID(ctx context.Context, id int64) (userdomain.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) UpdateByAdmin(ctx context.Context, input UpdateInput) error {
	if input.ID <= 0 || strings.TrimSpace(input.FullName) == "" || strings.TrimSpace(input.Phone) == "" {
		return ErrInvalidInput
	}

	if input.Role != userdomain.RoleAdmin && input.Role != userdomain.RoleManager && input.Role != userdomain.RoleStaff && input.Role != userdomain.RoleClient {
		return ErrInvalidInput
	}

	current, err := s.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	current.FullName = strings.TrimSpace(input.FullName)
	current.Phone = strings.TrimSpace(input.Phone)
	current.Email = strings.TrimSpace(input.Email)
	current.Role = input.Role
	current.IsActive = input.IsActive

	if err := s.repo.Update(ctx, current); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, input.ID, 0, "user_updated", fmt.Sprintf("role=%s,active=%t", input.Role, input.IsActive))
	return nil
}

func (s *Service) DeleteByAdmin(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidInput
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, id, 0, "user_deleted", "")
	return nil
}

func (s *Service) writeAudit(ctx context.Context, entityID, actorUserID int64, action, payload string) error {
	if s.auditor == nil {
		return nil
	}
	return s.auditor.Write(ctx, "user", entityID, actorUserID, action, payload)
}

func (s *Service) beginIdempotent(ctx context.Context, scope, key, requestHash string) (bool, string, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return false, "", nil
	}
	return s.idempotency.Begin(ctx, scope, strings.TrimSpace(key), requestHash)
}

func (s *Service) completeIdempotent(ctx context.Context, scope, key string, entityID int64) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	return s.idempotency.Complete(ctx, scope, strings.TrimSpace(key), strconv.FormatInt(entityID, 10))
}

func hashCreateInput(input CreateInput) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(input.Login),
		strings.TrimSpace(input.Password),
		string(input.Role),
		strings.TrimSpace(input.FullName),
		strings.TrimSpace(input.Phone),
		strings.TrimSpace(input.Email),
	}, "|")))
	return hex.EncodeToString(sum[:])
}
