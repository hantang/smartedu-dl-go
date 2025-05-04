package dl

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		return "", "", nil, fmt.Errorf("没有加密(无EXT-X-KEY标签)")
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

// 解密TS数据并写入MP4文件
func processSegment(segmentURL string, headers map[string]string, key []byte, iv []byte, saveFile *os.File, downloadedBytes *atomic.Int64) error {
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

	// 读取加密的TS数据
	encryptedData, err := io.ReadAll(segmentResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read segment data: %w", err)
	}

	// 解密数据
	var decryptedData []byte
	if key != nil {
		decryptedData, err = decryptAES_CBC(encryptedData, key, iv)
		if err != nil {
			return fmt.Errorf("failed to decrypt segment: %w", err)
		}
	} else {
		decryptedData = encryptedData // 未加密直接使用原始数据
	}

	// 写入MP4文件
	n, err := saveFile.Write(decryptedData)
	if err != nil {
		return fmt.Errorf("failed to write segment to file: %w", err)
	}

	// 更新下载字节数
	downloadedBytes.Add(int64(n))
	return nil
}

// downloads a M3U8 video and save it to MP4 file
func DownloadM3U8(m3u8URL, savePath string, headers map[string]string, downloadedBytes *atomic.Int64) error {
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

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
	key, err := getDecryptionKey(keyURL, keyID, headers)
	if err != nil {
		return err
	}
	slog.Debug(fmt.Sprintf("解密key: %s\n", key))

	saveFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer saveFile.Close()

	// TODO 并发
	baseURL := m3u8URL[:strings.LastIndex(m3u8URL, "/")+1]
	for i, segment := range segments {
		if segment == nil {
			continue
		}

		segmentURL := segment.URI
		if !strings.HasPrefix(segmentURL, "http") {
			segmentURL = baseURL + segmentURL
		}
		err := processSegment(segmentURL, headers, key, iv, saveFile, downloadedBytes)
		if err != nil {
			return fmt.Errorf("下载失败 %d (%s): %w", i, segmentURL, err)
		}
	}
	return nil
}
