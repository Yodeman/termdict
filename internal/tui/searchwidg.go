package tui

import (
	"unicode"

	"github.com/rivo/tview"
)

// initializeSearchWidgets initializes the search and suggestion widgets.
func (u *UI) initializeSearchWidgets() {
	u.searchGrid = tview.NewGrid().
		SetBorders(false).
		SetRows(3, -1)

	u.searchInputField = tview.NewInputField().
		SetPlaceholder("enter a word...").
		SetFieldWidth(searchGridWidth).
		SetFieldTextColor(u.theme.PrimaryText).
		SetPlaceholderTextColor(u.theme.Muted).
		SetAcceptanceFunc(func(_ string, ch rune) bool {
			return unicode.IsPrint(ch)
		})
	u.searchInputField.SetBorder(true).SetBorderColor(u.theme.Border)
	u.searchInputField.SetTitle("[::bi]search").SetTitleAlign(tview.AlignLeft)

	u.searchListField = tview.NewList()
	u.searchListField.SetBorder(true)
	u.searchListField.SetBorderColor(u.theme.Border)
	u.searchListField.SetTitle("[::bi]suggestions").SetTitleAlign(tview.AlignLeft)

	u.searchGrid.AddItem(u.searchInputField, 0, 0, 1, 1, 0, 0, false)
	u.searchGrid.AddItem(u.searchListField, 1, 0, 1, 1, 0, 0, false)
}
