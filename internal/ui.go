package internal

import (
	// "fmt"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func createOptionsTab(inputData binding.String) *fyne.Container {
	// TODO

	inputData.Set("")

	// left side: checkbox list
	dataList := []string{
		"Option 1",
		"Option 2",
		"Option 3",
		"Option 4",
	}
	var checkboxes []fyne.CanvasObject
	for _, value := range dataList {
		checkbox := widget.NewCheck(value, func(checked bool) {
			// 处理复选框状态变化的逻辑
		})
		checkboxes = append(checkboxes, checkbox)
	}

	selectButton := widget.NewButton("全选", func() {
		// Handle query action
	})
	deselectButton := widget.NewButton("清空", func() {
		// Handle query action
	})

	// Right side: Dropdown and query button
	dropdown := widget.NewSelect([]string{"Option 1", "Option 2"}, func(selected string) {
		// Handle selection
	})
	queryButton := widget.NewButton("查询", func() {
		// Handle query action
	})

	left := container.NewBorder(
		nil, container.NewCenter(container.NewHBox(selectButton, deselectButton)), nil, nil,
		container.NewVBox(checkboxes...),
	)

	right := container.NewBorder(
		nil, container.NewCenter(queryButton), nil, nil,
		container.NewVBox(dropdown),
	)
	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}

func createInputTab(inputData binding.String) *fyne.Container {
	// Multi-line text input for URL
	urlInput := widget.NewMultiLineEntry()
	urlInput.SetPlaceHolder("输入smart.edu资源链接")

	// Bind the input to inputData
	urlInput.OnChanged = func(text string) {
		if err := inputData.Set(text); err != nil {
			slog.Error("Failed to update inputData", "error", err)
		}
	}

	// Clear button
	clearButton := widget.NewButton("清空", func() {
		urlInput.SetText("")
		if err := inputData.Set(""); err != nil {
			slog.Error("清空失败", "error", err)
		}
	})

	// Description text
	info := "支持的URL格式示例：" +
		"\n- 教材URL: https://basic.smartedu.cn/tchMaterial/detail?contentType=assets_document&contentId={contentId}" +
		"\n- 课件URL: https://basic.smartedu.cn/syncClassroom/classActivity?activityId={activityId}" +
		"\n\n可以直接从浏览器地址复制URL。"

	// Create label
	description := widget.NewLabel(info)
	return container.NewBorder(nil, container.NewVBox(clearButton, description), nil, nil, urlInput)
}

func createFormatCheckboxes() []fyne.CanvasObject {
	var checkboxes []fyne.CanvasObject

	for _, format := range FORMAT_LIST {
		checkbox := widget.NewCheck(format.name, func(checked bool) {
			// 处理复选框状态变化的逻辑
		})

		if !format.status {
			checkbox.Disable()
		} else {
			checkbox.SetChecked(format.check)
		}
		checkboxes = append(checkboxes, checkbox)
	}
	return checkboxes
}

func createOperationArea(w fyne.Window, inputData binding.String) *fyne.Container {
	// Progress bar
	progressBar := widget.NewProgressBar()
	var wg sync.WaitGroup

	// Resource type checkboxes
	formatLabel := widget.NewLabel("资源类型: ")
	checkboxes := createFormatCheckboxes()

	// Save path display and button
	defaultPath, _ := os.UserHomeDir()
	downloadPath := path.Join(defaultPath, "Downloads")
	pathLabel := widget.NewLabel("保存目录: ")
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("从“选择目录”中更新路径，输入无效，默认下载目录【Downloads】")

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
			if ValidURL(link) {
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
				formatList = append(formatList, FORMAT_LIST[i].suffix)
			}
		}

		if len(formatList) == 0 {
			dialog.NewInformation("警告", "请勾选至少1个资源类型", w).Show()
			return
		}
		slog.Info(fmt.Sprintf("formatList is %v", len(formatList)))

		progressBar.SetValue(0)
		// 启动下载过程
		wg.Add(1)
		go func() {
			defer wg.Done()

			resourceURLs := ExtractResources(filteredURLs, formatList, false)
			slog.Info(fmt.Sprintf("resourceURLs is %v", len(resourceURLs)))
			baseValue := 10.0
			progressBar.SetValue(baseValue)

			outputDir := downloadPath
			if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
				slog.Warn(fmt.Sprintf("创建目录失败: %v", err))
				dialog.NewInformation("错误", "下载失败: \n"+err.Error(), w).Show()
			}

			var success, failed int
			for i, link := range resourceURLs {
				filename := fmt.Sprintf("%s.%s", link.title, link.format)
				filePath := path.Join(outputDir, filename)
				err := DownloadFile(link.url, filePath)
				if err != nil {
					failed++
				} else {
					success++
				}
				progressBar.SetValue(baseValue + float64(i)/float64(len(resourceURLs))*(100-baseValue))
			}

			statsInfo := fmt.Sprintf("- 成功: %d\n- 失败: %d", success, failed)
			if success > 0 {
				statsInfo += fmt.Sprintf("\n(已保存至%v)", downloadPath)
			}
			dialog.NewInformation("完成", "文件下载完成\n"+statsInfo, w).Show()
		}()
	})

	separator := widget.NewSeparator()
	return container.NewVBox(
		separator,
		container.NewHBox(formatLabel, container.NewHBox(checkboxes...)),
		container.NewBorder(nil, nil, pathLabel, container.NewHBox(selectPathButton, downloadButton), pathEntry),
		progressBar,
	)
}

func InitUI() {
	// a := app.New()
	a := app.NewWithID("io.github.hantang.smartedudl")

	// TODO 设置应用图标
	w := a.NewWindow(APP_NAME)

	// Menu and title
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
		}),
		widget.NewToolbarAction(theme.InfoIcon(), func() {
			dialog.NewInformation("关于", APP_DESC, w).Show()

		}),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			dialog.ShowInformation("帮助", "选择需要下载的资源，点击下载按钮即可", w)
		}),
	)

	// Tab container
	inputData := binding.NewString()
	tabContainer := container.NewAppTabs(
		container.NewTabItem("输入链接", createInputTab(inputData)),
		container.NewTabItem("教材列表", createOptionsTab(inputData)),
	)

	// Bottom operation area
	operationArea := createOperationArea(w, inputData)

	content := container.NewBorder(toolbar, operationArea, nil, nil, tabContainer)
	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}