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
	"github.com/hantang/smartedudlgo/internal/dl"
)

func CreateInputTab(w fyne.Window, inputData binding.String) *fyne.Container {
	// Multi-line text input for URL
	urlInput := widget.NewMultiLineEntry()
	urlInput.SetPlaceHolder("输入 smartedu.cn 资源链接")

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
	info := fmt.Sprintf(
		"支持的URL格式示例：\n• 教材URL: %s\n• 课程URL: %s\n\n可以直接从浏览器地址复制URL。",
		fmt.Sprintf(dl.TchMaterialInfo.Detail, "{contentId}"),
		fmt.Sprintf(dl.SyncClassroomInfo.Detail, "{activityId}"),
	)
	bottom := container.NewVBox(
		container.NewCenter(clearButton),
		container.NewPadded(),
		container.NewHBox(container.NewPadded(), widget.NewLabel(info)),
	)
	return container.NewBorder(nil, bottom, nil, nil, urlInput)
}
