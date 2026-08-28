package dict

import (
	"fmt"
	"io"
	"strings"
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
// misses the loaded dictionary. warningTag is the caller's themed bold
// markup tag (an empty tag degrades to plain bold). With the embedded
// core installed, a miss usually means a rare word that the full
// (downloadable) database still contains.
func NotFoundMessage(query, warningTag string) string {
	if warningTag == "" {
		warningTag = "[::b]"
	}
	return fmt.Sprintf(`
[::b]%s

%[2]sNo results found.[-:-:-]

This word isn't in the offline library.
Press ctrl+u and choose "Download Full Dictionary" to get
the complete word list (internet connection required).
`, query, warningTag)
}

// RenderPlainText writes entity to w in the plain-text format used by
// the CLI: definition payload only, no color markup, safe to pipe.
//
//	word
//
//	  part of speech: n.
//	  └definition…
//
//	alternate spellings: x, y      (only when present)
func RenderPlainText(w io.Writer, entity Entity) error {
	if _, err := fmt.Fprintf(w, "%s\n", entity.Word); err != nil {
		return err
	}
	for _, def := range entity.WordDefinitions {
		pos := strings.TrimSpace(def.PartOfSpeech)
		if pos != "" {
			if _, err := fmt.Fprintf(w, "\n  part of speech: %s\n", pos); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, "  └%s\n",
				strings.TrimRight(def.WordDefinition, "\n")); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "\n  └%s\n",
			strings.TrimRight(def.WordDefinition, "\n")); err != nil {
			return err
		}
	}
	if len(entity.Spellings) > 0 {
		_, err := fmt.Fprintf(w, "\nalternate spellings: %s\n",
			strings.Join(entity.Spellings, ", "))
		return err
	}
	return nil
}

// PlainTextNotFound returns the single-line not-found notice printed to
// stderr by the CLI (stdout stays empty so pipes stay clean).
func PlainTextNotFound(query string) string {
	return fmt.Sprintf("No results for %q. Run 'termdict download' to get the full dictionary.\n", query)
}
