package auth

import (
	"context"
	"errors"
	"strings"

	userdomain "project_cleaning/internal/domain/user"
	platformauth "project_cleaning/internal/platform/auth"
	sqliterepo "project_cleaning/internal/repository/sqlite"
)

var ErrInvalidCredentials = errors.New("неверный логин или пароль")
var ErrInactiveUser = errors.New("учетная запись деактивирована")

type UserFinder interface {
	FindByLogin(ctx context.Context, login string) (userdomain.User, error)
}

type Service struct {
	repo UserFinder
}

func NewService(repo UserFinder) *Service {
	return &Service{repo: repo}
}

func (s *Service) Login(ctx context.Context, login, password string) (userdomain.User, error) {
	login = strings.TrimSpace(login)
	password = strings.TrimSpace(password)
	if login == "" || password == "" {
		return userdomain.User{}, ErrInvalidCredentials
	}

	user, err := s.repo.FindByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, sqliterepo.ErrUserNotFound) {
			return userdomain.User{}, ErrInvalidCredentials
		}
		return userdomain.User{}, err
	}

	if !user.IsActive {
		return userdomain.User{}, ErrInactiveUser
	}

	if err := platformauth.VerifyPassword(user.PasswordHash, password); err != nil {
		return userdomain.User{}, ErrInvalidCredentials
	}

	return user, nil
}
