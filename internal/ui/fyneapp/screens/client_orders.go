package screens

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	ordersapp "project_cleaning/internal/app/orders"
	userdomain "project_cleaning/internal/domain/user"
)

func NewClientOrdersScreen(router *Router, user userdomain.User) fyne.CanvasObject {
	orders, err := router.LoadClientOrders()
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}
	services, err := router.LoadServices()
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}

	serviceOptions := make([]string, 0, len(services))
	serviceByLabel := make(map[string]ordersapp.ServiceView, len(services))
	serviceLabelByCode := make(map[string]string, len(services))
	for _, service := range services {
		serviceOptions = append(serviceOptions, service.Label)
		serviceByLabel[service.Label] = service
		serviceLabelByCode[service.Code] = service.Label
	}

	allOrders := orders
	filteredOrders := append([]ordersapp.ClientOrderView(nil), allOrders...)

	selectedOrderID := int64(0)
	selectedOrderCanModify := false
	selectedOrderCanDelete := false
	createOrderKey := newIdempotencyKey()
	createSubmitting := false

	orderDetails := widget.NewLabel("Выберите заказ из списка.")
	orderDetails.Wrapping = fyne.TextWrapWord
	historyDetails := widget.NewLabel("История статусов недоступна.")
	historyDetails.Wrapping = fyne.TextWrapWord
	summaryLabel := widget.NewLabel("")

	cityEntry := widget.NewEntry()
	cityEntry.SetPlaceHolder("Город")
	streetEntry := widget.NewEntry()
	streetEntry.SetPlaceHolder("Улица")
	houseEntry := widget.NewEntry()
	houseEntry.SetPlaceHolder("Дом")
	floorEntry := widget.NewEntry()
	floorEntry.SetPlaceHolder("Этаж")
	flatEntry := widget.NewEntry()
	flatEntry.SetPlaceHolder("Квартира")
	entranceEntry := widget.NewEntry()
	entranceEntry.SetPlaceHolder("Подъезд")
	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("ГГГГ-ММ-ДД")
	timeFromEntry := widget.NewEntry()
	timeFromEntry.SetPlaceHolder("ЧЧ:ММ")
	timeToEntry := widget.NewEntry()
	timeToEntry.SetPlaceHolder("ЧЧ:ММ")
	squareEntry := widget.NewEntry()
	squareEntry.SetPlaceHolder("Площадь, м2")
	windowEntry := widget.NewEntry()
	windowEntry.SetPlaceHolder("Окна")
	addressCommentEntry := widget.NewEntry()
	addressCommentEntry.SetPlaceHolder("Комментарий к адресу")
	detailsEntry := widget.NewMultiLineEntry()
	detailsEntry.SetPlaceHolder("Детали заказа")
	pricingHint := widget.NewLabel("")
	pricingHint.Wrapping = fyne.TextWrapWord

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Поиск по адресу, услуге или деталям")
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

	serviceSelect := widget.NewSelect(serviceOptions, func(selected string) {
		if selected == "" {
			pricingHint.SetText("")
			return
		}
		pricingHint.SetText(serviceByLabel[selected].PricingHint)
	})
	if len(serviceOptions) > 0 {
		serviceSelect.SetSelected(serviceOptions[0])
	}

	clearForm := func() {
		selectedOrderID = 0
		selectedOrderCanModify = false
		selectedOrderCanDelete = false
		createOrderKey = newIdempotencyKey()
		cityEntry.SetText("")
		streetEntry.SetText("")
		houseEntry.SetText("")
		floorEntry.SetText("")
		flatEntry.SetText("")
		entranceEntry.SetText("")
		dateEntry.SetText("")
		timeFromEntry.SetText("")
		timeToEntry.SetText("")
		squareEntry.SetText("")
		windowEntry.SetText("")
		addressCommentEntry.SetText("")
		detailsEntry.SetText("")
		if len(serviceOptions) > 0 {
			serviceSelect.SetSelected(serviceOptions[0])
		}
	}

	fillForm := func(order ordersapp.ClientOrderView) {
		selectedOrderID = order.ID
		selectedOrderCanModify = order.CanModify
		selectedOrderCanDelete = order.CanDelete
		cityEntry.SetText(order.City)
		streetEntry.SetText(order.Street)
		houseEntry.SetText(order.House)
		floorEntry.SetText(order.Floor)
		flatEntry.SetText(order.Flat)
		entranceEntry.SetText(order.Entrance)
		dateEntry.SetText(order.ScheduledDate)
		timeFromEntry.SetText(order.ScheduledTimeFrom)
		timeToEntry.SetText(order.ScheduledTimeTo)
		squareEntry.SetText(strconv.Itoa(order.Square))
		windowEntry.SetText(strconv.Itoa(order.WindowCount))
		addressCommentEntry.SetText(order.AddressComment)
		detailsEntry.SetText(order.Details)
		if label, ok := serviceLabelByCode[order.ServiceCode]; ok {
			serviceSelect.SetSelected(label)
		}
	}

	parseForm := func() (ordersapp.UpdateOrderInput, error) {
		selectedService, ok := serviceByLabel[serviceSelect.Selected]
		if !ok {
			return ordersapp.UpdateOrderInput{}, fmt.Errorf("выберите тип услуги")
		}
		square, err := strconv.Atoi(strings.TrimSpace(squareEntry.Text))
		if err != nil {
			return ordersapp.UpdateOrderInput{}, fmt.Errorf("площадь должна быть целым числом")
		}
		windows, err := strconv.Atoi(strings.TrimSpace(windowEntry.Text))
		if err != nil {
			return ordersapp.UpdateOrderInput{}, fmt.Errorf("количество окон должно быть целым числом")
		}
		return ordersapp.UpdateOrderInput{
			OrderID:           selectedOrderID,
			City:              strings.TrimSpace(cityEntry.Text),
			Street:            strings.TrimSpace(streetEntry.Text),
			House:             strings.TrimSpace(houseEntry.Text),
			Floor:             strings.TrimSpace(floorEntry.Text),
			Flat:              strings.TrimSpace(flatEntry.Text),
			Entrance:          strings.TrimSpace(entranceEntry.Text),
			AddressComment:    strings.TrimSpace(addressCommentEntry.Text),
			ScheduledDate:     strings.TrimSpace(dateEntry.Text),
			ScheduledTimeFrom: strings.TrimSpace(timeFromEntry.Text),
			ScheduledTimeTo:   strings.TrimSpace(timeToEntry.Text),
			ServiceType:       selectedService.Code,
			Details:           strings.TrimSpace(detailsEntry.Text),
			Square:            square,
			WindowCount:       windows,
		}, nil
	}

	loadHistory := func(orderID int64) {
		history, historyErr := router.LoadClientOrderHistory(orderID)
		if historyErr != nil {
			historyDetails.SetText(historyErr.Error())
			return
		}
		if len(history) == 0 {
			historyDetails.SetText("История статусов пока пуста.")
			return
		}
		lines := make([]string, 0, len(history))
		for _, item := range history {
			lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", item.ChangedAt, item.ActorName, item.Status, item.Comment))
		}
		historyDetails.SetText(strings.Join(lines, "\n"))
	}

	var createButton *widget.Button
	createButton = widget.NewButton("Создать заказ", func() {
		if createSubmitting {
			return
		}
		createSubmitting = true
		createButton.Disable()
		defer func() {
			createSubmitting = false
			createButton.Enable()
		}()
		input, err := parseForm()
		if err != nil {
			dialog.ShowError(err, router.window)
			return
		}
		router.CreateClientOrder(ordersapp.CreateOrderInput{
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
			IdempotencyKey:    createOrderKey,
		})
	})

	updateButton := widget.NewButton("Сохранить изменения", func() {
		if selectedOrderID == 0 || !selectedOrderCanModify {
			dialog.ShowInformation("Заказы", "Выберите заказ в статусе «Новый» для изменения.", router.window)
			return
		}
		input, err := parseForm()
		if err != nil {
			dialog.ShowError(err, router.window)
			return
		}
		router.UpdateClientOrder(input)
	})

	cancelButton := widget.NewButton("Отменить заказ", func() {
		if selectedOrderID == 0 || !selectedOrderCanModify {
			dialog.ShowInformation("Заказы", "Выберите заказ в статусе «Новый» для отмены.", router.window)
			return
		}
		dialog.ShowConfirm("Отмена заказа", "Отменить выбранный заказ?", func(ok bool) {
			if ok {
				router.CancelClientOrder(selectedOrderID)
			}
		}, router.window)
	})

	deleteHistoryButton := widget.NewButton("Удалить", func() {
		if selectedOrderID == 0 || !selectedOrderCanDelete {
			dialog.ShowInformation("Заказы", "Можно удалить только выполненный, закрытый или отмененный заказ.", router.window)
			return
		}
		dialog.ShowConfirm("Удаление заказа", "Удалить выбранный заказ полностью без возможности восстановления?", func(ok bool) {
			if ok {
				router.DeleteClientHistoricalOrder(selectedOrderID)
			}
		}, router.window)
	})
	deleteHistoryButton.Disable()

	resetButton := widget.NewButton("Очистить форму", clearForm)

	setButtonsState := func() {
		if selectedOrderCanModify {
			updateButton.Enable()
			cancelButton.Enable()
		} else {
			updateButton.Disable()
			cancelButton.Disable()
		}
		if selectedOrderCanDelete {
			deleteHistoryButton.Enable()
		} else {
			deleteHistoryButton.Disable()
		}
		deleteHistoryButton.Refresh()
	}

	list := widget.NewList(
		func() int { return len(filteredOrders) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			order := filteredOrders[id]
			obj.(*widget.Label).SetText(fmt.Sprintf("#%d | %s | %s | %s", order.ID, order.ScheduledDate, order.ServiceType, order.Status))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		order := filteredOrders[id]
		selectedOrderID = order.ID
		selectedOrderCanModify = order.CanModify
		selectedOrderCanDelete = order.CanDelete
		orderDetails.SetText(fmt.Sprintf("Заказ #%d\nУслуга: %s\nСтатус: %s\nОплата: %s\nДата: %s\nВремя: %s\nАдрес: %s\nСтоимость: %s\nДетали: %s",
			order.ID,
			order.ServiceType,
			order.Status,
			order.PaymentStatus,
			order.ScheduledDate,
			order.ScheduledTime,
			order.AddressLine,
			order.PriceTotal,
			order.Details,
		))
		loadHistory(order.ID)
		if order.CanModify {
			fillForm(order)
		}
		setButtonsState()
	}

	applyFilters := func() {
		query := strings.TrimSpace(searchEntry.Text)
		status := statusSelect.Selected
		filteredOrders = filteredOrders[:0]
		for _, order := range allOrders {
			if status != "" && status != "Все статусы" && order.Status != status {
				continue
			}
			if !containsFold(order.AddressLine+" "+order.ServiceType+" "+order.Details, query) {
				continue
			}
			filteredOrders = append(filteredOrders, order)
		}
		summaryLabel.SetText(fmt.Sprintf("Найдено: %d из %d", len(filteredOrders), len(allOrders)))
		list.UnselectAll()
		selectedOrderID = 0
		selectedOrderCanModify = false
		selectedOrderCanDelete = false
		setButtonsState()
		clearForm()
		orderDetails.SetText("Выберите заказ из списка.")
		historyDetails.SetText("История статусов недоступна.")
		if len(filteredOrders) > 0 {
			list.Select(0)
		}
		list.Refresh()
	}
	searchEntry.OnChanged = func(string) { applyFilters() }
	statusSelect.OnChanged = func(string) { applyFilters() }
	applyFilters()

	form := container.NewVBox(
		serviceSelect,
		pricingHint,
		container.NewGridWithColumns(3, cityEntry, streetEntry, houseEntry),
		container.NewGridWithColumns(3, floorEntry, flatEntry, entranceEntry),
		container.NewGridWithColumns(3, dateEntry, timeFromEntry, timeToEntry),
		container.NewGridWithColumns(2, squareEntry, windowEntry),
		addressCommentEntry,
		detailsEntry,
		container.NewGridWithColumns(2, createButton, updateButton),
		container.NewGridWithColumns(3, cancelButton, deleteHistoryButton, resetButton),
	)

	ordersPanel := widget.NewCard("Мои заказы", "Активные и исторические заказы клиента.", container.NewBorder(
		container.NewVBox(
			summaryLabel,
			container.NewGridWithColumns(2, searchEntry, statusSelect),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewHSplit(
			container.NewPadded(list),
			container.NewVBox(
				container.NewPadded(orderDetails),
				widget.NewSeparator(),
				widget.NewLabel("История статусов"),
				container.NewPadded(historyDetails),
			),
		),
	))

	createPanel := widget.NewCard("Форма заказа", "Новый заказ или редактирование заказа со статусом «Новый».", container.NewVScroll(form))
	main := container.NewHSplit(container.NewPadded(ordersPanel), container.NewPadded(createPanel))
	main.Offset = 0.56

	header := newWorkspaceHeader("Кабинет клиента", "Клиент: "+user.FullName, func() { router.ShowLogin() })

	return newWorkspaceScreen(header, container.New(layout.NewMaxLayout(), main))
}





