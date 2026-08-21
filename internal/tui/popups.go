package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/yodeman/termdict/internal/config"
)

// initializePopups initializes the popup widgets used to display
// messages.
func (u *UI) initializePopups() {
	helpWidget := tview.NewTextView().
		SetDoneFunc(func(_ tcell.Key) {
			u.pages.HidePage("help page")
			u.app.SetFocus(u.searchInputField)
		}).
		SetText(helpMessage).
		SetDynamicColors(true)
	helpWidget.SetBorder(true)
	helpWidget.SetBackgroundColor(borderColor)

	u.helpPopup = tview.NewGrid().
		SetBorders(false).
		SetColumns(0, popupWidth, 0).
		SetRows(0, popupHeight, 0).
		AddItem(helpWidget, 1, 1, 1, 1, 0, 0, true)

	u.aboutPopup = tview.NewModal().
		AddButtons([]string{"close"}).
		SetText(aboutMessage(config.AppVersion)).
		SetDoneFunc(func(_ int, _ string) {
			u.pages.HidePage("about page")
			u.app.SetFocus(u.searchInputField)
		})

	u.updateWidget = tview.NewTextView().
		SetDoneFunc(func(_ tcell.Key) {
			u.pages.HidePage("update page")
			u.app.SetFocus(u.searchInputField)
		}).
		SetText("Updating database...").
		SetChangedFunc(func() { u.app.Draw() }).
		SetDynamicColors(true)
	u.updateWidget.SetBorder(true)
	u.updateWidget.SetBackgroundColor(borderColor)

	u.updatePopup = tview.NewGrid().
		SetBorders(false).
		SetColumns(0, popupWidth, 0).
		SetRows(0, popupHeight, 0).
		AddItem(u.updateWidget, 1, 1, 1, 1, 0, 0, true)
}
