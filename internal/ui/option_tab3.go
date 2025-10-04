package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
)

func CreateReadingOptionsTab(w fyne.Window, linkItemMaps map[string][]dl.LinkItem, name string, isLocal bool, arrayLen int) *fyne.Container {
	var tabData = OptionTabData{
		InitTabData:     false,
		ComboLabelArray: make([]*widget.Label, arrayLen),
		ComboboxArray:   make([]*widget.Select, arrayLen),
		QueryButton:     widget.NewButtonWithIcon("查询", theme.SearchIcon(), nil),
		QueryLabel:      widget.NewLabel(""),
		QueryText:       binding.NewString(),

		CheckLabel:      widget.NewLabel(""),
		CheckText:       binding.NewString(),
		CheckGroup:      widget.NewCheckGroup(nil, nil),
		SelectAllButton: widget.NewButtonWithIcon("全选", theme.ConfirmIcon(), nil),
		CancelAllButton: widget.NewButtonWithIcon("清空", theme.CancelIcon(), nil),
	}

	// 绑定文本
	labels := dl.TAB_NAMES_LABEL[dl.TAB_NAMES[3]]
	tabData.QueryLabel.Bind(tabData.QueryText)
	tabData.CheckLabel.Bind(tabData.CheckText)
	tabData.QueryText.Set(labels[0])
	tabData.CheckText.Set(labels[1])

	tabData.SelectAllButton.Disable()
	tabData.CancelAllButton.Disable()

	buttonContainer := container.NewCenter(container.NewHBox(tabData.SelectAllButton, tabData.CancelAllButton))
	bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)

	// 限制多选框高度
	scrollContainer := container.NewVScroll(tabData.CheckGroup)
	scrollContainer.SetMinSize(fyne.NewSize(300, 400))
	left := container.NewBorder(tabData.CheckLabel, bottom, nil, nil, scrollContainer)

	right := initRightPart(w, linkItemMaps, tabData, name, isLocal, arrayLen)
	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}
