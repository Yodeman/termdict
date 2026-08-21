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
// misses the loaded dictionary. With the embedded core installed, a
// miss usually means a rare word that the full (downloadable) database
// still contains.
func NotFoundMessage(query string) string {
	return fmt.Sprintf(`
[::b]%s

[yellow::b]No results found.[-:-:-]

This word isn't in the offline library.
Press ctrl+u and choose "Download Full Dictionary" to get
the complete word list (internet connection required).
`, query)
}
