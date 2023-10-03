// Everything related to definition box widget

package tui

import (
    "github.com/rivo/tview"
)

func initializeDefinitionWidget() {
    definitionBox = tview.NewTextView().
        SetScrollable(true).
        SetDynamicColors(true)
    definitionBox.SetBorder(true)
    definitionBox.SetTitle("[::bi]Definition")
    definitionBox.SetBorderColor(borderColor)
}
