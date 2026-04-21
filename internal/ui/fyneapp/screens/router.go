package screens

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	auditapp "project_cleaning/internal/app/audit"
	authapp "project_cleaning/internal/app/auth"
	ordersapp "project_cleaning/internal/app/orders"
	staffapp "project_cleaning/internal/app/staff"
	usersapp "project_cleaning/internal/app/users"
	userdomain "project_cleaning/internal/domain/user"
	"project_cleaning/internal/platform/config"
	"project_cleaning/internal/platform/logging"
)

type Router struct {
	window        fyne.Window
	cfg           config.Config
	logger        *logging.Logger
	authService   *authapp.Service
	usersService  *usersapp.Service
	ordersService *ordersapp.Service
	staffService  *staffapp.Service
	auditService  *auditapp.Service
	currentUser   *userdomain.User
}

func NewRouter(window fyne.Window, cfg config.Config, logger *logging.Logger, authService *authapp.Service, usersService *usersapp.Service, ordersService *ordersapp.Service, staffService *staffapp.Service, auditService *auditapp.Service) *Router {
	return &Router{
		window:        window,
		cfg:           cfg,
		logger:        logger,
		authService:   authService,
		usersService:  usersService,
		ordersService: ordersService,
		staffService:  staffService,
		auditService:  auditService,
	}
}

func (r *Router) normalizeWindow() {
	r.window.SetFullScreen(false)
	r.window.Resize(fyne.NewSize(1360, 860))
	r.window.CenterOnScreen()
}
func (r *Router) showError(err error) {
	if err == nil {
		return
	}
	dialog.ShowError(r.localizeError(err), r.window)
}

func (r *Router) localizeError(err error) error {
	if err == nil {
		return nil
	}

	var unavailable *ordersapp.StaffUnavailableError
	if errors.As(err, &unavailable) {
		return errors.New(r.staffUnavailableMessage(unavailable))
	}

	switch {
	case errors.Is(err, authapp.ErrInvalidCredentials):
		return errors.New("Неверный логин или пароль")
	case errors.Is(err, authapp.ErrInactiveUser):
		return errors.New("Учетная запись деактивирована")
	case errors.Is(err, usersapp.ErrInvalidInput):
		return errors.New("Некорректно заполнены обязательные поля")
	case errors.Is(err, usersapp.ErrLoginExists):
		return errors.New("Пользователь с таким логином уже существует")
	case errors.Is(err, ordersapp.ErrInvalidOrderInput):
		return errors.New("Некорректные данные заказа")
	case errors.Is(err, ordersapp.ErrOrderNotEditable):
		return errors.New("Заказ можно изменять или отменять только в статусе «Новый»")
	case errors.Is(err, ordersapp.ErrOrderNotFound):
		return errors.New("Заказ не найден")
	case errors.Is(err, ordersapp.ErrOrderNotManageable):
		return errors.New("Это действие недоступно для выбранного заказа")
	case errors.Is(err, ordersapp.ErrStaffUnavailable):
		return errors.New("Этот сотрудник недоступен в выбранное время")
	case errors.Is(err, staffapp.ErrInvalidAvailability):
		return errors.New("Некорректно заполнен период недоступности")
	case errors.Is(err, staffapp.ErrAvailabilityOverlap):
		return errors.New("Период недоступности пересекается с уже существующим")
	}

	text := err.Error()
	replacer := strings.NewReplacer(
		"load client orders:", "загрузка заказов клиента:",
		"load manager orders:", "загрузка заказов менеджера:",
		"load staff orders:", "загрузка заказов сотрудника:",
		"load staff availability:", "загрузка недоступности сотрудника:",
		"load services:", "загрузка услуг:",
		"load users:", "загрузка пользователей:",
		"load audit entries:", "загрузка аудита:",
		"load order history:", "загрузка истории заказа:",
		"parse scheduled date:", "ошибка разбора даты заказа:",
		"parse scheduled time from:", "ошибка разбора времени начала заказа:",
		"parse scheduled time to:", "ошибка разбора времени окончания заказа:",
		"parse availability start:", "ошибка разбора начала периода недоступности:",
		"parse availability end:", "ошибка разбора окончания периода недоступности:",
		"get service by code:", "не удалось получить услугу:",
		"create address:", "не удалось создать адрес:",
		"create order:", "не удалось создать заказ:",
		"update address:", "не удалось обновить адрес:",
		"update order:", "не удалось обновить заказ:",
		"cancel order:", "не удалось отменить заказ:",
		"assign manager:", "не удалось закрепить заказ за менеджером:",
		"assign staff:", "не удалось назначить сотрудника:",
		"confirm payment:", "не удалось подтвердить оплату:",
		"close order:", "не удалось закрыть заказ:",
		"check staff availability:", "не удалось проверить доступность сотрудника:",
		"удаление заказа клиента:", "не удалось удалить заказ клиента:",
		"удаление заказа менеджером:", "не удалось удалить заказ менеджером:",
	)
	localized := replacer.Replace(text)
	if localized == text {
		return err
	}
	return errors.New(localized)
}

func (r *Router) staffUnavailableMessage(err *ordersapp.StaffUnavailableError) string {
	reason := strings.TrimSpace(err.Reason)
	message := "Этот сотрудник недоступен в выбранные дату и время."
	if reason != "" {
		if strings.Contains(strings.ToLower(reason), "отпуск") {
			message = "Этот сотрудник в отпуске."
		} else {
			message = "Этот сотрудник недоступен."
		}
		message += "\nПричина: " + reason
	}
	if !err.StartsAt.IsZero() && !err.EndsAt.IsZero() {
		if err.StartsAt.Format("15:04") == "00:00" && err.EndsAt.Format("15:04") == "23:59" {
			message += fmt.Sprintf("\nПериод: %s - %s", err.StartsAt.Format("2006-01-02"), err.EndsAt.Format("2006-01-02"))
		} else {
			message += fmt.Sprintf("\nПериод: %s - %s", err.StartsAt.Format("2006-01-02 15:04"), err.EndsAt.Format("2006-01-02 15:04"))
		}
	}
	return message
}

func (r *Router) ShowLogin() {
	r.currentUser = nil
	r.normalizeWindow()
	r.window.SetContent(NewLoginScreen(r))
}

func (r *Router) ShowClientRegistration() {
	r.normalizeWindow()
	r.window.SetContent(NewClientRegistrationScreen(r))
}

func (r *Router) HandleLogin(login, password string) {
	user, err := r.authService.Login(context.Background(), login, password)
	if err != nil {
		r.showError(err)
		return
	}

	r.currentUser = &user
	r.logger.Info("login successful", "login", user.Login, "role", user.Role)
	r.ShowHome(user)
}

func (r *Router) RegisterClient(input usersapp.CreateInput) {
	_, err := r.usersService.RegisterClient(context.Background(), input)
	if err != nil {
		r.showError(err)
		return
	}

	r.logger.Info("client registered", "login", input.Login)
	dialog.ShowInformation("Регистрация", "Учетная запись клиента создана. Используйте логин и пароль для входа.", r.window)
	r.ShowLogin()
}

func (r *Router) CreateClientOrder(input ordersapp.CreateOrderInput) {
	if r.currentUser == nil {
		r.showError(errors.New("Нет активной пользовательской сессии"))
		return
	}
	input.ClientID = r.currentUser.ID
	_, err := r.ordersService.CreateOrder(context.Background(), input)
	if err != nil {
		r.showError(err)
		return
	}
	r.logger.Info("client order created", "client_id", r.currentUser.ID, "service_type", input.ServiceType)
	dialog.ShowInformation("Заказы", "Заказ успешно создан.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) UpdateClientOrder(input ordersapp.UpdateOrderInput) {
	if r.currentUser == nil {
		r.showError(errors.New("Нет активной пользовательской сессии"))
		return
	}
	input.ClientID = r.currentUser.ID
	if err := r.ordersService.UpdateClientOrder(context.Background(), input); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ обновлен.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) CancelClientOrder(orderID int64) {
	if r.currentUser == nil {
		r.showError(errors.New("Нет активной пользовательской сессии"))
		return
	}
	if err := r.ordersService.CancelClientOrder(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ отменен.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) DeleteClientHistoricalOrder(orderID int64) {
	if r.currentUser == nil {
		r.showError(errors.New("Нет активной пользовательской сессии"))
		return
	}
	if err := r.ordersService.DeleteClientHistoricalOrder(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ удален полностью.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) LoadClientOrders() ([]ordersapp.ClientOrderView, error) {
	if r.currentUser == nil {
		return nil, errors.New("Нет активной пользовательской сессии")
	}
	orders, err := r.ordersService.ListClientOrders(context.Background(), r.currentUser.ID)
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load client orders: %w", err))
	}
	views := make([]ordersapp.ClientOrderView, 0, len(orders))
	for _, order := range orders {
		views = append(views, ordersapp.ClientOrderViewFromOrder(order))
	}
	return views, nil
}

func (r *Router) LoadClientOrderHistory(orderID int64) ([]ordersapp.OrderStatusHistoryView, error) {
	if r.currentUser == nil {
		return nil, errors.New("Нет активной пользовательской сессии")
	}
	orders, err := r.ordersService.ListClientOrders(context.Background(), r.currentUser.ID)
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load client orders: %w", err))
	}
	for _, order := range orders {
		if order.ID == orderID {
			return r.loadOrderHistory(orderID)
		}
	}
	return nil, errors.New("Заказ недоступен")
}

func (r *Router) LoadManagerOrders() ([]ordersapp.ManagerOrderView, error) {
	if r.currentUser == nil {
		return nil, errors.New("Нет активной сессии")
	}
	orders, err := r.ordersService.ListManagerOrders(context.Background())
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load manager orders: %w", err))
	}
	views := make([]ordersapp.ManagerOrderView, 0, len(orders))
	for _, order := range orders {
		views = append(views, ordersapp.ManagerOrderViewFromOrder(order, r.currentUser.ID))
	}
	return views, nil
}

func (r *Router) LoadManagerOrderHistory(orderID int64) ([]ordersapp.OrderStatusHistoryView, error) {
	if r.currentUser == nil {
		return nil, errors.New("Нет активной сессии")
	}
	return r.loadOrderHistory(orderID)
}

func (r *Router) LoadStaffOrders() ([]staffapp.OrderView, error) {
	if r.currentUser == nil {
		return nil, errors.New("Нет активной сессии")
	}
	orders, err := r.staffService.ListOrders(context.Background(), r.currentUser.ID)
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load staff orders: %w", err))
	}
	views := make([]staffapp.OrderView, 0, len(orders))
	for _, order := range orders {
		views = append(views, staffapp.OrderViewFromOrder(order))
	}
	return views, nil
}

func (r *Router) LoadStaffOrderHistory(orderID int64) ([]ordersapp.OrderStatusHistoryView, error) {
	if r.currentUser == nil {
		return nil, errors.New("Нет активной сессии")
	}
	orders, err := r.staffService.ListOrders(context.Background(), r.currentUser.ID)
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load staff orders: %w", err))
	}
	for _, order := range orders {
		if order.ID == orderID {
			return r.loadOrderHistory(orderID)
		}
	}
	return nil, errors.New("Заказ недоступен")
}

func (r *Router) LoadStaffAvailability() ([]staffapp.AvailabilityView, error) {
	if r.currentUser == nil {
		return nil, errors.New("Нет активной сессии")
	}
	items, err := r.staffService.ListAvailability(context.Background(), r.currentUser.ID)
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load staff availability: %w", err))
	}
	views := make([]staffapp.AvailabilityView, 0, len(items))
	for _, item := range items {
		views = append(views, staffapp.AvailabilityViewFromPeriod(item))
	}
	return views, nil
}

func (r *Router) AcceptStaffOrder(orderID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.staffService.AcceptOrder(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ подтвержден сотрудником.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) DeclineStaffOrder(orderID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.staffService.DeclineOrder(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ возвращен менеджеру для переназначения.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) StartStaffOrder(orderID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.staffService.StartOrder(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Работы по заказу начаты.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) CompleteStaffOrder(orderID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.staffService.CompleteOrder(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ переведен в статус «Выполнен».", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) AddStaffAvailability(input staffapp.AvailabilityInput) {
	if r.currentUser == nil {
		return
	}
	input.StaffID = r.currentUser.ID
	if err := r.staffService.AddAvailability(context.Background(), input); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Календарь", "Период недоступности сохранен.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) LoadStaffUsers() ([]userdomain.User, error) {
	users, err := r.usersService.List(context.Background())
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load users: %w", err))
	}
	staff := make([]userdomain.User, 0)
	for _, u := range users {
		if ordersapp.IsStaffRole(u.Role) && u.IsActive {
			staff = append(staff, u)
		}
	}
	return staff, nil
}

func (r *Router) AssignManagerToOrder(orderID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.ordersService.AssignManager(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ закреплен за менеджером.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) AssignStaffToOrder(orderID, staffID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.ordersService.AssignStaff(context.Background(), r.currentUser.ID, orderID, staffID); err != nil {
		var unavailable *ordersapp.StaffUnavailableError
		if errors.As(err, &unavailable) {
			dialog.ShowInformation("Назначение сотрудника", r.staffUnavailableMessage(unavailable), r.window)
			return
		}
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Сотрудник назначен на заказ.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) DeleteManagerHistoricalOrder(orderID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.ordersService.DeleteManagerHistoricalOrder(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ удален полностью.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) ConfirmOrderPayment(orderID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.ordersService.ConfirmPayment(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Оплата подтверждена.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) CloseManagerOrder(orderID int64) {
	if r.currentUser == nil {
		return
	}
	if err := r.ordersService.CloseOrder(context.Background(), r.currentUser.ID, orderID); err != nil {
		r.showError(err)
		return
	}
	dialog.ShowInformation("Заказы", "Заказ закрыт.", r.window)
	r.ShowHome(*r.currentUser)
}

func (r *Router) LoadServices() ([]ordersapp.ServiceView, error) {
	services, err := r.ordersService.ListServices(context.Background())
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load services: %w", err))
	}
	views := make([]ordersapp.ServiceView, 0, len(services))
	for _, item := range services {
		views = append(views, ordersapp.ServiceViewFromCatalog(item))
	}
	return views, nil
}

func (r *Router) CreateUserByAdmin(input usersapp.CreateInput, onDone func()) {
	_, err := r.usersService.CreateByAdmin(context.Background(), input)
	if err != nil {
		r.showError(err)
		return
	}
	r.logger.Info("user created by admin", "login", input.Login, "role", input.Role)
	if onDone != nil {
		onDone()
	}
}

func (r *Router) UpdateUserByAdmin(input usersapp.UpdateInput, onDone func()) {
	if err := r.usersService.UpdateByAdmin(context.Background(), input); err != nil {
		r.showError(err)
		return
	}
	r.logger.Info("user updated by admin", "user_id", input.ID, "role", input.Role, "active", input.IsActive)
	if onDone != nil {
		onDone()
	}
}

func (r *Router) DeleteUserByAdmin(id int64, onDone func()) {
	if err := r.usersService.DeleteByAdmin(context.Background(), id); err != nil {
		r.showError(err)
		return
	}
	r.logger.Info("user deleted by admin", "user_id", id)
	if onDone != nil {
		onDone()
	}
}

func (r *Router) LoadUsers() ([]userdomain.User, error) {
	users, err := r.usersService.List(context.Background())
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load users: %w", err))
	}
	return users, nil
}

func (r *Router) LoadAuditEntries() ([]auditapp.EntryView, error) {
	entries, err := r.auditService.ListRecent(context.Background(), 200)
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load audit entries: %w", err))
	}
	views := make([]auditapp.EntryView, 0, len(entries))
	for _, entry := range entries {
		views = append(views, auditapp.EntryViewFromEntry(entry))
	}
	return views, nil
}

func (r *Router) loadOrderHistory(orderID int64) ([]ordersapp.OrderStatusHistoryView, error) {
	entries, err := r.ordersService.ListOrderStatusHistory(context.Background(), orderID)
	if err != nil {
		return nil, r.localizeError(fmt.Errorf("load order history: %w", err))
	}
	views := make([]ordersapp.OrderStatusHistoryView, 0, len(entries))
	for _, entry := range entries {
		views = append(views, ordersapp.OrderStatusHistoryViewFromEntry(entry))
	}
	return views, nil
}

func (r *Router) ShowHome(user userdomain.User) {
	r.normalizeWindow()
	r.normalizeWindow()
	r.normalizeWindow()
	switch user.Role {
	case userdomain.RoleAdmin:
		r.window.SetContent(NewAdminUsersScreen(r, user))
	case userdomain.RoleManager:
		r.window.SetContent(NewManagerOrdersScreen(r, user))
	case userdomain.RoleStaff:
		r.window.SetContent(NewStaffOrdersScreen(r, user))
	case userdomain.RoleClient:
		r.window.SetContent(NewClientOrdersScreen(r, user))
	default:
		r.showError(fmt.Errorf("неподдерживаемая роль: %s", user.Role))
	}
}









