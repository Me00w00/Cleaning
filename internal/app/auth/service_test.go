package auth

import (
	"context"
	"testing"

	userdomain "project_cleaning/internal/domain/user"
	platformauth "project_cleaning/internal/platform/auth"
)

type stubUserFinder struct {
	user userdomain.User
	err  error
}

func (s stubUserFinder) FindByLogin(context.Context, string) (userdomain.User, error) {
	return s.user, s.err
}

func TestLoginSuccess(t *testing.T) {
	hash, err := platformauth.HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	svc := NewService(stubUserFinder{user: userdomain.User{
		Login:        "admin",
		PasswordHash: hash,
		Role:         userdomain.RoleAdmin,
		IsActive:     true,
	}})

	user, err := svc.Login(context.Background(), "admin", "secret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if user.Login != "admin" {
		t.Fatalf("expected admin login, got %s", user.Login)
	}
}

func TestLoginRejectsInactiveUser(t *testing.T) {
	hash, err := platformauth.HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	svc := NewService(stubUserFinder{user: userdomain.User{
		Login:        "admin",
		PasswordHash: hash,
		Role:         userdomain.RoleAdmin,
		IsActive:     false,
	}})

	if _, err := svc.Login(context.Background(), "admin", "secret"); err != ErrInactiveUser {
		t.Fatalf("expected ErrInactiveUser, got %v", err)
	}
}
