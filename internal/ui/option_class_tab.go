package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func initRightPart2(w fyne.Window, optionData binding.StringList, isLocal bool, arrayLen int) *fyne.Container {
	right := container.NewBorder(nil, nil, nil, nil, nil)
	return right
}

func CreateClassroomOptionsTab(w fyne.Window, optionData binding.StringList, isLocal bool, arrayLen int) *fyne.Container {
	// 左侧多选框
	statsLabel := widget.NewLabel("课程教学")
	// checkGroup = widget.NewCheckGroup(nil, nil)

	// selectButton = widget.NewButtonWithIcon("全选", theme.ConfirmIcon(), nil)
	// deselectButton = widget.NewButtonWithIcon("清空", theme.CancelIcon(), nil)
	// selectButton.Disable()
	// deselectButton.Disable()

	// buttonContainer := container.NewCenter(container.NewHBox(selectButton, deselectButton))
	// bottom := container.NewVBox(widget.NewSeparator(), buttonContainer)
	// left := container.NewBorder(statsLabel, bottom, nil, nil, checkGroup)
	left := container.NewBorder(statsLabel, nil, nil, nil, nil)

	// right := initRightPart(w, optionData, local, total)

	return container.NewBorder(nil, nil, nil, nil, left)
}
