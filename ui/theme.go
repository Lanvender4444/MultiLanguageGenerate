package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type customTheme struct {
	fyne.Theme
}

var _ fyne.Theme = (*customTheme)(nil)

func NewCustomTheme() fyne.Theme {
	return &customTheme{Theme: theme.DefaultTheme()}
}

func (c *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}