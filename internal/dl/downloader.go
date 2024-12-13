package dl

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
)

func getFilename(savePath string) string {
	dest := savePath
	destDir := path.Dir(dest)
	base := path.Base(dest)
	ext := path.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	if _, err := os.Stat(dest); err == nil {
		i := 1
		for {
			dest = path.Join(destDir, fmt.Sprintf("%s(%d)%s", stem, i, ext))
			if _, err := os.Stat(dest); err != nil {
				break
			}
			i++
		}
	}
	return dest
}

func DownloadFile(url, savePath string) error {
	slog.Debug("URL = " + url)
	dest := getFilename(savePath)
	slog.Info("Save file to " + dest)

	resp, err := http.Get(url)
	slog.Debug(fmt.Sprintf("status code %v", resp.StatusCode))
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download file: status code %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	slog.Debug("Downloading file...")
	_, err = io.Copy(out, resp.Body)
	slog.Debug("Download finished...")
	return err
}

func FetchJsonData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	slog.Debug(fmt.Sprintf("status code %v", resp.StatusCode))
	if err != nil {
		fmt.Println("Error fetching JSON data:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	return body, err
}
