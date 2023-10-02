// Everything related to definition box widget

package tui

import (
    "github.com/rivo/tview"
)

func defComponent() *tview.TextView {
    definitionBox := tview.NewTextView().
        SetScrollable(true)
    definitionBox.SetBorder(true)
    definitionBox.SetTitle("[::bi]Definition")
    definitionBox.SetBorderColor(borderColor)

    return definitionBox
}
