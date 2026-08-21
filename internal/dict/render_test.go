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
	if err := RenderTUI(&b, entity); err != nil {
		t.Fatalf("RenderTUI: %v", err)
	}

	out := b.String()
	for _, want := range []string{
		"[::b]test",
		"part of speech: [::bi]n.",
		"[::BI]└A trial.",
		"part of speech: [::bi]v. t.",
		"[::BI]└To try.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q;\ngot:\n%s", want, out)
		}
	}
}

func TestRenderTUIEmptyEntity(t *testing.T) {
	var b strings.Builder
	if err := RenderTUI(&b, Entity{}); err != nil {
		t.Fatalf("RenderTUI: %v", err)
	}
	if strings.Contains(b.String(), "└") {
		t.Error("empty entity should render no definitions")
	}
}

func TestNotFoundMessage(t *testing.T) {
	msg := NotFoundMessage("zzz")
	for _, want := range []string{"zzz", "No results found", "ctrl+u"} {
		if !strings.Contains(msg, want) {
			t.Errorf("NotFoundMessage missing %q; got:\n%s", want, msg)
		}
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
