package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
)

func createFormatCheckboxes() []fyne.CanvasObject {
	var checkboxes []fyne.CanvasObject

	for _, format := range dl.FORMAT_LIST {
		checkbox := widget.NewCheck(format.Name, func(checked bool) {
			// 处理复选框状态变化的逻辑
		})

		if !format.Status {
			checkbox.Disable()
		} else {
			checkbox.SetChecked(format.Check)
		}
		checkboxes = append(checkboxes, checkbox)
	}
	return checkboxes
}

func createOperationArea(w fyne.Window, inputData binding.String) *fyne.Container {
	// Progress bar
	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel("当前无下载内容")

	// Resource type checkboxes
	formatLabel := widget.NewLabel("资源类型: ")
	checkboxes := createFormatCheckboxes()

	// Save path display and button
	defaultPath, _ := os.UserHomeDir()
	downloadPath := path.Join(defaultPath, "Downloads")
	pathLabel := widget.NewLabel("保存目录: ")
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("从“选择目录”中更新路径，输入无效，默认下载目录【Downloads】")
	// pathEntry.Disable()

	selectPathButton := widget.NewButton("选择目录", func() {
		dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if dir == nil {
				return
			}
			downloadPath = dir.Path()
			pathEntry.SetText("更新为：" + downloadPath)
			// pathLabel.SetText("保存目录: " + downloadPath)
		}, w).Show()
	})

	// Download button
	downloadButton := widget.NewButton("下载", func() {
		if pathEntry.Text == "" {
			downloadPath = path.Join(defaultPath, "Downloads")
		}
		slog.Info(fmt.Sprintf("downloadPath is %v", downloadPath))
		if downloadPath == "" {
			dialog.NewInformation("警告", "下载目录为空，请选择", w).Show()
			return
		}
		// 从 urlInput 获取输入内容
		urlContent, err := inputData.Get()
		if err != nil {
			dialog.NewInformation("警告", "获取失败", w).Show()
			return
		}
		urlContent = strings.TrimSpace(urlContent)
		if urlContent == "" {
			dialog.NewInformation("警告", "请输入 URL，数据不能为空", w).Show()
			return
		}
		urlList := strings.Split(urlContent, "\n")
		filteredURLs := []string{}
		for _, link := range urlList {
			if dl.ValidURL(link) {
				filteredURLs = append(filteredURLs, link)
			}
		}
		if len(filteredURLs) == 0 {
			dialog.NewInformation("警告", "请输入有效的 URL", w).Show()
			return
		}
		slog.Info(fmt.Sprintf("filteredURLs is %v", len(filteredURLs)))

		// 遍历获取勾选状态
		var formatList []string
		for i, checkbox := range checkboxes {
			if checkbox.(*widget.Check).Checked {
				// slog.Debug("format", FORMAT_LIST[i].suffix, checkbox.(*widget.Check).Text)
				formatList = append(formatList, dl.FORMAT_LIST[i].Suffix)
			}
		}

		if len(formatList) == 0 {
			dialog.NewInformation("警告", "请勾选至少1个资源类型", w).Show()
			return
		}
		slog.Info(fmt.Sprintf("formatList is %v", len(formatList)))
		slog.Debug(fmt.Sprintf("formatList =\n %v", formatList))

		resourceURLs := dl.ExtractResources(filteredURLs, formatList, true)
		progressLabel.SetText(fmt.Sprintf("共解析到%d个资源", len(resourceURLs)))

		if len(resourceURLs) == 0 {
			dialog.NewError(fmt.Errorf("未解析到有效资源"), w).Show()
			return
		}

		// 下载任务 更新进度条
		downloadManager := dl.NewDownloadManager(w, progressBar, progressLabel, downloadPath, resourceURLs)
		downloadManager.StartDownload()
	})

	separator := widget.NewSeparator()
	return container.NewVBox(
		separator,
		container.NewHBox(formatLabel, container.NewHBox(checkboxes...)),
		container.NewBorder(nil, nil, pathLabel, container.NewHBox(selectPathButton, downloadButton), pathEntry),
		progressBar, progressLabel,
	)
}

func InitUI() {
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
	inputData := binding.NewString()
	tabContainer := container.NewAppTabs(
		container.NewTabItem("输入链接", CreateInputTab(inputData)),
		container.NewTabItem("教材列表", CreateOptionsTab(inputData)),
	)

	// Bottom operation area
	operationArea := createOperationArea(w, inputData)

	content := container.NewBorder(toolbar, operationArea, nil, nil, tabContainer)
	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}
