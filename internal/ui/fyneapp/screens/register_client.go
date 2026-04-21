package screens

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	usersapp "project_cleaning/internal/app/users"
)

func NewClientRegistrationScreen(router *Router) fyne.CanvasObject {
	registrationKey := newIdempotencyKey()
	submitting := false

	loginEntry := widget.NewEntry()
	loginEntry.SetPlaceHolder("Логин")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Пароль")

	passwordConfirmEntry := widget.NewPasswordEntry()
	passwordConfirmEntry.SetPlaceHolder("Повторите пароль")

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("ФИО")

	phoneEntry := widget.NewEntry()
	phoneEntry.SetPlaceHolder("Телефон")

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Email")

	statusLabel := widget.NewLabel("Заполните логин, пароль, ФИО и телефон.")
	statusLabel.Wrapping = fyne.TextWrapWord

	var registerButton *widget.Button
	var updateRegistrationState func()
	registerButton = widget.NewButtonWithIcon("Создать аккаунт", theme.ConfirmIcon(), func() {
		if submitting {
			return
		}
		login := strings.TrimSpace(loginEntry.Text)
		password := strings.TrimSpace(passwordEntry.Text)
		passwordConfirm := strings.TrimSpace(passwordConfirmEntry.Text)
		fullName := strings.TrimSpace(nameEntry.Text)
		phone := strings.TrimSpace(phoneEntry.Text)
		email := strings.TrimSpace(emailEntry.Text)
		if login == "" || password == "" || fullName == "" || phone == "" {
			statusLabel.SetText("Логин, пароль, ФИО и телефон обязательны.")
			return
		}
		if password != passwordConfirm {
			statusLabel.SetText("Пароли не совпадают.")
			return
		}
		submitting = true
		registerButton.Disable()
		statusLabel.SetText("Идет регистрация...")
		defer func() {
			submitting = false
			updateRegistrationState()
		}()

		router.RegisterClient(usersapp.CreateInput{
			Login:          login,
			Password:       password,
			FullName:       fullName,
			Phone:          phone,
			Email:          email,
			IdempotencyKey: registrationKey,
		})
	})
	registerButton.Disable()

	updateRegistrationState = func() {
		login := strings.TrimSpace(loginEntry.Text)
		password := strings.TrimSpace(passwordEntry.Text)
		passwordConfirm := strings.TrimSpace(passwordConfirmEntry.Text)
		fullName := strings.TrimSpace(nameEntry.Text)
		phone := strings.TrimSpace(phoneEntry.Text)
		switch {
		case login == "" || password == "" || fullName == "" || phone == "":
			registerButton.Disable()
			statusLabel.SetText("Заполните логин, пароль, ФИО и телефон.")
		case passwordConfirm == "":
			registerButton.Disable()
			statusLabel.SetText("Повторите пароль для проверки.")
		case password != passwordConfirm:
			registerButton.Disable()
			statusLabel.SetText("Пароли не совпадают.")
		default:
			registerButton.Enable()
			statusLabel.SetText("Форма готова к отправке.")
		}
	}
	loginEntry.OnChanged = func(string) { updateRegistrationState() }
	passwordEntry.OnChanged = func(string) { updateRegistrationState() }
	passwordConfirmEntry.OnChanged = func(string) { updateRegistrationState() }
	nameEntry.OnChanged = func(string) { updateRegistrationState() }
	phoneEntry.OnChanged = func(string) { updateRegistrationState() }

	backButton := widget.NewButtonWithIcon("Назад ко входу", theme.NavigateBackIcon(), func() {
		router.ShowLogin()
	})

	heroTitle := widget.NewRichTextFromMarkdown("# Новый клиент за пару минут")
	heroText := widget.NewLabel("Регистрация оставляет только нужные поля: контактные данные и доступ к личному кабинету, где можно быстро оформить уборку и следить за статусами.")
	heroText.Wrapping = fyne.TextWrapWord
	benefits := widget.NewLabel("Что будет после регистрации:\n• создание и редактирование заказа\n• история изменений статуса\n• контакты менеджера и сотрудника\n• хранение всех деталей адреса и работ")
	benefits.Wrapping = fyne.TextWrapWord

	heroCard := widget.NewCard("Личный кабинет клиента", "Чистый и понятный старт без лишних шагов", container.NewVBox(
		heroTitle,
		heroText,
		widget.NewSeparator(),
		benefits,
	))

	formCard := widget.NewCard("Регистрация клиента", "Самостоятельная регистрация доступна только для клиентов", container.NewVBox(
		loginEntry,
		passwordEntry,
		passwordConfirmEntry,
		nameEntry,
		phoneEntry,
		emailEntry,
		statusLabel,
		container.NewGridWithColumns(2, backButton, registerButton),
	))

	accent := canvas.NewRectangle(theme.PrimaryColor())
	accent.SetMinSize(fyne.NewSize(10, 0))
	leftBox := container.NewBorder(nil, nil, accent, nil, container.NewPadded(heroCard))

	split := container.NewHSplit(
		container.NewPadded(leftBox),
		container.NewPadded(formCard),
	)
	split.Offset = 0.5

	background := canvas.NewRectangle(theme.BackgroundColor())
	return container.New(layout.NewMaxLayout(), background, container.NewPadded(split))
}
