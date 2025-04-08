package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func CreateFileListContainer(dndFileList binding.StringList, sizeEntry *widget.Entry) fyne.CanvasObject {
	list := widget.NewListWithData(dndFileList, func() fyne.CanvasObject {
		return widget.NewLabel("")
	}, func(i binding.DataItem, o fyne.CanvasObject) {
		o.(*widget.Label).Bind(i.(binding.String))
	})

	sizeEntryDialog := widget.NewEntry()
	sizeEntryDialog.SetPlaceHolder("Enter target size (MB) - 9.5 for Discord.")
	sizeEntryDialog.SetText("9.5 MB")

	sizeEntryDialog.OnChanged = func(s string) {
		sizeEntry.SetText(s)
	}

	listBorder := container.NewBorder(nil, container.NewVBox(widget.NewLabel("Target Size (MB):"), sizeEntryDialog), nil, nil, list)

	return listBorder
}

func ShowFileListDialog(dndFileList binding.StringList, sizeEntry *widget.Entry, window fyne.Window, onCompressButtonPressed func()) {
	customConfirm := dialog.NewCustomConfirm("Files to Compress", "Compress", "Cancel", CreateFileListContainer(dndFileList, sizeEntry), func(ok bool) {
		if ok {
			onCompressButtonPressed()
		}
	}, window)
	customConfirm.Resize(fyne.NewSize(400, 300)) // Set the desired size of the customConfirm
	customConfirm.Show()
}
