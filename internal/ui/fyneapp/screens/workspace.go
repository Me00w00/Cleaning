package screens

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func newWorkspaceHeader(title, userLabel string, onLogout func()) fyne.CanvasObject {
	accent := canvas.NewRectangle(theme.PrimaryColor())
	accent.SetMinSize(fyne.NewSize(8, 0))

	titleBlock := widget.NewRichTextFromMarkdown("## " + title)
	return widget.NewCard("", "", container.NewBorder(
		nil,
		nil,
		container.NewVBox(widget.NewLabel(userLabel)),
		widget.NewButtonWithIcon("Выход", theme.LogoutIcon(), onLogout),
		container.NewBorder(nil, nil, accent, nil, container.NewPadded(titleBlock)),
	))
}

func newWorkspaceScreen(header, body fyne.CanvasObject) fyne.CanvasObject {
	background := canvas.NewRectangle(theme.BackgroundColor())
	scrolledBody := container.NewVScroll(container.NewPadded(body))
	return container.New(layout.NewMaxLayout(), background, container.NewBorder(container.NewPadded(header), nil, nil, nil, scrolledBody))
}
