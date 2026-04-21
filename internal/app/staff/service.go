package staff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	availabilitydomain "project_cleaning/internal/domain/availability"
	orderdomain "project_cleaning/internal/domain/order"
)

var ErrOrderNotActionable = errors.New("действие с заказом недоступно")
var ErrInvalidAvailability = errors.New("некорректные данные периода недоступности")
var ErrAvailabilityOverlap = errors.New("период недоступности пересекается с существующим")

type Repository interface {
	ListForStaff(ctx context.Context, staffID int64) ([]orderdomain.Order, error)
	StaffAcceptOrder(ctx context.Context, orderID, staffID int64) error
	StaffDeclineOrder(ctx context.Context, orderID, staffID int64) error
	StaffStartOrder(ctx context.Context, orderID, staffID int64) error
	StaffCompleteOrder(ctx context.Context, orderID, staffID int64) error
	CreateStatusHistory(ctx context.Context, orderID, actorID int64, newStatus orderdomain.Status, comment string) error
	ListStaffUnavailability(ctx context.Context, staffID int64) ([]availabilitydomain.Period, error)
	CreateStaffUnavailability(ctx context.Context, period availabilitydomain.Period) error
	HasAvailabilityOverlap(ctx context.Context, staffID int64, startsAt, endsAt time.Time) (bool, error)
}

type Auditor interface {
	Write(ctx context.Context, entityType string, entityID, actorUserID int64, action, payload string) error
}

type IdempotencyStore interface {
	Begin(ctx context.Context, scope, key, requestHash string) (bool, string, error)
	Complete(ctx context.Context, scope, key, responseRef string) error
}

type Service struct {
	repo        Repository
	auditor     Auditor
	idempotency IdempotencyStore
}

type AvailabilityInput struct {
	StaffID        int64
	StartDate      string
	StartTime      string
	EndDate        string
	EndTime        string
	Reason         string
	IdempotencyKey string
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

func (s *Service) ListOrders(ctx context.Context, staffID int64) ([]orderdomain.Order, error) {
	return s.repo.ListForStaff(ctx, staffID)
}

func (s *Service) AcceptOrder(ctx context.Context, staffID, orderID int64) error {
	if err := s.repo.StaffAcceptOrder(ctx, orderID, staffID); err != nil {
		return fmt.Errorf("accept order: %w", err)
	}
	if err := s.repo.CreateStatusHistory(ctx, orderID, staffID, orderdomain.StatusStaffConfirmed, "Staff accepted order"); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, "order", orderID, staffID, "staff_accepted", "status=staff_confirmed")
	return nil
}

func (s *Service) DeclineOrder(ctx context.Context, staffID, orderID int64) error {
	if err := s.repo.StaffDeclineOrder(ctx, orderID, staffID); err != nil {
		return fmt.Errorf("decline order: %w", err)
	}
	if err := s.repo.CreateStatusHistory(ctx, orderID, staffID, orderdomain.StatusAssignedManager, "Staff declined order"); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, "order", orderID, staffID, "staff_declined", "status=assigned_manager")
	return nil
}

func (s *Service) StartOrder(ctx context.Context, staffID, orderID int64) error {
	if err := s.repo.StaffStartOrder(ctx, orderID, staffID); err != nil {
		return fmt.Errorf("start order: %w", err)
	}
	if err := s.repo.CreateStatusHistory(ctx, orderID, staffID, orderdomain.StatusInProgress, "Staff started order"); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, "order", orderID, staffID, "order_started", "status=in_progress")
	return nil
}

func (s *Service) CompleteOrder(ctx context.Context, staffID, orderID int64) error {
	if err := s.repo.StaffCompleteOrder(ctx, orderID, staffID); err != nil {
		return fmt.Errorf("complete order: %w", err)
	}
	if err := s.repo.CreateStatusHistory(ctx, orderID, staffID, orderdomain.StatusCompleted, "Staff completed order"); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, "order", orderID, staffID, "order_completed", "status=completed")
	return nil
}

func (s *Service) ListAvailability(ctx context.Context, staffID int64) ([]availabilitydomain.Period, error) {
	return s.repo.ListStaffUnavailability(ctx, staffID)
}

func (s *Service) AddAvailability(ctx context.Context, input AvailabilityInput) error {
	if input.StaffID <= 0 || strings.TrimSpace(input.StartDate) == "" || strings.TrimSpace(input.EndDate) == "" {
		return ErrInvalidAvailability
	}
	startsAt, endsAt, err := parseAvailabilityBounds(input.StartDate, input.StartTime, input.EndDate, input.EndTime)
	if err != nil {
		return err
	}
	input.Reason = strings.TrimSpace(input.Reason)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)

	if reused, _, err := s.beginIdempotent(ctx, "staff.availability.create", input.IdempotencyKey, hashAvailabilityInput(input, startsAt, endsAt)); err != nil {
		return err
	} else if reused {
		return nil
	}

	overlap, err := s.repo.HasAvailabilityOverlap(ctx, input.StaffID, startsAt, endsAt)
	if err != nil {
		return fmt.Errorf("check availability overlap: %w", err)
	}
	if overlap {
		return ErrAvailabilityOverlap
	}
	if err := s.repo.CreateStaffUnavailability(ctx, availabilitydomain.Period{
		StaffID:  input.StaffID,
		StartsAt: startsAt,
		EndsAt:   endsAt,
		Reason:   input.Reason,
	}); err != nil {
		return err
	}
	if err := s.completeIdempotent(ctx, "staff.availability.create", input.IdempotencyKey, input.StaffID); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, "availability", input.StaffID, input.StaffID, "availability_added", fmt.Sprintf("from=%s,to=%s", startsAt.Format("2006-01-02 15:04"), endsAt.Format("2006-01-02 15:04")))
	return nil
}

func parseAvailabilityBounds(startDate, startTime, endDate, endTime string) (time.Time, time.Time, error) {
	startDate = strings.TrimSpace(startDate)
	startTime = strings.TrimSpace(startTime)
	endDate = strings.TrimSpace(endDate)
	endTime = strings.TrimSpace(endTime)
	if startDate == "" || endDate == "" {
		return time.Time{}, time.Time{}, ErrInvalidAvailability
	}
	if startTime == "" {
		startTime = "00:00"
	}
	if endTime == "" {
		endTime = "23:59"
	}
	startsAt, err := time.Parse("2006-01-02 15:04", startDate+" "+startTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse availability start: %w", err)
	}
	endsAt, err := time.Parse("2006-01-02 15:04", endDate+" "+endTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse availability end: %w", err)
	}
	if !endsAt.After(startsAt) {
		return time.Time{}, time.Time{}, ErrInvalidAvailability
	}
	return startsAt, endsAt, nil
}

func (s *Service) writeAudit(ctx context.Context, entityType string, entityID, actorUserID int64, action, payload string) error {
	if s.auditor == nil {
		return nil
	}
	return s.auditor.Write(ctx, entityType, entityID, actorUserID, action, payload)
}

func (s *Service) beginIdempotent(ctx context.Context, scope, key, requestHash string) (bool, string, error) {
	if s.idempotency == nil || key == "" {
		return false, "", nil
	}
	return s.idempotency.Begin(ctx, scope, key, requestHash)
}

func (s *Service) completeIdempotent(ctx context.Context, scope, key string, entityID int64) error {
	if s.idempotency == nil || key == "" {
		return nil
	}
	return s.idempotency.Complete(ctx, scope, key, strconv.FormatInt(entityID, 10))
}

func hashAvailabilityInput(input AvailabilityInput, startsAt, endsAt time.Time) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strconv.FormatInt(input.StaffID, 10),
		startsAt.Format("2006-01-02 15:04"),
		endsAt.Format("2006-01-02 15:04"),
		input.Reason,
	}, "|")))
	return hex.EncodeToString(sum[:])
}
