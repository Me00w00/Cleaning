package fyneapp

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type appTheme struct{}

func newTheme() fyne.Theme {
	return &appTheme{}
}

func (t *appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 241, G: 248, B: 247, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 102, G: 198, B: 212, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 44, G: 156, B: 181, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 26, G: 53, B: 63, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 252, G: 254, B: 254, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 114, G: 141, B: 148, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 174, G: 197, B: 201, A: 255}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 196, G: 222, B: 226, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 223, G: 242, B: 245, A: 255}
	case theme.ColorNameFocus:
		return color.NRGBA{R: 125, G: 209, B: 217, A: 255}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 144, G: 198, B: 207, A: 180}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *appTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 14
	case theme.SizeNameInlineIcon:
		return 22
	case theme.SizeNameScrollBar:
		return 12
	case theme.SizeNameInputRadius:
		return 10
	case theme.SizeNameSelectionRadius:
		return 10
	default:
		return theme.DefaultTheme().Size(name)
	}
}
