package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func initRightPart(w fyne.Window, left *fyne.Container, optionData binding.StringList, local bool) *fyne.Container {
	// var tabItemsHistory []dl.TagItem // 计算当前tag层级状态
	// right part: comboboxes for categories
	comboboxContainer := container.NewVBox()
	queryButton := widget.NewButtonWithIcon("查询", theme.SearchIcon(), nil)
	infoLabel := widget.NewLabel("点击查询加载教材信息")

	queryButton.Disable()

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
