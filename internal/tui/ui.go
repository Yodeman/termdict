// Package tui renders the terminal user interface. It owns only widget
// wiring: lookups come from *dict.Service and database updates from the
// Updater implementation, so the package contains no business logic.
package tui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/yodeman/termdict/internal/config"
	"github.com/yodeman/termdict/internal/data"
	"github.com/yodeman/termdict/internal/dict"
)

// Configurations.
const (
	maxMatchWords   = 50 // maximum numbers of search suggestions
	searchGridWidth = 60 // search and suggestions widget width
	commandsWidth   = 13 // width for each command options
	popupWidth      = 80 // message box width
	popupHeight     = 25 // message box height
)

// Updater supplies database update operations to the UI. Implemented by
// *data.Client.
type Updater interface {
	// Update downloads only files that changed on the remote.
	Update(ctx context.Context, progress data.ProgressFn) (updated int, err error)
	// DownloadFull fetches the complete letter-file set, adding every
	// headword missing from the embedded core.
	DownloadFull(ctx context.Context, progress data.ProgressFn) (downloaded int, err error)
}

// message shown upon pressing/clicking help command
const helpMessage = `
                [yellow:blue:b]press escape to exit!
[-:-:-]
Welcome to Terminal Dictionary Help!

Terminal Dictionary was built with [::bu:https://github.com/rivo/tview]tview[:::-]

[::Ub]			General Keys

[::B]Key		            Command
-----------------------------------------------
ctrl+q			Quit the application
ctrl+u          Update dictionary words database
F1              This help
F2			    Details about Terminal Dictionary
tab | shf+tab   Move between widgets (search, suggestions, definition)


[::b]           Dictionary Symbols

[::B]Symbol                 Meaning
-----------------------------------------------
n.              Noun
v.              Verb
v. t.           Transitive verb
v. i.           Intransitive verb
a.              Adjective
adv.            Adverb
prep.           Preposition
pron.           Pronoun
pl.             Plural



                [yellow:blue:b]press escape to exit!
`

// message shown after a successful database update
const updateDoneMsg = `
Done updating database.

Please restart to load newly updated database.

[yellow::b]press escape to exit!
`

// message shown when an update run finds nothing to do
const upToDateMsg = `
Local database is up to date.

[yellow::b]press escape to exit!
`

// message shown after the user cancels a run with Escape
const cancelledMsg = `
[yellow::b]Cancelled.[-:-:-]

Nothing was broken; partially downloaded files resume on the next run.

[yellow::b]press escape to exit!
`

// aboutMessage returns the about-popup content for the given version.
func aboutMessage(version string) string {
	return fmt.Sprintf(`
TermDict v%s

Built with [::bu:https://github.com/rivo/tview]tview[:::-]

[::u:https://github.com/yodeman/termdict] https://github.com/yodeman/termdict[-:-:-]
`, version)
}

// UI wires the dictionary service into a tview application.
type UI struct {
	cfg     config.Paths
	svc     *dict.Service
	updater Updater
	theme   Theme

	app              *tview.Application
	pages            *tview.Pages
	definitionBox    *tview.TextView
	searchGrid       *tview.Grid
	searchInputField *tview.InputField
	searchListField  *tview.List
	commandsGrid     *tview.Grid

	helpPopup    *tview.Grid
	aboutPopup   *tview.Modal
	updatePopup  *tview.Grid
	updateWidget *tview.TextView

	helpButton         *tview.Button
	aboutButton        *tview.Button
	quitButton         *tview.Button
	updateButton       *tview.Button
	updateDbButton     *tview.Button
	downloadFullButton *tview.Button

	updating     atomic.Bool
	updateCtx    context.Context
	cancelUpdate context.CancelFunc
	cancelAction context.CancelFunc
}

// New creates a UI for cfg using svc for lookups and updater for
// database updates. The color theme is resolved from the environment
// (TERMDICT_THEME; NO_COLOR forces a color-free palette).
func New(cfg config.Paths, svc *dict.Service, updater Updater) *UI {
	theme, themeErr := Select(os.Getenv)
	if themeErr != nil {
		log.Printf("Warning: %v.", themeErr)
	}
	return &UI{cfg: cfg, svc: svc, updater: updater, theme: theme}
}

// Run builds the layout and blocks until the user quits.
func (u *UI) Run() error {
	u.updateCtx, u.cancelUpdate = context.WithCancel(context.Background())
	defer u.cancelUpdate()

	// tview primitives capture border/title colors from tview.Styles at
	// construction time, so the theme must be applied first.
	u.theme.applyStyles()
	applyRoundedBorders()

	u.app = tview.NewApplication().EnableMouse(true)
	u.pages = tview.NewPages()

	// root widget
	rootGrid := tview.NewGrid().
		SetBorders(false).
		SetRows(-1, 1).
		SetColumns(searchGridWidth, -1)

	// Shrink the fixed-width search column on narrow terminals so the
	// definition pane stays readable down to ~60 columns.
	u.app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		width, _ := screen.Size()
		columns := searchGridWidth
		if width > 0 && width < searchGridWidth+24 {
			columns = width * 3 / 5
			if columns < 14 {
				columns = 14
			}
		}
		rootGrid.SetColumns(columns, -1)
		return false
	})

	u.initializeDefinitionWidget()
	u.initializeSearchWidgets()
	u.searchInputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			u.searchWord(u.searchInputField.GetText())
		}
	})
	u.searchInputField.SetChangedFunc(func(text string) {
		u.listSuggestions(text)
	})
	u.searchListField.SetChangedFunc(func(_ int, mainText, _ string, _ rune) {
		u.searchWord(mainText)
	})

	// commands
	u.commandsGrid = tview.NewGrid().
		SetBorders(false).
		SetColumns(
			commandsWidth, commandsWidth, commandsWidth+10, commandsWidth, -1)
	u.commandsGrid.SetBackgroundColor(u.theme.Border)

	u.initializePopups()
	u.initializeButtons()

	u.commandsGrid.AddItem(u.helpButton, 0, 0, 1, 1, 0, 0, false)
	u.commandsGrid.AddItem(u.aboutButton, 0, 1, 1, 1, 0, 0, false)
	u.commandsGrid.AddItem(u.updateButton, 0, 2, 1, 1, 0, 0, false)
	u.commandsGrid.AddItem(u.quitButton, 0, 3, 1, 1, 0, 0, false)

	rootGrid.AddItem(u.searchGrid, 0, 0, 1, 1, 0, 0, false)
	rootGrid.AddItem(u.definitionBox, 0, 1, 1, 1, 0, 0, false)
	rootGrid.AddItem(u.commandsGrid, 1, 0, 1, 2, 0, 0, false)

	u.applyFocusAccent(u.searchInputField.Box, u.searchListField.Box, u.definitionBox.Box)

	u.pages.AddPage("root widget", rootGrid, true, true)
	u.pages.AddPage("help page", u.helpPopup, true, false)
	u.pages.AddPage("about page", u.aboutPopup, true, false)
	u.pages.AddPage("update page", u.updatePopup, true, false)

	u.wireWidgetNavigation()
	u.wireGlobalKeys()

	if err := u.app.SetRoot(u.pages, true).SetFocus(u.searchInputField).Run(); err != nil {
		return fmt.Errorf("running interface: %w", err)
	}
	return nil
}

// wireWidgetNavigation lets tab and shift+tab move between widgets.
func (u *UI) wireWidgetNavigation() {
	selections := []*tview.Box{
		u.searchInputField.Box,
		u.searchListField.Box,
		u.definitionBox.Box,
	}
	for idx := range selections {
		box := selections[idx]
		box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyTab:
				u.app.SetFocus(selections[(idx+1)%len(selections)])
				return nil
			case tcell.KeyBacktab:
				u.app.SetFocus(
					selections[(idx+len(selections)-1)%len(selections)])
				return nil
			}
			return event
		})
	}
}

// wireGlobalKeys installs application-wide keybindings.
func (u *UI) wireGlobalKeys() {
	u.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			u.pages.ShowPage("help page")
		case tcell.KeyF2:
			u.pages.ShowPage("about page")
		case tcell.KeyCtrlQ:
			u.app.Stop()
		case tcell.KeyCtrlU:
			u.startUpdate()
		}
		return event
	})
}

func (u *UI) initializeDefinitionWidget() {
	u.definitionBox = tview.NewTextView().
		SetScrollable(true).
		SetDynamicColors(true)
	u.definitionBox.SetBorder(true)
	u.definitionBox.SetTitle("[::bi]Definition")
	u.definitionBox.SetBorderColor(u.theme.Border)
	u.definitionBox.SetChangedFunc(func() {
		u.app.Draw()
	})
}

// searchWord looks word up in the dictionary service and renders the
// result in the definition box. Lookup misses and render errors are
// surfaced in the box instead of terminating the program.
func (u *UI) searchWord(word string) {
	writer := new(strings.Builder)

	entity, found := u.svc.Lookup(word)
	switch {
	case !found:
		fmt.Fprintf(writer, "%s",
			dict.NotFoundMessage(strings.TrimSpace(word), u.theme.Tag(u.theme.Warning)))
		if suggestions := u.svc.Fuzzy(word, dict.MaxFuzzySuggestions); len(suggestions) > 0 {
			fmt.Fprintf(writer, "\n%sDid you mean:[::-] %s\n",
				u.theme.Tag(u.theme.Warning), strings.Join(suggestions, ", "))
		}
	default:
		if err := dict.RenderTUI(writer, entity); err != nil {
			log.Printf("rendering definition of %q: %v", word, err)
			writer.Reset()
			writer.WriteString("[red::b]Something went wrong while showing this definition.[red::-]")
		}
	}

	u.definitionBox.SetText(writer.String())
}

// listSuggestions fills the suggestion list with words matching text.
func (u *UI) listSuggestions(text string) {
	u.searchListField.Clear()
	for _, suggestion := range u.svc.Suggest(text, maxMatchWords) {
		u.searchListField.AddItem(suggestion, "", 0, nil)
	}
}

// startUpdate launches a single background database update and shows
// its progress in the update popup. Concurrent invocations are ignored.
func (u *UI) startUpdate() {
	u.runUpdateAction("Updating database", u.updater.Update)
}

// startDownloadFull launches a full-dictionary download in the
// background. Concurrent invocations are ignored.
func (u *UI) startDownloadFull() {
	u.runUpdateAction("Downloading full dictionary", u.updater.DownloadFull)
}

// progressText renders the popup body during a download: title,
// counts, a bar and the current file.
func progressText(title string, done, total int, current string) string {
	const barWidth = 24
	filled := 0
	if total > 0 {
		filled = done * barWidth / total
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("%s… (%d/%d)\n[%s]\n%s\n\n[dim::b]press Esc to cancel[-::-]",
		title, done, total, bar, current)
}

// runUpdateAction guards against concurrent runs, shows the update
// popup and reports progress/result through updateWidget.
func (u *UI) runUpdateAction(runningText string, action func(context.Context, data.ProgressFn) (int, error)) {
	if !u.updating.CompareAndSwap(false, true) {
		u.pages.ShowPage("update page")
		return
	}

	ctx, cancel := context.WithCancel(u.updateCtx)
	u.cancelAction = cancel
	go func() {
		defer cancel()

		updated, err := action(ctx, func(done, total int, current string) {
			u.app.QueueUpdateDraw(func() {
				if u.updating.Load() {
					u.updateWidget.SetText(
						progressText(runningText, done, total, current))
				}
			})
		})

		u.app.QueueUpdateDraw(func() {
			defer func() { u.updating.Store(false) }()
			switch {
			case errors.Is(err, context.Canceled):
				u.updateWidget.SetText(cancelledMsg)
			case err == nil && updated == 0:
				u.updateWidget.SetText(upToDateMsg)
			case err == nil:
				u.updateWidget.SetText(updateDoneMsg)
			default:
				var fetchErr *data.FetchError
				if errors.As(err, &fetchErr) {
					log.Printf("database update failures:\n%v\n", err)
					u.updateWidget.SetText(fmt.Sprintf(
						"Update finished with errors.\n%d file(s) failed.\n\nDetails were logged to stderr.\n\n[yellow::b]press escape to exit!",
						len(fetchErr.Failures)))
					return
				}
				log.Printf("database update failed:\n%v\n", err)
				u.updateWidget.SetText(fmt.Sprintf(
					"Update failed.\n\n%v\n\n[yellow::b]press escape to exit!", err))
			}
		})
	}()

	u.pages.ShowPage("update page")
}
