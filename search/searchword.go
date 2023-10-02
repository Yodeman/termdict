// Everything related to searching and displaying words definition

package search

import (
    "log"
    "strings"
    "text/template"

    "github.com/rivo/tview"
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

const defTempl = `
[::b]{{.Word}}

Definitions:
{{range .WordDefinitions}}
    [::Bi]part of speech: [::bi]{{.PartOfSpeech}}
    [::BI]└{{.WordDefinition}}

{{end}}
`
var definition *template.Template

func init() {
    definition = template.Must(
        template.New("definition").
        Parse(defTempl))
}


func SearchWord(
    word string,
    wordDbase map[string]DictEntity,
    definitionBox *tview.TextView) {

    entity := wordDbase[strings.ToLower(word)]
    writer := new(strings.Builder)

    if err := definition.Execute(writer, entity); err != nil {
        log.Fatalf("%v", err)
    }

    definitionBox.SetText(writer.String())
}
