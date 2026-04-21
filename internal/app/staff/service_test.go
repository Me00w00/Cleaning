package staff

import (
	"context"
	"testing"
	"time"

	availabilitydomain "project_cleaning/internal/domain/availability"
	orderdomain "project_cleaning/internal/domain/order"
)

type stubRepository struct {
	listForStaffFn            func(ctx context.Context, staffID int64) ([]orderdomain.Order, error)
	staffAcceptOrderFn        func(ctx context.Context, orderID, staffID int64) error
	staffDeclineOrderFn       func(ctx context.Context, orderID, staffID int64) error
	staffStartOrderFn         func(ctx context.Context, orderID, staffID int64) error
	staffCompleteOrderFn      func(ctx context.Context, orderID, staffID int64) error
	createStatusHistoryFn     func(ctx context.Context, orderID, actorID int64, newStatus orderdomain.Status, comment string) error
	listStaffUnavailabilityFn func(ctx context.Context, staffID int64) ([]availabilitydomain.Period, error)
	createStaffUnavailableFn  func(ctx context.Context, period availabilitydomain.Period) error
	hasAvailabilityOverlapFn  func(ctx context.Context, staffID int64, startsAt, endsAt time.Time) (bool, error)
}

func (s stubRepository) ListForStaff(ctx context.Context, staffID int64) ([]orderdomain.Order, error) {
	return s.listForStaffFn(ctx, staffID)
}
func (s stubRepository) StaffAcceptOrder(ctx context.Context, orderID, staffID int64) error {
	return s.staffAcceptOrderFn(ctx, orderID, staffID)
}
func (s stubRepository) StaffDeclineOrder(ctx context.Context, orderID, staffID int64) error {
	return s.staffDeclineOrderFn(ctx, orderID, staffID)
}
func (s stubRepository) StaffStartOrder(ctx context.Context, orderID, staffID int64) error {
	return s.staffStartOrderFn(ctx, orderID, staffID)
}
func (s stubRepository) StaffCompleteOrder(ctx context.Context, orderID, staffID int64) error {
	return s.staffCompleteOrderFn(ctx, orderID, staffID)
}
func (s stubRepository) CreateStatusHistory(ctx context.Context, orderID, actorID int64, newStatus orderdomain.Status, comment string) error {
	return s.createStatusHistoryFn(ctx, orderID, actorID, newStatus, comment)
}
func (s stubRepository) ListStaffUnavailability(ctx context.Context, staffID int64) ([]availabilitydomain.Period, error) {
	return s.listStaffUnavailabilityFn(ctx, staffID)
}
func (s stubRepository) CreateStaffUnavailability(ctx context.Context, period availabilitydomain.Period) error {
	return s.createStaffUnavailableFn(ctx, period)
}
func (s stubRepository) HasAvailabilityOverlap(ctx context.Context, staffID int64, startsAt, endsAt time.Time) (bool, error) {
	return s.hasAvailabilityOverlapFn(ctx, staffID, startsAt, endsAt)
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

func TestStaffOrderActionsWriteStatusHistory(t *testing.T) {
	accepted := false
	declined := false
	started := false
	completed := false
	historyStatuses := make([]orderdomain.Status, 0, 4)

	svc := NewService(stubRepository{
		listForStaffFn:       func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		staffAcceptOrderFn:   func(context.Context, int64, int64) error { accepted = true; return nil },
		staffDeclineOrderFn:  func(context.Context, int64, int64) error { declined = true; return nil },
		staffStartOrderFn:    func(context.Context, int64, int64) error { started = true; return nil },
		staffCompleteOrderFn: func(context.Context, int64, int64) error { completed = true; return nil },
		createStatusHistoryFn: func(_ context.Context, _ int64, _ int64, newStatus orderdomain.Status, _ string) error {
			historyStatuses = append(historyStatuses, newStatus)
			return nil
		},
		listStaffUnavailabilityFn: func(context.Context, int64) ([]availabilitydomain.Period, error) { return nil, nil },
		createStaffUnavailableFn:  func(context.Context, availabilitydomain.Period) error { return nil },
		hasAvailabilityOverlapFn:  func(context.Context, int64, time.Time, time.Time) (bool, error) { return false, nil },
	})

	if err := svc.AcceptOrder(context.Background(), 5, 10); err != nil {
		t.Fatalf("accept order: %v", err)
	}
	if err := svc.DeclineOrder(context.Background(), 5, 10); err != nil {
		t.Fatalf("decline order: %v", err)
	}
	if err := svc.StartOrder(context.Background(), 5, 10); err != nil {
		t.Fatalf("start order: %v", err)
	}
	if err := svc.CompleteOrder(context.Background(), 5, 10); err != nil {
		t.Fatalf("complete order: %v", err)
	}

	if !accepted || !declined || !started || !completed {
		t.Fatalf("expected all staff actions to execute")
	}
	if len(historyStatuses) != 4 {
		t.Fatalf("expected 4 history writes, got %d", len(historyStatuses))
	}
	if historyStatuses[0] != orderdomain.StatusStaffConfirmed || historyStatuses[1] != orderdomain.StatusAssignedManager || historyStatuses[2] != orderdomain.StatusInProgress || historyStatuses[3] != orderdomain.StatusCompleted {
		t.Fatalf("unexpected history statuses: %#v", historyStatuses)
	}
}

func TestAddAvailabilityValidatesDates(t *testing.T) {
	var created availabilitydomain.Period
	svc := NewService(stubRepository{
		listForStaffFn:            func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		staffAcceptOrderFn:        func(context.Context, int64, int64) error { return nil },
		staffDeclineOrderFn:       func(context.Context, int64, int64) error { return nil },
		staffStartOrderFn:         func(context.Context, int64, int64) error { return nil },
		staffCompleteOrderFn:      func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn:     func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStaffUnavailabilityFn: func(context.Context, int64) ([]availabilitydomain.Period, error) { return nil, nil },
		createStaffUnavailableFn:  func(_ context.Context, period availabilitydomain.Period) error { created = period; return nil },
		hasAvailabilityOverlapFn:  func(context.Context, int64, time.Time, time.Time) (bool, error) { return false, nil },
	})

	if err := svc.AddAvailability(context.Background(), AvailabilityInput{StaffID: 7, StartDate: "2026-05-01", StartTime: "09:00", EndDate: "2026-05-03", EndTime: "18:00", Reason: "Отпуск"}); err != nil {
		t.Fatalf("add availability: %v", err)
	}
	if created.StaffID != 7 {
		t.Fatalf("expected staff id 7, got %d", created.StaffID)
	}
	if created.StartsAt.Format("2006-01-02 15:04") != "2026-05-01 09:00" || created.EndsAt.Format("2006-01-02 15:04") != "2026-05-03 18:00" {
		t.Fatalf("unexpected bounds: %s - %s", created.StartsAt, created.EndsAt)
	}
	if created.Reason != "Отпуск" {
		t.Fatalf("unexpected reason: %s", created.Reason)
	}
}

func TestAddAvailabilityRejectsOverlap(t *testing.T) {
	svc := NewService(stubRepository{
		listForStaffFn:            func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		staffAcceptOrderFn:        func(context.Context, int64, int64) error { return nil },
		staffDeclineOrderFn:       func(context.Context, int64, int64) error { return nil },
		staffStartOrderFn:         func(context.Context, int64, int64) error { return nil },
		staffCompleteOrderFn:      func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn:     func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStaffUnavailabilityFn: func(context.Context, int64) ([]availabilitydomain.Period, error) { return nil, nil },
		createStaffUnavailableFn: func(context.Context, availabilitydomain.Period) error {
			t.Fatal("should not create overlap")
			return nil
		},
		hasAvailabilityOverlapFn: func(context.Context, int64, time.Time, time.Time) (bool, error) { return true, nil },
	})

	if err := svc.AddAvailability(context.Background(), AvailabilityInput{StaffID: 7, StartDate: "2026-05-01", StartTime: "09:00", EndDate: "2026-05-03", EndTime: "18:00", Reason: "Отпуск"}); err != ErrAvailabilityOverlap {
		t.Fatalf("expected ErrAvailabilityOverlap, got %v", err)
	}
}

func TestAddAvailabilityIdempotent(t *testing.T) {
	createCalls := 0
	svc := NewService(stubRepository{
		listForStaffFn:            func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		staffAcceptOrderFn:        func(context.Context, int64, int64) error { return nil },
		staffDeclineOrderFn:       func(context.Context, int64, int64) error { return nil },
		staffStartOrderFn:         func(context.Context, int64, int64) error { return nil },
		staffCompleteOrderFn:      func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn:     func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStaffUnavailabilityFn: func(context.Context, int64) ([]availabilitydomain.Period, error) { return nil, nil },
		createStaffUnavailableFn:  func(context.Context, availabilitydomain.Period) error { createCalls++; return nil },
		hasAvailabilityOverlapFn:  func(context.Context, int64, time.Time, time.Time) (bool, error) { return false, nil },
	}, stubIdempotency{
		beginFn:    func(context.Context, string, string, string) (bool, string, error) { return true, "7", nil },
		completeFn: func(context.Context, string, string, string) error { return nil },
	})

	if err := svc.AddAvailability(context.Background(), AvailabilityInput{StaffID: 7, StartDate: "2026-05-01", StartTime: "09:00", EndDate: "2026-05-03", EndTime: "18:00", Reason: "Отпуск", IdempotencyKey: "same-key"}); err != nil {
		t.Fatalf("idempotent add availability: %v", err)
	}
	if createCalls != 0 {
		t.Fatalf("expected no create calls, got %d", createCalls)
	}
}

func TestListAvailability(t *testing.T) {
	expectedStart := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	svc := NewService(stubRepository{
		listForStaffFn:       func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		staffAcceptOrderFn:   func(context.Context, int64, int64) error { return nil },
		staffDeclineOrderFn:  func(context.Context, int64, int64) error { return nil },
		staffStartOrderFn:    func(context.Context, int64, int64) error { return nil },
		staffCompleteOrderFn: func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn: func(context.Context, int64, int64, orderdomain.Status, string) error {
			return nil
		},
		listStaffUnavailabilityFn: func(context.Context, int64) ([]availabilitydomain.Period, error) {
			return []availabilitydomain.Period{{ID: 1, StaffID: 9, StartsAt: expectedStart, EndsAt: expectedEnd}}, nil
		},
		createStaffUnavailableFn: func(context.Context, availabilitydomain.Period) error { return nil },
		hasAvailabilityOverlapFn: func(context.Context, int64, time.Time, time.Time) (bool, error) { return false, nil },
	})

	items, err := svc.ListAvailability(context.Background(), 9)
	if err != nil {
		t.Fatalf("list availability: %v", err)
	}
	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("unexpected items: %#v", items)
	}
}
