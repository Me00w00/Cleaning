package screens

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	staffapp "project_cleaning/internal/app/staff"
	userdomain "project_cleaning/internal/domain/user"
)

func NewStaffOrdersScreen(router *Router, user userdomain.User) fyne.CanvasObject {
	orders, err := router.LoadStaffOrders()
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}
	availability, err := router.LoadStaffAvailability()
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}

	allOrders := orders
	filteredOrders := append([]staffapp.OrderView(nil), allOrders...)

	selectedOrderID := int64(0)
	availabilityKey := newIdempotencyKey()
	selectedCanAccept := false
	selectedCanDecline := false
	selectedCanStart := false
	selectedCanComplete := false
	availabilitySubmitting := false

	orderInfo := widget.NewLabel("")
	orderInfo.Wrapping = fyne.TextWrapWord
	historyInfo := widget.NewLabel("")
	historyInfo.Wrapping = fyne.TextWrapWord
	summaryLabel := widget.NewLabel("")

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("\u041f\u043e\u0438\u0441\u043a \u043f\u043e \u043a\u043b\u0438\u0435\u043d\u0442\u0443, \u0430\u0434\u0440\u0435\u0441\u0443 \u0438\u043b\u0438 \u0443\u0441\u043b\u0443\u0433\u0435")
	statusSelect := widget.NewSelect([]string{
		"\u0412\u0441\u0435 \u0441\u0442\u0430\u0442\u0443\u0441\u044b",
		"\u041d\u0430\u0437\u043d\u0430\u0447\u0435\u043d \u0441\u043e\u0442\u0440\u0443\u0434\u043d\u0438\u043a",
		"\u041f\u043e\u0434\u0442\u0432\u0435\u0440\u0436\u0434\u0435\u043d \u0441\u043e\u0442\u0440\u0443\u0434\u043d\u0438\u043a\u043e\u043c",
		"\u0412 \u0440\u0430\u0431\u043e\u0442\u0435",
		"\u0412\u044b\u043f\u043e\u043b\u043d\u0435\u043d",
		"\u0417\u0430\u043a\u0440\u044b\u0442",
		"\u041e\u0442\u043c\u0435\u043d\u0435\u043d",
	}, nil)
	statusSelect.SetSelected("\u0412\u0441\u0435 \u0441\u0442\u0430\u0442\u0443\u0441\u044b")

	setOrderState := func(message string) {
		orderInfo.SetText(message)
		historyInfo.SetText("\u0418\u0441\u0442\u043e\u0440\u0438\u044f \u0441\u0442\u0430\u0442\u0443\u0441\u043e\u0432 \u043d\u0435\u0434\u043e\u0441\u0442\u0443\u043f\u043d\u0430.")
	}

	acceptButton := widget.NewButton("\u041f\u043e\u0434\u0442\u0432\u0435\u0440\u0434\u0438\u0442\u044c \u0437\u0430\u043a\u0430\u0437", func() {
		if selectedOrderID == 0 || !selectedCanAccept {
			dialog.ShowInformation("\u0417\u0430\u043a\u0430\u0437\u044b", "\u0412\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u0437\u0430\u043a\u0430\u0437 \u0432 \u0441\u0442\u0430\u0442\u0443\u0441\u0435 \u00ab\u041d\u0430\u0437\u043d\u0430\u0447\u0435\u043d \u0441\u043e\u0442\u0440\u0443\u0434\u043d\u0438\u043a\u00bb.", router.window)
			return
		}
		router.AcceptStaffOrder(selectedOrderID)
	})

	declineButton := widget.NewButton("\u041e\u0442\u043a\u0430\u0437\u0430\u0442\u044c\u0441\u044f", func() {
		if selectedOrderID == 0 || !selectedCanDecline {
			dialog.ShowInformation("\u0417\u0430\u043a\u0430\u0437\u044b", "\u0412\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u0437\u0430\u043a\u0430\u0437 \u0432 \u0441\u0442\u0430\u0442\u0443\u0441\u0435 \u00ab\u041d\u0430\u0437\u043d\u0430\u0447\u0435\u043d \u0441\u043e\u0442\u0440\u0443\u0434\u043d\u0438\u043a\u00bb.", router.window)
			return
		}
		dialog.ShowConfirm("\u041e\u0442\u043a\u0430\u0437 \u043e\u0442 \u0437\u0430\u043a\u0430\u0437\u0430", "\u0412\u0435\u0440\u043d\u0443\u0442\u044c \u0437\u0430\u043a\u0430\u0437 \u043c\u0435\u043d\u0435\u0434\u0436\u0435\u0440\u0443 \u0434\u043b\u044f \u043f\u0435\u0440\u0435\u043d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u0438\u044f?", func(ok bool) {
			if !ok {
				return
			}
			router.DeclineStaffOrder(selectedOrderID)
		}, router.window)
	})

	startButton := widget.NewButton("\u041d\u0430\u0447\u0430\u0442\u044c \u0440\u0430\u0431\u043e\u0442\u0443", func() {
		if selectedOrderID == 0 || !selectedCanStart {
			dialog.ShowInformation("\u0417\u0430\u043a\u0430\u0437\u044b", "\u0412\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u043f\u043e\u0434\u0442\u0432\u0435\u0440\u0436\u0434\u0435\u043d\u043d\u044b\u0439 \u0437\u0430\u043a\u0430\u0437.", router.window)
			return
		}
		router.StartStaffOrder(selectedOrderID)
	})

	completeButton := widget.NewButton("\u0417\u0430\u0432\u0435\u0440\u0448\u0438\u0442\u044c", func() {
		if selectedOrderID == 0 || !selectedCanComplete {
			dialog.ShowInformation("\u0417\u0430\u043a\u0430\u0437\u044b", "\u0412\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u0437\u0430\u043a\u0430\u0437 \u0432 \u0441\u0442\u0430\u0442\u0443\u0441\u0435 \u00ab\u0412 \u0440\u0430\u0431\u043e\u0442\u0435\u00bb.", router.window)
			return
		}
		router.CompleteStaffOrder(selectedOrderID)
	})

	setActionState := func() {
		if selectedCanAccept {
			acceptButton.Enable()
		} else {
			acceptButton.Disable()
		}
		if selectedCanDecline {
			declineButton.Enable()
		} else {
			declineButton.Disable()
		}
		if selectedCanStart {
			startButton.Enable()
		} else {
			startButton.Disable()
		}
		if selectedCanComplete {
			completeButton.Enable()
		} else {
			completeButton.Disable()
		}
	}

	ordersList := widget.NewList(
		func() int { return len(filteredOrders) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			order := filteredOrders[id]
			obj.(*widget.Label).SetText(fmt.Sprintf("#%d | %s | %s | %s", order.ID, order.ScheduledDate, order.ClientName, order.Status))
		},
	)
	ordersList.OnSelected = func(id widget.ListItemID) {
		order := filteredOrders[id]
		selectedOrderID = order.ID
		selectedCanAccept = order.CanAccept
		selectedCanDecline = order.CanDecline
		selectedCanStart = order.CanStart
		selectedCanComplete = order.CanComplete
		orderInfo.SetText(fmt.Sprintf("\u0417\u0430\u043a\u0430\u0437 #%d\n\u0421\u0442\u0430\u0442\u0443\u0441: %s\n\u0423\u0441\u043b\u0443\u0433\u0430: %s\n\u041a\u043b\u0438\u0435\u043d\u0442: %s\n\u0422\u0435\u043b\u0435\u0444\u043e\u043d: %s\n\u0410\u0434\u0440\u0435\u0441: %s\n\u0414\u0430\u0442\u0430: %s\n\u0412\u0440\u0435\u043c\u044f: %s\n\u0421\u0442\u043e\u0438\u043c\u043e\u0441\u0442\u044c: %s",
			order.ID,
			order.Status,
			order.ServiceType,
			order.ClientName,
			order.ClientPhone,
			order.AddressLine,
			order.ScheduledDate,
			order.ScheduledTime,
			order.PriceTotal,
		))
		history, historyErr := router.LoadStaffOrderHistory(order.ID)
		if historyErr != nil {
			historyInfo.SetText(historyErr.Error())
			return
		}
		if len(history) == 0 {
			historyInfo.SetText("\u0418\u0441\u0442\u043e\u0440\u0438\u044f \u0441\u0442\u0430\u0442\u0443\u0441\u043e\u0432 \u043f\u043e\u043a\u0430 \u043f\u0443\u0441\u0442\u0430.")
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
			if status != "" && status != "\u0412\u0441\u0435 \u0441\u0442\u0430\u0442\u0443\u0441\u044b" && order.Status != status {
				continue
			}
			if !containsFold(order.ClientName+" "+order.AddressLine+" "+order.ServiceType, query) {
				continue
			}
			filteredOrders = append(filteredOrders, order)
		}
		summaryLabel.SetText(fmt.Sprintf("\u041d\u0430\u0439\u0434\u0435\u043d\u043e: %d \u0438\u0437 %d", len(filteredOrders), len(allOrders)))
		ordersList.UnselectAll()
		selectedOrderID = 0
		selectedCanAccept = false
		selectedCanDecline = false
		selectedCanStart = false
		selectedCanComplete = false
		setActionState()
		if len(filteredOrders) == 0 {
			setOrderState("\u041f\u043e \u0442\u0435\u043a\u0443\u0449\u0438\u043c \u0444\u0438\u043b\u044c\u0442\u0440\u0430\u043c \u0437\u0430\u043a\u0430\u0437\u044b \u043d\u0435 \u043d\u0430\u0439\u0434\u0435\u043d\u044b.")
		} else {
			ordersList.Select(0)
		}
		ordersList.Refresh()
	}
	searchEntry.OnChanged = func(string) { applyFilters() }
	statusSelect.OnChanged = func(string) { applyFilters() }
	applyFilters()

	refreshButton := widget.NewButton("\u041e\u0431\u043d\u043e\u0432\u0438\u0442\u044c", func() { router.ShowHome(user) })

	startDateSelectors := newDateSelectors(time.Now())
	endDateSelectors := newDateSelectors(time.Now())

	reasonEntry := widget.NewEntry()
	reasonEntry.SetPlaceHolder("\u041f\u0440\u0438\u0447\u0438\u043d\u0430")
	availabilityHint := widget.NewLabel("\u0417\u0430\u0434\u0430\u0439\u0442\u0435 \u0434\u0438\u0430\u043f\u0430\u0437\u043e\u043d \u0434\u0430\u0442 \u043d\u0435\u0434\u043e\u0441\u0442\u0443\u043f\u043d\u043e\u0441\u0442\u0438. \u0412\u0435\u0441\u044c \u0434\u0435\u043d\u044c \u0431\u0443\u0434\u0435\u0442 \u0441\u0447\u0438\u0442\u0430\u0442\u044c\u0441\u044f \u043d\u0435\u0434\u043e\u0441\u0442\u0443\u043f\u043d\u044b\u043c, \u043f\u043e\u044d\u0442\u043e\u043c\u0443 \u043c\u0435\u043d\u0435\u0434\u0436\u0435\u0440 \u043d\u0435 \u0441\u043c\u043e\u0436\u0435\u0442 \u043d\u0430\u0437\u043d\u0430\u0447\u0438\u0442\u044c \u0432\u0430\u0441 \u043d\u0430 \u0437\u0430\u043a\u0430\u0437 \u0432 \u044d\u0442\u0438 \u0434\u0430\u0442\u044b.")
	availabilityHint.Wrapping = fyne.TextWrapWord

	availabilityList := widget.NewList(
		func() int { return len(availability) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			item := availability[id]
			label := fmt.Sprintf("%s - %s", item.StartsAt, item.EndsAt)
			if strings.TrimSpace(item.Reason) != "" {
				label += " | " + item.Reason
			}
			obj.(*widget.Label).SetText(label)
		},
	)

	var addAvailabilityButton *widget.Button
	addAvailabilityButton = widget.NewButton("\u0414\u043e\u0431\u0430\u0432\u0438\u0442\u044c \u043f\u0435\u0440\u0438\u043e\u0434", func() {
		if availabilitySubmitting {
			return
		}
		availabilitySubmitting = true
		addAvailabilityButton.Disable()
		defer func() {
			availabilitySubmitting = false
			addAvailabilityButton.Enable()
		}()
		router.AddStaffAvailability(staffapp.AvailabilityInput{
			StartDate:      selectedDateValue(startDateSelectors),
			EndDate:        selectedDateValue(endDateSelectors),
			Reason:         strings.TrimSpace(reasonEntry.Text),
			IdempotencyKey: availabilityKey,
		})
	})

	ordersPanel := widget.NewCard(
		"\u041c\u043e\u0438 \u0437\u0430\u043a\u0430\u0437\u044b",
		fmt.Sprintf("\u041d\u0430\u0437\u043d\u0430\u0447\u0435\u043d\u043e: %s", strconv.Itoa(len(allOrders))),
		container.NewBorder(
			container.NewVBox(
				summaryLabel,
				container.NewGridWithColumns(2, searchEntry, statusSelect),
				widget.NewSeparator(),
			),
			nil,
			nil,
			nil,
			container.NewHSplit(
				container.NewPadded(ordersList),
				container.NewVBox(container.NewPadded(orderInfo), widget.NewSeparator(), widget.NewLabel("\u0418\u0441\u0442\u043e\u0440\u0438\u044f \u0441\u0442\u0430\u0442\u0443\u0441\u043e\u0432"), container.NewPadded(historyInfo)),
			),
		),
	)

	actionsPanel := widget.NewCard(
		"\u0414\u0435\u0439\u0441\u0442\u0432\u0438\u044f \u043f\u043e \u0437\u0430\u043a\u0430\u0437\u0443",
		"\u041f\u043e\u0434\u0442\u0432\u0435\u0440\u0436\u0434\u0435\u043d\u0438\u0435, \u043e\u0442\u043a\u0430\u0437 \u0438 \u0441\u043c\u0435\u043d\u0430 \u0441\u0442\u0430\u0442\u0443\u0441\u0430.",
		container.NewVBox(
			container.NewGridWithColumns(2, acceptButton, declineButton),
			container.NewGridWithColumns(2, startButton, completeButton),
			refreshButton,
		),
	)

	availabilityForm := container.NewVBox(
		availabilityHint,
		widget.NewLabel("\u041d\u0430\u0447\u0430\u043b\u043e"),
		container.NewGridWithColumns(3, startDateSelectors.Year, startDateSelectors.Month, startDateSelectors.Day),
		widget.NewLabel("\u041a\u043e\u043d\u0435\u0446"),
		container.NewGridWithColumns(3, endDateSelectors.Year, endDateSelectors.Month, endDateSelectors.Day),
		reasonEntry,
		addAvailabilityButton,
		widget.NewSeparator(),
		availabilityList,
	)

	availabilityPanel := widget.NewCard(
		"\u041a\u0430\u043b\u0435\u043d\u0434\u0430\u0440\u044c \u043d\u0435\u0434\u043e\u0441\u0442\u0443\u043f\u043d\u043e\u0441\u0442\u0438",
		fmt.Sprintf("\u041f\u0435\u0440\u0438\u043e\u0434\u043e\u0432: %s", strconv.Itoa(len(availability))),
		container.NewVScroll(availabilityForm),
	)

	left := container.NewVSplit(container.NewPadded(ordersPanel), container.NewPadded(actionsPanel))
	left.Offset = 0.76
	right := container.NewPadded(availabilityPanel)
	content := container.NewHSplit(left, right)
	content.Offset = 0.58

	header := newWorkspaceHeader("Рабочее место сотрудника", "Сотрудник: "+user.FullName, func() { router.ShowLogin() })

	return newWorkspaceScreen(header, container.New(layout.NewMaxLayout(), content))
}

type dateSelectors struct {
	Year  *widget.Select
	Month *widget.Select
	Day   *widget.Select
}

func newDateSelectors(value time.Time) dateSelectors {
	years := make([]string, 0, 7)
	for year := value.Year() - 1; year <= value.Year()+5; year++ {
		years = append(years, fmt.Sprintf("%04d", year))
	}
	months := []string{"01", "02", "03", "04", "05", "06", "07", "08", "09", "10", "11", "12"}
	days := make([]string, 0, 31)
	for day := 1; day <= 31; day++ {
		days = append(days, fmt.Sprintf("%02d", day))
	}

	yearSelect := widget.NewSelect(years, nil)
	monthSelect := widget.NewSelect(months, nil)
	daySelect := widget.NewSelect(days, nil)
	yearSelect.SetSelected(fmt.Sprintf("%04d", value.Year()))
	monthSelect.SetSelected(fmt.Sprintf("%02d", value.Month()))
	daySelect.SetSelected(fmt.Sprintf("%02d", value.Day()))
	return dateSelectors{Year: yearSelect, Month: monthSelect, Day: daySelect}
}


func selectedDateValue(selectors dateSelectors) string {
	return selectors.Year.Selected + "-" + selectors.Month.Selected + "-" + selectors.Day.Selected
}





