// Everything related to popups

package tui

import (
    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
)

func initializePopups() {
    helpWidget := tview.NewTextView().
        SetDoneFunc(func(key tcell.Key){
            pages.HidePage("help page")
            app.SetFocus(searchInputField)
        }).
        SetText(helpMessage).
        SetDynamicColors(true)
    helpWidget.SetBorder(true)
    helpWidget.SetBackgroundColor(borderColor)
    helpWidget.SetDisabled(true)

    helpPopup = tview.NewGrid().
        SetBorders(false).
        SetColumns(0, popupWidth, 0).
        SetRows(0, popupHeight, 0).
        AddItem(helpWidget, 1, 1, 1, 1, 0, 0, true)

    aboutPopup = tview.NewModal().
        AddButtons([]string{"close"}).
        SetText(aboutMessage).
        SetDoneFunc(func(buttonIdx int, buttonLbl string){
            pages.HidePage("about page")
            app.SetFocus(searchInputField)
        })

    updateWidget = tview.NewTextView().
        SetDoneFunc(func(key tcell.Key){
            pages.HidePage("update page")
            app.SetFocus(searchInputField)
        }).
        SetText("Updating database...").
        SetChangedFunc(func(){app.Draw()}).
        SetDynamicColors(true)
    updateWidget.SetBorder(true)
    updateWidget.SetBackgroundColor(borderColor)
    updateWidget.SetDisabled(true)

    updatePopup = tview.NewGrid().
        SetBorders(false).
        SetColumns(0, popupWidth, 0).
        SetRows(0, popupHeight, 0).
        AddItem(updateWidget, 1, 1, 1, 1, 0, 0, true)

}
