package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
)

func InitUI(isLocal bool) {
	a := app.New()

	customTheme := NewCustomTheme()
	a.Settings().SetTheme(customTheme)

	metadata := a.Metadata()
	w := a.NewWindow(metadata.Name)
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
			dialog.NewInformation("关于", fmt.Sprintf("%s\n当前版本：%s", dl.APP_DESC, metadata.Version), w).Show()
		}),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			dialog.ShowInformation("帮助",
				"选择➀先选择教材请课程或输入链接，➁然后需要下载的资源类型、修改下载目录，\n"+
					"➂最后点击下载按钮即可；➃若下载视频请用“仅下载视频”按钮。\n\n"+
					"如果出现下载失败等问题，请配置登录信息（X-Nd-Auth值或者Access Token）。\n"+
					"若使用“备用下载”，请注意可能下载得到非最新版本。", w)
		}),
	)

	// Tab container
	tabs := dl.TAB_NAMES
	linkItemMaps := make(map[string][]dl.LinkItem)
	for _, name := range tabs {
		linkItemMaps[name] = []dl.LinkItem{}
	}

	tabContainer := container.NewAppTabs(
		container.NewTabItemWithIcon(tabs[1], theme.ListIcon(), CreateMaterialOptionsTab(w, linkItemMaps, tabs[1], isLocal, 5)),
		container.NewTabItemWithIcon(tabs[2], theme.MediaVideoIcon(), CreateClassroomOptionsTab(w, linkItemMaps, tabs[2], isLocal, 6)),
		container.NewTabItemWithIcon(tabs[0], theme.ContentPasteIcon(), CreateInputTab(w, linkItemMaps, tabs[0], false, 0)),
	)

	// Bottom operation area
	operationArea := CreateOperationArea(w, tabContainer, linkItemMaps)

	content := container.NewBorder(toolbar, operationArea, nil, nil, tabContainer)
	w.SetContent(content)
	// w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}
