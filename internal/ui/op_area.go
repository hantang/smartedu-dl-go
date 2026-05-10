package ui

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
	"github.com/hantang/smartedudlgo/internal/util"
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

func createAudioCheckboxes() []fyne.CanvasObject {
	// 资源类型复选框
	var checkboxes []fyne.CanvasObject
	for _, format := range dl.FORMAT_LIST {
		checkbox := widget.NewCheck(format.Name, func(checked bool) {
		})
		if strings.Contains(format.Name, "音频") {
			checkbox.SetChecked(format.Suffix == "mp3")
		} else {
			checkbox.Disable()
		}
		checkboxes = append(checkboxes, checkbox)
	}
	return checkboxes
}

func extractDownloadInfo(w fyne.Window, pathEntry *widget.Entry, defaultPath string, ignores string) string {
	downloadPath := pathEntry.Text
	if downloadPath == "" {
		downloadPath = path.Join(defaultPath, "Downloads")
	} else if strings.HasPrefix(downloadPath, ignores) {
		downloadPath = downloadPath[len(ignores):]
	}
	slog.Info(fmt.Sprintf("downloadPath is %v", downloadPath))
	// if downloadPath == "" {
	// 	dialog.NewInformation("警告", "下载目录为空，请选择", w).Show()
	// }
	return downloadPath
}

func initHeaders(token string) map[string]string {
	headers := map[string]string{}
	authInfo := util.FulfillToken(token)
	if authInfo != "" {
		headers["x-nd-auth"] = authInfo
	}
	slog.Debug(fmt.Sprintf("headers is %v", headers))
	return headers
}

func extractDownloadLinks(w fyne.Window, tab *container.AppTabs, linkItemMaps map[string][]dl.LinkItem) []string {
	// random := true
	filteredURLs := []string{}
	currentTab := tab.Selected().Text
	slog.Debug(fmt.Sprintf("current tab = %v", currentTab))
	slog.Debug(fmt.Sprintf("linkItemMaps = %s", linkItemMaps))

	linkItems, ok := linkItemMaps[currentTab]
	if !ok {
		return filteredURLs
	}
	slog.Debug(fmt.Sprintf("linkItems = %d", len(linkItems)))

	if len(linkItems) == 0 {
		if currentTab == dl.TAB_NAMES[0] {
			dialog.NewInformation("警告", "请输入 URL，数据不能为空", w).Show()
		} else {
			dialog.NewInformation("警告", "至少选择1份教材/课程包", w).Show()
		}

		return filteredURLs
	}
	var urlList []string
	if currentTab == dl.TAB_NAMES[3] {
		urlList = dl.GenerateURLFromID2(linkItems)
		return urlList
	} else {
		urlList = dl.GenerateURLFromID(linkItems)
	}
	slog.Debug(fmt.Sprintf("urlList = %d, %s", len(urlList), urlList))

	for _, link := range urlList {
		if dl.ValidURL(link) {
			filteredURLs = append(filteredURLs, link)
		}
	}
	if len(filteredURLs) == 0 {
		info := "请从右侧下拉框中选择教材，再从左侧多选框选择课本"
		if currentTab == dl.TAB_NAMES[0] {
			info = "请在上方的输入框输入有效的 URL"
		}
		dialog.NewInformation("警告", info, w).Show()
		return filteredURLs
	}
	return filteredURLs
}

func expandAIEducationListForVideoSelection(links []string) ([]string, bool) {
	expanded := make([]string, 0, len(links))
	hasListPage := false
	for _, link := range links {
		parsedURL, err := url.Parse(link)
		if err == nil && parsedURL.Path == dl.AIEducationListPath {
			hasListPage = true
			query := parsedURL.Query()
			if query.Get("all") == "" && query.Get("downloadAll") == "" {
				query.Set("all", "1")
				parsedURL.RawQuery = query.Encode()
				link = parsedURL.String()
			}
		}
		expanded = append(expanded, link)
	}
	return expanded, hasListPage
}

func formatVideoSize(size int64) string {
	if size <= 0 {
		return ""
	}
	const mb = 1024 * 1024
	if size < mb {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/mb)
}

func showVideoSelectionDialog(w fyne.Window, resources []dl.LinkData, pageSize int, startDownload func([]dl.LinkData), onCancel func()) {
	selected := make([]bool, len(resources))
	checks := make([]*widget.Check, len(resources))
	checkContainer := container.NewVBox()
	countLabel := widget.NewLabel("")

	updateCount := func() {
		count := 0
		for _, checked := range selected {
			if checked {
				count++
			}
		}
		countLabel.SetText(fmt.Sprintf("已选择 %d/%d 个视频", count, len(resources)))
	}

	for i, resource := range resources {
		index := i
		selected[index] = true
		prefix := fmt.Sprintf("%03d.", i+1)
		if pageSize > 0 {
			prefix = fmt.Sprintf("第%d页 %03d.", i/pageSize+1, i+1)
		}
		label := fmt.Sprintf("%s %s", prefix, resource.Title)
		if sizeText := formatVideoSize(resource.Size); sizeText != "" {
			label = fmt.Sprintf("%s (%s)", label, sizeText)
		}
		check := widget.NewCheck(label, func(checked bool) {
			selected[index] = checked
			updateCount()
		})
		check.SetChecked(true)
		checks[index] = check
		checkContainer.Add(check)
	}

	selectAllButton := widget.NewButtonWithIcon("全选", theme.ConfirmIcon(), func() {
		for i, check := range checks {
			selected[i] = true
			check.SetChecked(true)
		}
		updateCount()
	})
	clearAllButton := widget.NewButtonWithIcon("全不选", theme.CancelIcon(), func() {
		for i, check := range checks {
			selected[i] = false
			check.SetChecked(false)
		}
		updateCount()
	})

	scroll := container.NewVScroll(checkContainer)
	scroll.SetMinSize(fyne.NewSize(720, 420))
	contentItems := []fyne.CanvasObject{
		widget.NewLabel(fmt.Sprintf("解析到 %d 个视频，请勾选要下载的内容。", len(resources))),
	}
	if pageSize > 0 && len(resources) > pageSize {
		contentItems = append(contentItems, widget.NewLabel(fmt.Sprintf("网页每页通常是 %d 个视频；第 2 页一般对应 %03d-%03d。", pageSize, pageSize+1, pageSize*2)))
	}
	contentItems = append(contentItems, container.NewHBox(selectAllButton, clearAllButton, countLabel), scroll)
	content := container.NewVBox(contentItems...)
	updateCount()

	dialog.NewCustomConfirm("选择要下载的视频", "开始下载", "取消", content, func(ok bool) {
		if !ok {
			onCancel()
			return
		}

		selectedResources := make([]dl.LinkData, 0, len(resources))
		for i, checked := range selected {
			if checked {
				selectedResources = append(selectedResources, resources[i])
			}
		}
		if len(selectedResources) == 0 {
			dialog.NewInformation("提示", "请至少选择 1 个视频", w).Show()
			onCancel()
			return
		}
		startDownload(selectedResources)
	}, w).Show()
}

func CreateOperationArea(w fyne.Window, tab *container.AppTabs, linkItemMaps map[string][]dl.LinkItem, maxConcurrency int) *fyne.Container {
	random := true
	// Progress bar
	progressBar := widget.NewProgressBar()
	progressLabel := widget.NewLabel("当前无下载内容")

	// Download buttons
	downloadButton := widget.NewButtonWithIcon("下载已选择资源", theme.DownloadIcon(), nil)
	downloadVideoButton := widget.NewButtonWithIcon("仅下载视频", theme.FileVideoIcon(), nil)

	// Resource type checkboxes
	formatLabel := widget.NewLabelWithStyle("🔖 资源类型: ", fyne.TextAlign(fyne.TextAlignLeading), fyne.TextStyle{Bold: true})
	formatContainer := container.NewHBox()
	updateCheckboxes := func() {
		formatContainer.Objects = nil
		var checkboxes []fyne.CanvasObject
		if tab.Selected() != nil && tab.Selected().Text == dl.TAB_NAMES[3] {
			checkboxes = createAudioCheckboxes()
			downloadVideoButton.Disable()
		} else {
			checkboxes = createFormatCheckboxes()
			downloadVideoButton.Enable()
		}
		formatContainer.Objects = checkboxes
		formatContainer.Refresh()
	}
	updateCheckboxes()

	// 根据tab更新资源列表
	originalOnSelected := tab.OnSelected
	tab.OnSelected = func(tab *container.TabItem) {
		if originalOnSelected != nil {
			originalOnSelected(tab)
		}
		updateCheckboxes()
	}

	// backup links
	backupCheckbox := widget.NewCheck("备用解析", func(checked bool) {})
	logCheckbox := widget.NewCheck("记录日志", func(checked bool) {})

	// user log info
	loginLabel := widget.NewLabelWithStyle("🍪 登录信息: ", fyne.TextAlign(fyne.TextAlignLeading), fyne.TextStyle{Bold: true})
	loginEntry := NewTokenEntry()

	// 预读取token
	token, err := util.GetToken()
	if err == nil {
		slog.Info("配置登录信息成功")
		loginEntry.SetText(token)
	} else {
		loginEntry.SetPlaceHolder("请在浏览器登录账号后，填写X-Nd-Auth值或者Access Token")
	}

	// Save path display and button
	defaultPath, _ := os.UserHomeDir()
	// downloadPath := path.Join(defaultPath, "Downloads")
	pathLabel := widget.NewLabelWithStyle("🗂️ 保存目录: ", fyne.TextAlign(fyne.TextAlignLeading), fyne.TextStyle{Bold: true})
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("从“选择目录”中更新路径，输入无效，默认【用户下载目录】")
	// pathEntry.Disable()
	pathComment := "更新为："

	selectPathButton := widget.NewButtonWithIcon("选择目录", theme.FolderIcon(), func() {
		dialog.NewFolderOpen(func(dir fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if dir == nil {
				return
			}
			// downloadPath = dir.Path()
			pathEntry.SetText(pathComment + dir.Path())
		}, w).Show()
	})

	downloadButton.OnTapped = func() {
		isParse := tab.Selected().Text != dl.TAB_NAMES[3]
		filteredURLs := extractDownloadLinks(w, tab, linkItemMaps)
		slog.Info(fmt.Sprintf("filteredURLs count = %d", len(filteredURLs)))
		slog.Debug(fmt.Sprintf("filteredURLs list = %s", filteredURLs))

		if len(filteredURLs) == 0 {
			return
		}
		downloadPath := extractDownloadInfo(w, pathEntry, defaultPath, pathComment)
		headers := initHeaders(loginEntry.Text)

		// 下载进行中禁止再次点击
		downloadButton.Disable()
		downloadVideoButton.Disable()

		// 遍历获取勾选状态
		var formatList []string
		checkboxes := formatContainer.Objects
		for i, checkbox := range checkboxes {
			if checkbox.(*widget.Check).Checked {
				formatList = append(formatList, dl.FORMAT_LIST[i].Suffix)
			}
		}

		if len(formatList) == 0 {
			dialog.NewInformation("警告", "请勾选至少1个资源类型", w).Show()
			downloadButton.Enable()
			downloadVideoButton.Enable()
			return
		}
		slog.Info(fmt.Sprintf("formatList count = %d", len(formatList)))
		slog.Debug(fmt.Sprintf("formatList =\n %v", formatList))

		resourceURLs := dl.ExtractResources(filteredURLs, formatList, random, backupCheckbox.Checked, isParse)
		resourceStats := make(map[string]int)
		formatDict := make(map[string]string)
		for _, item := range dl.FORMAT_LIST {
			formatDict[item.Suffix] = item.Name
		}

		// 遍历统计每个文件类型个数
		for _, item := range resourceURLs {
			resourceStats[item.Format]++
		}
		var resultStrBuilder strings.Builder
		for key, value := range resourceStats {
			resultStrBuilder.WriteString(fmt.Sprintf("%s=%d ", formatDict[key], value))
		}
		resultStr := resultStrBuilder.String()
		infoStr := fmt.Sprintf("共解析到%d个资源：%s", len(resourceURLs), resultStr)
		progressLabel.SetText(infoStr)
		slog.Info(infoStr)

		if len(resourceURLs) == 0 {
			dialog.NewError(fmt.Errorf("未解析到有效资源"), w).Show()
			downloadButton.Enable()
			downloadVideoButton.Enable()
			return
		}

		// 下载任务 更新进度条
		downloadManager := dl.NewDownloadManager(w, progressBar, progressLabel, downloadPath, resourceURLs)
		downloadManager.StartDownload(downloadButton, downloadVideoButton, headers, logCheckbox.Checked, false, maxConcurrency)
	}

	downloadVideoButton.OnTapped = func() {
		filteredURLs := extractDownloadLinks(w, tab, linkItemMaps)
		slog.Info(fmt.Sprintf("filteredURLs count = %d", len(filteredURLs)))
		if len(filteredURLs) == 0 {
			return
		}
		downloadPath := extractDownloadInfo(w, pathEntry, defaultPath, pathComment)
		headers := initHeaders(loginEntry.Text)
		useBackup := backupCheckbox.Checked
		enableLog := logCheckbox.Checked

		// 下载进行中禁止再次点击
		downloadButton.Disable()
		downloadVideoButton.Disable()
		progressLabel.SetText("正在解析视频资源...")

		videoLinks, hasAIEducationListPage := expandAIEducationListForVideoSelection(filteredURLs)
		selectionPageSize := 0
		if hasAIEducationListPage {
			selectionPageSize = dl.AIEducationDefaultPageSize
		}
		go func() {
			formatList := dl.FORMAT_VIDEO
			resourceURLs := dl.ExtractResources(videoLinks, formatList, random, useBackup, true)
			fyne.Do(func() {
				if len(resourceURLs) == 0 {
					dialog.NewError(fmt.Errorf("未解析到有效资源"), w).Show()
					downloadButton.Enable()
					downloadVideoButton.Enable()
					progressLabel.SetText("当前无下载内容")
					return
				}

				startDownload := func(selectedResources []dl.LinkData) {
					downloadManager := dl.NewDownloadManager(w, progressBar, progressLabel, downloadPath, selectedResources)
					downloadManager.StartDownload(downloadButton, downloadVideoButton, headers, enableLog, true, maxConcurrency)
				}
				cancelDownload := func() {
					downloadButton.Enable()
					downloadVideoButton.Enable()
					progressLabel.SetText("已取消视频下载")
				}

				if len(resourceURLs) == 1 {
					startDownload(resourceURLs)
					return
				}

				progressLabel.SetText(fmt.Sprintf("解析到 %d 个视频，请选择要下载的内容", len(resourceURLs)))
				showVideoSelectionDialog(w, resourceURLs, selectionPageSize, startDownload, cancelDownload)
			})
		}()
	}

	downloadPart := container.NewCenter(
		container.New(layout.NewCustomPaddedHBoxLayout(20), downloadButton, downloadVideoButton),
	)
	return container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(),
		container.NewBorder(nil, nil, nil, logCheckbox, downloadPart),
		container.NewPadded(),
		container.NewHBox(formatLabel, formatContainer),
		container.NewBorder(nil, nil, pathLabel, container.NewHBox(selectPathButton), pathEntry),
		container.NewBorder(nil, nil, loginLabel, backupCheckbox, loginEntry),
		container.NewPadded(),
		progressBar,
		progressLabel,
	)
}
