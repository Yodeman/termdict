// TUI interface for terminal dictionary.

package tui

import (
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/rivo/tview"
    "github.com/gdamore/tcell/v2"
)

// configurations
const (
    maxMatchWords           = 50
    searchGridWidth         = 60
    commandsWidth           = 13
    popupWidth              = 80
    popupHeight             = 25
    borderColor             = tcell.ColorBlue
    inputFieldColor         = tcell.ColorWhite
    buttonFocusColor        = tcell.ColorYellow
)

var (
    app                     *tview.Application
    pages                   *tview.Pages
    definitionBox           *tview.TextView
    searchGrid              *tview.Grid
    searchInputField        *tview.InputField
    searchListField         *tview.List
    commandsGrid            *tview.Grid
    helpPopup               *tview.Grid
    aboutPopup              *tview.Modal
    updatePopup             *tview.Grid
    helpButton              *tview.Button
    aboutButton             *tview.Button
    quitButton              *tview.Button
    updateButton            *tview.Button
    updateWidget            *tview.TextView

)

type Definition struct {
    PartOfSpeech    string `json:"part_of_speech"`
    WordDefinition  string  `json:"definition"`
}
type DictEntity struct {
    Word            string          `json:"word"`
    Spellings       []string        `json:"alternate_spellings,omitempty"`
    WordDefinitions []Definition    `json:"definitions"`
}

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
const updateDoneMsg = `
Done updating database.

Please restart to load newly updated database.
`

var DbaseDir string
var err error
func init() {
    DbaseDir, err = os.UserHomeDir()
    if err != nil {
        log.Fatalf("Error accessing home directory.\n%v\n", err)
    }

    DbaseDir = strings.Join(
        []string{DbaseDir, ".termdict", "dbase", "json"},
        string(os.PathSeparator))
    DbaseDir += string(os.PathSeparator)
}

func RenderLayout(dbase map[string]DictEntity, words []string) {
    // app
    app = tview.NewApplication().EnableMouse(true)
    pages = tview.NewPages()

    // root widget
    rootGrid := tview.NewGrid().
        SetBorders(false).
        SetRows(-1, 1).
        SetColumns(searchGridWidth, -1)

    // definition widget
    initializeDefinitionWidget()
    definitionBox.SetChangedFunc(func(){
        app.Draw()
    })

    // search widgets
    initializeSearchWidgets()
    searchInputField.SetDoneFunc(func(key tcell.Key){
        switch key {
            case tcell.KeyEnter:
                searchWord(searchInputField.GetText(), dbase)
        }
    })
    searchInputField.SetChangedFunc(func(text string){
        listSuggestions(maxMatchWords, text, words)
    })
    
    searchListField.SetChangedFunc(func(idx int, mainText, s string, r rune){
        searchWord(mainText, dbase)
    })

    // commands
    commandsGrid = tview.NewGrid().
        SetBorders(false).
        SetColumns(
            commandsWidth, commandsWidth, commandsWidth+10, commandsWidth, -1)
    commandsGrid.SetBackgroundColor(borderColor)

    initializePopups()
    initializeButtons()

    commandsGrid.AddItem(helpButton, 0, 0, 1, 1, 0, 0, false)
    commandsGrid.AddItem(aboutButton, 0, 1, 1, 1, 0, 0, false)
    commandsGrid.AddItem(updateButton, 0, 2, 1, 1, 0, 0, false)
    commandsGrid.AddItem(quitButton, 0, 3, 1, 1, 0, 0, false)


    rootGrid.AddItem(searchGrid, 0, 0, 1, 1, 0, 0, false)
    rootGrid.AddItem(definitionBox, 0, 1, 1, 1, 0, 0, false)
    rootGrid.AddItem(commandsGrid, 1, 0, 1, 2, 0, 0, false)

    pages.AddPage("root widget", rootGrid, true, true)
    pages.AddPage("help page", helpPopup, true, false)
    pages.AddPage("about page", aboutPopup, true, false)
    pages.AddPage("update page", updatePopup, true, false)

    // moving between widgets
    selections := []*tview.Box{
                    searchInputField.Box,
                    searchListField.Box,
                    definitionBox.Box,
                }
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
            case tcell.KeyCtrlU:
                pages.ShowPage("update page")
                go func () {
                    err = FetchDbase()
                    if err != nil {
                        updateWidget.SetText(fmt.Sprintf("%s", err))
                    } else {
                        updateWidget.SetText(updateDoneMsg)
                    }
                } ()
        }
        return event
    })

    if err := app.SetRoot(pages, true).SetFocus(searchInputField).Run(); err != nil {
        panic(err)
    }
}
