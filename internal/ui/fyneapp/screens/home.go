package screens

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	userdomain "project_cleaning/internal/domain/user"
)

func NewBasicHomeScreen(router *Router, user userdomain.User, titleText, bodyText string) fyne.CanvasObject {
	header := container.NewBorder(nil, nil, nil, widget.NewButton("Выход", func() {
		router.ShowLogin()
	}), widget.NewLabel("Выполнен вход: "+user.FullName+" ("+string(user.Role)+")"))

	body := widget.NewCard(titleText, "", container.NewVBox(
		widget.NewLabel(bodyText),
		widget.NewSeparator(),
		widget.NewLabel("На этом этапе реализованы авторизация, регистрация клиента и управление пользователями для администратора."),
	))

	return container.NewBorder(header, nil, nil, nil, container.NewPadded(body))
}
