package ui

import (
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/hantang/smartedudlgo/internal/dl"
)

func CreateInputTab(w fyne.Window, linkItemMaps map[string][]dl.LinkItem, name string, isLocal bool, arrayLen int) *fyne.Container {
	// Multi-line text input for URL
	urlInput := widget.NewMultiLineEntry()
	urlInput.SetPlaceHolder("输入 smartedu.cn 资源链接")

	// Update the input to linkItemMaps[name]
	urlInput.OnChanged = func(text string) {
		lines := strings.Split(text, "\n")
		linkItemMaps[name] = []dl.LinkItem{} // 清空现有数据
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			linkItem := dl.LinkItem{
				Link: line,
				Type: dl.InputInfo.Type,
			}
			linkItemMaps[name] = append(linkItemMaps[name], linkItem)
		}
		slog.Debug(fmt.Sprintf("text = %s, lines = %d, options = %d", text, len(lines), len(linkItemMaps[name])))
	}

	// Clear button
	clearButton := widget.NewButtonWithIcon("清空", theme.DeleteIcon(), func() {
		urlInput.SetText("")
		linkItemMaps[name] = []dl.LinkItem{}
	})

	// Description text
	info := fmt.Sprintf(
		"支持的URL格式示例：\n• 📚 教材URL: %s\n• 📹 课程URL: %s\n• 🎥 精品课程: %s\n• 🌐 资源链接: %s\n\n📝 可直接从浏览器地址栏复制URL。",
		fmt.Sprintf(dl.TchMaterialInfo.Detail, "{contentId}"),
		fmt.Sprintf(dl.SyncClassroomInfo.Detail, "{activityId}"),
		fmt.Sprintf(dl.EliteSyncClassroomInfo.Detail, "{courseId}"),
		fmt.Sprintf("完整的PDF、m3u8等URL（%s）", dl.RESOURCES_PATH),
	)
	bottom := container.NewVBox(
		container.NewCenter(clearButton),
		container.NewPadded(),
		container.NewHBox(container.NewPadded(), widget.NewLabel(info)),
	)
	return container.NewBorder(nil, bottom, nil, nil, urlInput)
}
