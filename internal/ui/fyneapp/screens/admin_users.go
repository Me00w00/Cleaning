package screens

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	auditapp "project_cleaning/internal/app/audit"
	usersapp "project_cleaning/internal/app/users"
	userdomain "project_cleaning/internal/domain/user"
)

func NewAdminUsersScreen(router *Router, currentUser userdomain.User) fyne.CanvasObject {
	users, err := router.LoadUsers()
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}
	auditEntries, err := router.LoadAuditEntries()
	if err != nil {
		return container.NewCenter(widget.NewLabel(err.Error()))
	}

	allUsers := users
	filteredUsers := append([]userdomain.User(nil), allUsers...)
	allAuditEntries := auditEntries
	filteredAuditEntries := append([]auditapp.EntryView(nil), allAuditEntries...)
	selectedID := int64(0)

	loginValue := widget.NewLabel("\u0412\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f \u0438\u0437 \u0441\u043f\u0438\u0441\u043a\u0430")
	fullNameEntry := widget.NewEntry()
	phoneEntry := widget.NewEntry()
	emailEntry := widget.NewEntry()
	roleSelect := widget.NewSelect([]string{"admin", "manager", "staff", "client"}, nil)
	activeCheck := widget.NewCheck("\u0410\u043a\u0442\u0438\u0432\u0435\u043d", nil)
	userSearchEntry := widget.NewEntry()
	userSearchEntry.SetPlaceHolder("\u041f\u043e\u0438\u0441\u043a \u043f\u043e \u043b\u043e\u0433\u0438\u043d\u0443, \u0424\u0418\u041e, \u0442\u0435\u043b\u0435\u0444\u043e\u043d\u0443 \u0438\u043b\u0438 \u0440\u043e\u043b\u0438")
	userSummaryLabel := widget.NewLabel("")

	refreshScreen := func() {
		router.ShowHome(currentUser)
	}

	loadUser := func(user userdomain.User) {
		selectedID = user.ID
		loginValue.SetText(user.Login)
		fullNameEntry.SetText(user.FullName)
		phoneEntry.SetText(user.Phone)
		emailEntry.SetText(user.Email)
		roleSelect.SetSelected(string(user.Role))
		activeCheck.SetChecked(user.IsActive)
	}

	list := widget.NewList(
		func() int { return len(filteredUsers) },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			user := filteredUsers[id]
			status := "\u043d\u0435\u0430\u043a\u0442\u0438\u0432\u0435\u043d"
			if user.IsActive {
				status = "\u0430\u043a\u0442\u0438\u0432\u0435\u043d"
			}
			obj.(*widget.Label).SetText(fmt.Sprintf("%s | %s | %s", user.Login, user.Role, status))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		loadUser(filteredUsers[id])
	}

	saveButton := widget.NewButton("\u0421\u043e\u0445\u0440\u0430\u043d\u0438\u0442\u044c \u0438\u0437\u043c\u0435\u043d\u0435\u043d\u0438\u044f", func() {
		if selectedID == 0 {
			dialog.ShowInformation("\u041f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u0438", "\u0421\u043d\u0430\u0447\u0430\u043b\u0430 \u0432\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f.", router.window)
			return
		}

		router.UpdateUserByAdmin(usersapp.UpdateInput{
			ID:       selectedID,
			FullName: strings.TrimSpace(fullNameEntry.Text),
			Phone:    strings.TrimSpace(phoneEntry.Text),
			Email:    strings.TrimSpace(emailEntry.Text),
			Role:     userdomain.Role(roleSelect.Selected),
			IsActive: activeCheck.Checked,
		}, refreshScreen)
	})

	deleteButton := widget.NewButton("\u0423\u0434\u0430\u043b\u0438\u0442\u044c \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f", func() {
		if selectedID == 0 {
			dialog.ShowInformation("\u041f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u0438", "\u0421\u043d\u0430\u0447\u0430\u043b\u0430 \u0432\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f.", router.window)
			return
		}
		if selectedID == currentUser.ID {
			dialog.ShowInformation("\u041f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u0438", "\u0422\u0435\u043a\u0443\u0449\u0438\u0439 \u0430\u0434\u043c\u0438\u043d\u0438\u0441\u0442\u0440\u0430\u0442\u043e\u0440 \u043d\u0435 \u043c\u043e\u0436\u0435\u0442 \u0443\u0434\u0430\u043b\u0438\u0442\u044c \u0441\u0430\u043c\u043e\u0433\u043e \u0441\u0435\u0431\u044f \u0432 \u0430\u043a\u0442\u0438\u0432\u043d\u043e\u0439 \u0441\u0435\u0441\u0441\u0438\u0438.", router.window)
			return
		}

		dialog.ShowConfirm("\u0423\u0434\u0430\u043b\u0435\u043d\u0438\u0435 \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f", "\u0423\u0434\u0430\u043b\u0438\u0442\u044c \u0432\u044b\u0431\u0440\u0430\u043d\u043d\u043e\u0433\u043e \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f \u0431\u0435\u0437 \u0432\u043e\u0437\u043c\u043e\u0436\u043d\u043e\u0441\u0442\u0438 \u0432\u043e\u0441\u0441\u0442\u0430\u043d\u043e\u0432\u043b\u0435\u043d\u0438\u044f?", func(ok bool) {
			if !ok {
				return
			}
			router.DeleteUserByAdmin(selectedID, refreshScreen)
		}, router.window)
	})

	reloadButton := widget.NewButton("\u041e\u0431\u043d\u043e\u0432\u0438\u0442\u044c", func() {
		refreshScreen()
	})

	editCard := widget.NewCard("\u0414\u0430\u043d\u043d\u044b\u0435 \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f", "", container.NewVBox(
		widget.NewLabel("\u041b\u043e\u0433\u0438\u043d"),
		loginValue,
		widget.NewLabel("\u0424\u0418\u041e"),
		fullNameEntry,
		widget.NewLabel("\u0422\u0435\u043b\u0435\u0444\u043e\u043d"),
		phoneEntry,
		widget.NewLabel("Email"),
		emailEntry,
		widget.NewLabel("\u0420\u043e\u043b\u044c"),
		roleSelect,
		activeCheck,
		container.NewGridWithColumns(3, reloadButton, deleteButton, saveButton),
	))

	createLogin := widget.NewEntry()
	createPassword := widget.NewPasswordEntry()
	createName := widget.NewEntry()
	createPhone := widget.NewEntry()
	createEmail := widget.NewEntry()
	createRole := widget.NewSelect([]string{"manager", "staff", "admin"}, nil)
	createRole.SetSelected("manager")
	createStatusLabel := widget.NewLabel("\u0417\u0430\u043f\u043e\u043b\u043d\u0438\u0442\u0435 \u043b\u043e\u0433\u0438\u043d, \u0432\u0440\u0435\u043c\u0435\u043d\u043d\u044b\u0439 \u043f\u0430\u0440\u043e\u043b\u044c, \u0424\u0418\u041e \u0438 \u0442\u0435\u043b\u0435\u0444\u043e\u043d.")
	createStatusLabel.Wrapping = fyne.TextWrapWord
	createSubmitting := false

	var createButton *widget.Button
	var updateCreateState func()
	createButton = widget.NewButton("\u0421\u043e\u0437\u0434\u0430\u0442\u044c \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f", func() {
		if createSubmitting {
			return
		}
		login := strings.TrimSpace(createLogin.Text)
		password := strings.TrimSpace(createPassword.Text)
		fullName := strings.TrimSpace(createName.Text)
		phone := strings.TrimSpace(createPhone.Text)
		email := strings.TrimSpace(createEmail.Text)
		if login == "" || password == "" || fullName == "" || phone == "" || createRole.Selected == "" {
			createStatusLabel.SetText("\u041b\u043e\u0433\u0438\u043d, \u043f\u0430\u0440\u043e\u043b\u044c, \u0424\u0418\u041e, \u0442\u0435\u043b\u0435\u0444\u043e\u043d \u0438 \u0440\u043e\u043b\u044c \u043e\u0431\u044f\u0437\u0430\u0442\u0435\u043b\u044c\u043d\u044b.")
			return
		}
		createSubmitting = true
		createButton.Disable()
		createStatusLabel.SetText("\u0421\u043e\u0437\u0434\u0430\u0435\u043c \u0443\u0447\u0435\u0442\u043d\u0443\u044e \u0437\u0430\u043f\u0438\u0441\u044c...")
		defer func() {
			createSubmitting = false
			updateCreateState()
		}()
		router.CreateUserByAdmin(usersapp.CreateInput{
			Login:    login,
			Password: password,
			FullName: fullName,
			Phone:    phone,
			Email:    email,
			Role:     userdomain.Role(createRole.Selected),
		}, func() {
			createLogin.SetText("")
			createPassword.SetText("")
			createName.SetText("")
			createPhone.SetText("")
			createEmail.SetText("")
			createRole.SetSelected("manager")
			refreshScreen()
		})
	})
	createButton.Disable()

	updateCreateState = func() {
		if createSubmitting {
			return
		}
		if strings.TrimSpace(createLogin.Text) == "" ||
			strings.TrimSpace(createPassword.Text) == "" ||
			strings.TrimSpace(createName.Text) == "" ||
			strings.TrimSpace(createPhone.Text) == "" ||
			strings.TrimSpace(createRole.Selected) == "" {
			createButton.Disable()
			createStatusLabel.SetText("\u0417\u0430\u043f\u043e\u043b\u043d\u0438\u0442\u0435 \u043b\u043e\u0433\u0438\u043d, \u0432\u0440\u0435\u043c\u0435\u043d\u043d\u044b\u0439 \u043f\u0430\u0440\u043e\u043b\u044c, \u0424\u0418\u041e \u0438 \u0442\u0435\u043b\u0435\u0444\u043e\u043d.")
			return
		}
		createButton.Enable()
		createStatusLabel.SetText("\u0424\u043e\u0440\u043c\u0430 \u0433\u043e\u0442\u043e\u0432\u0430 \u043a \u0441\u043e\u0437\u0434\u0430\u043d\u0438\u044e.")
	}
	createLogin.OnChanged = func(string) { updateCreateState() }
	createPassword.OnChanged = func(string) { updateCreateState() }
	createName.OnChanged = func(string) { updateCreateState() }
	createPhone.OnChanged = func(string) { updateCreateState() }
	createRole.OnChanged = func(string) { updateCreateState() }

	createCard := widget.NewCard("\u0421\u043e\u0437\u0434\u0430\u043d\u0438\u0435 \u0443\u0447\u0435\u0442\u043d\u043e\u0439 \u0437\u0430\u043f\u0438\u0441\u0438", "", container.NewVBox(
		createLogin,
		createPassword,
		createName,
		createPhone,
		createEmail,
		createRole,
		createStatusLabel,
		createButton,
	))

	createLogin.SetPlaceHolder("\u041b\u043e\u0433\u0438\u043d")
	createPassword.SetPlaceHolder("\u0412\u0440\u0435\u043c\u0435\u043d\u043d\u044b\u0439 \u043f\u0430\u0440\u043e\u043b\u044c")
	createName.SetPlaceHolder("\u0424\u0418\u041e")
	createPhone.SetPlaceHolder("\u0422\u0435\u043b\u0435\u0444\u043e\u043d")
	createEmail.SetPlaceHolder("Email")

	auditSearchEntry := widget.NewEntry()
	auditSearchEntry.SetPlaceHolder("\u041f\u043e\u0438\u0441\u043a \u043f\u043e \u0438\u0441\u043f\u043e\u043b\u043d\u0438\u0442\u0435\u043b\u044e, \u0434\u0435\u0439\u0441\u0442\u0432\u0438\u044e, \u0441\u0443\u0449\u043d\u043e\u0441\u0442\u0438 \u0438\u043b\u0438 \u0434\u0435\u0442\u0430\u043b\u044f\u043c")
	auditSummaryLabel := widget.NewLabel("")
	auditList := widget.NewList(
		func() int { return len(filteredAuditEntries) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			entry := filteredAuditEntries[id]
			obj.(*widget.Label).SetText(fmt.Sprintf("%s | %s | %s | %s", entry.CreatedAt, entry.ActorName, entry.Action, entry.Entity))
		},
	)
	auditInfo := widget.NewLabel("\u0412\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u0437\u0430\u043f\u0438\u0441\u044c \u0430\u0443\u0434\u0438\u0442\u0430")
	auditInfo.Wrapping = fyne.TextWrapWord
	auditList.OnSelected = func(id widget.ListItemID) {
		entry := filteredAuditEntries[id]
		auditInfo.SetText(fmt.Sprintf("\u0412\u0440\u0435\u043c\u044f: %s\n\u0418\u0441\u043f\u043e\u043b\u043d\u0438\u0442\u0435\u043b\u044c: %s\n\u0414\u0435\u0439\u0441\u0442\u0432\u0438\u0435: %s\n\u0421\u0443\u0449\u043d\u043e\u0441\u0442\u044c: %s\n\u0414\u0435\u0442\u0430\u043b\u0438: %s", entry.CreatedAt, entry.ActorName, entry.Action, entry.Entity, entry.Payload))
	}

	applyUserFilter := func() {
		query := strings.TrimSpace(userSearchEntry.Text)
		filteredUsers = filteredUsers[:0]
		for _, user := range allUsers {
			if !containsFold(user.Login+" "+user.FullName+" "+user.Phone+" "+string(user.Role), query) {
				continue
			}
			filteredUsers = append(filteredUsers, user)
		}
		userSummaryLabel.SetText(fmt.Sprintf("\u041d\u0430\u0439\u0434\u0435\u043d\u043e: %d \u0438\u0437 %d", len(filteredUsers), len(allUsers)))
		list.UnselectAll()
		selectedID = 0
		loginValue.SetText("\u0412\u044b\u0431\u0435\u0440\u0438\u0442\u0435 \u043f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044f \u0438\u0437 \u0441\u043f\u0438\u0441\u043a\u0430")
		fullNameEntry.SetText("")
		phoneEntry.SetText("")
		emailEntry.SetText("")
		roleSelect.ClearSelected()
		activeCheck.SetChecked(false)
		list.Refresh()
		if len(filteredUsers) > 0 {
			list.Select(0)
		}
	}

	applyAuditFilter := func() {
		query := strings.TrimSpace(auditSearchEntry.Text)
		filteredAuditEntries = filteredAuditEntries[:0]
		for _, entry := range allAuditEntries {
			if !containsFold(entry.ActorName+" "+entry.Action+" "+entry.Entity+" "+entry.Payload, query) {
				continue
			}
			filteredAuditEntries = append(filteredAuditEntries, entry)
		}
		auditSummaryLabel.SetText(fmt.Sprintf("\u041d\u0430\u0439\u0434\u0435\u043d\u043e: %d \u0438\u0437 %d", len(filteredAuditEntries), len(allAuditEntries)))
		auditList.UnselectAll()
		auditList.Refresh()
		if len(filteredAuditEntries) == 0 {
			auditInfo.SetText("\u041f\u043e \u0442\u0435\u043a\u0443\u0449\u0435\u043c\u0443 \u0444\u0438\u043b\u044c\u0442\u0440\u0443 \u0437\u0430\u043f\u0438\u0441\u0438 \u0430\u0443\u0434\u0438\u0442\u0430 \u043d\u0435 \u043d\u0430\u0439\u0434\u0435\u043d\u044b.")
			return
		}
		auditList.Select(0)
	}

	userSearchEntry.OnChanged = func(string) { applyUserFilter() }
	auditSearchEntry.OnChanged = func(string) { applyAuditFilter() }
	applyUserFilter()
	applyAuditFilter()
	updateCreateState()

	header := newWorkspaceHeader("Пользователи и аудит", "Администратор: "+currentUser.FullName, func() {
	router.ShowLogin()
})
usersTab := container.NewHSplit(container.NewBorder(
		container.NewVBox(
			userSummaryLabel,
			userSearchEntry,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewPadded(list),
	), container.NewPadded(editCard))
	usersTab.Offset = 0.36
	auditTab := container.NewHSplit(container.NewBorder(
		container.NewVBox(
			auditSummaryLabel,
			auditSearchEntry,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewPadded(auditList),
	), container.NewPadded(auditInfo))
	auditTab.Offset = 0.55

	content := container.NewAppTabs(
		container.NewTabItem("\u041f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u0438", usersTab),
		container.NewTabItem("\u0421\u043e\u0437\u0434\u0430\u043d\u0438\u0435", container.NewPadded(createCard)),
		container.NewTabItem("\u0410\u0443\u0434\u0438\u0442", auditTab),
	)

	return newWorkspaceScreen(header, content)
}



