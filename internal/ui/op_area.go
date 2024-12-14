package ui

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
)

func createFormatCheckboxes() []fyne.CanvasObject {
	// 资源类型复选框
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

func CreateOperationArea(w fyne.Window, tab *container.AppTabs, inputData binding.String, optionData binding.StringList) *fyne.Container {
	// Progress bar
	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel("当前无下载内容")

	// Resource type checkboxes
	formatLabel := widget.NewLabelWithStyle("资源类型: ", fyne.TextAlign(fyne.TextAlignLeading), fyne.TextStyle{Bold: true})
	checkboxes := createFormatCheckboxes()

	// Save path display and button
	defaultPath, _ := os.UserHomeDir()
	downloadPath := path.Join(defaultPath, "Downloads")
	pathLabel := widget.NewLabelWithStyle("保存目录: ", fyne.TextAlign(fyne.TextAlignLeading), fyne.TextStyle{Bold: true})
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("从“选择目录”中更新路径，输入无效，默认下载目录【Downloads】")
	// pathEntry.Disable()

	selectPathButton := widget.NewButtonWithIcon("选择目录", theme.FolderIcon(), func() {
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
	downloadButton := widget.NewButtonWithIcon("下载", theme.DownloadIcon(), nil)
	downloadButton.OnTapped = func() {
		downloadButton.Disable() // 下载进行中禁止再次点击
		random := true
		var urlList []string

		if pathEntry.Text == "" {
			downloadPath = path.Join(defaultPath, "Downloads")
		}
		slog.Info(fmt.Sprintf("downloadPath is %v", downloadPath))
		if downloadPath == "" {
			dialog.NewInformation("警告", "下载目录为空，请选择", w).Show()
			return
		}

		currentTab := tab.Selected().Text
		slog.Debug(fmt.Sprintf("current tab = %v", currentTab))
		if currentTab == dl.TAB_NAMES[0] {
			// 从 urlInput 获取输入内容
			urlContent, err := inputData.Get()
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			urlContent = strings.TrimSpace(urlContent)
			if urlContent == "" {
				dialog.NewInformation("警告", "请输入 URL，数据不能为空", w).Show()
				return
			}
			urlList = strings.Split(urlContent, "\n")

		} else {
			bookIdList, err := optionData.Get()
			slog.Debug(fmt.Sprintf("op: bookIdList = %v, err = %v", bookIdList, err))
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if len(bookIdList) == 0 {
				dialog.NewInformation("警告", "至少选择1个多选框", w).Show()
				return
			}
			// urlList =
			urlList = dl.GenerateURLFromID(bookIdList)
			slog.Debug(fmt.Sprintf("urlList count = %d\n%v", len(urlList), urlList))
		}
		filteredURLs := []string{}
		for _, link := range urlList {
			if dl.ValidURL(link) {
				filteredURLs = append(filteredURLs, link)
			}
		}
		if len(filteredURLs) == 0 {
			info := "请右侧下拉框中选择教材，再从左侧多选框选择课本"
			if currentTab == dl.TAB_NAMES[0] {
				info = "请在上方的输入框输入有效的 URL"
			}
			dialog.NewInformation("警告", info, w).Show()
			return
		}
		slog.Info(fmt.Sprintf("filteredURLs count = %d", len(filteredURLs)))

		// 遍历获取勾选状态
		var formatList []string
		for i, checkbox := range checkboxes {
			if checkbox.(*widget.Check).Checked {
				formatList = append(formatList, dl.FORMAT_LIST[i].Suffix)
			}
		}

		if len(formatList) == 0 {
			dialog.NewInformation("警告", "请勾选至少1个资源类型", w).Show()
			return
		}
		slog.Info(fmt.Sprintf("formatList count = %d", len(formatList)))
		slog.Debug(fmt.Sprintf("formatList =\n %v", formatList))

		resourceURLs := dl.ExtractResources(filteredURLs, formatList, random)
		progressLabel.SetText(fmt.Sprintf("共解析到%d个资源", len(resourceURLs)))

		if len(resourceURLs) == 0 {
			dialog.NewError(fmt.Errorf("未解析到有效资源"), w).Show()
			return
		}

		// 下载任务 更新进度条
		downloadManager := dl.NewDownloadManager(w, progressBar, progressLabel, downloadPath, resourceURLs)
		downloadManager.StartDownload(downloadButton)
	}

	separator := widget.NewSeparator()
	return container.NewVBox(
		separator,
		container.NewHBox(formatLabel, container.NewHBox(checkboxes...)),
		container.NewBorder(nil, nil, pathLabel, container.NewHBox(selectPathButton, downloadButton), pathEntry),
		progressBar, progressLabel,
	)
}
