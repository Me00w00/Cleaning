package screens

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func NewLoginScreen(router *Router) fyne.CanvasObject {
	statusLabel := widget.NewLabel("Введите логин и пароль для входа.")
	statusLabel.Wrapping = fyne.TextWrapWord

	loginEntry := widget.NewEntry()
	loginEntry.SetPlaceHolder("Логин")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Пароль")

	loginButton := widget.NewButtonWithIcon("Войти", theme.LoginIcon(), func() {
		login := strings.TrimSpace(loginEntry.Text)
		password := strings.TrimSpace(passwordEntry.Text)
		if login == "" || password == "" {
			statusLabel.SetText("Логин и пароль обязательны.")
			return
		}
		statusLabel.SetText("Выполняется вход...")
		router.HandleLogin(login, password)
	})
	loginButton.Disable()

	registerButton := widget.NewButtonWithIcon("Регистрация клиента", theme.AccountIcon(), func() {
		router.ShowClientRegistration()
	})

	updateLoginState := func() {
		if strings.TrimSpace(loginEntry.Text) != "" && strings.TrimSpace(passwordEntry.Text) != "" {
			loginButton.Enable()
			statusLabel.SetText("Можно войти в систему.")
			return
		}
		loginButton.Disable()
		statusLabel.SetText("Введите логин и пароль для входа.")
	}
	loginEntry.OnChanged = func(string) { updateLoginState() }
	passwordEntry.OnChanged = func(string) { updateLoginState() }

	heroTitle := widget.NewRichTextFromMarkdown("# Свежо, прозрачно, без лишней рутины")
	heroText := widget.NewLabel("Интерфейс построен вокруг ежедневной работы клинингового агентства: оформить заказ, назначить исполнителя, подтвердить оплату и отследить результат.")
	heroText.Wrapping = fyne.TextWrapWord
	chip1 := widget.NewCard("Чистые статусы", "Каждый этап заказа виден сразу", nil)
	chip2 := widget.NewCard("Быстрое назначение", "Менеджер сразу видит доступность сотрудника", nil)
	chip3 := widget.NewCard("Контакты под рукой", "Клиент, менеджер и исполнитель в одной карточке", nil)

	heroPanel := container.NewVBox(
		heroTitle,
		heroText,
		widget.NewSeparator(),
		chip1,
		chip2,
		chip3,
	)

	formCard := widget.NewCard("Вход в систему", "", container.NewVBox(
		loginEntry,
		passwordEntry,
		statusLabel,
		container.NewGridWithColumns(2, loginButton, registerButton),
	))

	accent := canvas.NewRectangle(theme.PrimaryColor())
	accent.SetMinSize(fyne.NewSize(10, 0))
	leftBox := container.NewBorder(nil, nil, accent, nil, container.NewPadded(heroPanel))

	split := container.NewHSplit(container.NewPadded(leftBox), container.NewPadded(formCard))
	split.Offset = 0.54

	background := canvas.NewLinearGradient(
		color.NRGBA{R: 241, G: 248, B: 247, A: 255},
		color.NRGBA{R: 223, G: 242, B: 245, A: 255},
		35,
	)

	return container.New(layout.NewMaxLayout(), background, container.NewPadded(split))
}
