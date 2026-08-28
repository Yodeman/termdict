package dict

import (
	"fmt"
	"io"
	"strings"
)

// maxMeasure caps the definition text measure so long terminal widths
// stay readable (planning report v2 B.3 wrap-width requirement).
const maxMeasure = 96

// RenderOptions carries the themed tview markup snippets and layout
// parameters RenderTUI embeds. The dict package stays free of tview
// imports: callers pass color tags derived from their active theme.
// Zero-value options degrade to attribute-only markup (bold/italic),
// which is also what the mono (NO_COLOR) theme produces.
type RenderOptions struct {
	HeaderTag      string // bold accent for the headword
	AccentTag      string // bold accent for POS headers and sense numbers
	MutedTag       string // muted for box borders and dividers
	MutedItalicTag string // muted italic for alternate spellings
	ResetTag       string // closes a themed tag ("[-:-:-]")

	// Boxed selects the POS-grouped box layout; otherwise a flat
	// fallback (headers + thin rules, no side borders) is rendered.
	// Width is the available content width of the definition pane.
	Boxed bool
	Width int
}

// DefaultRenderOptions returns attribute-only options (no color).
func DefaultRenderOptions() RenderOptions {
	return RenderOptions{
		HeaderTag:      "[::b]",
		AccentTag:      "[::b]",
		MutedTag:       "[::]",
		MutedItalicTag: "[::i]",
		ResetTag:       "[-:-:-]",
	}
}

// RenderTUI writes entity to w formatted with tview color markup.
//
// Senses are grouped by part of speech (source order preserved); each
// group renders as a boxed section whose header is the full spelled-out
// label, with senses separated by thin horizontal rules and numbered
// restarting at 1. Alternate spellings render as a muted italic
// trailing line. Definition prose is verbatim.
func RenderTUI(w io.Writer, entity Entity, opts RenderOptions) error {
	var b strings.Builder

	fmt.Fprintf(&b, "%s%s%s\n", opts.HeaderTag, entity.Word, opts.ResetTag)
	fmt.Fprintf(&b, "%s%s%s\n\n", opts.MutedTag, strings.Repeat("─", 32), opts.ResetTag)

	groups := GroupByPOS(entity)
	if opts.Boxed && opts.Width >= 20 {
		renderBoxed(&b, groups, opts)
	} else {
		renderFlat(&b, groups, opts)
	}

	if len(entity.Spellings) > 0 && len(groups) > 0 {
		fmt.Fprintf(&b, "\n%salternate spellings: %s%s\n",
			opts.MutedItalicTag, strings.Join(entity.Spellings, ", "), opts.ResetTag)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func renderBoxed(b *strings.Builder, groups []POSGroup, opts RenderOptions) {
	boxOuter := min(opts.Width, maxMeasure+4)
	boxInner := boxOuter - 4 // "│ " prefix + " │" suffix

	for _, group := range groups {
		label, _ := POSLabel(group.POS)
		b.WriteString(boxTop(boxOuter, label, opts))
		b.WriteString("\n")

		for i, def := range group.Senses {
			number := fmt.Sprintf("%d.", i+1)
			visiblePrefix := len([]rune(number)) + 1
			lines := WrapText(strings.TrimRight(def.WordDefinition, "\n"),
				boxInner-visiblePrefix)
			for j, line := range lines {
				var styled, visible string
				if j == 0 {
					styled = opts.AccentTag + number + opts.ResetTag + " "
					visible = number + " "
				} else {
					visible = strings.Repeat(" ", visiblePrefix)
					styled = visible
				}
				styled += line
				visible += line
				writeBoxLine(b, styled, len([]rune(visible)), boxInner, opts)
			}
			if i < len(group.Senses)-1 {
				rule := strings.Repeat("─", boxInner)
				writeBoxLine(b, opts.MutedTag+rule+opts.ResetTag, boxInner, boxInner, opts)
			}
		}
		b.WriteString(boxBottom(boxOuter, opts))
		b.WriteString("\n\n")
	}
}

func boxTop(boxOuter int, label string, opts RenderOptions) string {
	if label == "" {
		return opts.MutedTag + "┌" + strings.Repeat("─", boxOuter-2) + "┐" + opts.ResetTag
	}
	budget := boxOuter - 6 // "┌─ " + space + "┐" minus room for ≥1 fill
	if len([]rune(label)) > budget {
		label = string([]rune(label)[:max(budget-1, 1)]) + "…"
	}
	fill := strings.Repeat("─", max(boxOuter-5-len([]rune(label)), 1))
	return opts.MutedTag + "┌─ " + opts.ResetTag +
		opts.AccentTag + label + opts.ResetTag +
		opts.MutedTag + " " + fill + "┐" + opts.ResetTag
}

func boxBottom(boxOuter int, opts RenderOptions) string {
	return opts.MutedTag + "└" + strings.Repeat("─", boxOuter-2) + "┘" + opts.ResetTag
}

// writeBoxLine emits one box row: a muted left border, the styled
// content padded with spaces to the inner width (visibleWidth counts
// only cells that render — markup tags are invisible), then the muted
// right border.
func writeBoxLine(b *strings.Builder, styled string, visibleWidth, boxInner int, opts RenderOptions) {
	pad := strings.Repeat(" ", max(boxInner-visibleWidth, 0))
	b.WriteString(opts.MutedTag + "│ " + opts.ResetTag +
		styled + pad +
		opts.MutedTag + " │" + opts.ResetTag)
	b.WriteString("\n")
}

// renderFlat is the narrow-terminal fallback: plain accent headers and
// thin rules instead of boxes (the pane is too small for the frame to
// earn its keep).
func renderFlat(b *strings.Builder, groups []POSGroup, opts RenderOptions) {
	width := opts.Width
	if width <= 0 || width > maxMeasure {
		width = maxMeasure
	}

	for _, group := range groups {
		label, _ := POSLabel(group.POS)
		if label != "" {
			fmt.Fprintf(b, "%s%s%s\n", opts.AccentTag, label, opts.ResetTag)
		}
		for i, def := range group.Senses {
			lines := WrapText(strings.TrimRight(def.WordDefinition, "\n"), width-4)
			for j, line := range lines {
				if j == 0 {
					fmt.Fprintf(b, " %s%d.%s %s\n",
						opts.AccentTag, i+1, opts.ResetTag, line)
					continue
				}
				fmt.Fprintf(b, "   %s\n", line)
			}
			if i < len(group.Senses)-1 {
				fmt.Fprintf(b, "%s%s%s\n",
					opts.MutedTag, strings.Repeat("─", min(width-2, 40)), opts.ResetTag)
			}
		}
		b.WriteString("\n")
	}
}

// WrapText breaks text into lines of at most width cells, breaking on
// spaces where possible. Tokens longer than a full line are hard-split.
func WrapText(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if width < 4 {
		width = 4
	}

	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := ""
		for _, word := range words {
			switch {
			case current == "":
				for len([]rune(word)) > width { // token longer than a line
					chunk := string([]rune(word)[:width])
					word = string([]rune(word)[width:])
					lines = append(lines, chunk)
				}
				current = word
			case len([]rune(current))+1+len([]rune(word)) <= width:
				current += " " + word
			default:
				lines = append(lines, current)
				for len([]rune(word)) > width {
					chunk := string([]rune(word)[:width])
					word = string([]rune(word)[width:])
					lines = append(lines, chunk)
				}
				current = word
			}
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return lines
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
