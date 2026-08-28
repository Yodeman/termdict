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
	"sync"
	"sync/atomic"
	"time"

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

// helpMessage renders the F1 pane: key table first (mirroring the
// footer hints), then the part-of-speech legend, then data hints.
// Short lines and plain words — the audience includes kids.
func helpMessage(theme Theme) string {
	header := theme.TagStyle(theme.Header, "b")
	hint := theme.TagStyle(theme.Muted, "i")
	reset := "[-:-:-]"

	return fmt.Sprintf(`
%[1]s Help %[3]s  %[2]shint: press Esc to close%[3]s

KEYS
  type             search as you type
  enter            define the word
  tab / shft+tab   move between panes
  F1               this help
  F2               about TermDict
  ctrl+u           update or download the dictionary
  ctrl+q           quit

PARTS OF SPEECH
  Definitions are grouped under full labels
  (Noun, Adjective, Verb (transitive), ...).

%[2]sctrl+u and "Download Full Dictionary" adds the complete word list.%[3]s
%[2]sBuilt with tview · github.com/yodeman/termdict%[3]s
`, header, hint, reset)
}

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

	footerGrid   *tview.Grid
	footerHints  *tview.TextView
	footerStatus *tview.TextView

	currentSuggestions []string // clean words behind the styled list items
	buttonsHidden      bool     // button bar hidden on narrow terminals
	spinnerActive      atomic.Bool
	pageReturnFocus    tview.Primitive // focus to restore when a popup closes
	lastWord           string          // word currently shown in the definition pane
	lastRenderWidth    int             // terminal width of the last definition render

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
		SetRows(-1, 1, 1).
		SetColumns(searchGridWidth, -1)

	// Responsive layout: shrink the fixed-width search column on narrow
	// terminals, and below 70 columns hide the button bar entirely —
	// the persistent footer carries the key hints instead.
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
		narrow := width > 0 && width < 70
		if narrow != u.buttonsHidden {
			u.buttonsHidden = narrow
			if narrow {
				rootGrid.RemoveItem(u.commandsGrid)
			} else {
				rootGrid.AddItem(u.commandsGrid, 1, 0, 1, 2, 0, 0, false)
			}
		}
		// The definition renderer wraps to the pane width, so a width
		// change must re-render the current content. The re-render is
		// queued (SetText from inside a draw would re-enter app.Draw).
		if width > 0 && width != u.lastRenderWidth {
			widthChanged := u.lastRenderWidth != 0
			u.lastRenderWidth = width
			if widthChanged {
				u.app.QueueUpdateDraw(func() { u.rerenderDefinition() })
			}
		}
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
	u.searchListField.SetChangedFunc(func(idx int, mainText, secondaryText string, _ rune) {
		// List items carry themed markup; the clean word for the lookup
		// lives in currentSuggestions, keyed by row index. The trailing
		// "… N more" row has no clean word and is skipped.
		_ = mainText
		_ = secondaryText
		if idx < len(u.currentSuggestions) {
			u.searchWord(u.currentSuggestions[idx])
		}
	})

	// commands
	u.commandsGrid = tview.NewGrid().
		SetBorders(false).
		SetColumns(
			commandsWidth, commandsWidth, commandsWidth+10, commandsWidth, -1)
	u.commandsGrid.SetBackgroundColor(u.theme.Border)

	u.initializePopups()
	u.initializeButtons()
	u.initializeFooter()

	u.commandsGrid.AddItem(u.helpButton, 0, 0, 1, 1, 0, 0, false)
	u.commandsGrid.AddItem(u.aboutButton, 0, 1, 1, 1, 0, 0, false)
	u.commandsGrid.AddItem(u.updateButton, 0, 2, 1, 1, 0, 0, false)
	u.commandsGrid.AddItem(u.quitButton, 0, 3, 1, 1, 0, 0, false)

	rootGrid.AddItem(u.searchGrid, 0, 0, 1, 1, 0, 0, false)
	rootGrid.AddItem(u.definitionBox, 0, 1, 1, 1, 0, 0, false)
	rootGrid.AddItem(u.commandsGrid, 1, 0, 1, 2, 0, 0, false)
	rootGrid.AddItem(u.footerGrid, 2, 0, 1, 2, 0, 0, false)

	u.applyFocusAccent(u.searchInputField.Box, u.searchListField.Box, u.definitionBox.Box)

	u.pages.AddPage("root widget", rootGrid, true, true)
	u.pages.AddPage("help page", u.helpPopup, true, false)
	u.pages.AddPage("about page", u.aboutPopup, true, false)
	u.pages.AddPage("update page", u.updatePopup, true, false)

	u.wireWidgetNavigation()
	u.wireGlobalKeys()
	u.showWelcome()
	u.setFooterReady()

	if err := u.app.SetRoot(u.pages, true).SetFocus(u.searchInputField).Run(); err != nil {
		return fmt.Errorf("running interface: %w", err)
	}
	return nil
}

// wireWidgetNavigation lets tab and shift+tab rotate focus through
// search -> suggestions -> definition and back.
//
// The cycle focuses the PRIMITIVES, never their bare Box wrappers:
// tview's InputField mirrors InputField.HasFocus() into its inner text
// area on every draw (inputfield.go: textArea.hasFocus = i.HasFocus()),
// and only InputField.Blur() clears that flag. Focusing the raw Box
// leaves the flag latched, so InputField.HasFocus() stays true and the
// grid's focus delegation (input field listed before the suggestions
// list) permanently routes every Tab back to the search pane.
func (u *UI) wireWidgetNavigation() {
	panes := []tview.Primitive{u.searchInputField, u.searchListField, u.definitionBox}
	for idx := range panes {
		// tview's chainable setters return *Box, so the capture-target
		// interface must match that exact signature.
		capturer := panes[idx].(interface {
			SetInputCapture(func(*tcell.EventKey) *tcell.EventKey) *tview.Box
		})
		capturer.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyTab:
				u.app.SetFocus(panes[(idx+1)%len(panes)])
				return nil
			case tcell.KeyBacktab:
				u.app.SetFocus(panes[(idx+len(panes)-1)%len(panes)])
				return nil
			}
			return event
		})
	}
}

// showPopup opens a named page, remembering the current focus so
// hidePopup can return the user to wherever they were (QA v2 issue 3).
//
// The popup must also RECEIVE focus: tview's Pages routes key events to
// the page containing the focused primitive, so a visible-but-unfocused
// popup silently sends Esc/Tab to the main panes beneath it — Esc could
// never reach the popup's close handler and Tab desynced the pane cycle
// under the overlay. GetPage + SetFocus lets each page's own focus flags
// pick the right widget (help/update text views, the About modal's
// button form).
func (u *UI) showPopup(name string) {
	u.pageReturnFocus = u.app.GetFocus()
	u.pages.ShowPage(name)
	if page := u.pages.GetPage(name); page != nil {
		u.app.SetFocus(page)
	}
}

// hidePopup closes a named page and restores the focus saved when it
// opened, falling back to the search field.
func (u *UI) hidePopup(name string) {
	u.pages.HidePage(name)
	if u.pageReturnFocus != nil {
		u.app.SetFocus(u.pageReturnFocus)
		u.pageReturnFocus = nil
	} else {
		u.app.SetFocus(u.searchInputField)
	}
}

// wireGlobalKeys installs application-wide keybindings.
func (u *UI) wireGlobalKeys() {
	u.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			u.showPopup("help page")
		case tcell.KeyF2:
			u.showPopup("about page")
		case tcell.KeyCtrlQ:
			u.app.Stop()
		case tcell.KeyCtrlU:
			u.startUpdate()
		}
		return event
	})
}

// initializeFooter builds the persistent status row: key hints on the
// left, contextual state (word count / update progress) on the right.
func (u *UI) initializeFooter() {
	u.footerHints = tview.NewTextView().
		SetText(" type to search · enter define · tab panes · F1 help · ctrl+u update · ctrl+q quit").
		SetTextAlign(tview.AlignLeft)
	u.footerStatus = tview.NewTextView().SetTextAlign(tview.AlignRight)
	u.footerGrid = tview.NewGrid().
		SetBorders(false).
		SetColumns(-2, -1).
		AddItem(u.footerHints, 0, 0, 1, 1, 0, 0, false).
		AddItem(u.footerStatus, 0, 1, 1, 1, 0, 0, false)
}

// setFooterStatus replaces the right-hand footer text. Safe to call
// from any goroutine when wrapped in app.QueueUpdateDraw.
func (u *UI) setFooterStatus(text string) {
	if u.footerStatus != nil {
		u.footerStatus.SetText(text)
	}
}

// setFooterReady restores the idle status line.
func (u *UI) setFooterReady() {
	u.setFooterStatus(fmt.Sprintf("%d words ready · theme: %s ", u.svc.Len(), u.theme.Name))
}

// renderOptions derives the themed markup snippets for dict.RenderTUI.
// The part-of-speech badge reverses the theme colors (theme background
// text on an accent chip) so the badge reads as a distinct element, not
// as body text.
func (u *UI) renderOptions() dict.RenderOptions {
	defWidth := 0
	if u.lastRenderWidth > 0 {
		searchCol := searchGridWidth
		if u.lastRenderWidth < searchGridWidth+24 {
			searchCol = u.lastRenderWidth * 3 / 5
		}
		defWidth = u.lastRenderWidth - searchCol - 2 // pane borders
	}

	return dict.RenderOptions{
		HeaderTag:      u.theme.TagStyle(u.theme.Header, "b"),
		AccentTag:      u.theme.TagStyle(u.theme.Accent, "b"),
		MutedTag:       u.theme.TagStyle(u.theme.Muted, ""),
		MutedItalicTag: u.theme.TagStyle(u.theme.Muted, "i"),
		ResetTag:       "[-:-:-]",
		Boxed:          !u.buttonsHidden && defWidth >= 24,
		Width:          defWidth,
	}
}

func (u *UI) initializeDefinitionWidget() {
	u.definitionBox = tview.NewTextView().
		SetScrollable(true).
		SetWrap(false). // the renderer wraps to the pane width itself
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
	u.lastWord = strings.TrimSpace(word)
	u.rerenderDefinition()
}

// rerenderDefinition redraws the definition pane for the current word
// at the current terminal width (used on first lookup and on resize).
func (u *UI) rerenderDefinition() {
	writer := new(strings.Builder)

	if u.lastWord == "" {
		u.showWelcome()
		return
	}

	entity, found := u.svc.Lookup(u.lastWord)
	switch {
	case !found:
		fmt.Fprintf(writer, "%s",
			dict.NotFoundMessage(u.lastWord, u.theme.Tag(u.theme.Warning)))
		if suggestions := u.svc.Fuzzy(u.lastWord, dict.MaxFuzzySuggestions); len(suggestions) > 0 {
			fmt.Fprintf(writer, "\n%sDid you mean:[::-] %s\n",
				u.theme.Tag(u.theme.Warning), strings.Join(suggestions, ", "))
		}
	default:
		if err := dict.RenderTUI(writer, entity, u.renderOptions()); err != nil {
			log.Printf("rendering definition of %q: %v", u.lastWord, err)
			writer.Reset()
			writer.WriteString("[red::b]Something went wrong while showing this definition.[red::-]")
		}
	}

	u.definitionBox.SetText(writer.String())
}

// listSuggestions fills the suggestion list with words matching text,
// emphasizing the typed prefix in accent color and appending a muted
// truncation hint when the cap hides further matches.
func (u *UI) listSuggestions(text string) {
	u.searchListField.Clear()

	suggestions := u.svc.Suggest(text, maxMatchWords)
	u.currentSuggestions = suggestions

	prefix := strings.ToLower(strings.TrimSpace(text))
	var accentTag string
	if prefix != "" {
		accentTag = u.theme.TagStyle(u.theme.Accent, "")
	}
	for _, suggestion := range suggestions {
		main := suggestion
		if accentTag != "" && strings.HasPrefix(suggestion, prefix) {
			main = accentTag + suggestion[:len(prefix)] + "[-::-]" + suggestion[len(prefix):]
		}
		u.searchListField.AddItem(main, "", 0, nil)
	}

	if total := u.svc.CountPrefix(text); total > len(suggestions) {
		u.searchListField.AddItem(
			u.theme.TagStyle(u.theme.Muted, "i")+
				fmt.Sprintf("… %d more — keep typing", total-len(suggestions))+
				"[-::-]", "", 0, nil)
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

// spinnerFrames is the braille spinner used while the update channel
// is being checked (before per-file progress is known). Paired with
// explanatory text at all times — color/motion never carries meaning
// alone.
const spinnerFrames = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"

// startSpinner animates the update popup with a braille spinner until
// stop is called (idempotent; the returned wait guarantees the last
// tick has drained when stop returns). footerStatus mirrors the state.
func (u *UI) startSpinner(prefix string) (stop func()) {
	u.spinnerActive.Store(true)
	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		frames := []rune(spinnerFrames)
		for i := 0; ; i++ {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				frame := frames[i%len(frames)]
				u.app.QueueUpdateDraw(func() {
					if !u.spinnerActive.Load() || !u.updating.Load() {
						return
					}
					u.updateWidget.SetText(fmt.Sprintf(
						"%s… %c\nChecking for updates…\n\n[dim::b]press Esc to cancel[-::-]",
						prefix, frame))
					u.setFooterStatus(fmt.Sprintf("%s… ", prefix))
				})
			}
		}
	}()

	var once sync.Once
	return func() {
		once.Do(func() {
			u.spinnerActive.Store(false)
			close(stopCh)
			<-done
		})
	}
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

		stopSpinner := u.startSpinner(runningText)
		var spinnerOnce sync.Once
		updated, err := action(ctx, func(done, total int, current string) {
			spinnerOnce.Do(func() { u.spinnerActive.Store(false) })
			u.app.QueueUpdateDraw(func() {
				if u.updating.Load() {
					u.updateWidget.SetText(
						progressText(runningText, done, total, current))
					u.setFooterStatus(fmt.Sprintf("%s… %d/%d ", runningText, done, total))
				}
			})
		})
		stopSpinner() // drain the goroutine before drawing the result

		u.app.QueueUpdateDraw(func() {
			defer func() {
				u.updating.Store(false)
				u.setFooterReady()
			}()
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
