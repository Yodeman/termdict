// Everything related to searching and displaying words definition

package tui

import (
    "log"
    "strings"
    "text/template"
)

// Text template used to display word definition
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


// searchWord searches for word in the dictionary database
// and renders the definition of the word.
func searchWord(word string, wordDbase map[string]DictEntity) {
    entity := wordDbase[strings.ToLower(word)]
    writer := new(strings.Builder)

    if err := definition.Execute(writer, entity); err != nil {
        log.Fatalf("%v", err)
    }

    definitionBox.SetText(writer.String())
}
