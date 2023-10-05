// Everythin related to buttons

package tui

import (
    "fmt"

    "github.com/rivo/tview"
)

// initializeButtons initializes the tui commands buttons
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

    updateButton = tview.NewButton("").
        SetLabel("Update Dbase [::b][CTRL+U[]").
        SetBackgroundColorActivated(buttonFocusColor).
        SetSelectedFunc(func(){
            pages.ShowPage("update page")
            go func () {
                err := UpdateDbase()
                if err != nil {
                    updateWidget.SetText(fmt.Sprintf("%s", err))
                } else {
                    updateWidget.SetText(updateDoneMsg)
                }
            } ()
        })
}
