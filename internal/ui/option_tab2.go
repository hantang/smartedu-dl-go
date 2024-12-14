package ui

import (
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
)

func cleanData(left *fyne.Container, comboboxContainer *fyne.Container, index int, optionData binding.StringList) {
	// Clear checkboxes
	left.Objects = nil
	left.Refresh()

	// Clear data
	optionData.Set(nil)

	if index <= 0 {
		comboboxContainer.Objects = nil
		comboboxContainer.Refresh()

	} else {
		if index <= len(comboboxContainer.Objects)-1 {
			comboboxContainer.Objects = comboboxContainer.Objects[:index]
			comboboxContainer.Refresh()
		}
	}

}

func createComboboxes(left *fyne.Container, comboboxContainer *fyne.Container, index int, optionData binding.StringList, docPDFMap map[string]dl.DocPDFData, tabItemsHistory []dl.TagItem) {
	if tabItemsHistory == nil {
		slog.Debug("tabItemsHistory is nil")
		return
	}
	if index < 0 || index >= len(tabItemsHistory) {
		slog.Debug(fmt.Sprintf("index = %d, tabItemsHistory = %d", index, len(tabItemsHistory)))
		return
	}

	cleanData(left, comboboxContainer, index, optionData)
	tagItem := tabItemsHistory[index]
	title, optionNames, _, children := dl.Query(tagItem, docPDFMap)
	if len(optionNames) > 0 {
		if len(children) > 0 {
			label := widget.NewLabel(title)
			combobox := widget.NewSelect(optionNames, func(selected string) {
				// 创建下一个下拉框
				optIndex := slices.Index(optionNames, selected)
				childItem := children[optIndex]
				tabItemsHistory = append(tabItemsHistory[:index+1], childItem)

				createComboboxes(left, comboboxContainer, index+1, optionData, docPDFMap, tabItemsHistory)
			})
			comboboxContainer.Add(container.NewBorder(nil, nil, label, nil, combobox))
		} else {
			// 最后一层，创建复选框
			// createCheckboxes(left, optionNames, optionIDs, optionData)
		}
	} else {
		dialog.ShowError(fmt.Errorf("数据查询为空"), nil)
		// initData(nil, &docPDFMap, &tabItemsHistory)
	}
	comboboxContainer.Refresh()
}

func initRightPart(w fyne.Window, left *fyne.Container, optionData binding.StringList, local bool) *fyne.Container {
	var tabItemsHistory []dl.TagItem // 计算当前tag层级状态
	// right part: comboboxes for categories
	comboboxContainer := container.NewVBox()
	queryButton := widget.NewButtonWithIcon("查询", theme.SearchIcon(), nil)
	infoLabel := widget.NewLabel("点击查询加载教材信息")

	queryButton.OnTapped = func() {
		infoLabel.SetText("加载中...")
		tagItems, _, docPDFMap := dl.FetchRawData("", local)
		tabItemsHistory = []dl.TagItem{tagItems[0]}
		if tabItemsHistory != nil {
			infoLabel.SetText("请选择教材")
			slog.Debug("加载数组" + fmt.Sprintf("tabItemsHistory = %d", len(tabItemsHistory)))
			slog.Debug(fmt.Sprintf("docPDFMap = %d", len(docPDFMap)))
			queryButton.SetText("重置")
			cleanData(left, comboboxContainer, 0, optionData)
			createComboboxes(left, comboboxContainer, 0, optionData, docPDFMap, tabItemsHistory)

		} else {
			infoLabel.SetText("教材加载失败，稍后重试")
			dialog.ShowError(fmt.Errorf("数据初始化失败"), w)
		}
	}

	bottom := container.NewVBox(widget.NewSeparator(), container.NewCenter(queryButton))
	right := container.NewBorder(infoLabel, bottom, nil, nil, comboboxContainer)
	return right
}

func CreateOptionsTab(w fyne.Window, optionData binding.StringList) *fyne.Container {
	local := false // 是否使用本地数据
	left := container.NewHBox()
	right := initRightPart(w, left, optionData, local)

	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}
