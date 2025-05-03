package ui

import (
	"image/color"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
)

func InitUI(isLocal bool) {
	arrayLen := 5

	a := app.NewWithID(dl.APP_ID)
	customTheme := NewCustomTheme()
	a.Settings().SetTheme(customTheme)

	w := a.NewWindow(dl.APP_NAME)

	// Menu and title
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			picker := dialog.NewColorPicker("主题设置", "选择主题颜色", func(c color.Color) {
				customTheme.primaryColor = c
				a.Settings().SetTheme(customTheme)
			}, w)
			picker.Show()
		}),
		widget.NewToolbarAction(theme.InfoIcon(), func() {
			dialog.NewInformation("关于", dl.APP_DESC, w).Show()
		}),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			dialog.ShowInformation("帮助", "选择需要下载的资源，点击下载按钮即可", w)
		}),
	)

	// Tab container
	tabs := dl.TAB_NAMES
	linkItemMaps := make(map[string][]dl.LinkItem)
	for _, name := range tabs {
		linkItemMaps[name] = []dl.LinkItem{}
	}

	tabContainer := container.NewAppTabs(
		container.NewTabItemWithIcon(tabs[1], theme.ListIcon(), CreateMaterialOptionsTab(w, linkItemMaps, tabs[1], isLocal, arrayLen)),
		container.NewTabItemWithIcon(tabs[2], theme.MediaVideoIcon(), CreateClassroomOptionsTab(w, linkItemMaps, tabs[2], isLocal, 7)),
		container.NewTabItemWithIcon(tabs[0], theme.ContentPasteIcon(), CreateInputTab(w, linkItemMaps, tabs[0], false, 0)),
	)

	// Bottom operation area
	operationArea := CreateOperationArea(w, tabContainer, linkItemMaps)

	content := container.NewBorder(toolbar, operationArea, nil, nil, tabContainer)
	w.SetContent(content)
	// w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}
