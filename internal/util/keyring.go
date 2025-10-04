package util

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	service  = "smartedu-dl"
	username = "smartuser"
	envKey   = "SMARTEDU_TOKEN"
)

// GetToken 获取 Token：优先环境变量，其次系统 Keyring
func GetToken() (string, error) {
	// 1. 优先环境变量
	slog.Debug("尝试读取token")
	if token := os.Getenv(envKey); token != "" {
		slog.Debug(fmt.Sprintf("读取环境变量 token is %v", token))
		return token, nil
	}

	// 2. Keyring
	token, err := keyring.Get(service, username)
	if err != nil {
		slog.Debug("token读取失败")
		return "", errors.New("未找到token")
	}
	slog.Debug(fmt.Sprintf("读取 token is %v", token))
	return token, nil
}

// SaveToken 保存 Token 到系统 Keyring
func SaveToken(token string) error {
	return keyring.Set(service, username, token)
}

// DeleteToken 从系统 Keyring 删除 Token
func DeleteToken() error {
	return keyring.Delete(service, username)
}

func ExtractToken(authInfo string) string {
	re := regexp.MustCompile(`(?i)MAC\s+id="([^"]+)",nonce="0",mac="0"`)
	m := re.FindStringSubmatch(authInfo)
	if len(m) < 2 {
		return authInfo
	}
	return m[1]
}

func FulfillToken(token string) string {
	// 拼接 access token 得到完整的 x-nd-auth
	if !strings.HasPrefix(token, "MAC id") {
		token = fmt.Sprintf(`MAC id="%s",nonce="0",mac="0"`, token)
	}
	return token
}
