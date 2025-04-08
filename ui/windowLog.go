package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowLogWindow(application fyne.App) (fyne.Window, *widget.Entry, *container.Scroll) {
	logWindow := application.NewWindow("Compression Logs")
	logWindow.Resize(fyne.NewSize(600, 400))

	logContent := widget.NewMultiLineEntry()

	logScrollContainer := container.NewScroll(logContent)
	logWindow.SetContent(logScrollContainer)
	logWindow.Show()

	return logWindow, logContent, logScrollContainer
}
