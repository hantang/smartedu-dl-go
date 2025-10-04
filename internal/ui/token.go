package ui

import (
	"log/slog"
	"strings"

	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/util"
)

type TokenEntry struct {
	widget.Entry
}

func NewTokenEntry() *TokenEntry {
	// 支持回车或失去焦点时自动保存token
	e := &TokenEntry{}
	e.ExtendBaseWidget(e)
	// 回车提交时保存
	e.Entry.OnSubmitted = func(text string) {
		e.saveToken()
	}

	return e
}

func (e *TokenEntry) saveToken() {
	text := strings.TrimSpace(e.Text)
	if text != "" {
		authInfo := util.ExtractToken(text)
		if err := util.SaveToken(authInfo); err != nil {
			slog.Error("保存登录信息失败", "error", err)
		} else {
			slog.Debug("已保存登录信息")
		}
	} else {
		if err := util.DeleteToken(); err != nil {
			slog.Error("删除登录信息失败", "error", err)
		} else {
			slog.Debug("已删除登录信息")
		}
	}
}

// 失去焦点时保存
func (e *TokenEntry) FocusLost() {
	e.saveToken()
	e.Entry.FocusLost() // 保持默认行为
}
