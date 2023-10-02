// Everything related to listing search suggestions

package search

import (
    "slices"
    "strings"

    "github.com/rivo/tview"
)

func ListSuggestions(
    maxMatch int,
    word string,
    words []string,
    searchListField *tview.List,
) {

    searchListField.Clear()
    wordsLen := len(words)
    wIdx, _ := slices.BinarySearch(words, word)

    for i := 0; (i < maxMatch) && ((i + wIdx) < wordsLen); i++ {
        if strings.HasPrefix(words[i + wIdx], word) {
            searchListField.AddItem(words[i + wIdx], "", 0, nil)
        } else if words[i + wIdx] > word {
            break
        }
    }
}
