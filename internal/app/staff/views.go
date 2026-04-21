package staff

import (
	"fmt"

	availabilitydomain "project_cleaning/internal/domain/availability"
	orderdomain "project_cleaning/internal/domain/order"
)

type OrderView struct {
	ID            int64
	Status        string
	StatusCode    string
	ServiceType   string
	ClientName    string
	ClientPhone   string
	AddressLine   string
	ScheduledDate string
	ScheduledTime string
	PriceTotal    string
	CanAccept     bool
	CanDecline    bool
	CanStart      bool
	CanComplete   bool
}

type AvailabilityView struct {
	ID       int64
	StartsAt string
	EndsAt   string
	Reason   string
}

func OrderViewFromOrder(order orderdomain.Order) OrderView {
	scheduledTime := "Весь день"
	switch {
	case order.ScheduledTimeFrom != "" && order.ScheduledTimeTo != "":
		scheduledTime = fmt.Sprintf("%s - %s", order.ScheduledTimeFrom, order.ScheduledTimeTo)
	case order.ScheduledTimeFrom != "":
		scheduledTime = fmt.Sprintf("с %s", order.ScheduledTimeFrom)
	case order.ScheduledTimeTo != "":
		scheduledTime = fmt.Sprintf("до %s", order.ScheduledTimeTo)
	}

	return OrderView{
		ID:            order.ID,
		Status:        localizeOrderStatus(order.Status),
		StatusCode:    string(order.Status),
		ServiceType:   order.ServiceType,
		ClientName:    order.Client.FullName,
		ClientPhone:   order.Client.Phone,
		AddressLine:   fmt.Sprintf("%s, %s, %s", order.Address.City, order.Address.Street, order.Address.House),
		ScheduledDate: order.ScheduledDate.Format("2006-01-02"),
		ScheduledTime: scheduledTime,
		PriceTotal:    fmt.Sprintf("%d", order.PriceTotal),
		CanAccept:     order.Status == orderdomain.StatusAssignedStaff,
		CanDecline:    order.Status == orderdomain.StatusAssignedStaff,
		CanStart:      order.Status == orderdomain.StatusStaffConfirmed,
		CanComplete:   order.Status == orderdomain.StatusInProgress,
	}
}

func AvailabilityViewFromPeriod(period availabilitydomain.Period) AvailabilityView {
	startsAt := period.StartsAt.Format("2006-01-02")
	endsAt := period.EndsAt.Format("2006-01-02")
	if period.StartsAt.Format("2006-01-02") == period.EndsAt.Format("2006-01-02") {
		endsAt = startsAt
	}

	return AvailabilityView{
		ID:       period.ID,
		StartsAt: startsAt,
		EndsAt:   endsAt,
		Reason:   period.Reason,
	}
}

func localizeOrderStatus(status orderdomain.Status) string {
	switch status {
	case orderdomain.StatusAssignedStaff:
		return "Назначен сотрудник"
	case orderdomain.StatusStaffConfirmed:
		return "Подтвержден сотрудником"
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

