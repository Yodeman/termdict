package dict

import (
	"fmt"
	"io"
	"text/template"
)

// defTemplate renders an entity with tview color markup for the
// definition box. Output format is unchanged from v0.1.0.
const defTemplate = `
[::b]{{.Word}}

Definitions:
{{range .WordDefinitions}}
    [::Bi]part of speech: [::bi]{{.PartOfSpeech}}
    [::BI]└{{.WordDefinition}}

{{end}}
`

var definitionTmpl = template.Must(template.New("definition").Parse(defTemplate))

// RenderTUI writes entity to w formatted with tview color markup.
func RenderTUI(w io.Writer, entity Entity) error {
	return definitionTmpl.Execute(w, entity)
}

// NotFoundMessage returns the TUI-formatted block shown when a lookup
// misses the loaded dictionary. downloadHint tells the user how to get
// the full dictionary; it is empty in offline-only builds.
func NotFoundMessage(query string) string {
	return fmt.Sprintf(`
[::b]%s

[yellow::b]No results found.[-:-:-]

The word isn't in the loaded dictionary.
Press ctrl+u to update or download the words database.
`, query)
}
