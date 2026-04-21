package audit

import (
	"fmt"

	auditdomain "project_cleaning/internal/domain/audit"
)

type EntryView struct {
	CreatedAt string
	ActorName string
	Action    string
	Entity    string
	Payload   string
}

func EntryViewFromEntry(entry auditdomain.Entry) EntryView {
	actor := entry.ActorName
	if actor == "" {
		actor = "Система"
	}
	payload := entry.PayloadJSON
	if payload == "" {
		payload = "-"
	}
	return EntryView{
		CreatedAt: entry.CreatedAt.Format("2006-01-02 15:04:05"),
		ActorName: actor,
		Action:    localizeAction(entry.Action),
		Entity:    fmt.Sprintf("%s #%d", localizeEntity(entry.EntityType), entry.EntityID),
		Payload:   payload,
	}
}

func localizeEntity(entity string) string {
	switch entity {
	case "order":
		return "Заказ"
	case "user":
		return "Пользователь"
	case "availability":
		return "Недоступность"
	default:
		return entity
	}
}

func localizeAction(action string) string {
	switch action {
	case "order_created":
		return "Создание заказа"
	case "order_updated":
		return "Изменение заказа"
	case "order_cancelled":
		return "Отмена заказа"
	case "manager_assigned":
		return "Назначение менеджера"
	case "staff_assigned":
		return "Назначение сотрудника"
	case "payment_confirmed":
		return "Подтверждение оплаты"
	case "order_closed":
		return "Закрытие заказа"
	case "staff_accepted":
		return "Подтверждение заказа сотрудником"
	case "staff_declined":
		return "Отказ сотрудника от заказа"
	case "order_started":
		return "Начало работ"
	case "order_completed":
		return "Завершение работ"
	case "availability_added":
		return "Добавление недоступности"
	case "user_created":
		return "Создание пользователя"
	case "user_updated":
		return "Изменение пользователя"
	case "user_deleted":
		return "Удаление пользователя"
	default:
		return action
	}
}
