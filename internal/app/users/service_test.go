package users

import (
	"context"
	"errors"
	"testing"

	userdomain "project_cleaning/internal/domain/user"
	sqliterepo "project_cleaning/internal/repository/sqlite"
)

type stubRepository struct {
	createFn      func(ctx context.Context, user userdomain.User) (int64, error)
	updateFn      func(ctx context.Context, user userdomain.User) error
	deleteFn      func(ctx context.Context, id int64) error
	listFn        func(ctx context.Context) ([]userdomain.User, error)
	findByLoginFn func(ctx context.Context, login string) (userdomain.User, error)
	findByIDFn    func(ctx context.Context, id int64) (userdomain.User, error)
}

func (s stubRepository) Create(ctx context.Context, user userdomain.User) (int64, error) {
	return s.createFn(ctx, user)
}
func (s stubRepository) Update(ctx context.Context, user userdomain.User) error {
	return s.updateFn(ctx, user)
}
func (s stubRepository) Delete(ctx context.Context, id int64) error {
	return s.deleteFn(ctx, id)
}
func (s stubRepository) List(ctx context.Context) ([]userdomain.User, error) {
	return s.listFn(ctx)
}
func (s stubRepository) FindByLogin(ctx context.Context, login string) (userdomain.User, error) {
	return s.findByLoginFn(ctx, login)
}
func (s stubRepository) FindByID(ctx context.Context, id int64) (userdomain.User, error) {
	return s.findByIDFn(ctx, id)
}

type stubIdempotency struct {
	beginFn    func(ctx context.Context, scope, key, requestHash string) (bool, string, error)
	completeFn func(ctx context.Context, scope, key, responseRef string) error
}

func (s stubIdempotency) Begin(ctx context.Context, scope, key, requestHash string) (bool, string, error) {
	return s.beginFn(ctx, scope, key, requestHash)
}
func (s stubIdempotency) Complete(ctx context.Context, scope, key, responseRef string) error {
	return s.completeFn(ctx, scope, key, responseRef)
}

func TestRegisterClientSetsClientRole(t *testing.T) {
	var created userdomain.User
	svc := NewService(stubRepository{
		createFn: func(_ context.Context, user userdomain.User) (int64, error) {
			created = user
			return 5, nil
		},
		updateFn: func(context.Context, userdomain.User) error { return nil },
		deleteFn: func(context.Context, int64) error { return nil },
		listFn:   func(context.Context) ([]userdomain.User, error) { return nil, nil },
		findByLoginFn: func(context.Context, string) (userdomain.User, error) {
			return userdomain.User{}, sqliterepo.ErrUserNotFound
		},
		findByIDFn: func(context.Context, int64) (userdomain.User, error) {
			return userdomain.User{}, nil
		},
	})

	id, err := svc.RegisterClient(context.Background(), CreateInput{
		Login:    "client1",
		Password: "secret",
		FullName: "Client One",
		Phone:    "+100",
	})
	if err != nil {
		t.Fatalf("register client: %v", err)
	}

	if id != 5 {
		t.Fatalf("expected id 5, got %d", id)
	}
	if created.Role != userdomain.RoleClient {
		t.Fatalf("expected role client, got %s", created.Role)
	}
	if created.PasswordHash == "secret" || created.PasswordHash == "" {
		t.Fatalf("expected hashed password, got %q", created.PasswordHash)
	}
}

func TestRegisterClientIdempotentReturnsExistingID(t *testing.T) {
	createCalls := 0
	svc := NewService(stubRepository{
		createFn: func(context.Context, userdomain.User) (int64, error) {
			createCalls++
			return 0, nil
		},
		updateFn: func(context.Context, userdomain.User) error { return nil },
		deleteFn: func(context.Context, int64) error { return nil },
		listFn:   func(context.Context) ([]userdomain.User, error) { return nil, nil },
		findByLoginFn: func(context.Context, string) (userdomain.User, error) {
			return userdomain.User{}, sqliterepo.ErrUserNotFound
		},
		findByIDFn: func(context.Context, int64) (userdomain.User, error) { return userdomain.User{}, nil },
	}, stubIdempotency{
		beginFn:    func(context.Context, string, string, string) (bool, string, error) { return true, "11", nil },
		completeFn: func(context.Context, string, string, string) error { return nil },
	})

	id, err := svc.RegisterClient(context.Background(), CreateInput{
		Login:          "client1",
		Password:       "secret",
		FullName:       "Client One",
		Phone:          "+100",
		IdempotencyKey: "same-key",
	})
	if err != nil {
		t.Fatalf("register client idempotent: %v", err)
	}
	if id != 11 {
		t.Fatalf("expected id 11, got %d", id)
	}
	if createCalls != 0 {
		t.Fatalf("expected no create calls, got %d", createCalls)
	}
}

func TestCreateRejectsDuplicateLogin(t *testing.T) {
	svc := NewService(stubRepository{
		createFn: func(context.Context, userdomain.User) (int64, error) { return 0, nil },
		updateFn: func(context.Context, userdomain.User) error { return nil },
		deleteFn: func(context.Context, int64) error { return nil },
		listFn:   func(context.Context) ([]userdomain.User, error) { return nil, nil },
		findByLoginFn: func(context.Context, string) (userdomain.User, error) {
			return userdomain.User{ID: 1}, nil
		},
		findByIDFn: func(context.Context, int64) (userdomain.User, error) { return userdomain.User{}, nil },
	})

	_, err := svc.RegisterClient(context.Background(), CreateInput{
		Login:    "client1",
		Password: "secret",
		FullName: "Client One",
		Phone:    "+100",
	})
	if !errors.Is(err, ErrLoginExists) {
		t.Fatalf("expected ErrLoginExists, got %v", err)
	}
}
