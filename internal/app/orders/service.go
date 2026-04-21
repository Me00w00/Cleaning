package orders

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
	userdomain "project_cleaning/internal/domain/user"
)

var ErrInvalidOrderInput = errors.New("некорректные данные заказа")
var ErrOrderNotEditable = errors.New("заказ нельзя изменить")
var ErrOrderNotFound = errors.New("заказ не найден")
var ErrOrderNotManageable = errors.New("действие с заказом недоступно")
var ErrStaffUnavailable = errors.New("сотрудник недоступен в выбранный период")

type StaffUnavailableError struct {
	Reason   string
	StartsAt time.Time
	EndsAt   time.Time
}

func (e *StaffUnavailableError) Error() string {
	reason := strings.TrimSpace(e.Reason)
	if reason == "" {
		return ErrStaffUnavailable.Error()
	}
	return fmt.Sprintf("%s: %s", ErrStaffUnavailable.Error(), reason)
}

func (e *StaffUnavailableError) Unwrap() error {
	return ErrStaffUnavailable
}

type Repository interface {
	CreateAddress(ctx context.Context, address orderdomain.Address) (int64, error)
	UpdateAddress(ctx context.Context, address orderdomain.Address) error
	CreateOrder(ctx context.Context, order orderdomain.Order) (int64, error)
	UpdateOrderForClient(ctx context.Context, order orderdomain.Order) error
	CancelOrderForClient(ctx context.Context, orderID, clientID int64, cancelReason string) error
	DeleteHistoricalForClient(ctx context.Context, orderID, clientID int64) error
	DeleteHistoricalForManager(ctx context.Context, orderID int64) error
	AssignManager(ctx context.Context, orderID, managerID int64) error
	AssignStaff(ctx context.Context, orderID, managerID, staffID int64) error
	ConfirmPayment(ctx context.Context, orderID, managerID int64) error
	CloseOrder(ctx context.Context, orderID, managerID int64) error
	CreateStatusHistory(ctx context.Context, orderID, actorID int64, newStatus orderdomain.Status, comment string) error
	ListStatusHistory(ctx context.Context, orderID int64) ([]orderdomain.StatusHistoryEntry, error)
	ListByClient(ctx context.Context, clientID int64) ([]orderdomain.Order, error)
	ListForManager(ctx context.Context) ([]orderdomain.Order, error)
	ListServices(ctx context.Context) ([]orderdomain.ServiceCatalogItem, error)
	GetServiceByCode(ctx context.Context, code string) (orderdomain.ServiceCatalogItem, error)
	FindStaffUnavailability(ctx context.Context, staffID int64, scheduledStart, scheduledEnd time.Time) (availabilitydomain.Period, bool, error)
}

type Auditor interface {
	Write(ctx context.Context, entityType string, entityID, actorUserID int64, action, payload string) error
}

type IdempotencyStore interface {
	Begin(ctx context.Context, scope, key, requestHash string) (bool, string, error)
	Complete(ctx context.Context, scope, key, responseRef string) error
}

type CreateOrderInput struct {
	ClientID          int64
	City              string
	Street            string
	House             string
	Floor             string
	Flat              string
	Entrance          string
	AddressComment    string
	ScheduledDate     string
	ScheduledTimeFrom string
	ScheduledTimeTo   string
	ServiceType       string
	Details           string
	Square            int
	WindowCount       int
	IdempotencyKey    string
}

type UpdateOrderInput struct {
	OrderID           int64
	ClientID          int64
	City              string
	Street            string
	House             string
	Floor             string
	Flat              string
	Entrance          string
	AddressComment    string
	ScheduledDate     string
	ScheduledTimeFrom string
	ScheduledTimeTo   string
	ServiceType       string
	Details           string
	Square            int
	WindowCount       int
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

func (s *Service) ListClientOrders(ctx context.Context, clientID int64) ([]orderdomain.Order, error) {
	return s.repo.ListByClient(ctx, clientID)
}

func (s *Service) ListManagerOrders(ctx context.Context) ([]orderdomain.Order, error) {
	return s.repo.ListForManager(ctx)
}

func (s *Service) ListServices(ctx context.Context) ([]orderdomain.ServiceCatalogItem, error) {
	return s.repo.ListServices(ctx)
}

func (s *Service) ListOrderStatusHistory(ctx context.Context, orderID int64) ([]orderdomain.StatusHistoryEntry, error) {
	return s.repo.ListStatusHistory(ctx, orderID)
}

func (s *Service) CreateOrder(ctx context.Context, input CreateOrderInput) (int64, error) {
	normalized, service, err := s.normalizeInput(ctx, CreateOrderInput(input))
	if err != nil {
		return 0, err
	}

	if reused, responseRef, err := s.beginIdempotent(ctx, "order.create", normalized.IdempotencyKey, hashCreateOrderInput(normalized)); err != nil {
		return 0, err
	} else if reused {
		return strconv.ParseInt(responseRef, 10, 64)
	}

	priceTotal := service.BasePrice + service.PricePerSquareMeter*normalized.Square + service.PricePerWindow*normalized.WindowCount

	addressID, err := s.repo.CreateAddress(ctx, orderdomain.Address{
		City:     normalized.City,
		Street:   normalized.Street,
		House:    normalized.House,
		Floor:    normalized.Floor,
		Flat:     normalized.Flat,
		Entrance: normalized.Entrance,
		Comment:  normalized.AddressComment,
	})
	if err != nil {
		return 0, fmt.Errorf("create address: %w", err)
	}

	scheduledDate, _ := time.Parse("2006-01-02", normalized.ScheduledDate)
	orderID, err := s.repo.CreateOrder(ctx, orderdomain.Order{
		ClientID:          normalized.ClientID,
		Address:           orderdomain.Address{ID: addressID},
		ScheduledDate:     scheduledDate,
		ScheduledTimeFrom: normalized.ScheduledTimeFrom,
		ScheduledTimeTo:   normalized.ScheduledTimeTo,
		ServiceType:       service.Code,
		Details:           normalized.Details,
		Square:            normalized.Square,
		WindowCount:       normalized.WindowCount,
		Status:            orderdomain.StatusNew,
		PaymentStatus:     orderdomain.PaymentStatusUnpaid,
		PriceTotal:        priceTotal,
	})
	if err != nil {
		return 0, fmt.Errorf("create order: %w", err)
	}

	if err := s.repo.CreateStatusHistory(ctx, orderID, normalized.ClientID, orderdomain.StatusNew, "Order created by client"); err != nil {
		return 0, fmt.Errorf("create order status history: %w", err)
	}
	if err := s.completeIdempotent(ctx, "order.create", normalized.IdempotencyKey, orderID); err != nil {
		return 0, err
	}
	_ = s.writeAudit(ctx, "order", orderID, normalized.ClientID, "order_created", fmt.Sprintf("service=%s,price=%d", service.Code, priceTotal))

	return orderID, nil
}

func (s *Service) UpdateClientOrder(ctx context.Context, input UpdateOrderInput) error {
	normalized, service, err := s.normalizeInput(ctx, CreateOrderInput{
		ClientID:          input.ClientID,
		City:              input.City,
		Street:            input.Street,
		House:             input.House,
		Floor:             input.Floor,
		Flat:              input.Flat,
		Entrance:          input.Entrance,
		AddressComment:    input.AddressComment,
		ScheduledDate:     input.ScheduledDate,
		ScheduledTimeFrom: input.ScheduledTimeFrom,
		ScheduledTimeTo:   input.ScheduledTimeTo,
		ServiceType:       input.ServiceType,
		Details:           input.Details,
		Square:            input.Square,
		WindowCount:       input.WindowCount,
	})
	if err != nil {
		return err
	}

	existing, err := s.findClientOrder(ctx, input.ClientID, input.OrderID)
	if err != nil {
		return err
	}
	if existing.Status != orderdomain.StatusNew {
		return ErrOrderNotEditable
	}

	if err := s.repo.UpdateAddress(ctx, orderdomain.Address{
		ID:       existing.Address.ID,
		City:     normalized.City,
		Street:   normalized.Street,
		House:    normalized.House,
		Floor:    normalized.Floor,
		Flat:     normalized.Flat,
		Entrance: normalized.Entrance,
		Comment:  normalized.AddressComment,
	}); err != nil {
		return fmt.Errorf("update address: %w", err)
	}

	scheduledDate, _ := time.Parse("2006-01-02", normalized.ScheduledDate)
	priceTotal := service.BasePrice + service.PricePerSquareMeter*normalized.Square + service.PricePerWindow*normalized.WindowCount
	if err := s.repo.UpdateOrderForClient(ctx, orderdomain.Order{
		ID:                input.OrderID,
		ClientID:          input.ClientID,
		Address:           orderdomain.Address{ID: existing.Address.ID},
		ScheduledDate:     scheduledDate,
		ScheduledTimeFrom: normalized.ScheduledTimeFrom,
		ScheduledTimeTo:   normalized.ScheduledTimeTo,
		ServiceType:       service.Code,
		Details:           normalized.Details,
		Square:            normalized.Square,
		WindowCount:       normalized.WindowCount,
		Status:            orderdomain.StatusNew,
		PaymentStatus:     existing.PaymentStatus,
		PriceTotal:        priceTotal,
	}); err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	_ = s.writeAudit(ctx, "order", input.OrderID, input.ClientID, "order_updated", fmt.Sprintf("service=%s,price=%d", service.Code, priceTotal))

	return nil
}

func (s *Service) CancelClientOrder(ctx context.Context, clientID, orderID int64) error {
	existing, err := s.findClientOrder(ctx, clientID, orderID)
	if err != nil {
		return err
	}
	if existing.Status != orderdomain.StatusNew {
		return ErrOrderNotEditable
	}

	if err := s.repo.CancelOrderForClient(ctx, orderID, clientID, "Cancelled by client"); err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}
	if err := s.repo.CreateStatusHistory(ctx, orderID, clientID, orderdomain.StatusCancelled, "Order cancelled by client"); err != nil {
		return fmt.Errorf("create cancellation history: %w", err)
	}
	_ = s.writeAudit(ctx, "order", orderID, clientID, "order_cancelled", "initiator=client")
	return nil
}

func (s *Service) DeleteClientHistoricalOrder(ctx context.Context, clientID, orderID int64) error {
	order, err := s.findClientOrder(ctx, clientID, orderID)
	if err != nil {
		return err
	}
	if !canDeleteHistorical(order.Status) {
		return ErrOrderNotManageable
	}
	if err := s.repo.DeleteHistoricalForClient(ctx, orderID, clientID); err != nil {
		return fmt.Errorf("удаление заказа клиента: %w", err)
	}
	_ = s.writeAudit(ctx, "order", orderID, clientID, "order_deleted", "initiator=client")
	return nil
}

func (s *Service) DeleteManagerHistoricalOrder(ctx context.Context, managerID, orderID int64) error {
	order, err := s.findManagerOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if !canDeleteHistorical(order.Status) {
		return ErrOrderNotManageable
	}
	if err := s.repo.DeleteHistoricalForManager(ctx, orderID); err != nil {
		return fmt.Errorf("удаление заказа менеджером: %w", err)
	}
	_ = s.writeAudit(ctx, "order", orderID, managerID, "order_deleted", "initiator=manager")
	return nil
}

func (s *Service) AssignManager(ctx context.Context, managerID, orderID int64) error {
	order, err := s.findManagerOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != orderdomain.StatusNew {
		return ErrOrderNotManageable
	}
	if err := s.repo.AssignManager(ctx, orderID, managerID); err != nil {
		return fmt.Errorf("assign manager: %w", err)
	}
	if err := s.repo.CreateStatusHistory(ctx, orderID, managerID, orderdomain.StatusAssignedManager, "Manager assigned to order"); err != nil {
		return fmt.Errorf("create manager assignment history: %w", err)
	}
	_ = s.writeAudit(ctx, "order", orderID, managerID, "manager_assigned", fmt.Sprintf("manager_id=%d", managerID))
	return nil
}

func (s *Service) AssignStaff(ctx context.Context, managerID, orderID, staffID int64) error {
	order, err := s.findManagerOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != orderdomain.StatusAssignedManager || order.ManagerID != managerID {
		return ErrOrderNotManageable
	}
	scheduledStart, scheduledEnd, err := orderScheduleBounds(order)
	if err != nil {
		return fmt.Errorf("calculate order schedule bounds: %w", err)
	}
	period, hasConflict, err := s.repo.FindStaffUnavailability(ctx, staffID, scheduledStart, scheduledEnd)
	if err != nil {
		return fmt.Errorf("check staff availability: %w", err)
	}
	if hasConflict {
		return &StaffUnavailableError{Reason: period.Reason, StartsAt: period.StartsAt, EndsAt: period.EndsAt}
	}
	if err := s.repo.AssignStaff(ctx, orderID, managerID, staffID); err != nil {
		return fmt.Errorf("assign staff: %w", err)
	}
	if err := s.repo.CreateStatusHistory(ctx, orderID, managerID, orderdomain.StatusAssignedStaff, "Staff assigned to order"); err != nil {
		return fmt.Errorf("create staff assignment history: %w", err)
	}
	_ = s.writeAudit(ctx, "order", orderID, managerID, "staff_assigned", fmt.Sprintf("staff_id=%d", staffID))
	return nil
}

func orderScheduleBounds(order orderdomain.Order) (time.Time, time.Time, error) {
	date := order.ScheduledDate.Format("2006-01-02")
	startTime := strings.TrimSpace(order.ScheduledTimeFrom)
	endTime := strings.TrimSpace(order.ScheduledTimeTo)
	if startTime == "" {
		startTime = "00:00"
	}
	if endTime == "" {
		endTime = "23:59"
	}
	startsAt, err := time.Parse("2006-01-02 15:04", date+" "+startTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endsAt, err := time.Parse("2006-01-02 15:04", date+" "+endTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !endsAt.After(startsAt) {
		return time.Time{}, time.Time{}, ErrInvalidOrderInput
	}
	return startsAt, endsAt, nil
}

func (s *Service) ConfirmPayment(ctx context.Context, managerID, orderID int64) error {
	order, err := s.findManagerOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.ManagerID != managerID && order.Status != orderdomain.StatusNew {
		return ErrOrderNotManageable
	}
	if err := s.repo.ConfirmPayment(ctx, orderID, managerID); err != nil {
		return fmt.Errorf("confirm payment: %w", err)
	}
	_ = s.writeAudit(ctx, "order", orderID, managerID, "payment_confirmed", "payment_status=paid")
	return nil
}

func (s *Service) CloseOrder(ctx context.Context, managerID, orderID int64) error {
	order, err := s.findManagerOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.ManagerID != managerID || order.Status != orderdomain.StatusCompleted {
		return ErrOrderNotManageable
	}
	if err := s.repo.CloseOrder(ctx, orderID, managerID); err != nil {
		return fmt.Errorf("close order: %w", err)
	}
	if err := s.repo.CreateStatusHistory(ctx, orderID, managerID, orderdomain.StatusClosed, "Order closed by manager"); err != nil {
		return fmt.Errorf("create close history: %w", err)
	}
	_ = s.writeAudit(ctx, "order", orderID, managerID, "order_closed", "status=closed")
	return nil
}

func (s *Service) normalizeInput(ctx context.Context, input CreateOrderInput) (CreateOrderInput, orderdomain.ServiceCatalogItem, error) {
	if input.ClientID <= 0 || strings.TrimSpace(input.City) == "" || strings.TrimSpace(input.Street) == "" || strings.TrimSpace(input.House) == "" || strings.TrimSpace(input.ScheduledDate) == "" || strings.TrimSpace(input.ServiceType) == "" || input.Square < 0 || input.WindowCount < 0 {
		return CreateOrderInput{}, orderdomain.ServiceCatalogItem{}, ErrInvalidOrderInput
	}

	if _, err := time.Parse("2006-01-02", strings.TrimSpace(input.ScheduledDate)); err != nil {
		return CreateOrderInput{}, orderdomain.ServiceCatalogItem{}, fmt.Errorf("parse scheduled date: %w", err)
	}
	if err := validateOptionalTimeRange(strings.TrimSpace(input.ScheduledTimeFrom), strings.TrimSpace(input.ScheduledTimeTo)); err != nil {
		return CreateOrderInput{}, orderdomain.ServiceCatalogItem{}, err
	}

	service, err := s.repo.GetServiceByCode(ctx, strings.TrimSpace(input.ServiceType))
	if err != nil {
		return CreateOrderInput{}, orderdomain.ServiceCatalogItem{}, fmt.Errorf("get service by code: %w", err)
	}

	input.City = strings.TrimSpace(input.City)
	input.Street = strings.TrimSpace(input.Street)
	input.House = strings.TrimSpace(input.House)
	input.Floor = strings.TrimSpace(input.Floor)
	input.Flat = strings.TrimSpace(input.Flat)
	input.Entrance = strings.TrimSpace(input.Entrance)
	input.AddressComment = strings.TrimSpace(input.AddressComment)
	input.ScheduledDate = strings.TrimSpace(input.ScheduledDate)
	input.ScheduledTimeFrom = strings.TrimSpace(input.ScheduledTimeFrom)
	input.ScheduledTimeTo = strings.TrimSpace(input.ScheduledTimeTo)
	input.ServiceType = strings.TrimSpace(input.ServiceType)
	input.Details = strings.TrimSpace(input.Details)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	return input, service, nil
}

func validateOptionalTimeRange(timeFrom, timeTo string) error {
	if timeFrom == "" && timeTo == "" {
		return nil
	}
	if timeFrom != "" {
		if _, err := time.Parse("15:04", timeFrom); err != nil {
			return fmt.Errorf("parse scheduled time from: %w", err)
		}
	}
	if timeTo != "" {
		if _, err := time.Parse("15:04", timeTo); err != nil {
			return fmt.Errorf("parse scheduled time to: %w", err)
		}
	}
	if timeFrom != "" && timeTo != "" {
		from, _ := time.Parse("15:04", timeFrom)
		to, _ := time.Parse("15:04", timeTo)
		if !to.After(from) {
			return ErrInvalidOrderInput
		}
	}
	return nil
}

func (s *Service) findClientOrder(ctx context.Context, clientID, orderID int64) (orderdomain.Order, error) {
	orders, err := s.repo.ListByClient(ctx, clientID)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("list client orders: %w", err)
	}
	for _, order := range orders {
		if order.ID == orderID {
			return order, nil
		}
	}
	return orderdomain.Order{}, ErrOrderNotFound
}

func (s *Service) findManagerOrder(ctx context.Context, orderID int64) (orderdomain.Order, error) {
	orders, err := s.repo.ListForManager(ctx)
	if err != nil {
		return orderdomain.Order{}, fmt.Errorf("list manager orders: %w", err)
	}
	for _, order := range orders {
		if order.ID == orderID {
			return order, nil
		}
	}
	return orderdomain.Order{}, ErrOrderNotFound
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

func hashCreateOrderInput(input CreateOrderInput) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strconv.FormatInt(input.ClientID, 10),
		input.City,
		input.Street,
		input.House,
		input.Floor,
		input.Flat,
		input.Entrance,
		input.AddressComment,
		input.ScheduledDate,
		input.ScheduledTimeFrom,
		input.ScheduledTimeTo,
		input.ServiceType,
		input.Details,
		strconv.Itoa(input.Square),
		strconv.Itoa(input.WindowCount),
	}, "|")))
	return hex.EncodeToString(sum[:])
}

func canDeleteHistorical(status orderdomain.Status) bool {
	return status == orderdomain.StatusCompleted || status == orderdomain.StatusClosed || status == orderdomain.StatusCancelled
}

func IsStaffRole(role userdomain.Role) bool {
	return role == userdomain.RoleStaff
}
