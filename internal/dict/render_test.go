package dict

import (
	"strings"
	"testing"
)

func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

func TestRenderTUIBoxed(t *testing.T) {
	entity := Entity{
		Word: "test",
		WordDefinitions: []Definition{
			{PartOfSpeech: "a.", WordDefinition: "A trial of something."},
			{PartOfSpeech: "a.", WordDefinition: "Second adjective sense."},
			{PartOfSpeech: "n.", WordDefinition: "The thing being tried."},
		},
	}
	opts := DefaultRenderOptions()
	opts.Boxed = true
	opts.Width = 80

	var b strings.Builder
	if err := RenderTUI(&b, entity, opts); err != nil {
		t.Fatalf("RenderTUI: %v", err)
	}
	out := b.String()

	// Full spelled-out labels, one box per POS group, source order.
	if strings.Index(out, "Adjective") > strings.Index(out, "Noun") {
		t.Error("group order must follow the source (adjective first)")
	}
	if strings.Contains(out, "[a.]") || strings.Contains(stripTags(out), " a. ") {
		t.Error("abbreviation leaked into the box header")
	}
	for _, want := range []string{
		"┌─ Adjective", "└", "┌─ Noun",
		"1. A trial of something.", "2. Second adjective sense.",
		"1. The thing being tried.", // numbering restarts per group
		"─",                         // divider rule between senses
	} {
		if !strings.Contains(stripTags(out), want) && !strings.Contains(out, want) {
			t.Errorf("output missing %q;\ngot:\n%s", want, stripTags(out))
		}
	}
	// Blank line between the two boxes.
	if !strings.Contains(stripTags(out), "┘\n\n┌─ Noun") {
		t.Error("boxes should be separated by a blank line")
	}
}

func TestRenderTUIBoxedLineWidths(t *testing.T) {
	entity := Entity{
		Word: "wrap",
		WordDefinitions: []Definition{
			{PartOfSpeech: "v. t.", WordDefinition: strings.Repeat("word ", 40)},
		},
	}
	opts := DefaultRenderOptions()
	opts.Boxed = true
	opts.Width = 70

	var b strings.Builder
	if err := RenderTUI(&b, entity, opts); err != nil {
		t.Fatal(err)
	}

	width := 0
	for _, line := range strings.Split(stripTags(b.String()), "\n") {
		if strings.HasPrefix(line, "│") || strings.HasPrefix(line, "┌") ||
			strings.HasPrefix(line, "└") {
			if width == 0 {
				width = len([]rune(line))
			} else if len([]rune(line)) != width {
				t.Errorf("box line width %d differs from %d: %q",
					len([]rune(line)), width, line)
			}
		}
	}
	if width != 70 {
		t.Errorf("box outer width = %d, want the full available width 70", width)
	}
}

func TestRenderTUIBoxedSingleSense(t *testing.T) {
	entity := Entity{
		Word:            "solo",
		WordDefinitions: []Definition{{PartOfSpeech: "a.", WordDefinition: "Alone."}},
	}
	opts := DefaultRenderOptions()
	opts.Boxed = true
	opts.Width = 60

	var b strings.Builder
	if err := RenderTUI(&b, entity, opts); err != nil {
		t.Fatal(err)
	}
	out := stripTags(b.String())
	if !strings.Contains(out, "┌─ Adjective") || !strings.Contains(out, "1. Alone.") {
		t.Errorf("single-sense box malformed:\n%s", out)
	}
	if strings.Count(out, "─") < 3 {
		t.Error("expected top border, header fill and bottom border")
	}
}

func TestRenderTUIFlat(t *testing.T) {
	entity := Entity{
		Word: "test",
		WordDefinitions: []Definition{
			{PartOfSpeech: "a.", WordDefinition: "A trial."},
			{PartOfSpeech: "a.", WordDefinition: "Second sense."},
		},
	}
	opts := DefaultRenderOptions()
	opts.Boxed = false
	opts.Width = 40

	var b strings.Builder
	if err := RenderTUI(&b, entity, opts); err != nil {
		t.Fatal(err)
	}
	out := stripTags(b.String())
	if strings.ContainsAny(out, "│┌└") {
		t.Error("flat fallback must not draw box borders")
	}
	if !strings.Contains(out, "Adjective") || !strings.Contains(out, "1. A trial.") {
		t.Errorf("flat output missing header/number:\n%s", out)
	}
	if !strings.Contains(out, "──") {
		t.Error("flat fallback should separate senses with thin rules")
	}
}

func TestRenderTUIEmptyPOSGroup(t *testing.T) {
	entity := Entity{
		Word:            "void",
		WordDefinitions: []Definition{{PartOfSpeech: "", WordDefinition: "Unlabeled sense."}},
	}
	opts := DefaultRenderOptions()
	opts.Boxed = true
	opts.Width = 60

	var b strings.Builder
	if err := RenderTUI(&b, entity, opts); err != nil {
		t.Fatal(err)
	}
	out := stripTags(b.String())
	if !strings.Contains(out, "Unlabeled sense.") {
		t.Errorf("sense missing:\n%s", out)
	}
	// The unlabeled group renders a header-less box (no label text).
	if strings.Contains(out, "┌─  ") {
		t.Errorf("empty label should omit the header text:\n%s", out)
	}
}

func TestRenderTUIEmptyEntity(t *testing.T) {
	var b strings.Builder
	if err := RenderTUI(&b, Entity{}, DefaultRenderOptions()); err != nil {
		t.Fatalf("RenderTUI: %v", err)
	}
	if strings.Contains(stripTags(b.String()), "1.") {
		t.Error("empty entity should render no senses")
	}
}

func TestRenderTUIMutedSpellings(t *testing.T) {
	var b strings.Builder
	entity := Entity{
		Word:            "solo",
		Spellings:       []string{"solos"},
		WordDefinitions: []Definition{{PartOfSpeech: "a.", WordDefinition: "Alone."}},
	}
	opts := DefaultRenderOptions()
	opts.Boxed = true
	opts.Width = 60
	if err := RenderTUI(&b, entity, opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "[::i]alternate spellings: solos[-:-:-]") {
		t.Errorf("spellings should render muted italic:\n%s", b.String())
	}
}

func TestWrapText(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{"empty", "", 10, nil},
		{"fits", "hello world", 20, []string{"hello world"}},
		{"greedy", "hello world again", 11, []string{"hello world", "again"}},
		{"exact", "hello world", 11, []string{"hello world"}},
		{"long token", "abcdefghij", 4, []string{"abcd", "efgh", "ij"}},
		{"multiple paragraphs", "one\n\ntwo", 10, []string{"one", "", "two"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WrapText(tc.text, tc.width)
			if len(got) != len(tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestPlainTextNotFound(t *testing.T) {
	msg := PlainTextNotFound("zzz")
	if !strings.Contains(msg, `No results for "zzz"`) ||
		!strings.Contains(msg, "termdict download") {
		t.Errorf("unexpected message: %q", msg)
	}
}
