package audit

import (
	"context"

	auditdomain "project_cleaning/internal/domain/audit"
)

type Repository interface {
	Create(ctx context.Context, entry auditdomain.Entry) error
	ListRecent(ctx context.Context, limit int) ([]auditdomain.Entry, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Write(ctx context.Context, entityType string, entityID, actorUserID int64, action, payload string) error {
	return s.repo.Create(ctx, auditdomain.Entry{
		EntityType:  entityType,
		EntityID:    entityID,
		Action:      action,
		ActorUserID: actorUserID,
		PayloadJSON: payload,
	})
}

func (s *Service) ListRecent(ctx context.Context, limit int) ([]auditdomain.Entry, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.ListRecent(ctx, limit)
}
