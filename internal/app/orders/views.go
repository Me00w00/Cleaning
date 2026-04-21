package orders

import (
	"fmt"

	orderdomain "project_cleaning/internal/domain/order"
)

type ClientOrderView struct {
	ID                int64
	ServiceType       string
	ServiceCode       string
	Status            string
	PaymentStatus     string
	PriceTotal        string
	ScheduledDate     string
	ScheduledTimeFrom string
	ScheduledTimeTo   string
	ScheduledTime     string
	AddressLine       string
	Details           string
	City              string
	Street            string
	House             string
	Floor             string
	Flat              string
	Entrance          string
	AddressComment    string
	Square            int
	WindowCount       int
	CanModify         bool
	CanDelete         bool
}

type ManagerOrderView struct {
	ID            int64
	Status        string
	StatusCode    string
	PaymentStatus string
	ClientName    string
	ClientPhone   string
	ManagerName   string
	StaffName     string
	StaffPhone    string
	ServiceType   string
	AddressLine   string
	ScheduledDate string
	ScheduledTime string
	PriceTotal    string
	CanTake       bool
	CanAssign     bool
	CanConfirmPay bool
	CanClose      bool
	CanDelete     bool
}

type OrderStatusHistoryView struct {
	ChangedAt string
	ActorName string
	Status    string
	Comment   string
}

type ServiceView struct {
	Code        string
	Label       string
	PricingHint string
}

func formatScheduledTime(timeFrom, timeTo string) string {
	switch {
	case timeFrom != "" && timeTo != "":
		return fmt.Sprintf("%s - %s", timeFrom, timeTo)
	case timeFrom != "":
		return fmt.Sprintf("с %s", timeFrom)
	case timeTo != "":
		return fmt.Sprintf("до %s", timeTo)
	default:
		return "Весь день"
	}
}

func ClientOrderViewFromOrder(order orderdomain.Order) ClientOrderView {
	canDelete := order.Status == orderdomain.StatusCompleted || order.Status == orderdomain.StatusClosed || order.Status == orderdomain.StatusCancelled
	return ClientOrderView{
		ID:                order.ID,
		ServiceType:       order.ServiceType,
		ServiceCode:       order.ServiceType,
		Status:            localizeOrderStatus(order.Status),
		PaymentStatus:     localizePaymentStatus(order.PaymentStatus),
		PriceTotal:        fmt.Sprintf("%d", order.PriceTotal),
		ScheduledDate:     order.ScheduledDate.Format("2006-01-02"),
		ScheduledTimeFrom: order.ScheduledTimeFrom,
		ScheduledTimeTo:   order.ScheduledTimeTo,
		ScheduledTime:     formatScheduledTime(order.ScheduledTimeFrom, order.ScheduledTimeTo),
		AddressLine:       fmt.Sprintf("%s, %s, %s", order.Address.City, order.Address.Street, order.Address.House),
		Details:           order.Details,
		City:              order.Address.City,
		Street:            order.Address.Street,
		House:             order.Address.House,
		Floor:             order.Address.Floor,
		Flat:              order.Address.Flat,
		Entrance:          order.Address.Entrance,
		AddressComment:    order.Address.Comment,
		Square:            order.Square,
		WindowCount:       order.WindowCount,
		CanModify:         order.Status == orderdomain.StatusNew,
		CanDelete:         canDelete,
	}
}

func ManagerOrderViewFromOrder(order orderdomain.Order, managerID int64) ManagerOrderView {
	canDelete := order.Status == orderdomain.StatusCompleted || order.Status == orderdomain.StatusClosed || order.Status == orderdomain.StatusCancelled
	return ManagerOrderView{
		ID:            order.ID,
		Status:        localizeOrderStatus(order.Status),
		StatusCode:    string(order.Status),
		PaymentStatus: localizePaymentStatus(order.PaymentStatus),
		ClientName:    order.Client.FullName,
		ClientPhone:   order.Client.Phone,
		ManagerName:   order.Manager.FullName,
		StaffName:     order.Staff.FullName,
		StaffPhone:    order.Staff.Phone,
		ServiceType:   order.ServiceType,
		AddressLine:   fmt.Sprintf("%s, %s, %s", order.Address.City, order.Address.Street, order.Address.House),
		ScheduledDate: order.ScheduledDate.Format("2006-01-02"),
		ScheduledTime: formatScheduledTime(order.ScheduledTimeFrom, order.ScheduledTimeTo),
		PriceTotal:    fmt.Sprintf("%d", order.PriceTotal),
		CanTake:       order.Status == orderdomain.StatusNew,
		CanAssign:     order.Status == orderdomain.StatusAssignedManager && order.ManagerID == managerID,
		CanConfirmPay: order.PaymentStatus == orderdomain.PaymentStatusUnpaid && (order.ManagerID == managerID || order.Status == orderdomain.StatusNew),
		CanClose:      order.Status == orderdomain.StatusCompleted && order.ManagerID == managerID,
		CanDelete:     canDelete,
	}
}

func OrderStatusHistoryViewFromEntry(entry orderdomain.StatusHistoryEntry) OrderStatusHistoryView {
	actor := entry.ChangedByName
	if actor == "" {
		actor = "Система"
	}
	comment := entry.Comment
	if comment == "" {
		comment = "-"
	}
	return OrderStatusHistoryView{
		ChangedAt: entry.ChangedAt.Format("2006-01-02 15:04:05"),
		ActorName: actor,
		Status:    localizeOrderStatus(entry.NewStatus),
		Comment:   comment,
	}
}

func ServiceViewFromCatalog(item orderdomain.ServiceCatalogItem) ServiceView {
	return ServiceView{
		Code:        item.Code,
		Label:       item.Name + " (" + item.Code + ")",
		PricingHint: fmt.Sprintf("база=%d, м2=%d, окно=%d", item.BasePrice, item.PricePerSquareMeter, item.PricePerWindow),
	}
}

func localizeOrderStatus(status orderdomain.Status) string {
	switch status {
	case orderdomain.StatusNew:
		return "Новый"
	case orderdomain.StatusAssignedManager:
		return "Назначен менеджер"
	case orderdomain.StatusAssignedStaff:
		return "Назначен сотрудник"
	case orderdomain.StatusStaffConfirmed:
		return "Сотрудник подтвердил"
	case orderdomain.StatusInProgress:
		return "В работе"
	case orderdomain.StatusCompleted:
		return "Выполнен"
	case orderdomain.StatusClosed:
		return "Закрыт"
	case orderdomain.StatusCancelled:
		return "Отменен"
	default:
		return string(status)
	}
}

func localizePaymentStatus(status orderdomain.PaymentStatus) string {
	switch status {
	case orderdomain.PaymentStatusUnpaid:
		return "Не оплачен"
	case orderdomain.PaymentStatusPaid:
		return "Оплачен"
	default:
		return string(status)
	}
}
