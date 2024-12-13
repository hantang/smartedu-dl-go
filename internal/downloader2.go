package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

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

func (dm *DownloadManager) startDownload() {
	if err := os.MkdirAll(dm.downloadsDir, 0755); err != nil {
		dialog.ShowError(fmt.Errorf("下载目录创建失败: %v", err), dm.window)
		dm.statusLabel.SetText("创建下载目录失败")
		return
	}

	// 计算文件大小
	var totalSize int64
	for i := range dm.links {
		resp, err := http.Head(dm.links[i].url)
		if err != nil {
			dialog.ShowError(err, dm.window)
			return
		}
		dm.links[i].size = resp.ContentLength
		totalSize += resp.ContentLength
	}

	var downloadedBytes atomic.Int64
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
				progress := downloaded / float64(totalSize)
				dm.progressBar.SetValue(progress)
			}
		}
	}()

	// Start downloads
	successCount := 0
	for _, file := range dm.links {
		wg.Add(1)
		go func(file LinkData) {
			success := dm.downloadFile(&wg, file, &downloadedBytes)
			if success {
				successCount++
			}
		}(file)
	}

	// Wait for completion in a goroutine
	go func() {
		wg.Wait()
		done <- true
		dm.progressBar.SetValue(1.0)

		failedCount := len(dm.links) - successCount
		statsInfo := fmt.Sprintf("- 成功: %d\n- 失败: %d", successCount, failedCount)
		if successCount > 0 {
			statsInfo += fmt.Sprintf("\n(已保存至%v)", dm.downloadsDir)
		}
		dm.statusLabel.SetText(fmt.Sprintf("下载完成: 成功/失败 = %d/%d", successCount, len(dm.links)))
		dialog.NewInformation("完成", "文件下载完成\n"+statsInfo, dm.window).Show()
	}()
}

func (dm *DownloadManager) downloadFile(wg *sync.WaitGroup, file LinkData, downloadedBytes *atomic.Int64) bool {
	defer wg.Done()

	resp, err := http.Get(file.url)
	if err != nil {
		fmt.Printf("下载 %s 出错: %v\n", file.title, err)
		return false
	}
	defer resp.Body.Close()

	filename := fmt.Sprintf("%s.%s", file.title, file.format)
	outputPath := filepath.Join(dm.downloadsDir, filename)
	out, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("创建文件 %s 出错: %v\n", outputPath, err)
		return false
	}
	defer out.Close()

	buffer := make([]byte, 32*1024) // 32KB大小
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// Write to file
			if _, err := out.Write(buffer[:n]); err != nil {
				fmt.Printf("写入文件 %s 出错: %v\n", file.title, err)
				return false
			}
			downloadedBytes.Add(int64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("读取 %s 出错: %v\n", file.title, err)
			return false
		}
	}
	return true
}
