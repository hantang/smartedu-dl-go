package internal

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type CustomTheme struct {
	primaryColor color.Color
	defaultTheme fyne.Theme
}

func NewCustomTheme() *CustomTheme {
	return &CustomTheme{
		primaryColor: theme.DefaultTheme().Color(theme.ColorNamePrimary, theme.VariantLight),
		defaultTheme: theme.DefaultTheme(),
	}
}

func (t *CustomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return t.primaryColor
	}
	return t.defaultTheme.Color(name, variant)
}

func (t *CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.defaultTheme.Icon(name)
}

func (t *CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.defaultTheme.Font(style)
}

func (t *CustomTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.defaultTheme.Size(name)
}
