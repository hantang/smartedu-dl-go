package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	"github.com/hantang/smartedudlgo/internal/dl"
)

func createCombobox(title string, options []string) *fyne.Container {
	label := widget.NewLabel(title)
	dropdown := widget.NewSelect(options, func(selected string) {
		// Handle selection
	})
	return container.NewBorder(nil, nil, label, nil, dropdown)
}

func CreateOptionsTab(inputData binding.String) *fyne.Container {
	inputData.Set("")
	tagItem, _, docPDFMap := dl.FetchRawData("", true)

	queryButton := widget.NewButton("查询", func() {
		// Handle query action
	})

	dropdownContainer := container.NewVBox()
	title, optionNames, optionIDs, children := dl.Query(tagItem, docPDFMap)
	_ = optionIDs
	_ = children
	if len(optionNames) > 0 {
		// Create new dropdown for child options
		childDropdown := createCombobox(title, optionNames)
		dropdownContainer.Add(childDropdown)
	}

	dataList := []string{}
	var checkboxes []fyne.CanvasObject
	for _, value := range dataList {
		checkbox := widget.NewCheck(value, func(checked bool) {
			// Handle checkbox state change
		})
		checkboxes = append(checkboxes, checkbox)
	}

	selectButton := widget.NewButton("全选", func() {
		for _, obj := range checkboxes {
			if checkbox, ok := obj.(*widget.Check); ok {
				checkbox.SetChecked(true)
			}
		}
	})

	deselectButton := widget.NewButton("清空", func() {
		for _, obj := range checkboxes {
			if checkbox, ok := obj.(*widget.Check); ok {
				checkbox.SetChecked(false)
			}
		}
	})

	// Create containers
	left := container.NewBorder(
		nil, container.NewCenter(container.NewHBox(selectButton, deselectButton)), nil, nil,
		container.NewVBox(checkboxes...),
	)

	right := container.NewBorder(nil, container.NewCenter(queryButton), nil, nil, dropdownContainer)

	return container.NewBorder(nil, nil, nil, nil, container.NewHSplit(left, right))
}
