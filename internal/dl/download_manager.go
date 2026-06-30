package dl

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// DownloadStats 线程安全的下载统计
type DownloadStats struct {
	downloadedBytes atomic.Int64
	downloadedFiles atomic.Int64
	successCount    atomic.Int64
	retryCount      atomic.Int64
	statsMu         sync.RWMutex
}

// GetProgress 原子地获取进度信息，避免竞态条件
func (ds *DownloadStats) GetProgress(totalSize int64, totalFiles int) (progress float64, filesCount int64) {
	ds.statsMu.RLock()
	defer ds.statsMu.RUnlock()

	downloaded := float64(ds.downloadedBytes.Load())
	filesCount = ds.downloadedFiles.Load()
	progress = float64(filesCount) / float64(totalFiles)
	if totalSize > 0 {
		progress = downloaded / float64(totalSize)
	}
	return
}

// DownloadManager 处理下载逻辑
type DownloadManager struct {
	window       fyne.Window
	progressBar  *widget.ProgressBar
	statusLabel  *widget.Label
	downloadsDir string
	links        []LinkData
	savePathMu   sync.Mutex
}

func NewDownloadManager(window fyne.Window, progressBar *widget.ProgressBar, statusLabel *widget.Label, downloadsDir string, links []LinkData) *DownloadManager {
	return &DownloadManager{
		window:       window,
		progressBar:  progressBar,
		statusLabel:  statusLabel,
		downloadsDir: downloadsDir,
		links:        links,
	}
}

func (dm *DownloadManager) StartDownload(downloadButton *widget.Button, downloadVideoButton *widget.Button, headers map[string]string, enableLog bool, isVideo bool, maxConcurrency int) {
	if err := os.MkdirAll(dm.downloadsDir, 0755); err != nil {
		dialog.ShowError(fmt.Errorf("下载目录创建失败：%v", err), dm.window)
		dm.statusLabel.SetText("创建下载目录失败")
		downloadButton.Enable()
		downloadVideoButton.Enable()
		return
	}

	// 计算文件大小
	var totalSize int64
	for i := range dm.links {
		if dm.links[i].Size > 0 {
			totalSize += dm.links[i].Size
		}
	}
	slog.Debug(fmt.Sprintf("Total links = %d, total file size = %d", len(dm.links), totalSize))

	stats := &DownloadStats{}
	var tokenInvalid atomic.Bool
	var wg sync.WaitGroup
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	if maxConcurrency > len(dm.links) {
		maxConcurrency = len(dm.links)
	}

	// 初始化：禁用下载按钮
	downloadButton.Disable()
	downloadVideoButton.Disable()
	dm.statusLabel.SetText("正在准备下载...")
	dm.progressBar.SetValue(0)

	// Update progress in a separate goroutine
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				progress, numFiles := stats.GetProgress(totalSize, len(dm.links))
				retries := stats.retryCount.Load()

				fyne.DoAndWait(func() {
					dm.progressBar.SetValue(progress)
					statusText := fmt.Sprintf("下载中... %d/%d 个文件", numFiles, len(dm.links))
					if retries > 0 {
						statusText += fmt.Sprintf(" (重试: %d次)", retries)
					}
					dm.statusLabel.SetText(statusText)
				})
			}
		}
	}()

	// Start downloads
	resultCh := make(chan string, len(dm.links))
	jobs := make(chan LinkData)
	for range maxConcurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				isSuccess, statusCode, outputPath := false, 0, ""
				if isVideo {
					isSuccess, statusCode, outputPath = dm.downloadVideoFile(file, &stats.downloadedBytes, headers, maxConcurrency, &stats.retryCount)
				} else {
					isSuccess, statusCode, outputPath = dm.downloadFile(file, &stats.downloadedBytes, headers)
				}
				stats.downloadedFiles.Add(1)
				if isSuccess {
					stats.successCount.Add(1)
				}
				if statusCode == http.StatusUnauthorized { // token 失效
					tokenInvalid.Store(true)
				}

				// TODO 更好的日志格式，目前是csv
				now := time.Now().Format("2006-01-02 15:04:05 MST")
				resultCh <- fmt.Sprintf("%s,%v,%d,%s,%s,%s", now, isSuccess, file.Size, outputPath, file.RawURL, file.BackupURL)
			}
		}()
	}
	go func() {
		for _, file := range dm.links {
			jobs <- file
		}
		close(jobs)
	}()

	// Wait for completion in a goroutine
	go func() {
		wg.Wait()
		close(done)
		close(resultCh)
		results := []string{"\nlog-time,success,file-size,save-path,raw-url,extra-url"}
		for result := range resultCh {
			results = append(results, result)
		}

		// Update progress bar on main thread
		fyne.DoAndWait(func() {
			dm.progressBar.SetValue(1.0)
		})

		successes := int(stats.successCount.Load())
		failedCount := len(dm.links) - successes
		retries := stats.retryCount.Load()
		statsInfo := fmt.Sprintf("- 成功：%d\n- 失败：%d", successes, failedCount)
		if retries > 0 {
			statsInfo += fmt.Sprintf("\n- 重试：%d次", retries)
		}
		if successes > 0 {
			statsInfo += fmt.Sprintf("\n(已保存至%v)", dm.downloadsDir)
		}

		if enableLog {
			now := time.Now().Format("2006-01-02 15:04:05 MST")
			more := []string{
				"",
				"===============================================================",
				fmt.Sprintf("## %s 下载统计：成功/失败 = %d/%d", now, successes, failedCount),
				"---------------------------------------------------------------",
				"**详细信息：**",
			}
			results = append(more, results...)
			saveLogFile(dm.downloadsDir, results)
		}

		fyne.DoAndWait(func() {
			dm.statusLabel.SetText(fmt.Sprintf("下载完成：成功/失败 = %d/%d", successes, failedCount))
			if !tokenInvalid.Load() && successes > 0 {
				dialog.NewInformation("结果", "文件下载完成：\n"+statsInfo, dm.window).Show()
			} else {
				dialog.ShowError(fmt.Errorf("⚠️  【登录信息】可能错误或者失效\n\n文件下载结果：\n%s", statsInfo), dm.window)
			}
			downloadButton.Enable()
			downloadVideoButton.Enable()
		})
	}()
}

func saveLogFile(downloadsDir string, results []string) {
	filename := LOG_FILE
	savePath := filepath.Join(downloadsDir, filename)
	content := strings.Join(append(results, ""), "\n")
	slog.Info(fmt.Sprintf("Save log to %s", savePath))

	file, err := os.OpenFile(savePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("Error opening file: %v", err))
		return
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		slog.Error(fmt.Sprintf("Error writing file: %v", err))
		return
	}
}

func (dm *DownloadManager) reserveSavePath(
	folder string,
	stem string,
	suffix string,
	autoRename bool,
) (string, *os.File, error) {

	dm.savePathMu.Lock()
	defer dm.savePathMu.Unlock()

	// 修正后缀 m3u8 -> ts
	if suffix == "m3u8" {
		suffix = "ts"
	}

	// 去除文件中特殊字符
	if folder != "" {
		folder = sanitizeFilename(folder)
	}
	stem = sanitizeFilename(stem)

	index := 0
	for {
		name := stem

		if index > 0 {
			if suffix != "" {
				name = fmt.Sprintf("%s (%d).%s", stem, index, suffix)
			} else {
				name = fmt.Sprintf("%s (%d)", stem, index)
			}
		} else {
			if suffix != "" {
				name = fmt.Sprintf("%s.%s", stem, suffix)
			}
		}

		// 构建路径（folder 可选）
		parts := []string{dm.downloadsDir}

		if folder != "" {
			parts = append(parts, folder)
		}

		parts = append(parts, name)
		outputPath := filepath.Join(parts...)

		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			// 目录创建失败
			return "", nil, err
		}
		file, err := os.OpenFile(
			outputPath,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			0644,
		)

		if err == nil {
			return outputPath, file, nil
		}

		// 文件已存在
		if os.IsExist(err) {
			if !autoRename {
				return "", nil, fmt.Errorf("file exists: %s", outputPath)
			}
			index++
			continue
		}

		// 其他错误
		return "", nil, err
	}
}
func sanitizeWindowsFilename(name string) string {
	// 替换所有 Windows 非法字符
	// 删除路径遍历尝试
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "~", "")

	name = strings.Map(func(r rune) rune {
		switch {
		case r < 32: // 忽略控制字符，如\n, \r, \t等
			return ' '
		case strings.ContainsRune(`<>:"/\|?*`, r): // Windows 非法字符
			return '_'
		default:
			return r
		}
	}, name)
	name = strings.TrimRight(name, " .") // 避免以空格或 . 结尾

	// Windows 保留名称
	base := strings.TrimSuffix(name, filepath.Ext(name))
	switch strings.ToUpper(base) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}
	return name
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)

	// 替换所有 Windows 非法字符
	name = sanitizeWindowsFilename(name)

	// 合并连续空白字符（空格、Tab、换行等）
	name = strings.Join(strings.Fields(name), " ")

	// 默认值
	if name == "" {
		name = "Untitled"
	}

	// 限制长度
	const maxRunes = 255
	runes := []rune(name)
	if len(runes) > maxRunes {
		name = strings.TrimRight(string(runes[:maxRunes]), " .")
	}

	return name
}

func (dm *DownloadManager) downloadFile(file LinkData, downloadedBytes *atomic.Int64, headers map[string]string) (bool, int, string) {
	url := file.BackupURL
	for _, v := range headers {
		if v != "" {
			url = file.RawURL
			break
		}
	}
	slog.Debug(fmt.Sprintf("Title = %s, URL = %s", file.Title, url))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Warn(fmt.Sprintf("创建下载请求 %s 出错: %v", file.Title, err))
		return false, -1, ""
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := defaultHTTPClient.Do(req)
	// resp, err := http.Get(file.URL)
	if err != nil {
		slog.Warn(fmt.Sprintf("下载 %s 出错: %v", file.Title, err))
		return false, -1, ""
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	if statusCode != 200 {
		slog.Warn(fmt.Sprintf("下载 %s 状态异常: %v", file.Title, resp.StatusCode))
		return false, statusCode, ""
	}
	outputPath, reservedFile, err := dm.reserveSavePath(file.Folder, file.Title, file.Format, true)
	if err != nil {
		slog.Warn(fmt.Sprintf("创建文件 %s 出错：%v\n", outputPath, err))
		return false, statusCode, outputPath
	}
	defer reservedFile.Close()

	isSuccess := true
	buffer := make([]byte, 32*1024) // 32KB大小
	for {
		n, err := resp.Body.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Warn(fmt.Sprintf("读取 %s 出错：%v\n", file.Title, err))
			isSuccess = false
			break

		}
		if n > 0 {
			// Write to file
			if _, err := reservedFile.Write(buffer[:n]); err != nil {
				slog.Warn(fmt.Sprintf("写入文件 %s 出错：%v\n", file.Title, err))
				isSuccess = false
				break
			}
			downloadedBytes.Add(int64(n))
		}
	}
	return isSuccess, statusCode, outputPath
}

func (dm *DownloadManager) downloadVideoFile(
	file LinkData,
	downloadedBytes *atomic.Int64,
	headers map[string]string,
	maxConcurrency int,
	retryStats *atomic.Int64,
) (bool, int, string) {
	url := file.BackupURL
	for _, v := range headers {
		if v != "" {
			url = file.RawURL
			break
		}
	}

	slog.Debug(fmt.Sprintf("URL = %s", url))
	outputPath, reservedFile, err := dm.reserveSavePath(file.Folder, file.Title, file.Format, true)
	if err != nil {
		slog.Warn(fmt.Sprintf("创建文件 %s 出错：%v\n", outputPath, err))
		return false, -1, outputPath
	}
	if err := reservedFile.Close(); err != nil {
		return false, -1, outputPath
	}

	statusCode, err := DownloadM3U8(url, outputPath, headers, downloadedBytes, maxConcurrency, retryStats)
	isSuccess := true
	if err != nil || statusCode != 200 {
		slog.Warn(fmt.Sprintf("下载出错 %v", err))
		isSuccess = false
		if removeErr := os.Remove(outputPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.Warn(fmt.Sprintf("删除失败视频文件 %s 出错：%v", outputPath, removeErr))
		}
	}
	return isSuccess, statusCode, outputPath
}
