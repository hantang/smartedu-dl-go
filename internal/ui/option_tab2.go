package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
)

func createRadiobuttons(w fyne.Window, name string, tabData OptionTabData, linkItemMaps map[string][]dl.LinkItem, bookOptions []dl.BookOption) {
	// left part: checkboxes for book(PDF)
	options := []string{}
	optionMap := map[string]string{}
	for i, opt := range bookOptions {
		name2 := fmt.Sprintf("%d. %s", i+1, opt.OptionName)
		options = append(options, name2)
		optionMap[name2] = opt.OptionID
	}

	tabData.StatsText.Set(fmt.Sprintf("💡 课程（共%d项）：", len(options)))
	tabData.RadioDict = optionMap
	tabData.RadioGroup.Options = options
	tabData.RadioGroup.OnChanged = radioCallback(w, name, tabData, linkItemMaps)
	tabData.RadioGroup.Enable()
}

func radioCallback(w fyne.Window, name string, tabData OptionTabData, linkItemMaps map[string][]dl.LinkItem) func(string) {
	return func(value string) {
		var courseToc []dl.CourseToc

		courseID, ok := tabData.RadioDict[value]
		if ok {
			slog.Debug(fmt.Sprintf("Radio courseID = %s", courseID))
			courseToc, ok = tabData.CourseDict[courseID]
			if !ok {
				tabData.CheckText.Set("查询课程单元中")
				courseToc = dl.ParseCourseID(courseID)
			}
		}

		options := []string{}
		optionToc := make(map[string]dl.CourseToc)
		for _, toc := range courseToc {
			options = append(options, toc.Title)
			optionToc[toc.Title] = toc
		}

		if len(options) == 0 {
			tabData.CheckText.Set("课程单元为空")
			// TODO 重新在在下拉框选择时后弹出
			// dialog.ShowError(fmt.Errorf("课程单元为空，请查询其他课程"), w)
			return
		} else {
			tabData.CheckText.Set(fmt.Sprintf("课程单元（共%d章）", len(options)))
		}

		tabData.Combobox.SetOptions(options)
		tabData.Combobox.SetSelected(options[0])
		tabData.Combobox.OnChanged = func(selected string) {
			createCheckboxes2(name, tabData, linkItemMaps, optionToc[selected])
		}
		tabData.Combobox.Enable()
	}
}

func createCheckboxes2(name string, tabData OptionTabData, linkItemMaps map[string][]dl.LinkItem, courseToc dl.CourseToc) {
	labels := dl.TAB_NAMES_LABEL[name]
	info := labels[1]
	quantifier := labels[3]

	options := []string{}
	optionMap := map[string]dl.CourseItem{}
	for _, opt := range courseToc.Children {
		options = append(options, opt.Title)
		optionMap[opt.Title] = opt
	}

	tabData.CheckText.Set(fmt.Sprintf("%s（共%d%s）：", info, len(options), quantifier))
	tabData.CheckGroup.Options = options
	tabData.CheckGroup.SetSelected([]string{})
	tabData.CheckGroup.OnChanged = func(items []string) {
		linkItemMaps[name] = []dl.LinkItem{}
		for _, item := range items {
			linkItem := dl.LinkItem{
				Link: optionMap[item].CourseID,
				Type: optionMap[item].ResourceType,
			}
			linkItemMaps[name] = append(linkItemMaps[name], linkItem)
		}
		tabData.CheckText.Set(fmt.Sprintf("%s（共%d%s，已选%d%s）：", info, len(options), quantifier, len(items), quantifier))
	}

	tabData.SelectAllButton.OnTapped = func() {
		tabData.CheckGroup.SetSelected(options)
	}
	tabData.CancelAllButton.OnTapped = func() {
		tabData.CheckGroup.SetSelected(nil)
		linkItemMaps[name] = []dl.LinkItem{}
	}

	tabData.CheckGroup.Enable()
	tabData.SelectAllButton.Enable()
	tabData.CancelAllButton.Enable()
}

func CreateClassroomOptionsTab(w fyne.Window, linkItemMaps map[string][]dl.LinkItem, name string, isLocal bool, arrayLen int) *fyne.Container {
	var tabData = OptionTabData{
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

		RadioGroup:      widget.NewRadioGroup([]string{}, nil),
		RadioStatsLabel: widget.NewLabel(""),
		StatsText:       binding.NewString(),
		Combobox:        widget.NewSelect([]string{}, nil),
		RadioDict:       make(map[string]string),
		CourseDict:      make(map[string][]dl.CourseToc),
	}

	labels := dl.TAB_NAMES_LABEL[dl.TAB_NAMES[2]]
	tabData.QueryLabel.Bind(tabData.QueryText)
	tabData.CheckLabel.Bind(tabData.CheckText)
	tabData.RadioStatsLabel.Bind(tabData.StatsText)
	tabData.QueryText.Set(labels[0])
	tabData.CheckText.Set(labels[1])
	tabData.StatsText.Set("💡 请选择某一课程")

	tabData.SelectAllButton.Disable()
	tabData.CancelAllButton.Disable()

	buttonContainer := container.NewCenter(container.NewHBox(tabData.SelectAllButton, tabData.CancelAllButton))
	bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)
	top := container.NewVBox(tabData.CheckLabel, tabData.Combobox)
	scrollCheckGroup := container.NewScroll(tabData.CheckGroup)
	leftDown := container.NewBorder(top, bottom, nil, nil, scrollCheckGroup)

	leftTop := container.NewBorder(tabData.RadioStatsLabel, nil, nil, nil, tabData.RadioGroup)
	left := container.NewVSplit(leftTop, leftDown)

	right := initRightPart(w, linkItemMaps, tabData, name, isLocal, arrayLen)
	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}
