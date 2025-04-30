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

type OptionTabData struct {
	StatsLabel     *widget.Label
	CheckGroup     *widget.CheckGroup
	SelectButton   *widget.Button
	DeselectButton *widget.Button
	LabelArray     []*widget.Label
	ComboboxArray  []*widget.Select
}

func cleanData(tabData OptionTabData, index int, optionData binding.StringList, labelArray []*widget.Label, comboboxArray []*widget.Select, placeholders []string) {
	// Clear data
	optionData.Set(nil)

	tabData.StatsLabel.SetText("课本")
	tabData.CheckGroup.Options = []string{}
	tabData.CheckGroup.SetSelected([]string{})
	tabData.SelectButton.OnTapped = nil
	tabData.DeselectButton.OnTapped = nil

	tabData.CheckGroup.Disable()
	tabData.SelectButton.Disable()
	tabData.DeselectButton.Disable()

	if index < 0 {
		return
	}
	for i := index; i < len(labelArray); i++ {
		labelArray[i].SetText(placeholders[i])
		comboboxArray[i].SetOptions(nil)
		comboboxArray[i].SetSelected("")
		comboboxArray[i].OnChanged = nil
		comboboxArray[i].Disable()
	}
}

func updateComboboxes(tabData OptionTabData, index int, optionData binding.StringList, w fyne.Window,
	placeholders []string, bookItemsHistory []dl.BookItem) {
	// 改成固定数量下拉框
	if index < 0 {
		slog.Debug(fmt.Sprintf("index = %d, labelArray = %d", index, len(tabData.LabelArray)))
		return
	}

	cleanData(tabData, index, optionData, tabData.LabelArray, tabData.ComboboxArray, placeholders)
	title, bookOptions, children := dl.Query2(bookItemsHistory[index])
	if len(bookOptions) == 0 {
		dialog.ShowError(fmt.Errorf("数据查询为空"), w)
		return
	}
	optionNames := []string{}
	for _, option := range bookOptions {
		optionNames = append(optionNames, option.OptionName)
	}
	if len(children) > 0 {
		if title == "电子教材" {
			title = "学段"
		} else if title == "新旧教材" {
			title = "教材"
		}

		tabData.LabelArray[index].SetText(fmt.Sprintf("%s〖%s〗", placeholders[index], title))
		tabData.ComboboxArray[index].SetOptions(optionNames)
		tabData.ComboboxArray[index].SetSelected(optionNames[0])
		tabData.ComboboxArray[index].OnChanged = func(selected string) {
			// 创建下一个下拉框
			optIndex := slices.Index(optionNames, selected)
			bookItemsHistory[index+1] = children[optIndex]
			updateComboboxes(tabData, index+1, optionData, w, placeholders, bookItemsHistory)
		}
		tabData.ComboboxArray[index].Enable()
	} else {
		// 最后一层，创建复选框
		createCheckboxes(tabData, optionData, bookOptions)
	}
}

func createCheckboxes(tabData OptionTabData, optionData binding.StringList, bookOptions []dl.BookOption) {
	// left part: checkboxes for book(PDF)
	options := []string{}
	optionMap := map[string]string{}
	for i, opt := range bookOptions {
		name2 := fmt.Sprintf("%d. %s", i+1, opt.OptionName)
		options = append(options, name2)
		optionMap[name2] = opt.OptionID
	}

	optionData.Set(nil)
	tabData.StatsLabel.SetText(fmt.Sprintf("课本（共%d项）：", len(options)))
	tabData.CheckGroup.Options = options
	tabData.CheckGroup.Selected = []string{}
	tabData.CheckGroup.OnChanged = func(items []string) {
		optionData.Set(nil)
		for _, item := range items {
			optionData.Append(optionMap[item])
		}
		tabData.StatsLabel.SetText(fmt.Sprintf("课本（共%d项，已选%d项）：", len(options), len(items)))
	}
	tabData.SelectButton.OnTapped = func() {
		tabData.CheckGroup.SetSelected(options)
	}
	tabData.DeselectButton.OnTapped = func() {
		tabData.CheckGroup.SetSelected(nil)
	}

	tabData.CheckGroup.Enable()
	tabData.SelectButton.Enable()
	tabData.DeselectButton.Enable()
}

func initRightPart(w fyne.Window, optionData binding.StringList, tabData OptionTabData, name string, isLocal bool, arrayLen int) *fyne.Container {
	// right part: comboboxes for categories
	comboContainers := make([]fyne.CanvasObject, arrayLen)

	initTabData := false
	bookItemsHistory := make([]dl.BookItem, arrayLen+1)
	placeholders := []string{"㊀", "㊁", "㊂", "㊃", "㊄", "㊅", "㊆", "㊇", "㊈", "㊉"}

	for i := range tabData.LabelArray {
		tabData.LabelArray[i] = widget.NewLabel(placeholders[i])
		tabData.ComboboxArray[i] = widget.NewSelect([]string{}, nil)
		tabData.ComboboxArray[i].Disable()
		comboContainers[i] = container.NewBorder(nil, nil, tabData.LabelArray[i], nil, tabData.ComboboxArray[i])
	}
	comboboxContainer := container.NewVBox(comboContainers...)
	queryButton := widget.NewButtonWithIcon("查询", theme.SearchIcon(), nil)
	infoLabel := widget.NewLabel("点击查询加载教材信息")

	queryButton.OnTapped = func() {
		infoLabel.SetText("加载中...")
		index := 0

		if !initTabData {
			bookBase := dl.FetchRawData2(name, isLocal)
			if len(bookBase.Children) > 0 {
				bookItemsHistory[index] = bookBase.Children[index]
				initTabData = true
			}
		}
		if initTabData {
			infoLabel.SetText("请选择教材")
			queryButton.SetText("重置")

			updateComboboxes(tabData, index, optionData, w, placeholders, bookItemsHistory)
		} else {
			infoLabel.SetText("教材加载失败，稍后重试")
			dialog.ShowError(fmt.Errorf("数据初始化失败"), w)
		}
	}

	bottom := container.NewVBox(widget.NewSeparator(), container.NewCenter(queryButton))
	right := container.NewBorder(infoLabel, bottom, nil, nil, comboboxContainer)
	return right
}

func CreateOptionsTab(w fyne.Window, optionData binding.StringList, name string, isLocal bool, arrayLen int) *fyne.Container {
	// 左侧多选框
	var tabData = OptionTabData{
		StatsLabel:     widget.NewLabel("课本"),
		CheckGroup:     widget.NewCheckGroup(nil, nil),
		SelectButton:   widget.NewButtonWithIcon("全选", theme.ConfirmIcon(), nil),
		DeselectButton: widget.NewButtonWithIcon("清空", theme.CancelIcon(), nil),
		LabelArray:     make([]*widget.Label, arrayLen),
		ComboboxArray:  make([]*widget.Select, arrayLen),
	}

	tabData.SelectButton.Disable()
	tabData.DeselectButton.Disable()

	buttonContainer := container.NewCenter(container.NewHBox(tabData.SelectButton, tabData.DeselectButton))
	bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)
	left := container.NewBorder(tabData.StatsLabel, bottom, nil, nil, tabData.CheckGroup)

	right := initRightPart(w, optionData, tabData, name, isLocal, arrayLen)
	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}

func initRightPart2(w fyne.Window, optionData binding.StringList, tabData OptionTabData, name string, isLocal bool, arrayLen int) *fyne.Container {
	/*
	课程列表 解析：
	data_version.json -> part_10x.json (得到教材信息， 没有tag_paths) -> "id" 字段 得到parts.json->part_100.json 课程单元
	national_lesson_tag.json -> 教材层级结构
	*/

	comboContainers := make([]fyne.CanvasObject, arrayLen)

	initTabData := false
	bookItemsHistory := make([]dl.BookItem, arrayLen+1)
	placeholders := []string{"㊀", "㊁", "㊂", "㊃", "㊄", "㊅", "㊆", "㊇", "㊈", "㊉"}

	for i := range tabData.LabelArray {
		tabData.LabelArray[i] = widget.NewLabel(placeholders[i])
		tabData.ComboboxArray[i] = widget.NewSelect([]string{}, nil)
		tabData.ComboboxArray[i].Disable()
		comboContainers[i] = container.NewBorder(nil, nil, tabData.LabelArray[i], nil, tabData.ComboboxArray[i])
	}
	comboboxContainer := container.NewVBox(comboContainers...)
	queryButton := widget.NewButtonWithIcon("查询", theme.SearchIcon(), nil)
	infoLabel := widget.NewLabel("点击查询加载课程教学内容")

	queryButton.OnTapped = func() {
		infoLabel.SetText("加载中...")
		index := 0

		if !initTabData {
			bookBase := dl.FetchRawData2(name, isLocal)
			if bookBase.Name != "" {
				bookItemsHistory[index] = bookBase
				initTabData = true
			}
		}
		if initTabData {
			infoLabel.SetText("请选择课程")
			queryButton.SetText("重置")

			updateComboboxes(tabData, index, optionData, w, placeholders, bookItemsHistory)
		} else {
			infoLabel.SetText("课程数据加载失败，稍后重试")
			dialog.ShowError(fmt.Errorf("数据初始化失败"), w)
		}
	}

	bottom := container.NewVBox(widget.NewSeparator(), container.NewCenter(queryButton))
	right := container.NewBorder(infoLabel, bottom, nil, nil, comboboxContainer)
	return right
}

func CreateClassroomOptionsTab(w fyne.Window, optionData binding.StringList, name string, isLocal bool, arrayLen int) *fyne.Container {
	var tabData = OptionTabData{
		StatsLabel:     widget.NewLabel("课程教学"),
		CheckGroup:     widget.NewCheckGroup(nil, nil),
		SelectButton:   widget.NewButtonWithIcon("全选", theme.ConfirmIcon(), nil),
		DeselectButton: widget.NewButtonWithIcon("清空", theme.CancelIcon(), nil),
		LabelArray:     make([]*widget.Label, arrayLen),
		ComboboxArray:  make([]*widget.Select, arrayLen),
	}

	tabData.SelectButton.Disable()
	tabData.DeselectButton.Disable()

	buttonContainer := container.NewCenter(container.NewHBox(tabData.SelectButton, tabData.DeselectButton))
	bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)
	left := container.NewBorder(tabData.StatsLabel, bottom, nil, nil, tabData.CheckGroup)

	right := initRightPart2(w, optionData, tabData, name, isLocal, arrayLen)
	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}
