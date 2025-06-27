package dl

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// DownloadManager 处理下载逻辑
type DownloadManager struct {
	window       fyne.Window
	progressBar  *widget.ProgressBar
	statusLabel  *widget.Label
	downloadsDir string
	links        []LinkData
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

func (dm *DownloadManager) StartDownload(downloadButton *widget.Button, downloadVideoButton *widget.Button, headers map[string]string, enableLog bool, isVideo bool) {
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
		totalSize += dm.links[i].Size
	}
	slog.Debug(fmt.Sprintf("Total links = %d, total file size = %d", len(dm.links), totalSize))

	var downloadedBytes atomic.Int64
	var downloadedFiles atomic.Int64
	var wg sync.WaitGroup

	// 初始化：禁用下载按钮
	downloadButton.Disable()
	downloadVideoButton.Disable()
	dm.statusLabel.SetText("正在准备下载...")
	dm.progressBar.SetValue(0)

	// Update progress in a separate goroutine
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				downloaded := float64(downloadedBytes.Load())
				downloadedFiles := int64(downloadedFiles.Load())
				progress := downloaded / float64(totalSize)
				fyne.DoAndWait(func() {
					dm.progressBar.SetValue(progress)
					dm.statusLabel.SetText(fmt.Sprintf("下载中... %d/%d 个文件", downloadedFiles, len(dm.links)))
				})
			}
		}
	}()

	// Start downloads
	successCount := 0
	results := []string{"\nlog-time,success,file-size,save-path,raw-url,extra-url"}
	for _, file := range dm.links {
		wg.Add(1)
		go func(file LinkData) {
			// TODO
			var isSuccess bool
			var outputPath string
			if isVideo {
				isSuccess, outputPath = dm.downloadVideoFile(&wg, file, &downloadedBytes, headers)
			} else {
				isSuccess, outputPath = dm.downloadFile(&wg, file, &downloadedBytes, headers)
			}
			downloadedFiles.Add(int64(1))
			if isSuccess {
				successCount++
			}
			now := time.Now().Format("2006-01-02 15:04:05 MST")
			result := fmt.Sprintf("%s,%v,%d,%s,%s,%s", now, isSuccess, file.Size, outputPath, file.RawURL, file.URL)
			results = append(results, result)
		}(file)
	}

	// Wait for completion in a goroutine
	go func() {
		wg.Wait()
		done <- true

		// Update progress bar on main thread
		fyne.DoAndWait(func() {
			dm.progressBar.SetValue(1.0)
		})

		failedCount := len(dm.links) - successCount
		statsInfo := fmt.Sprintf("- 成功：%d\n- 失败：%d", successCount, failedCount)
		if successCount > 0 {
			statsInfo += fmt.Sprintf("\n(已保存至%v)", dm.downloadsDir)
		}

		if enableLog {
			now := time.Now().Format("2006-01-02 15:04:05 MST")
			more := []string{
				"",
				"===============================================================",
				fmt.Sprintf("## %s 下载统计：成功/失败 = %d/%d", now, successCount, failedCount),
				"---------------------------------------------------------------",
				"**详细信息：**",
			}
			results = append(more, results...)
			saveLogFile(dm.downloadsDir, results)
		}

		fyne.DoAndWait(func() {
			dm.statusLabel.SetText(fmt.Sprintf("下载完成：成功/失败 = %d/%d", successCount, failedCount))
			dialog.NewInformation("完成", "文件下载完成\n"+statsInfo, dm.window).Show()

			downloadButton.Enable()
			downloadVideoButton.Enable()
		})
	}()
}

func saveLogFile(downloadsDir string, results []string) {
	filename := "log-smartedudl.txt"
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

func getSavePath(downloadsDir string, title string, suffix string) string {
	index := 0
	if suffix == "m3u8" { // TODO
		suffix = "ts"
	}
	filename := fmt.Sprintf("%s.%s", title, suffix)
	for {
		outputPath := filepath.Join(downloadsDir, filename)
		if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
			return outputPath
		}
		index += 1
		filename = fmt.Sprintf("%s (%d).%s", title, index, suffix)
	}
}

func sanitizeFilename(s string) string {
	// 删除非法字符
	invalidChars := regexp.MustCompile(`[/\\:*?"<>|]`)
	s = invalidChars.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	// 控制最大长度（比如 255 字符）
	maxLength := 255
	if len(s) > maxLength {
		s = strings.TrimSpace(s[:maxLength])
	}
	// 默认值
	if s == "" {
		return "Untitled"
	}
	return s
}

func (dm *DownloadManager) downloadFile(wg *sync.WaitGroup, file LinkData, downloadedBytes *atomic.Int64, headers map[string]string) (bool, string) {
	defer wg.Done()
	url := file.URL
	for _, v := range headers {
		if v != "" {
			url = file.RawURL
			break
		}
	}
	slog.Debug(fmt.Sprintf("Title = %s, URL = %s", file.Title, url))

	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := (&http.Client{}).Do(req)
	// resp, err := http.Get(file.URL)
	if err != nil || resp.StatusCode != 200 {
		slog.Warn(fmt.Sprintf("下载 %s 状态：%v 出错：%v", file.Title, resp.StatusCode, err))
		return false, ""
	}
	defer resp.Body.Close()

	// 去除标题中特殊字符
	filename := sanitizeFilename(file.Title)
	outputPath := getSavePath(dm.downloadsDir, filename, file.Format)
	out, err := os.Create(outputPath)
	if err != nil {
		slog.Warn(fmt.Sprintf("创建文件 %s 出错：%v\n", outputPath, err))
		return false, outputPath
	}
	defer out.Close()

	buffer := make([]byte, 32*1024) // 32KB大小
	for {
		n, err := resp.Body.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Warn(fmt.Sprintf("读取 %s 出错：%v\n", file.Title, err))
			return false, outputPath
		}
		if n > 0 {
			// Write to file
			if _, err := out.Write(buffer[:n]); err != nil {
				slog.Warn(fmt.Sprintf("写入文件 %s 出错：%v\n", file.Title, err))
				return false, outputPath
			}
			downloadedBytes.Add(int64(n))
		}
	}
	return true, outputPath
}

func (dm *DownloadManager) downloadVideoFile(wg *sync.WaitGroup, file LinkData, downloadedBytes *atomic.Int64, headers map[string]string) (bool, string) {
	defer wg.Done()
	url := file.URL
	for _, v := range headers {
		if v != "" {
			url = file.RawURL
			break
		}
	}

	slog.Debug(fmt.Sprintf("URL = %s", url))
	filename := sanitizeFilename(file.Title)
	outputPath := getSavePath(dm.downloadsDir, filename, file.Format)

	err := DownloadM3U8(url, outputPath, headers, downloadedBytes)
	if err != nil {
		slog.Warn(fmt.Sprintf("下载出错 %v", err))
		return false, outputPath
	}
	return true, outputPath
}
