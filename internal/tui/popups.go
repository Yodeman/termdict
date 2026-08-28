package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/yodeman/termdict/internal/config"
)

// trapTabKeys swallows Tab/Backtab on popup text views. tview's
// TextView fires DoneFunc for those keys exactly as for Escape, so an
// un-trapped Tab silently CLOSED the popup — which read as "Esc stops
// working" (QA v2 issue 3). Tab inside popups is a deliberate no-op:
// the help and update panes have a single focusable widget (Esc to
// leave) and the About modal cycles its own buttons via tview's
// built-in Modal handling.
func trapTabKeys(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
		return nil
	}
	return event
}

// initializePopups initializes the popup widgets used to display
// messages.
func (u *UI) initializePopups() {
	helpWidget := tview.NewTextView().
		SetDoneFunc(func(_ tcell.Key) {
			u.hidePopup("help page")
		}).
		SetText(helpMessage(u.theme)).
		SetDynamicColors(true)
	helpWidget.SetInputCapture(trapTabKeys)
	helpWidget.SetBorder(true)
	helpWidget.SetTitle("[::bi]Help — [Esc[] close")
	helpWidget.SetBackgroundColor(u.theme.Background)

	u.helpPopup = tview.NewGrid().
		SetBorders(false).
		SetColumns(0, popupWidth, 0).
		SetRows(0, popupHeight, 0).
		AddItem(helpWidget, 1, 1, 1, 1, 0, 0, true)

	u.aboutPopup = tview.NewModal().
		AddButtons([]string{"close"}).
		SetText(aboutMessage(config.AppVersion)).
		SetDoneFunc(func(_ int, _ string) {
			u.hidePopup("about page")
		})

	u.updateWidget = tview.NewTextView().
		SetDoneFunc(func(_ tcell.Key) {
			u.hidePopup("update page")
		}).
		SetText("Updating database...").
		SetChangedFunc(func() { u.app.Draw() }).
		SetDynamicColors(true)
	u.updateWidget.SetInputCapture(trapTabKeys)
	u.updateWidget.SetBorder(true)
	u.updateWidget.SetTitle("[::bi]Database — [Esc[] close / cancel")
	u.updateWidget.SetBackgroundColor(u.theme.Background)

	// Action row shown under the update log: incremental update and
	// full-dictionary download (the embedded core covers only the most
	// frequent headwords).
	u.updateDbButton = tview.NewButton("Update Dbase").
		SetBackgroundColorActivated(u.theme.Accent).
		SetSelectedFunc(func() { u.startUpdate() })
	u.downloadFullButton = tview.NewButton("Download Full Dictionary").
		SetBackgroundColorActivated(u.theme.Accent).
		SetSelectedFunc(func() { u.startDownloadFull() })

	buttonsGrid := tview.NewGrid().
		SetBorders(false).
		SetColumns(0, 18, 2, 30, 0).
		AddItem(tview.NewBox(), 0, 0, 1, 1, 0, 0, false).
		AddItem(u.updateDbButton, 0, 1, 1, 1, 0, 0, true).
		AddItem(tview.NewBox(), 0, 2, 1, 1, 0, 0, false).
		AddItem(u.downloadFullButton, 0, 3, 1, 1, 0, 0, false)

	updateStack := tview.NewGrid().
		SetBorders(false).
		SetRows(-1, 1).
		AddItem(u.updateWidget, 0, 0, 1, 1, 0, 0, false).
		AddItem(buttonsGrid, 1, 0, 1, 1, 0, 0, true)
	updateStack.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			if u.updating.Load() && u.cancelAction != nil {
				u.cancelAction() // first Esc cancels; result pane explains
				return nil
			}
			u.hidePopup("update page")
			return nil
		}
		return event
	})

	u.updatePopup = tview.NewGrid().
		SetBorders(false).
		SetColumns(0, popupWidth, 0).
		SetRows(0, popupHeight+3, 0).
		AddItem(updateStack, 1, 1, 1, 1, 0, 0, true)
}
