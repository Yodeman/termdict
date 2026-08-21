package tui

import (
	"github.com/rivo/tview"
)

// initializeButtons initializes the command buttons.
func (u *UI) initializeButtons() {
	u.helpButton = tview.NewButton("").
		SetLabel("Help [::b][F1[]").
		SetBackgroundColorActivated(buttonFocusColor).
		SetSelectedFunc(func() { u.pages.ShowPage("help page") })

	u.aboutButton = tview.NewButton("").
		SetLabel("About [::b][F2[]").
		SetBackgroundColorActivated(buttonFocusColor).
		SetSelectedFunc(func() { u.pages.ShowPage("about page") })

	u.quitButton = tview.NewButton("").
		SetLabel("Quit [::b][CTRL+Q[]").
		SetBackgroundColorActivated(buttonFocusColor).
		SetSelectedFunc(func() { u.app.Stop() })

	u.updateButton = tview.NewButton("").
		SetLabel("Update Dbase [::b][CTRL+U[]").
		SetBackgroundColorActivated(buttonFocusColor).
		SetSelectedFunc(func() {
			u.startUpdate()
		})
}
