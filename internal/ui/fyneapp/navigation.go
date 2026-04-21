package fyneapp

import "fyne.io/fyne/v2"

type Navigator struct {
	window fyne.Window
}

func NewNavigator(window fyne.Window) *Navigator {
	return &Navigator{window: window}
}

func (n *Navigator) Show(content fyne.CanvasObject) {
	n.window.SetContent(content)
}
