package dict

import (
	"strings"
	"testing"
)

func TestRenderTUI(t *testing.T) {
	entity := Entity{
		Word: "test",
		WordDefinitions: []Definition{
			{PartOfSpeech: "n.", WordDefinition: "A trial."},
			{PartOfSpeech: "v. t.", WordDefinition: "To try."},
		},
	}

	var b strings.Builder
	if err := RenderTUI(&b, entity, DefaultRenderOptions()); err != nil {
		t.Fatalf("RenderTUI: %v", err)
	}

	out := b.String()
	for _, want := range []string{
		"[::b]test", // headword header
		"─",         // muted rule
		" [::b]1.[-:-:-] [::b]n.[-:-:-] A trial.", // numbered sense + badge
		" [::b]2.[-:-:-] [::b]v. t.[-:-:-] To try.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q;\ngot:\n%s", want, out)
		}
	}
	if strings.Contains(out, "part of speech:") || strings.Contains(out, "└") {
		t.Error("old label/tree format leaked into new renderer")
	}
}

func TestRenderTUIEmptyPOS(t *testing.T) {
	var b strings.Builder
	if err := RenderTUI(&b, Entity{
		Word:            "solo",
		WordDefinitions: []Definition{{PartOfSpeech: "", WordDefinition: "Alone."}},
	}, DefaultRenderOptions()); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, " [::b]1.[-:-:-] Alone.") {
		t.Errorf("missing POS badge should render number + text only; got:\n%s", out)
	}
}

func TestRenderTUIEmptyEntity(t *testing.T) {
	var b strings.Builder
	if err := RenderTUI(&b, Entity{}, DefaultRenderOptions()); err != nil {
		t.Fatalf("RenderTUI: %v", err)
	}
	if strings.Contains(b.String(), "1.") {
		t.Error("empty entity should render no senses")
	}
}

func TestRenderTUIMutedSpellings(t *testing.T) {
	var b strings.Builder
	if err := RenderTUI(&b, Entity{
		Word:      "test",
		Spellings: []string{"teste"},
	}, DefaultRenderOptions()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "[::i]alternate spellings: teste[-:-:-]") {
		t.Errorf("spellings should render muted italic; got:\n%s", b.String())
	}
}

func TestNotFoundMessage(t *testing.T) {
	msg := NotFoundMessage("zzz", "")
	for _, want := range []string{"zzz", "No results found", "ctrl+u"} {
		if !strings.Contains(msg, want) {
			t.Errorf("NotFoundMessage missing %q; got:\n%s", want, msg)
		}
	}
	if !strings.Contains(msg, "[::b]No results found") {
		t.Errorf("empty tag should degrade to plain bold; got:\n%s", msg)
	}
	themed := NotFoundMessage("zzz", "[#f5dfa9::b]")
	if !strings.Contains(themed, "[#f5dfa9::b]No results found") {
		t.Errorf("themed tag not applied; got:\n%s", themed)
	}
}

func TestRenderPlainText(t *testing.T) {
	entity := Entity{
		Word:      "test",
		Spellings: []string{"teste", "testes"},
		WordDefinitions: []Definition{
			{PartOfSpeech: "n.", WordDefinition: "A trial."},
			{PartOfSpeech: "", WordDefinition: "An unlabeled sense."},
			{PartOfSpeech: "v. t.", WordDefinition: "To try."},
		},
	}

	var b strings.Builder
	if err := RenderPlainText(&b, entity); err != nil {
		t.Fatalf("RenderPlainText: %v", err)
	}

	want := "test\n" +
		"\n  part of speech: n.\n  └A trial.\n" +
		"\n  └An unlabeled sense.\n" +
		"\n  part of speech: v. t.\n  └To try.\n" +
		"\nalternate spellings: teste, testes\n"
	if b.String() != want {
		t.Errorf("RenderPlainText mismatch:\ngot:\n%q\nwant:\n%q", b.String(), want)
	}
}

func TestRenderPlainTextNoSpellings(t *testing.T) {
	var b strings.Builder
	if err := RenderPlainText(&b, Entity{
		Word:            "solo",
		WordDefinitions: []Definition{{PartOfSpeech: "a.", WordDefinition: "Alone."}},
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "alternate spellings") {
		t.Error("spellings block must be omitted when empty")
	}
}

func TestPlainTextNotFound(t *testing.T) {
	msg := PlainTextNotFound("zzz")
	if !strings.Contains(msg, `No results for "zzz"`) ||
		!strings.Contains(msg, "termdict download") {
		t.Errorf("unexpected message: %q", msg)
	}
}
