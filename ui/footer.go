package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"net/url"
)

func CreateFooter(resourceKofiIconPng fyne.Resource) *fyne.Container {
	buyMeACoffeeURL, _ := url.Parse("https://www.maxisjoe.xyz/maxfund")
	buyMeACoffeeLink := widget.NewHyperlink("buy me a coffee", buyMeACoffeeURL)

	dollarIcon := widget.NewIcon(theme.ContentAddIcon())
	dollarIcon.SetResource(theme.NewThemedResource(resourceKofiIconPng))

	footer := container.NewHBox(
		dollarIcon,
		buyMeACoffeeLink,
	)

	return footer
}
