package dict

import (
	"fmt"
	"io"
	"strings"
)

// RenderOptions carries the themed tview markup snippets RenderTUI
// embeds. The dict package stays free of tview imports: callers pass
// color tags derived from their active theme. Zero-value options
// degrade to attribute-only markup (bold/italic), which is also what
// the mono (NO_COLOR) theme produces.
type RenderOptions struct {
	HeaderTag string // bold accent for the headword line
	AccentTag string // bold accent for sense numbers and POS badges
	MutedTag  string // muted for the rule and alternate spellings
	ResetTag  string // closes a themed tag ("[-:-:-]")
}

// DefaultRenderOptions returns attribute-only options (no color).
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		HeaderTag: "[::b]",
		AccentTag: "[::b]",
		MutedTag:  "[::i]",
		ResetTag:  "[-:-:-]",
	}
}

// RenderTUI writes entity to w formatted with tview color markup:
// an accent headword header over a muted rule, then numbered senses
// with inline part-of-speech badges, and a muted alternate-spellings
// line when present. Definition prose itself is rendered verbatim.
func RenderTUI(w io.Writer, entity Entity, opts RenderOptions) error {
	var b strings.Builder

	fmt.Fprintf(&b, "%s%s%s\n", opts.HeaderTag, entity.Word, opts.ResetTag)
	fmt.Fprintf(&b, "%s%s%s\n\n", opts.MutedTag, strings.Repeat("─", 32), opts.ResetTag)

	for i, def := range entity.WordDefinitions {
		fmt.Fprintf(&b, " %s%d.%s ", opts.AccentTag, i+1, opts.ResetTag)
		if pos := strings.TrimSpace(def.PartOfSpeech); pos != "" {
			fmt.Fprintf(&b, "%s%s%s ", opts.AccentTag, pos, opts.ResetTag)
		}
		fmt.Fprintf(&b, "%s\n", strings.TrimRight(def.WordDefinition, "\n"))
	}

	if len(entity.Spellings) > 0 {
		fmt.Fprintf(&b, "\n%salternate spellings: %s%s\n",
			opts.MutedTag, strings.Join(entity.Spellings, ", "), opts.ResetTag)
	}

	_, err := io.WriteString(w, b.String())
	return err
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

// PlainTextNotFound returns the single-line not-found notice printed to
// stderr by the CLI (stdout stays empty so pipes stay clean).
func PlainTextNotFound(query string) string {
	return fmt.Sprintf("No results for %q. Run 'termdict download' to get the full dictionary.\n", query)
}

// RenderPlainText writes entity to w in the plain-text format used by
// the CLI: definition payload only, no color markup, safe to pipe.
// Format is stable since v0.2.0 — the piping contract depends on it.
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
