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
	// right
	InitTabData     bool
	ComboLabelArray []*widget.Label
	ComboboxArray   []*widget.Select
	QueryButton     *widget.Button
	QueryLabel      *widget.Label
	QueryText       binding.String

	// left bottom
	CheckLabel      *widget.Label
	CheckText       binding.String
	CheckGroup      *widget.CheckGroup // 课本或课程单元
	SelectAllButton *widget.Button
	CancelAllButton *widget.Button

	// left top 用于课程
	RadioGroup      *widget.RadioGroup // 选择课程
	RadioStatsLabel *widget.Label
	Combobox        *widget.Select // 选择章节
	RadioDict       map[string]string
	CourseDict      map[string][]dl.CourseToc
}

var placeholders = []string{"㊀", "㊁", "㊂", "㊃", "㊄", "㊅", "㊆", "㊇", "㊈", "㊉"}

// 重置选择组件（Select 或 RadioGroup）
func resetSelectComponent(selectWidget interface{}) {
	switch v := selectWidget.(type) {
	case *widget.Select:
		v.SetOptions(nil)
		v.SetSelected("")
		v.OnChanged = nil
		v.Disable()
	case *widget.RadioGroup:
		v.Options = []string{}
		v.SetSelected("")
		v.OnChanged = nil
		v.Disable()
	case *widget.CheckGroup:
		v.Options = []string{}
		v.SetSelected([]string{})
		v.OnChanged = nil
		v.Disable()
	}
}

// 重置所有组件状态
func resetComponents(tabData OptionTabData, index int) {
	// 重置多选框
	resetSelectComponent(tabData.CheckGroup)

	// 重置按钮
	tabData.SelectAllButton.OnTapped = nil
	tabData.SelectAllButton.Disable()
	tabData.CancelAllButton.OnTapped = nil
	tabData.CancelAllButton.Disable()

	for i, label := range tabData.ComboLabelArray {
		if i >= index {
			label.SetText(placeholders[i])
			resetSelectComponent(tabData.ComboboxArray[i])
		}
	}

	// 重置单个下拉框
	if tabData.Combobox != nil {
		resetSelectComponent(tabData.Combobox)
	}
	if tabData.RadioGroup != nil {
		resetSelectComponent(tabData.RadioGroup)
	}
}

// 清理数据
func cleanData(tabData OptionTabData, name string, index int, linkItemMaps map[string][]dl.LinkItem) {
	// 重置所有组件
	resetComponents(tabData, index)

	if name == dl.TAB_NAMES[2] {
		// tabData.QueryText.Set("")
		tabData.CheckText.Set("课程包列表")
	} else {
		tabData.CheckText.Set("电子教材")
	}

	// 清空数据
	linkItemMaps[name] = []dl.LinkItem{}
}

func updateComboboxes(w fyne.Window, tabData OptionTabData, name string, index int, linkItemMaps map[string][]dl.LinkItem, bookItemsHistory []dl.BookItem) {
	// 改成固定数量下拉框
	if index < 0 {
		slog.Debug(fmt.Sprintf("index = %d, labelArray = %d", index, len(tabData.ComboLabelArray)))
		return
	}

	cleanData(tabData, name, index, linkItemMaps)
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

		tabData.ComboLabelArray[index].SetText(fmt.Sprintf("%s〖%s〗", placeholders[index], title))
		tabData.ComboboxArray[index].SetOptions(optionNames)
		tabData.ComboboxArray[index].SetSelected(optionNames[0])
		tabData.ComboboxArray[index].OnChanged = func(selected string) {
			// 创建下一个下拉框
			optIndex := slices.Index(optionNames, selected)
			bookItemsHistory[index+1] = children[optIndex]
			updateComboboxes(w, tabData, name, index+1, linkItemMaps, bookItemsHistory)
		}
		tabData.ComboboxArray[index].Enable()
	} else {
		// 最后一层，创建复选框
		if name == dl.TAB_NAMES[2] {
			createRadiobuttons(w, name, tabData, linkItemMaps, bookOptions)
		} else {
			createCheckboxes(name, tabData, linkItemMaps, bookOptions)
		}
	}
}

func createCheckboxes(name string, tabData OptionTabData, linkItemMaps map[string][]dl.LinkItem, bookOptions []dl.BookOption) {
	// left part: checkboxes for book(PDF)
	info := "电子教材"
	options := []string{}
	optionMap := map[string]string{}
	for i, opt := range bookOptions {
		name2 := fmt.Sprintf("%d. %s", i+1, opt.OptionName)
		options = append(options, name2)
		optionMap[name2] = opt.OptionID
	}

	// linkItemMaps[name] = []dl.LinkItem{}
	tabData.CheckText.Set(fmt.Sprintf("%s（共%d项）：", info, len(options)))
	tabData.CheckGroup.Options = options
	tabData.CheckGroup.SetSelected([]string{})
	tabData.CheckGroup.OnChanged = func(items []string) {
		linkItemMaps[name] = []dl.LinkItem{}
		for _, item := range items {
			linkItem := dl.LinkItem{
				Link: optionMap[item],
				Type: dl.TchMaterialInfo.Type,
			}
			linkItemMaps[name] = append(linkItemMaps[name], linkItem)
		}
		tabData.CheckText.Set(fmt.Sprintf("%s（共%d项，已选%d项）：", info, len(options), len(items)))
	}

	tabData.SelectAllButton.OnTapped = func() {
		tabData.CheckGroup.SetSelected(options)
	}
	tabData.CancelAllButton.OnTapped = func() {
		tabData.CheckGroup.SetSelected(nil)
	}

	tabData.CheckGroup.Enable()
	tabData.SelectAllButton.Enable()
	tabData.CancelAllButton.Enable()
}

func initRightPart(w fyne.Window, linkItemMaps map[string][]dl.LinkItem, tabData OptionTabData, name string, isLocal bool, arrayLen int) *fyne.Container {
	// right part: comboboxes for categories
	bookItemsHistory := make([]dl.BookItem, arrayLen+1)
	comboContainers := make([]fyne.CanvasObject, arrayLen)
	info := "教材"
	if name == dl.TAB_NAMES[2] {
		info = "课程"
	}

	for i := range tabData.ComboLabelArray {
		tabData.ComboLabelArray[i] = widget.NewLabel(placeholders[i])
		tabData.ComboboxArray[i] = widget.NewSelect([]string{}, nil)
		tabData.ComboboxArray[i].Disable()
		comboContainers[i] = container.NewBorder(nil, nil, tabData.ComboLabelArray[i], nil, tabData.ComboboxArray[i])
	}

	comboboxContainer := container.NewVBox(comboContainers...)
	bottom := container.NewVBox(widget.NewSeparator(), container.NewCenter(tabData.QueryButton))
	right := container.NewBorder(tabData.QueryLabel, bottom, nil, nil, comboboxContainer)

	// 查询/重置按钮
	tabData.QueryButton.OnTapped = func() {
		tabData.QueryText.Set("加载中...")
		index := 0

		if !tabData.InitTabData {
			bookBase := dl.FetchRawData2(name, isLocal)
			if name == dl.TAB_NAMES[2] {
				if bookBase.Name != "" {
					bookItemsHistory[index] = bookBase
					tabData.InitTabData = true
				}
			} else {

				if len(bookBase.Children) > 0 {
					bookItemsHistory[index] = bookBase.Children[index]
					tabData.InitTabData = true
				}
			}
		}

		if tabData.InitTabData {
			tabData.QueryText.Set("请选择" + info)
			tabData.QueryButton.SetText("重置")
			tabData.QueryButton.SetIcon(theme.ViewRefreshIcon())

			updateComboboxes(w, tabData, name, index, linkItemMaps, bookItemsHistory)
		} else {
			tabData.QueryText.Set(info + "加载失败，稍后重试")
			dialog.ShowError(fmt.Errorf("数据初始化失败"), w)
		}
	}

	return right
}

func CreateMaterialOptionsTab(w fyne.Window, linkItemMaps map[string][]dl.LinkItem, name string, isLocal bool, arrayLen int) *fyne.Container {
	// 左侧多选框
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
	tabData.QueryLabel.Bind(tabData.QueryText)
	tabData.CheckLabel.Bind(tabData.CheckText)
	tabData.QueryText.Set("点击查询、加载教材信息")
	tabData.CheckText.Set("电子教材")

	tabData.SelectAllButton.Disable()
	tabData.CancelAllButton.Disable()

	buttonContainer := container.NewCenter(container.NewHBox(tabData.SelectAllButton, tabData.CancelAllButton))
	bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)
	left := container.NewBorder(tabData.CheckLabel, bottom, nil, nil, tabData.CheckGroup)

	right := initRightPart(w, linkItemMaps, tabData, name, isLocal, arrayLen)
	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}
