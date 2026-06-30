package dl

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Eyevinn/hls-m3u8/m3u8"
)

func encryptMD5(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func GetResponseBody(url string, headers map[string]string) ([]byte, error) {
	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加请求头
	for key, value := range headers {
		req.Header.Add(key, value)
	}

	// 发送请求
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求状态异常: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	return body, nil
}

func getKeyFromURL(url, key string, headers map[string]string) (string, error) {
	body, err := GetResponseBody(url, headers)
	if err != nil {
		return "", err
	}
	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	// 获取特定键值
	if value, exists := data[key]; exists {
		return value, nil
	}
	return "", fmt.Errorf("key=%s not found", key)
}

func getDecryptionKey(keyURL, keyID string, headers map[string]string) ([]byte, error) {
	// ts视频解码部分参考
	// - https://github.com/52beijixing/smartedu-download/blob/main/utils/download.py
	// - https://basic.smartedu.cn/fish/video/videoplayer.min.js

	signURL := keyURL + "/signs"
	nonce, err := getKeyFromURL(signURL, "nonce", headers)
	if err != nil {
		return nil, err
	}
	slog.Debug("获取视频解密 nonce 成功")

	sign := encryptMD5(nonce + keyID)[:16]
	keyIDURL := fmt.Sprintf("%s?nonce=%s&sign=%s", keyURL, nonce, sign)
	keyData, err := getKeyFromURL(keyIDURL, "key", headers)
	if err != nil {
		return nil, err
	}
	slog.Debug("获取视频加密 key 数据成功")

	keyText, err := base64.StdEncoding.DecodeString(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key: %w", err)
	}

	decryptionKey, err := decryptAES_ECB(keyText, []byte(sign))
	return decryptionKey, err
}

// PKCS7Unpadding 去除PKCS7填充
func PKCS7Unpadding(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	length := len(data)
	unpadding := int(data[length-1])
	if unpadding <= 0 || unpadding > length {
		return data
	}
	return data[:(length - unpadding)]
}

// decryptAES_CBC AES-CBC模式解密
func decryptAES_CBC(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %v", err)
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("密文长度必须是块大小(%d)的倍数", aes.BlockSize)
	}

	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("IV长度必须是%d字节", aes.BlockSize)
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	// 去除填充
	plaintext = PKCS7Unpadding(plaintext)
	return plaintext, nil
}

// decryptAES_ECB AES-ECB模式解密
func decryptAES_ECB(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %v", err)
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("密文长度必须是块大小(%d)的倍数", aes.BlockSize)
	}

	plaintext := make([]byte, len(ciphertext))
	for bs, be := 0, block.BlockSize(); bs < len(ciphertext); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(plaintext[bs:be], ciphertext[bs:be])
	}

	// 去除填充
	plaintext = PKCS7Unpadding(plaintext)
	return plaintext, nil
}

func GetM3U8Size(m3u8URL string, headers map[string]string) (int64, error) {
	req, err := http.NewRequest("HEAD", m3u8URL, nil)
	if err != nil {
		return 0, fmt.Errorf("创建 M3U8 头信息请求失败: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("获取 M3U8 头信息失败: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	// 获取 M3U8 播放列表
	req, err = http.NewRequest("GET", m3u8URL, nil)
	if err != nil {
		return 0, fmt.Errorf("创建 M3U8 播放列表请求失败: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err = defaultHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("获取 M3U8 播放列表失败: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	playlist, listType, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return 0, fmt.Errorf("解析 M3U8 播放列表失败: %w", err)
	}

	// 检查是否是媒体播放列表
	if listType != m3u8.MEDIA {
		return 0, fmt.Errorf("不是媒体播放列表")
	}

	var totalSize int64
	mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
	baseURL := m3u8URL[:strings.LastIndex(m3u8URL, "/")+1]
	for _, segment := range mediaPlaylist.Segments {
		if segment == nil {
			continue
		}
		segmentURL := segment.URI
		if !strings.HasPrefix(segmentURL, "http") {
			segmentURL = baseURL + segmentURL
		}
		// 获取每个分段的大小
		req, err := http.NewRequest("HEAD", segmentURL, nil)
		if err != nil {
			slog.Warn(fmt.Sprintf("创建分段大小请求失败: %s", err))
			continue
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		segmentResp, err := defaultHTTPClient.Do(req)
		if err != nil {
			slog.Warn(fmt.Sprintf("获取分段大小失败: %s", err))
			continue
		}

		sizeStr := segmentResp.Header.Get("Content-Length")
		io.Copy(io.Discard, segmentResp.Body)
		segmentResp.Body.Close()
		if sizeStr != "" {
			size, err := strconv.ParseInt(sizeStr, 10, 64)
			if err == nil {
				totalSize += size
			}
		}
	}

	return totalSize, nil
}

func extractM3u8Info(mediaPlaylist m3u8.MediaPlaylist) (keyURL, keyID string, iv []byte, err error) {
	if len(mediaPlaylist.Keys) == 0 {
		return "", "", nil, nil // 没有加密(无EXT-X-KEY标签)
	}

	// 取第一把 key（METHOD=NONE 的情况下 URI 通常为空，可按需跳过）
	key := mediaPlaylist.Keys[0]
	for _, k := range mediaPlaylist.Keys {
		if k.Method != "" && k.Method != "NONE" {
			key = k
			break
		}
	}

	slog.Debug("检测到加密密钥配置", "keyLen", len(key.URI))

	keyURL = key.URI
	parts := strings.Split(keyURL, "/")
	keyID = parts[len(parts)-1]

	if key.IV != "" {
		// 去除0x 0X前缀
		hexStr := strings.TrimPrefix(key.IV, "0x")
		hexStr = strings.TrimPrefix(hexStr, "0X")
		iv, err = hex.DecodeString(hexStr)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to decode IV: %w", err)
		}
		slog.Debug("成功提取加密参数", "ivLen", len(iv))
	}
	return keyURL, keyID, iv, nil
}

// isNetworkError 判断是否为网络相关错误，这类错误适合重试
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "reset") ||
		strings.Contains(errStr, "refused") ||
		strings.Contains(errStr, "temporary failure")
}

// 下载单个TS文件，增加header信息，更新进度条
func downloadTSFile(segmentURL, filename string, headers map[string]string, downloadedBytes *atomic.Int64) error {
	// 创建HTTP请求
	segmentReq, err := http.NewRequest("GET", segmentURL, nil)
	if err != nil {
		return fmt.Errorf("request error: %w", err)
	}
	for k, v := range headers {
		segmentReq.Header.Set(k, v)
	}

	// 发送请求
	segmentResp, err := defaultHTTPClient.Do(segmentReq)
	if err != nil {
		return fmt.Errorf("failed to download segment (%s): %w", segmentURL, err)
	}
	defer segmentResp.Body.Close()
	if segmentResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download segment (%s): status code %d", segmentURL, segmentResp.StatusCode)
	}

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	n, err := io.Copy(outFile, segmentResp.Body)
	if n > 0 {
		downloadedBytes.Add(n)
	}
	return err
}

// downloadTSFileWithRetry 带重试机制的TS文件下载（指数退避）
func downloadTSFileWithRetry(segmentURL, filename string, headers map[string]string, downloadedBytes *atomic.Int64, retryStats *atomic.Int64) error {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := downloadTSFile(segmentURL, filename, headers, downloadedBytes)
		if err == nil {
			return nil
		}

		// 只重试网络错误，不重试HTTP 4xx/5xx
		if !isNetworkError(err) {
			return err
		}

		lastErr = err
		if attempt < maxRetries-1 {
			// 指数退避：1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			slog.Debug("download segment failed, retrying",
				"url", segmentURL[:min(30, len(segmentURL))],
				"attempt", attempt+1,
				"backoff", backoff.String())
			time.Sleep(backoff)
			retryStats.Add(1)
		}
	}

	return fmt.Errorf("download failed after %d retries: %w", maxRetries, lastErr)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getDecryptedData(tsFile string, key []byte, iv []byte) ([]byte, error) {
	// 读取加密的TS数据
	// encryptedData, err := io.ReadAll(segmentResp.Body)
	encryptedData, err := os.ReadFile(tsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read segment data: %w", err)
	}

	// 解密数据
	var decryptedData []byte
	if key != nil {
		decryptedData, err = decryptAES_CBC(encryptedData, key, iv)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt segment: %w", err)
		}
	} else {
		decryptedData = encryptedData // 未加密直接使用原始数据
	}
	return decryptedData, nil

}

type segmentJob struct {
	index int
	url   string
}

func downloadAllTSWithRetry(tempDir string, urls []string, headers map[string]string, maxConcurrency int, downloadedBytes *atomic.Int64, retryStats *atomic.Int64) error {
	var wg sync.WaitGroup
	downloadChan := make(chan segmentJob, len(urls))
	errChan := make(chan error, len(urls))
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	if maxConcurrency > len(urls) {
		maxConcurrency = len(urls)
	}

	if retryStats == nil {
		retryStats = &atomic.Int64{}
	}

	// 控制并发数
	for i := 0; i < maxConcurrency; i++ {
		go func() {
			for job := range downloadChan {
				filename := filepath.Join(tempDir, fmt.Sprintf("%05d.ts", job.index))
				err := downloadTSFileWithRetry(job.url, filename, headers, downloadedBytes, retryStats)
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
				}
				wg.Done()
			}
		}()
	}

	// 分发任务
	for index, url := range urls {
		wg.Add(1)
		downloadChan <- segmentJob{index: index, url: url}
	}
	close(downloadChan)

	// 等待下载完成
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 检查是否有错误
	if err := <-errChan; err != nil {
		return err
	}
	return nil
}

// 按顺序合并 ts 文件
func mergeTSFiles(tempDir, outFile string, urls []string, key []byte, iv []byte) error {
	// 合并前先检查文件
	for i := 0; i < len(urls); i++ {
		tsFile := filepath.Join(tempDir, fmt.Sprintf("%05d.ts", i))
		info, err := os.Stat(tsFile)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("segment %d not found: %s", i, tsFile)
			}
			return fmt.Errorf("segment %d stat error: %w", i, err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("segment %d is empty: %s", i, tsFile)
		}
	}

	outFileHandle, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer outFileHandle.Close()

	for i := 0; i < len(urls); i++ {
		tsFile := filepath.Join(tempDir, fmt.Sprintf("%05d.ts", i))
		tsData, err := getDecryptedData(tsFile, key, iv)
		if err != nil {
			return err
		}
		if _, err := outFileHandle.Write(tsData); err != nil {
			return err
		}
	}

	return nil
}

func getTempDirFromHash(input string, prefix string) (string, error) {
	// hash作为临时目录
	h := fnv.New32a()
	h.Write([]byte(input))
	hashValue := h.Sum32()
	hashStr := strconv.FormatUint(uint64(hashValue), 16)
	tempDir := filepath.Join(os.TempDir(), prefix, "video_"+hashStr)

	if err := os.RemoveAll(tempDir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return "", err
	}

	return tempDir, nil
}

// downloads a M3U8 video and save it to MP4 file
func DownloadM3U8(m3u8URL, savePath string, headers map[string]string, downloadedBytes *atomic.Int64, maxConcurrency int, retryStats *atomic.Int64) (int, error) {
	if retryStats == nil {
		retryStats = &atomic.Int64{}
	}
	statusCode := -1
	req, err := http.NewRequest("GET", m3u8URL, nil)
	if err != nil {
		return statusCode, fmt.Errorf("创建 GET 请求失败: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 发送 GET 请求
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return statusCode, fmt.Errorf("获取 M3U8 播放列表失败: %w", err)
	}
	slog.Debug(fmt.Sprintf("Fetch Video status code %v", resp.StatusCode))
	statusCode = resp.StatusCode
	defer resp.Body.Close()
	if statusCode != http.StatusOK {
		return statusCode, fmt.Errorf("获取 M3U8 播放列表状态异常: %d", statusCode)
	}

	playlist, listType, err := m3u8.DecodeFrom(resp.Body, true)
	_ = playlist.DecodeFrom(resp.Body, true)
	if err != nil {
		return statusCode, fmt.Errorf("解析 M3U8 播放列表失败: %w", err)
	}

	var segments []*m3u8.MediaSegment
	if listType != m3u8.MEDIA {
		return statusCode, fmt.Errorf("不是媒体播放列表")
	}

	mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
	segments = mediaPlaylist.Segments
	keyURL, keyID, iv, err := extractM3u8Info(*mediaPlaylist)
	if err != nil {
		return statusCode, err
	}

	var key []byte
	if keyURL != "" {
		key, err = getDecryptionKey(keyURL, keyID, headers)
		if err != nil {
			return statusCode, fmt.Errorf("获取视频解密 key 失败: %w", err)
		}
		slog.Debug("获取视频解密 key 成功")
	}

	// 并发下载，临时目录保存单个ts片段，之后再合并
	baseURL := m3u8URL[:strings.LastIndex(m3u8URL, "/")+1]
	segmentURLList := []string{}
	for _, segment := range segments {
		if segment == nil {
			continue
		}
		segmentURL := segment.URI
		if !strings.HasPrefix(segmentURL, "http") {
			segmentURL = baseURL + segmentURL
		}
		segmentURLList = append(segmentURLList, segmentURL)
	}

	tempDir, err := getTempDirFromHash(baseURL, APP_NAME)
	if err != nil {
		return statusCode, err
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			slog.Warn("failed to cleanup temp dir", "path", tempDir, "err", err)
		}
	}()
	slog.Debug(fmt.Sprintf("tempDir: %s\nmaxConcurrency: %d\nTS count: %d", tempDir, maxConcurrency, len(segmentURLList)))

	if err := downloadAllTSWithRetry(tempDir, segmentURLList, headers, maxConcurrency, downloadedBytes, retryStats); err != nil {
		return statusCode, err
	}
	if err := mergeTSFiles(tempDir, savePath, segmentURLList, key, iv); err != nil {
		return statusCode, err
	}
	return statusCode, nil
}
