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

func (dm *DownloadManager) StartDownload(downloadButton *widget.Button, headers map[string]string) {
	if err := os.MkdirAll(dm.downloadsDir, 0755); err != nil {
		dialog.ShowError(fmt.Errorf("下载目录创建失败：%v", err), dm.window)
		dm.statusLabel.SetText("创建下载目录失败")
		downloadButton.Enable()
		return
	}

	// 计算文件大小
	var totalSize int64
	for i := range dm.links {
		// resp, err := http.Head(dm.links[i].URL)
		// if err != nil {
		// 	dialog.ShowError(err, dm.window)
		// 	return
		// }
		// dm.links[i].Size = resp.ContentLength
		totalSize += dm.links[i].Size
	}
	slog.Debug(fmt.Sprintf("Total links = %d, total file size = %d", len(dm.links), totalSize))

	var downloadedBytes atomic.Int64
	var downloadedFiles atomic.Int64
	var wg sync.WaitGroup

	// Update progress in a separate goroutine
	done := make(chan bool)
	go func() {
		dm.statusLabel.SetText("下载中...")
		for {
			select {
			case <-done:
				return
			default:
				downloaded := float64(downloadedBytes.Load())
				downloadedFiles := int64(downloadedFiles.Load())
				progress := downloaded / float64(totalSize)
				dm.progressBar.SetValue(progress)
				dm.statusLabel.SetText(fmt.Sprintf("下载中... %d/%d 个文件", downloadedFiles, len(dm.links)))
			}
		}
	}()

	// Start downloads
	successCount := 0
	results := []string{"\nlog-time,success,file-size,save-path,raw-url,extra-url"}
	for _, file := range dm.links {
		wg.Add(1)
		go func(file LinkData) {
			isSuccess, outputPath := dm.downloadFile(&wg, file, &downloadedBytes, headers)
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
		dm.progressBar.SetValue(1.0)

		failedCount := len(dm.links) - successCount
		statsInfo := fmt.Sprintf("- 成功：%d\n- 失败：%d", successCount, failedCount)
		if successCount > 0 {
			statsInfo += fmt.Sprintf("\n(已保存至%v)", dm.downloadsDir)
		}
		dm.statusLabel.SetText(fmt.Sprintf("下载完成：成功/失败 = %d/%d", successCount, failedCount))
		dialog.NewInformation("完成", "文件下载完成\n"+statsInfo, dm.window).Show()

		now := time.Now().Format("2006-01-02 15:04:05 MST")
		more_result := fmt.Sprintf("\n---\n%s 下载统计：成功/失败 = %d/%d\n\n", now, successCount, failedCount)
		results = append(results, more_result)
		saveLogFile(dm.downloadsDir, results)

		downloadButton.Enable()
	}()
}

func saveLogFile(downloadsDir string, results []string) {
	filename := "log-smartedudl.txt"
	savePath := filepath.Join(downloadsDir, filename)
	content := strings.Join(results, "\n")

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

func (dm *DownloadManager) downloadFile(wg *sync.WaitGroup, file LinkData, downloadedBytes *atomic.Int64, headers map[string]string) (bool, string) {
	defer wg.Done()
	url := file.URL
	for _, v := range headers {
		if v != "" {
			url = file.RawURL
			break
		}
	}
	slog.Debug(fmt.Sprintf("URL = %s", url))

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

	outputPath := getSavePath(dm.downloadsDir, file.Title, file.Format)
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
