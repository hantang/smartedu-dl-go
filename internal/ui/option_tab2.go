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
)

func cleanData(index int, optionData binding.StringList, labelArray []*widget.Label, comboboxArray []*widget.Select, placeholders []string) {
	// Clear data
	optionData.Set(nil)

	statsLabel.SetText("")
	checkGroup.Options = []string{}
	checkGroup.SetSelected([]string{})
	selectButton.OnTapped = nil
	deselectButton.OnTapped = nil

	checkGroup.Disable()
	selectButton.Disable()
	deselectButton.Disable()

	if (index < 0) || (index >= len(labelArray)) {
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

func updateComboboxes(index int, optionData binding.StringList,
	w fyne.Window, labelArray []*widget.Label, comboboxArray []*widget.Select,
	placeholders []string, docPDFMap map[string]dl.DocPDFData, tabItemsHistory []dl.TagItem) {

	if index < 0 || index >= len(labelArray) {
		slog.Debug(fmt.Sprintf("index = %d, labelArray = %d", index, len(labelArray)))
		return
	}

	cleanData(index, optionData, labelArray, comboboxArray, placeholders)
	title, optionNames, optionIDs, children := dl.Query(tabItemsHistory[index], docPDFMap)
	if len(optionNames) == 0 {
		dialog.ShowError(fmt.Errorf("数据查询为空"), w)
		return
	}
	if len(children) > 0 {
		labelArray[index].SetText(fmt.Sprintf("%s〖%s〗", placeholders[index], title))
		comboboxArray[index].SetOptions(optionNames)
		comboboxArray[index].SetSelected(optionNames[0])
		comboboxArray[index].OnChanged = func(selected string) {
			// 创建下一个下拉框
			optIndex := slices.Index(optionNames, selected)
			tabItemsHistory[index+1] = children[optIndex]
			updateComboboxes(index+1, optionData, w, labelArray[:], comboboxArray[:], placeholders, docPDFMap, tabItemsHistory)
		}
		comboboxArray[index].Enable()
	} else {
		// 最后一层，创建复选框
		createCheckboxes(optionData, optionNames, optionIDs)
	}
}

func createCheckboxes(optionData binding.StringList, optionNames []string, optionIDs []string) {
	// left part: checkboxes for book(PDF)
	options := []string{}
	optionMap := map[string]string{}
	for i, name := range optionNames {
		name2 := fmt.Sprintf("%d. 《%s》", i+1, name)
		options = append(options, name2)
		optionMap[name2] = optionIDs[i]
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

func initRightPart(w fyne.Window, optionData binding.StringList, local bool) *fyne.Container {
	// right part: comboboxes for categories
	total := 5 // TODO
	labelArray := make([]*widget.Label, total)
	comboboxArray := make([]*widget.Select, total)
	comboContainers := make([]fyne.CanvasObject, total)

	var docPDFMap map[string]dl.DocPDFData
	tabItemsHistory := make([]dl.TagItem, total+1)
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

		if docPDFMap == nil {
			tagItems, _, tmpDocPDFMap := dl.FetchRawData("", local)
			tabItemsHistory[index] = tagItems[index]
			docPDFMap = tmpDocPDFMap
		}
		if docPDFMap != nil {
			slog.Debug(fmt.Sprintf("docPDFMap = %d", len(docPDFMap)))
			infoLabel.SetText("请选择教材")
			queryButton.SetText("重置")

			updateComboboxes(index, optionData, w, labelArray[:], comboboxArray[:], placeholders, docPDFMap, tabItemsHistory)
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
	statsLabel = widget.NewLabel("")
	checkGroup = widget.NewCheckGroup(nil, nil)

	selectButton = widget.NewButtonWithIcon("全选", theme.ConfirmIcon(), nil)
	deselectButton = widget.NewButtonWithIcon("清空", theme.CancelIcon(), nil)
	selectButton.Disable()
	deselectButton.Disable()

	buttonContainer := container.NewCenter(container.NewHBox(selectButton, deselectButton))
	bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)
	left := container.NewBorder(statsLabel, bottom, nil, nil, checkGroup)

	right := initRightPart(w, optionData, local)

	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}
