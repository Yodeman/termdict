// TUI interface for terminal dictionary.

package tui

import (
    "github.com/rivo/tview"
    "github.com/gdamore/tcell/v2"
)

const (
    searchGridWidth         = 60
    commandsWidth           = 13
    popupWidth              = 80
    popupHeight             = 25
    borderColor             = tcell.ColorBlue
    inputFieldColor         = tcell.ColorWhite
    buttonFocusColor        = tcell.ColorYellow
)

const helpMessage = `
Welcome to Terminal Dictionary Help!

Terminal Dictionary was built with [::bu:https://github.com/rivo/tview]tview

[::Ub]			General Keys

[::B]Key		            Command
-----------------------------------------------
ctrl+c			Quit the application
F1              This help
F2			    Details about Terminal Dictionary



[yellow:blue:b]press escape to exit!
`
const aboutMessage = `
term-dict v0.1.0

Built with [::bu:https://github.com/rivo/tview]tview

[::u:https://github.com/Yodeman/term-dict] https://github.com/Yodeman/term-dict
`

func RenderLayout() {
    // app
    app := tview.NewApplication().EnableMouse(true)
    pages := tview.NewPages()

    // root widget
    rootGrid := tview.NewGrid().
        SetBorders(false).
        SetRows(-1, 1).
        SetColumns(searchGridWidth, -1)

    // definition widget
    definitionBox := defComponent()

    // search widgets
    searchGrid, searchInputField, searchListField := searchComponents()

    // commands
    commandsGrid := tview.NewGrid().
        SetBorders(false).
        SetColumns(commandsWidth, commandsWidth, commandsWidth, -1)
    commandsGrid.SetBackgroundColor(borderColor)

    helpPopup := tview.NewTextView().
        SetDoneFunc(func(key tcell.Key){
            pages.HidePage("help page")
            app.SetFocus(searchInputField)
        }).
        SetText(helpMessage).
        SetDynamicColors(true)
    helpPopup.SetBorder(true)
    helpPopup.SetBackgroundColor(borderColor)
    helpPopup.SetDisabled(true)

    helpModal := tview.NewGrid().
        SetBorders(false).
        SetColumns(0, popupWidth, 0).
        SetRows(0, popupHeight, 0).
        AddItem(helpPopup, 1, 1, 1, 1, 0, 0, true)

    aboutPopup := tview.NewModal().
        AddButtons([]string{"close"}).
        SetText(aboutMessage).
        SetDoneFunc(func(buttonIdx int, buttonLbl string){
            pages.HidePage("about page")
            app.SetFocus(searchInputField)
        })

    helpButton := tview.NewButton("").
        SetLabel("Help [::b][F1[]").
        SetBackgroundColorActivated(buttonFocusColor).
        SetSelectedFunc(func(){pages.ShowPage("help page")})

    aboutButton := tview.NewButton("").
        SetLabel("About [::b][F2[]").
        SetBackgroundColorActivated(buttonFocusColor).
        SetSelectedFunc(func(){pages.ShowPage("about page")})

    quitButton := tview.NewButton("").
        SetLabel("Quit [::b][CTRL+C[]").
        SetBackgroundColorActivated(buttonFocusColor).
        SetSelectedFunc(func(){app.Stop()})

    commandsGrid.AddItem(helpButton, 0, 0, 1, 1, 0, 0, false)
    commandsGrid.AddItem(aboutButton, 0, 1, 1, 1, 0, 0, false)
    commandsGrid.AddItem(quitButton, 0, 2, 1, 1, 0, 0, false)


    rootGrid.AddItem(searchGrid, 0, 0, 1, 1, 0, 0, false)
    rootGrid.AddItem(definitionBox, 0, 1, 1, 1, 0, 0, false)
    rootGrid.AddItem(commandsGrid, 1, 0, 1, 2, 0, 0, false)

    pages.AddPage("root widget", rootGrid, true, true)
    pages.AddPage("help page", helpModal, true, false)
    pages.AddPage("about page", aboutPopup, true, false)

    // moving between widgets
    selections := []*tview.Box{searchInputField.Box, searchListField.Box, definitionBox.Box}
    for i, box := range selections {
        (func(idx int) {
            box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
                switch event.Key() {
                    case tcell.KeyTab:
                        app.SetFocus(selections[(idx+1)%len(selections)])
                        return nil
                    case tcell.KeyBacktab:
                        app.SetFocus(selections[(idx+len(selections)-1)%len(selections)])
                        return nil
                }
                return event
            })
        })(i)
    }

    app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        switch event.Key() {
            case tcell.KeyF1:
                pages.ShowPage("help page")
            case tcell.KeyF2:
                pages.ShowPage("about page")
        }
        return event
    })

    if err := app.SetRoot(pages, true).SetFocus(searchInputField).Run(); err != nil {
        panic(err)
    }
}
