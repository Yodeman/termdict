// Everythin related to buttons

package tui

import (
    "github.com/rivo/tview"
)

func initializeButtons() {
    helpButton = tview.NewButton("").
        SetLabel("Help [::b][F1[]").
        SetBackgroundColorActivated(buttonFocusColor).
        SetSelectedFunc(func(){pages.ShowPage("help page")})

    aboutButton = tview.NewButton("").
        SetLabel("About [::b][F2[]").
        SetBackgroundColorActivated(buttonFocusColor).
        SetSelectedFunc(func(){pages.ShowPage("about page")})

    quitButton = tview.NewButton("").
        SetLabel("Quit [::b][CTRL+C[]").
        SetBackgroundColorActivated(buttonFocusColor).
        SetSelectedFunc(func(){app.Stop()})
}
