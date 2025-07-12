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
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/grafov/m3u8"
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
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

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
		fmt.Printf("解析JSON失败: %v\n", err)
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
	fmt.Printf("nonce = %s\n", nonce)

	sign := encryptMD5(nonce + keyID)[:16]
	keyIDURL := fmt.Sprintf("%s?nonce=%s&sign=%s", keyURL, nonce, sign)
	keyData, err := getKeyFromURL(keyIDURL, "key", headers)
	if err != nil {
		return nil, err
	}
	fmt.Printf("keyData = %s\n", keyData)

	keyText, err := base64.StdEncoding.DecodeString(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key: %w", err)
	}
	fmt.Printf("keyText = %v\n", keyText)

	decryptionKey, err := decryptAES_ECB(keyText, []byte(sign))
	fmt.Printf("decryptionKey = %s\n", decryptionKey)
	return decryptionKey, err
}

// PKCS7Unpadding 去除PKCS7填充
func PKCS7Unpadding(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
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
	resp, err := http.Head(m3u8URL)
	if err != nil {
		return 0, fmt.Errorf("获取 M3U8 头信息失败: %w", err)
	}
	defer resp.Body.Close()

	// 获取 M3U8 播放列表
	resp, err = http.Get(m3u8URL)
	if err != nil {
		return 0, fmt.Errorf("获取 M3U8 播放列表失败: %w", err)
	}
	defer resp.Body.Close()

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
	for _, segment := range mediaPlaylist.Segments {
		// 获取每个分段的大小
		segmentResp, err := http.Head(segment.URI)
		if err != nil {
			slog.Warn(fmt.Sprintf("获取分段大小失败: %s", err))
			continue
		}
		defer segmentResp.Body.Close()

		sizeStr := segmentResp.Header.Get("Content-Length")
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
	if mediaPlaylist.Key == nil {
		return "", "", nil, nil // fmt.Errorf("没有加密(无EXT-X-KEY标签)")
	}
	slog.Debug(fmt.Sprintf("加密方法(METHOD): %s\n", mediaPlaylist.Key.Method))
	slog.Debug(fmt.Sprintf("密钥URI: %s\n", mediaPlaylist.Key.URI))
	slog.Debug(fmt.Sprintf("IV值: %s\n", mediaPlaylist.Key.IV))

	keyURL = mediaPlaylist.Key.URI
	parts := strings.Split(keyURL, "/")
	keyID = parts[len(parts)-1]
	slog.Debug(fmt.Sprintf("keyID: %s", keyID))

	if mediaPlaylist.Key.IV != "" {
		// 去除0x前缀
		hexStr := strings.TrimPrefix(mediaPlaylist.Key.IV, "0x")
		iv, err = hex.DecodeString(hexStr)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to decode IV: %w", err)
		}
	}
	slog.Debug(fmt.Sprintf("iv: %s", iv))
	return keyURL, keyID, iv, nil
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
	client := &http.Client{}
	segmentResp, err := client.Do(segmentReq)
	if err != nil {
		return fmt.Errorf("failed to download segment (%s): %w", segmentURL, err)
	}
	defer segmentResp.Body.Close()

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, segmentResp.Body)
	if segmentResp.ContentLength > 0 {
		downloadedBytes.Add(int64(segmentResp.ContentLength))
	}
	return err
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

// 获取 url 在切片中的索引
func getIndex(urls []string, target string) int {
	for i, u := range urls {
		if u == target {
			return i
		}
	}
	return -1
}

// 并发下载 ts 文件并按顺序写入输出文件
func downloadAllTS(tempDir string, urls []string, headers map[string]string, maxConcurrency int, downloadedBytes *atomic.Int64) error {
	var wg sync.WaitGroup
	downloadChan := make(chan string, len(urls))
	errChan := make(chan error, 1)

	// 控制并发数
	for i := 0; i < maxConcurrency; i++ {
		go func() {
			for url := range downloadChan {
				filename := filepath.Join(tempDir, fmt.Sprintf("%05d.ts", getIndex(urls, url)))
				err := downloadTSFile(url, filename, headers, downloadedBytes)
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
	for _, url := range urls {
		wg.Add(1)
		downloadChan <- url
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
func DownloadM3U8(m3u8URL, savePath string, headers map[string]string, downloadedBytes *atomic.Int64, maxConcurrency int) error {
	req, err := http.NewRequest("GET", m3u8URL, nil)
	if err != nil {
		return fmt.Errorf("创建 GET 请求失败: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 发送 GET 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("获取 M3U8 播放列表失败: %w", err)
	}
	defer resp.Body.Close()

	playlist, listType, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return fmt.Errorf("解析 M3U8 播放列表失败: %w", err)
	}

	var segments []*m3u8.MediaSegment
	if listType != m3u8.MEDIA {
		return fmt.Errorf("不是媒体播放列表")
	}

	mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
	segments = mediaPlaylist.Segments
	keyURL, keyID, iv, err := extractM3u8Info(*mediaPlaylist)
	if err != nil {
		return err
	}

	// 允许key为空，不加密
	key, err := getDecryptionKey(keyURL, keyID, headers)
	if err != nil {
		slog.Debug("解密key为空\n")
	} else {
		slog.Debug(fmt.Sprintf("解密key: %s\n", key))
	}

	saveFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件(%s)失败: %w", savePath, err)
	}
	defer saveFile.Close()

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
		return err
	}
	slog.Debug(fmt.Sprintf("tempDir: %s\nmaxConcurrency: %d\nTS count: %d", tempDir, maxConcurrency, len(segmentURLList)))

	downloadAllTS(tempDir, segmentURLList, headers, maxConcurrency, downloadedBytes)
	mergeTSFiles(tempDir, savePath, segmentURLList, key, iv)
	return nil
}
