package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"strconv"
	"strings"
)

func CreateSizeEntry(sizeEntry *widget.Entry, compressBtn *widget.Button, isFileEmpty func() bool, updateEstimate func()) (*fyne.Container, *widget.Entry) {
	sizeEntry = widget.NewEntry()
	sizeEntry.SetPlaceHolder("Enter target size (MB) - 9.5 for Discord.")
	sizeEntry.SetText("9.5 MB")
	sizeEntry.Validator = func(s string) error {
		updateEstimate()
		if _, err := strconv.ParseFloat(strings.TrimSuffix(s, " MB"), 64); err != nil {
			compressBtn.Disable()
			return fmt.Errorf("invalid number")
		}
		if isFileEmpty() {
			compressBtn.Enable()
		}
		return nil
	}

	incrementBtn := widget.NewButton("+", func() {
		updateEstimate()
		if sizeEntry.Text == "" {
			sizeEntry.SetText("0.5 MB")
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(sizeEntry.Text, " MB"), 64)
		if err == nil {
			value += 0.5
			sizeEntry.SetText(fmt.Sprintf("%.1f MB", value))
		}
	})

	decrementBtn := widget.NewButton("-", func() {
		updateEstimate()
		if sizeEntry.Text == "" {
			sizeEntry.SetText("0.5 MB")
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(sizeEntry.Text, " MB"), 64)
		if err == nil && value > 0.5 {
			value -= 0.5
			sizeEntry.SetText(fmt.Sprintf("%.1f MB", value))
		}
	})

	/*
		prefilledSizes := []string{"5 MB", "9.5 MB", "14.5 MB", "19.5 MB", "24.5 MB", "29.5 MB", "34.5 MB", "39.5 MB", "44.5 MB"}
		menuItems := make([]*fyne.MenuItem, len(prefilledSizes))
		for i, sizeElement := range prefilledSizes {
			menuItems[i] = fyne.NewMenuItem(sizeElement, func() {
				sizeEntry.SetText(sizeElement)
			})
		}

		contextMenu := fyne.NewMenu("Set Size", menuItems...)
		sizeEntry.TappedSecondary()
		widget.ShowPopUpMenuAtPosition(contextMenu, fyne.CurrentApp().Driver().CanvasForObject(sizeEntry), pe.Position)
	*/

	finalResult := container.NewBorder(nil, nil, nil, container.NewHBox(decrementBtn, incrementBtn), sizeEntry)
	return finalResult, sizeEntry
}
