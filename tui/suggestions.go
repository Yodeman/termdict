// Everything related to listing search suggestions

package tui

import (
    "slices"
    "strings"
)

// listSuggestions receives text input from the search input field,
// searches for matching words in the list of words present in the
// dictionary database and then list the matching words in the
// search suggestion box.
func listSuggestions(word string) {
    searchListField.Clear()
    wordsLen := len(DictWords)
    wIdx, _ := slices.BinarySearch(DictWords, word)

    for i := 0; (i < maxMatchWords) && ((i + wIdx) < wordsLen); i++ {
        if strings.HasPrefix(DictWords[i + wIdx], word) {
            searchListField.AddItem(DictWords[i + wIdx], "", 0, nil)
        } else if DictWords[i + wIdx] > word {
            break
        }
    }
}
