package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	availabilitydomain "project_cleaning/internal/domain/availability"
	orderdomain "project_cleaning/internal/domain/order"
)

type stubRepository struct {
	createAddressFn       func(ctx context.Context, address orderdomain.Address) (int64, error)
	updateAddressFn       func(ctx context.Context, address orderdomain.Address) error
	createOrderFn         func(ctx context.Context, order orderdomain.Order) (int64, error)
	updateOrderFn         func(ctx context.Context, order orderdomain.Order) error
	cancelOrderFn         func(ctx context.Context, orderID, clientID int64, cancelReason string) error
	deleteClientOrderFn   func(ctx context.Context, orderID, clientID int64) error
	deleteManagerOrderFn  func(ctx context.Context, orderID int64) error
	assignManagerFn       func(ctx context.Context, orderID, managerID int64) error
	assignStaffFn         func(ctx context.Context, orderID, managerID, staffID int64) error
	confirmPaymentFn      func(ctx context.Context, orderID, managerID int64) error
	closeOrderFn          func(ctx context.Context, orderID, managerID int64) error
	createStatusHistoryFn func(ctx context.Context, orderID, actorID int64, newStatus orderdomain.Status, comment string) error
	listStatusHistoryFn   func(ctx context.Context, orderID int64) ([]orderdomain.StatusHistoryEntry, error)
	listByClientFn        func(ctx context.Context, clientID int64) ([]orderdomain.Order, error)
	listForManagerFn      func(ctx context.Context) ([]orderdomain.Order, error)
	listServicesFn        func(ctx context.Context) ([]orderdomain.ServiceCatalogItem, error)
	getServiceByCodeFn    func(ctx context.Context, code string) (orderdomain.ServiceCatalogItem, error)
	findStaffConflictFn   func(ctx context.Context, staffID int64, scheduledStart, scheduledEnd time.Time) (availabilitydomain.Period, bool, error)
}

func (s stubRepository) CreateAddress(ctx context.Context, address orderdomain.Address) (int64, error) {
	return s.createAddressFn(ctx, address)
}
func (s stubRepository) UpdateAddress(ctx context.Context, address orderdomain.Address) error {
	return s.updateAddressFn(ctx, address)
}
func (s stubRepository) CreateOrder(ctx context.Context, order orderdomain.Order) (int64, error) {
	return s.createOrderFn(ctx, order)
}
func (s stubRepository) UpdateOrderForClient(ctx context.Context, order orderdomain.Order) error {
	return s.updateOrderFn(ctx, order)
}
func (s stubRepository) CancelOrderForClient(ctx context.Context, orderID, clientID int64, cancelReason string) error {
	return s.cancelOrderFn(ctx, orderID, clientID, cancelReason)
}
func (s stubRepository) DeleteHistoricalForClient(ctx context.Context, orderID, clientID int64) error {
	if s.deleteClientOrderFn != nil {
		return s.deleteClientOrderFn(ctx, orderID, clientID)
	}
	return nil
}
func (s stubRepository) DeleteHistoricalForManager(ctx context.Context, orderID int64) error {
	if s.deleteManagerOrderFn != nil {
		return s.deleteManagerOrderFn(ctx, orderID)
	}
	return nil
}
func (s stubRepository) AssignManager(ctx context.Context, orderID, managerID int64) error {
	return s.assignManagerFn(ctx, orderID, managerID)
}
func (s stubRepository) AssignStaff(ctx context.Context, orderID, managerID, staffID int64) error {
	return s.assignStaffFn(ctx, orderID, managerID, staffID)
}
func (s stubRepository) ConfirmPayment(ctx context.Context, orderID, managerID int64) error {
	return s.confirmPaymentFn(ctx, orderID, managerID)
}
func (s stubRepository) CloseOrder(ctx context.Context, orderID, managerID int64) error {
	return s.closeOrderFn(ctx, orderID, managerID)
}
func (s stubRepository) CreateStatusHistory(ctx context.Context, orderID, actorID int64, newStatus orderdomain.Status, comment string) error {
	return s.createStatusHistoryFn(ctx, orderID, actorID, newStatus, comment)
}
func (s stubRepository) ListStatusHistory(ctx context.Context, orderID int64) ([]orderdomain.StatusHistoryEntry, error) {
	return s.listStatusHistoryFn(ctx, orderID)
}
func (s stubRepository) ListByClient(ctx context.Context, clientID int64) ([]orderdomain.Order, error) {
	return s.listByClientFn(ctx, clientID)
}
func (s stubRepository) ListForManager(ctx context.Context) ([]orderdomain.Order, error) {
	return s.listForManagerFn(ctx)
}
func (s stubRepository) ListServices(ctx context.Context) ([]orderdomain.ServiceCatalogItem, error) {
	return s.listServicesFn(ctx)
}
func (s stubRepository) GetServiceByCode(ctx context.Context, code string) (orderdomain.ServiceCatalogItem, error) {
	return s.getServiceByCodeFn(ctx, code)
}
func (s stubRepository) FindStaffUnavailability(ctx context.Context, staffID int64, scheduledStart, scheduledEnd time.Time) (availabilitydomain.Period, bool, error) {
	if s.findStaffConflictFn != nil {
		return s.findStaffConflictFn(ctx, staffID, scheduledStart, scheduledEnd)
	}
	return availabilitydomain.Period{}, false, nil
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

func TestCreateOrderCalculatesPriceAndInitialStatus(t *testing.T) {
	var createdOrder orderdomain.Order
	svc := NewService(stubRepository{
		createAddressFn: func(context.Context, orderdomain.Address) (int64, error) { return 10, nil },
		updateAddressFn: func(context.Context, orderdomain.Address) error { return nil },
		createOrderFn: func(_ context.Context, order orderdomain.Order) (int64, error) {
			createdOrder = order
			return 25, nil
		},
		updateOrderFn:         func(context.Context, orderdomain.Order) error { return nil },
		cancelOrderFn:         func(context.Context, int64, int64, string) error { return nil },
		assignManagerFn:       func(context.Context, int64, int64) error { return nil },
		assignStaffFn:         func(context.Context, int64, int64, int64) error { return nil },
		confirmPaymentFn:      func(context.Context, int64, int64) error { return nil },
		closeOrderFn:          func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn: func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStatusHistoryFn:   func(context.Context, int64) ([]orderdomain.StatusHistoryEntry, error) { return nil, nil },
		listByClientFn:        func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		listForManagerFn:      func(context.Context) ([]orderdomain.Order, error) { return nil, nil },
		listServicesFn:        func(context.Context) ([]orderdomain.ServiceCatalogItem, error) { return nil, nil },
		getServiceByCodeFn: func(context.Context, string) (orderdomain.ServiceCatalogItem, error) {
			return orderdomain.ServiceCatalogItem{Code: "basic_cleaning", BasePrice: 100, PricePerSquareMeter: 10, PricePerWindow: 5}, nil
		},
	})

	id, err := svc.CreateOrder(context.Background(), CreateOrderInput{ClientID: 7, City: "Moscow", Street: "Tverskaya", House: "1", ScheduledDate: "2026-04-01", ServiceType: "basic_cleaning", Square: 20, WindowCount: 3})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if id != 25 {
		t.Fatalf("expected order id 25, got %d", id)
	}
	if createdOrder.PriceTotal != 315 {
		t.Fatalf("expected price 315, got %d", createdOrder.PriceTotal)
	}
	if createdOrder.Status != orderdomain.StatusNew {
		t.Fatalf("expected status new, got %s", createdOrder.Status)
	}
	if createdOrder.Address.ID != 10 {
		t.Fatalf("expected address id 10, got %d", createdOrder.Address.ID)
	}
}

func TestCreateOrderIdempotentReturnsExistingID(t *testing.T) {
	createCalls := 0
	svc := NewService(stubRepository{
		createAddressFn: func(context.Context, orderdomain.Address) (int64, error) { createCalls++; return 10, nil },
		updateAddressFn: func(context.Context, orderdomain.Address) error { return nil },
		createOrderFn: func(context.Context, orderdomain.Order) (int64, error) {
			t.Fatal("should not create order")
			return 0, nil
		},
		updateOrderFn:         func(context.Context, orderdomain.Order) error { return nil },
		cancelOrderFn:         func(context.Context, int64, int64, string) error { return nil },
		assignManagerFn:       func(context.Context, int64, int64) error { return nil },
		assignStaffFn:         func(context.Context, int64, int64, int64) error { return nil },
		confirmPaymentFn:      func(context.Context, int64, int64) error { return nil },
		closeOrderFn:          func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn: func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStatusHistoryFn:   func(context.Context, int64) ([]orderdomain.StatusHistoryEntry, error) { return nil, nil },
		listByClientFn:        func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		listForManagerFn:      func(context.Context) ([]orderdomain.Order, error) { return nil, nil },
		listServicesFn:        func(context.Context) ([]orderdomain.ServiceCatalogItem, error) { return nil, nil },
		getServiceByCodeFn: func(context.Context, string) (orderdomain.ServiceCatalogItem, error) {
			return orderdomain.ServiceCatalogItem{Code: "basic_cleaning", BasePrice: 100, PricePerSquareMeter: 10, PricePerWindow: 5}, nil
		},
	}, stubIdempotency{
		beginFn:    func(context.Context, string, string, string) (bool, string, error) { return true, "77", nil },
		completeFn: func(context.Context, string, string, string) error { return nil },
	})

	id, err := svc.CreateOrder(context.Background(), CreateOrderInput{ClientID: 7, City: "Moscow", Street: "Tverskaya", House: "1", ScheduledDate: "2026-04-01", ServiceType: "basic_cleaning", Square: 20, WindowCount: 3, IdempotencyKey: "key-1"})
	if err != nil {
		t.Fatalf("create order idempotent: %v", err)
	}
	if id != 77 {
		t.Fatalf("expected order id 77, got %d", id)
	}
	if createCalls != 0 {
		t.Fatalf("expected no writes, got %d", createCalls)
	}
}

func TestUpdateClientOrderAllowedOnlyForNew(t *testing.T) {
	var updatedAddress orderdomain.Address
	var updatedOrder orderdomain.Order
	svc := NewService(stubRepository{
		createAddressFn:       func(context.Context, orderdomain.Address) (int64, error) { return 0, nil },
		updateAddressFn:       func(_ context.Context, address orderdomain.Address) error { updatedAddress = address; return nil },
		createOrderFn:         func(context.Context, orderdomain.Order) (int64, error) { return 0, nil },
		updateOrderFn:         func(_ context.Context, order orderdomain.Order) error { updatedOrder = order; return nil },
		cancelOrderFn:         func(context.Context, int64, int64, string) error { return nil },
		assignManagerFn:       func(context.Context, int64, int64) error { return nil },
		assignStaffFn:         func(context.Context, int64, int64, int64) error { return nil },
		confirmPaymentFn:      func(context.Context, int64, int64) error { return nil },
		closeOrderFn:          func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn: func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStatusHistoryFn:   func(context.Context, int64) ([]orderdomain.StatusHistoryEntry, error) { return nil, nil },
		listByClientFn: func(context.Context, int64) ([]orderdomain.Order, error) {
			return []orderdomain.Order{{ID: 12, ClientID: 7, Address: orderdomain.Address{ID: 44, City: "Moscow"}, Status: orderdomain.StatusNew, PaymentStatus: orderdomain.PaymentStatusUnpaid}}, nil
		},
		listForManagerFn: func(context.Context) ([]orderdomain.Order, error) { return nil, nil },
		listServicesFn:   func(context.Context) ([]orderdomain.ServiceCatalogItem, error) { return nil, nil },
		getServiceByCodeFn: func(context.Context, string) (orderdomain.ServiceCatalogItem, error) {
			return orderdomain.ServiceCatalogItem{Code: "general_cleaning", BasePrice: 1000, PricePerSquareMeter: 20, PricePerWindow: 10}, nil
		},
	})

	err := svc.UpdateClientOrder(context.Background(), UpdateOrderInput{OrderID: 12, ClientID: 7, City: "Saint-Petersburg", Street: "Nevsky", House: "10", ScheduledDate: "2026-04-03", ServiceType: "general_cleaning", Square: 10, WindowCount: 2})
	if err != nil {
		t.Fatalf("update order: %v", err)
	}
	if updatedAddress.ID != 44 {
		t.Fatalf("expected existing address id 44, got %d", updatedAddress.ID)
	}
	if updatedOrder.PriceTotal != 1220 {
		t.Fatalf("expected recalculated price 1220, got %d", updatedOrder.PriceTotal)
	}
}

func TestCancelClientOrderAllowedOnlyForNew(t *testing.T) {
	cancelled := false
	historyWritten := false
	svc := NewService(stubRepository{
		createAddressFn:  func(context.Context, orderdomain.Address) (int64, error) { return 0, nil },
		updateAddressFn:  func(context.Context, orderdomain.Address) error { return nil },
		createOrderFn:    func(context.Context, orderdomain.Order) (int64, error) { return 0, nil },
		updateOrderFn:    func(context.Context, orderdomain.Order) error { return nil },
		cancelOrderFn:    func(context.Context, int64, int64, string) error { cancelled = true; return nil },
		assignManagerFn:  func(context.Context, int64, int64) error { return nil },
		assignStaffFn:    func(context.Context, int64, int64, int64) error { return nil },
		confirmPaymentFn: func(context.Context, int64, int64) error { return nil },
		closeOrderFn:     func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn: func(context.Context, int64, int64, orderdomain.Status, string) error {
			historyWritten = true
			return nil
		},
		listStatusHistoryFn: func(context.Context, int64) ([]orderdomain.StatusHistoryEntry, error) { return nil, nil },
		listByClientFn: func(context.Context, int64) ([]orderdomain.Order, error) {
			return []orderdomain.Order{{ID: 21, ClientID: 7, Status: orderdomain.StatusNew, Address: orderdomain.Address{ID: 5}, CreatedAt: time.Now()}}, nil
		},
		listForManagerFn: func(context.Context) ([]orderdomain.Order, error) { return nil, nil },
		listServicesFn:   func(context.Context) ([]orderdomain.ServiceCatalogItem, error) { return nil, nil },
		getServiceByCodeFn: func(context.Context, string) (orderdomain.ServiceCatalogItem, error) {
			return orderdomain.ServiceCatalogItem{}, nil
		},
	})

	if err := svc.CancelClientOrder(context.Background(), 7, 21); err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	if !cancelled || !historyWritten {
		t.Fatalf("expected cancellation and history write")
	}
}

func TestDeleteHistoricalOrderAllowedOnlyForHistoricalStatuses(t *testing.T) {
	deleted := false
	svc := NewService(stubRepository{
		createAddressFn:       func(context.Context, orderdomain.Address) (int64, error) { return 0, nil },
		updateAddressFn:       func(context.Context, orderdomain.Address) error { return nil },
		createOrderFn:         func(context.Context, orderdomain.Order) (int64, error) { return 0, nil },
		updateOrderFn:         func(context.Context, orderdomain.Order) error { return nil },
		cancelOrderFn:         func(context.Context, int64, int64, string) error { return nil },
		deleteClientOrderFn:   func(context.Context, int64, int64) error { deleted = true; return nil },
		assignManagerFn:       func(context.Context, int64, int64) error { return nil },
		assignStaffFn:         func(context.Context, int64, int64, int64) error { return nil },
		confirmPaymentFn:      func(context.Context, int64, int64) error { return nil },
		closeOrderFn:          func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn: func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStatusHistoryFn:   func(context.Context, int64) ([]orderdomain.StatusHistoryEntry, error) { return nil, nil },
		listByClientFn: func(context.Context, int64) ([]orderdomain.Order, error) {
			return []orderdomain.Order{{ID: 50, ClientID: 7, Status: orderdomain.StatusClosed, Address: orderdomain.Address{ID: 5}}}, nil
		},
		listForManagerFn: func(context.Context) ([]orderdomain.Order, error) { return nil, nil },
		listServicesFn:   func(context.Context) ([]orderdomain.ServiceCatalogItem, error) { return nil, nil },
		getServiceByCodeFn: func(context.Context, string) (orderdomain.ServiceCatalogItem, error) {
			return orderdomain.ServiceCatalogItem{}, nil
		},
	})

	if err := svc.DeleteClientHistoricalOrder(context.Background(), 7, 50); err != nil {
		t.Fatalf("delete historical order: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete call")
	}
}

func TestAssignManagerAndStaffAndPaymentAndClose(t *testing.T) {
	managerAssigned := false
	staffAssigned := false
	paymentConfirmed := false
	orderClosed := false
	var availabilityStart time.Time
	var availabilityEnd time.Time
	svc := NewService(stubRepository{
		createAddressFn:       func(context.Context, orderdomain.Address) (int64, error) { return 0, nil },
		updateAddressFn:       func(context.Context, orderdomain.Address) error { return nil },
		createOrderFn:         func(context.Context, orderdomain.Order) (int64, error) { return 0, nil },
		updateOrderFn:         func(context.Context, orderdomain.Order) error { return nil },
		cancelOrderFn:         func(context.Context, int64, int64, string) error { return nil },
		assignManagerFn:       func(context.Context, int64, int64) error { managerAssigned = true; return nil },
		assignStaffFn:         func(context.Context, int64, int64, int64) error { staffAssigned = true; return nil },
		confirmPaymentFn:      func(context.Context, int64, int64) error { paymentConfirmed = true; return nil },
		closeOrderFn:          func(context.Context, int64, int64) error { orderClosed = true; return nil },
		createStatusHistoryFn: func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStatusHistoryFn:   func(context.Context, int64) ([]orderdomain.StatusHistoryEntry, error) { return nil, nil },
		listByClientFn:        func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		listForManagerFn: func(context.Context) ([]orderdomain.Order, error) {
			return []orderdomain.Order{
				{ID: 1, Status: orderdomain.StatusNew, PaymentStatus: orderdomain.PaymentStatusUnpaid},
				{ID: 2, Status: orderdomain.StatusAssignedManager, ManagerID: 9, PaymentStatus: orderdomain.PaymentStatusUnpaid, ScheduledDate: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), ScheduledTimeFrom: "10:00", ScheduledTimeTo: "12:00"},
				{ID: 3, Status: orderdomain.StatusCompleted, ManagerID: 9, PaymentStatus: orderdomain.PaymentStatusPaid},
			}, nil
		},
		listServicesFn: func(context.Context) ([]orderdomain.ServiceCatalogItem, error) { return nil, nil },
		findStaffConflictFn: func(_ context.Context, _ int64, scheduledStart, scheduledEnd time.Time) (availabilitydomain.Period, bool, error) {
			availabilityStart = scheduledStart
			availabilityEnd = scheduledEnd
			return availabilitydomain.Period{}, false, nil
		},
		getServiceByCodeFn: func(context.Context, string) (orderdomain.ServiceCatalogItem, error) {
			return orderdomain.ServiceCatalogItem{}, nil
		},
	})

	if err := svc.AssignManager(context.Background(), 9, 1); err != nil {
		t.Fatalf("assign manager: %v", err)
	}
	if err := svc.AssignStaff(context.Background(), 9, 2, 15); err != nil {
		t.Fatalf("assign staff: %v", err)
	}
	if err := svc.ConfirmPayment(context.Background(), 9, 2); err != nil {
		t.Fatalf("confirm payment: %v", err)
	}
	if err := svc.CloseOrder(context.Background(), 9, 3); err != nil {
		t.Fatalf("close order: %v", err)
	}
	if !managerAssigned || !staffAssigned || !paymentConfirmed || !orderClosed {
		t.Fatalf("expected all manager actions to execute")
	}
	if availabilityStart.Format("2006-01-02 15:04") != "2026-04-10 10:00" || availabilityEnd.Format("2006-01-02 15:04") != "2026-04-10 12:00" {
		t.Fatalf("unexpected availability bounds: %s - %s", availabilityStart, availabilityEnd)
	}
}

func TestAssignStaffRejectsUnavailableStaffWithReason(t *testing.T) {
	svc := NewService(stubRepository{
		createAddressFn:       func(context.Context, orderdomain.Address) (int64, error) { return 0, nil },
		updateAddressFn:       func(context.Context, orderdomain.Address) error { return nil },
		createOrderFn:         func(context.Context, orderdomain.Order) (int64, error) { return 0, nil },
		updateOrderFn:         func(context.Context, orderdomain.Order) error { return nil },
		cancelOrderFn:         func(context.Context, int64, int64, string) error { return nil },
		assignManagerFn:       func(context.Context, int64, int64) error { return nil },
		assignStaffFn:         func(context.Context, int64, int64, int64) error { t.Fatal("should not assign"); return nil },
		confirmPaymentFn:      func(context.Context, int64, int64) error { return nil },
		closeOrderFn:          func(context.Context, int64, int64) error { return nil },
		createStatusHistoryFn: func(context.Context, int64, int64, orderdomain.Status, string) error { return nil },
		listStatusHistoryFn:   func(context.Context, int64) ([]orderdomain.StatusHistoryEntry, error) { return nil, nil },
		listByClientFn:        func(context.Context, int64) ([]orderdomain.Order, error) { return nil, nil },
		listForManagerFn: func(context.Context) ([]orderdomain.Order, error) {
			return []orderdomain.Order{{ID: 2, Status: orderdomain.StatusAssignedManager, ManagerID: 9, ScheduledDate: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), ScheduledTimeFrom: "10:00", ScheduledTimeTo: "12:00"}}, nil
		},
		listServicesFn: func(context.Context) ([]orderdomain.ServiceCatalogItem, error) { return nil, nil },
		findStaffConflictFn: func(context.Context, int64, time.Time, time.Time) (availabilitydomain.Period, bool, error) {
			return availabilitydomain.Period{Reason: "отпуск", StartsAt: time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC), EndsAt: time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC)}, true, nil
		},
		getServiceByCodeFn: func(context.Context, string) (orderdomain.ServiceCatalogItem, error) {
			return orderdomain.ServiceCatalogItem{}, nil
		},
	})

	err := svc.AssignStaff(context.Background(), 9, 2, 15)
	if err == nil {
		t.Fatal("expected unavailable error")
	}
	var unavailable *StaffUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("expected StaffUnavailableError, got %v", err)
	}
	if unavailable.Reason != "отпуск" {
		t.Fatalf("expected reason отпуск, got %q", unavailable.Reason)
	}
}
