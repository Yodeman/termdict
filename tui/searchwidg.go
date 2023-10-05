// Everything related to search and suggestions widgets

package tui

import (
    "unicode"

    "github.com/rivo/tview"
)

// initializeSearchWidgets initializes tui widgets for search and
// suggestions.
func initializeSearchWidgets() {
    searchGrid = tview.NewGrid().
        SetBorders(false).
        SetRows(3, -1)

    searchInputField = tview.NewInputField().
        SetPlaceholder("enter a word...").
        SetFieldWidth(searchGridWidth).
        SetFieldTextColor(inputFieldColor).
        SetPlaceholderTextColor(inputFieldColor).
        SetAcceptanceFunc(func(text string, ch rune) bool {
            return unicode.IsPrint(ch)
        })
    searchInputField.SetBorder(true).SetBorderColor(borderColor)
    searchInputField.SetTitle("[::bi]search").SetTitleAlign(tview.AlignLeft)

    searchListField = tview.NewList()
    searchListField.SetBorder(true)
    searchListField.SetBorderColor(borderColor)
    searchListField.SetTitle("[::bi]suggestions").SetTitleAlign(tview.AlignLeft)

    searchGrid.AddItem(searchInputField, 0, 0, 1, 1, 0, 0, false)
    searchGrid.AddItem(searchListField, 1, 0, 1, 1, 0, 0, false)
}
