package screens

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	ordersapp "project_cleaning/internal/app/orders"
	userdomain "project_cleaning/internal/domain/user"
)

func NewManagerOrdersScreen(router *Router, user userdomain.User) fyne.CanvasObject {
	orders, err := router.LoadManagerOrders()
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}
	staffUsers, err := router.LoadStaffUsers()
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}

	staffOptions := make([]string, 0, len(staffUsers))
	staffByLabel := make(map[string]int64, len(staffUsers))
	for _, staff := range staffUsers {
		label := fmt.Sprintf("%s | %s", staff.FullName, staff.Phone)
		staffOptions = append(staffOptions, label)
		staffByLabel[label] = staff.ID
	}

	allOrders := orders
	filteredOrders := append([]ordersapp.ManagerOrderView(nil), allOrders...)

	selectedOrderID := int64(0)
	selectedCanTake := false
	selectedCanAssign := false
	selectedCanPay := false
	selectedCanClose := false
	selectedCanDelete := false

	info := widget.NewLabel("Выберите заказ из списка.")
	info.Wrapping = fyne.TextWrapWord
	historyInfo := widget.NewLabel("История статусов недоступна.")
	historyInfo.Wrapping = fyne.TextWrapWord
	staffSelect := widget.NewSelect(staffOptions, nil)
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск по клиенту, адресу или услуге")
	statusSelect := widget.NewSelect([]string{
		"Все статусы",
		"Новый",
		"Назначен менеджер",
		"Назначен сотрудник",
		"Сотрудник подтвердил",
		"В работе",
		"Выполнен",
		"Закрыт",
		"Отменен",
	}, nil)
	statusSelect.SetSelected("Все статусы")
	summaryLabel := widget.NewLabel("")

	takeButton := widget.NewButton("Взять в работу", func() {
		if selectedOrderID == 0 || !selectedCanTake {
			dialog.ShowInformation("Заказы", "Для закрепления выберите новый заказ.", router.window)
			return
		}
		router.AssignManagerToOrder(selectedOrderID)
	})

	assignStaffButton := widget.NewButton("Назначить сотрудника", func() {
		if selectedOrderID == 0 || !selectedCanAssign {
			dialog.ShowInformation("Заказы", "Выберите заказ, закрепленный за вами.", router.window)
			return
		}
		staffID, ok := staffByLabel[staffSelect.Selected]
		if !ok {
			dialog.ShowInformation("Заказы", "Выберите сотрудника.", router.window)
			return
		}
		router.AssignStaffToOrder(selectedOrderID, staffID)
	})

	payButton := widget.NewButton("Подтвердить оплату", func() {
		if selectedOrderID == 0 || !selectedCanPay {
			dialog.ShowInformation("Заказы", "Выберите заказ с неподтвержденной оплатой.", router.window)
			return
		}
		router.ConfirmOrderPayment(selectedOrderID)
	})

	closeButton := widget.NewButton("Закрыть заказ", func() {
		if selectedOrderID == 0 || !selectedCanClose {
			dialog.ShowInformation("Заказы", "Выберите выполненный заказ, закрепленный за вами.", router.window)
			return
		}
		router.CloseManagerOrder(selectedOrderID)
	})

	deleteButton := widget.NewButton("Удалить", func() {
		if selectedOrderID == 0 || !selectedCanDelete {
			dialog.ShowInformation("Заказы", "Можно удалить только выполненный, закрытый или отмененный заказ.", router.window)
			return
		}
		dialog.ShowConfirm("Удаление заказа", "Удалить выбранный заказ полностью без возможности восстановления?", func(ok bool) {
			if ok {
				router.DeleteManagerHistoricalOrder(selectedOrderID)
			}
		}, router.window)
	})
	deleteButton.Disable()

	setActionState := func() {
		if selectedCanTake {
			takeButton.Enable()
		} else {
			takeButton.Disable()
		}
		if selectedCanAssign {
			assignStaffButton.Enable()
		} else {
			assignStaffButton.Disable()
		}
		if selectedCanPay {
			payButton.Enable()
		} else {
			payButton.Disable()
		}
		if selectedCanClose {
			closeButton.Enable()
		} else {
			closeButton.Disable()
		}
		if selectedCanDelete {
			deleteButton.Enable()
		} else {
			deleteButton.Disable()
		}
		deleteButton.Refresh()
	}

	list := widget.NewList(
		func() int { return len(filteredOrders) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			order := filteredOrders[id]
			obj.(*widget.Label).SetText(fmt.Sprintf("#%d | %s | %s | %s", order.ID, order.ScheduledDate, order.ClientName, order.Status))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		order := filteredOrders[id]
		selectedOrderID = order.ID
		selectedCanTake = order.CanTake
		selectedCanAssign = order.CanAssign
		selectedCanPay = order.CanConfirmPay
		selectedCanClose = order.CanClose
		selectedCanDelete = order.CanDelete
		info.SetText(fmt.Sprintf("Заказ #%d\nСтатус: %s\nОплата: %s\nКлиент: %s\nТелефон клиента: %s\nМенеджер: %s\nСотрудник: %s\nТелефон сотрудника: %s\nУслуга: %s\nАдрес: %s\nДата: %s\nВремя: %s\nСтоимость: %s",
			order.ID,
			order.Status,
			order.PaymentStatus,
			order.ClientName,
			order.ClientPhone,
			order.ManagerName,
			order.StaffName,
			order.StaffPhone,
			order.ServiceType,
			order.AddressLine,
			order.ScheduledDate,
			order.ScheduledTime,
			order.PriceTotal,
		))
		history, historyErr := router.LoadManagerOrderHistory(order.ID)
		if historyErr != nil {
			historyInfo.SetText(historyErr.Error())
			setActionState()
			return
		}
		if len(history) == 0 {
			historyInfo.SetText("История статусов пока пуста.")
			setActionState()
			return
		}
		lines := make([]string, 0, len(history))
		for _, item := range history {
			lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", item.ChangedAt, item.ActorName, item.Status, item.Comment))
		}
		historyInfo.SetText(strings.Join(lines, "\n"))
		setActionState()
	}

	applyFilters := func() {
		query := strings.TrimSpace(searchEntry.Text)
		status := statusSelect.Selected
		filteredOrders = filteredOrders[:0]
		for _, order := range allOrders {
			if status != "" && status != "Все статусы" && order.Status != status {
				continue
			}
			if !containsFold(order.ClientName+" "+order.AddressLine+" "+order.ServiceType, query) {
				continue
			}
			filteredOrders = append(filteredOrders, order)
		}
		summaryLabel.SetText(fmt.Sprintf("Найдено: %d из %d", len(filteredOrders), len(allOrders)))
		list.UnselectAll()
		selectedOrderID = 0
		selectedCanTake = false
		selectedCanAssign = false
		selectedCanPay = false
		selectedCanClose = false
		selectedCanDelete = false
		setActionState()
		info.SetText("Выберите заказ из списка.")
		historyInfo.SetText("История статусов недоступна.")
		if len(filteredOrders) > 0 {
			list.Select(0)
		}
		list.Refresh()
	}
	searchEntry.OnChanged = func(string) { applyFilters() }
	statusSelect.OnChanged = func(string) { applyFilters() }
	applyFilters()

	refreshButton := widget.NewButton("Обновить", func() { router.ShowHome(user) })

	header := newWorkspaceHeader("Заказы менеджера", "Менеджер: "+user.FullName, func() { router.ShowLogin() })

	right := widget.NewCard("Карточка заказа", "Просмотр деталей, истории и действий менеджера.", container.NewVBox(
		info,
		widget.NewSeparator(),
		widget.NewLabel("История статусов"),
		historyInfo,
		widget.NewSeparator(),
		staffSelect,
		container.NewGridWithColumns(2, takeButton, assignStaffButton),
		container.NewGridWithColumns(2, payButton, closeButton),
		deleteButton,
		refreshButton,
	))

	left := widget.NewCard("Заказы", fmt.Sprintf("Всего заказов: %s", strconv.Itoa(len(allOrders))), container.NewBorder(
		container.NewVBox(
			summaryLabel,
			container.NewGridWithColumns(2, searchEntry, statusSelect),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewPadded(list),
	))
	content := container.NewHSplit(left, right)
	content.Offset = 0.42

	return newWorkspaceScreen(header, container.NewPadded(content))
}





