package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func CreateInputTab(w fyne.Window, inputData binding.String) *fyne.Container {
	// Multi-line text input for URL
	urlInput := widget.NewMultiLineEntry()
	urlInput.SetPlaceHolder("输入 smart.edu 资源链接")

	// Bind the input to inputData
	urlInput.OnChanged = func(text string) {
		if err := inputData.Set(text); err != nil {
			slog.Error("Failed to update inputData", "error", err)
		}
	}

	// Clear button
	clearButton := widget.NewButtonWithIcon("清空", theme.DeleteIcon(), func() {
		urlInput.SetText("")
		if err := inputData.Set(""); err != nil {
			slog.Error("清空失败", "error", err)
			dialog.ShowError(err, w)
		}
	})

	// Description text
	
	info := "支持的URL格式示例：" +
		"\n- 教材URL: https://basic.smartedu.cn/tchMaterial/detail?contentType=assets_document&contentId={contentId}" +
		"\n- 课件URL: https://basic.smartedu.cn/syncClassroom/classActivity?activityId={activityId}" +
		"\n\n可以直接从浏览器地址复制URL。"

	// Create label
	bottom := container.NewVBox(container.NewCenter(clearButton),
		widget.NewSeparator(),
		container.NewHBox(widget.NewLabel(""), widget.NewLabel(info)))
	return container.NewBorder(nil, bottom, nil, nil, urlInput)
}
