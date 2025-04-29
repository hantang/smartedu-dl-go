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

var (
	statsLabel     *widget.Label
	checkGroup     *widget.CheckGroup
	selectButton   *widget.Button
	deselectButton *widget.Button
	labelArray     []*widget.Label
	comboboxArray  []*widget.Select
)

func cleanData(index int, optionData binding.StringList, labelArray []*widget.Label, comboboxArray []*widget.Select, placeholders []string) {
	// Clear data
	optionData.Set(nil)

	statsLabel.SetText("课本")
	checkGroup.Options = []string{}
	checkGroup.SetSelected([]string{})
	selectButton.OnTapped = nil
	deselectButton.OnTapped = nil

	checkGroup.Disable()
	selectButton.Disable()
	deselectButton.Disable()

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

func updateComboboxes(index int, optionData binding.StringList, w fyne.Window,
	placeholders []string, bookItemsHistory []dl.BookItem) {
	// 改成固定数量下拉框

	if index < 0 {
		slog.Debug(fmt.Sprintf("index = %d, labelArray = %d", index, len(labelArray)))
		return
	}

	cleanData(index, optionData, labelArray, comboboxArray, placeholders)
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
		labelArray[index].SetText(fmt.Sprintf("%s〖%s〗", placeholders[index], title))
		comboboxArray[index].SetOptions(optionNames)
		comboboxArray[index].SetSelected(optionNames[0])
		comboboxArray[index].OnChanged = func(selected string) {
			// 创建下一个下拉框
			optIndex := slices.Index(optionNames, selected)
			bookItemsHistory[index+1] = children[optIndex]
			updateComboboxes(index+1, optionData, w, placeholders, bookItemsHistory)
		}
		comboboxArray[index].Enable()
	} else {
		// 最后一层，创建复选框
		createCheckboxes(optionData, bookOptions)
	}
}

func createCheckboxes(optionData binding.StringList, bookOptions []dl.BookOption) {
	// left part: checkboxes for book(PDF)
	options := []string{}
	optionMap := map[string]string{}
	for i, opt := range bookOptions {
		name2 := fmt.Sprintf("%d. %s", i+1, opt.OptionName)
		options = append(options, name2)
		optionMap[name2] = opt.OptionID
	}

	optionData.Set(nil)
	statsLabel.SetText(fmt.Sprintf("课本（共%d项）：", len(options)))
	checkGroup.Options = options
	checkGroup.Selected = []string{}
	checkGroup.OnChanged = func(items []string) {
		optionData.Set(nil)
		for _, item := range items {
			optionData.Append(optionMap[item])
		}
		statsLabel.SetText(fmt.Sprintf("课本（共%d项，已选%d项）：", len(options), len(items)))
	}
	selectButton.OnTapped = func() {
		checkGroup.SetSelected(options)
	}
	deselectButton.OnTapped = func() {
		checkGroup.SetSelected(nil)
	}

	checkGroup.Enable()
	selectButton.Enable()
	deselectButton.Enable()
}

func initRightPart(w fyne.Window, optionData binding.StringList, local bool, total int) *fyne.Container {
	// right part: comboboxes for categories
	labelArray = make([]*widget.Label, total)
	comboboxArray = make([]*widget.Select, total)
	comboContainers := make([]fyne.CanvasObject, total)

	initTabData := false
	bookItemsHistory := make([]dl.BookItem, total+1)
	placeholders := []string{"㊀", "㊁", "㊂", "㊃", "㊄", "㊅", "㊆", "㊇", "㊈", "㊉"}

	for i := 0; i < len(labelArray); i++ {
		labelArray[i] = widget.NewLabel(placeholders[i])
		comboboxArray[i] = widget.NewSelect([]string{}, nil)
		comboboxArray[i].Disable()
		comboContainers[i] = container.NewBorder(nil, nil, labelArray[i], nil, comboboxArray[i])
	}
	comboboxContainer := container.NewVBox(comboContainers...)
	queryButton := widget.NewButtonWithIcon("查询", theme.SearchIcon(), nil)
	infoLabel := widget.NewLabel("点击查询加载教材信息")

	queryButton.OnTapped = func() {
		infoLabel.SetText("加载中...")
		index := 0

		if !initTabData {
			bookBase := dl.FetchRawData2("", local)
			if len(bookBase.Children) > 0 {
				bookItemsHistory[index] = bookBase.Children[index]
				initTabData = true
			}
		}
		if initTabData {
			infoLabel.SetText("请选择教材")
			queryButton.SetText("重置")

			updateComboboxes(index, optionData, w, placeholders, bookItemsHistory)
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
	total := 5     // TODO 层级（下拉框）数量

	// 左侧多选框
	statsLabel = widget.NewLabel("课本")
	checkGroup = widget.NewCheckGroup(nil, nil)

	selectButton = widget.NewButtonWithIcon("全选", theme.ConfirmIcon(), nil)
	deselectButton = widget.NewButtonWithIcon("清空", theme.CancelIcon(), nil)
	selectButton.Disable()
	deselectButton.Disable()

	buttonContainer := container.NewCenter(container.NewHBox(selectButton, deselectButton))
	bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)
	left := container.NewBorder(statsLabel, bottom, nil, nil, checkGroup)

	right := initRightPart(w, optionData, local, total)

	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}

func CreateClassroomOptionsTab(w fyne.Window, optionData binding.StringList) *fyne.Container {
	// local := false
	// total := 5

	// 左侧多选框
	statsLabel = widget.NewLabel("课程教学")
	checkGroup = widget.NewCheckGroup(nil, nil)

	selectButton = widget.NewButtonWithIcon("全选", theme.ConfirmIcon(), nil)
	deselectButton = widget.NewButtonWithIcon("清空", theme.CancelIcon(), nil)
	selectButton.Disable()
	deselectButton.Disable()

	buttonContainer := container.NewCenter(container.NewHBox(selectButton, deselectButton))
	bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)
	left := container.NewBorder(statsLabel, bottom, nil, nil, checkGroup)

	// right := initRightPart(w, optionData, local, total)

	return container.NewBorder(nil, nil, nil, nil, left)
}
