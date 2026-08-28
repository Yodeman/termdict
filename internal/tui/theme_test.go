package tui

import (
	"math"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/yodeman/termdict/internal/dict"
)

func TestSelect(t *testing.T) {
	ocean, _ := Select(func(string) string { return "" })

	cases := []struct {
		name    string
		env     map[string]string
		want    Theme
		wantErr bool
	}{
		{"unset selects ocean", map[string]string{}, ocean, false},
		{"empty selects ocean", map[string]string{"TERMDICT_THEME": ""}, ocean, false},
		{"case-insensitive catppuccin", map[string]string{"TERMDICT_THEME": "CATPPUCCIN"}, catppuccinTheme, false},
		{"mocha alias", map[string]string{"TERMDICT_THEME": "mocha"}, catppuccinTheme, false},
		{"paper", map[string]string{"TERMDICT_THEME": "paper"}, paperTheme, false},
		{"light alias", map[string]string{"TERMDICT_THEME": "light"}, paperTheme, false},
		{"unknown falls back to ocean with error", map[string]string{"TERMDICT_THEME": "dracula"}, ocean, true},
		{"NO_COLOR forces mono", map[string]string{"NO_COLOR": "1", "TERMDICT_THEME": "catppuccin"}, monoTheme, false},
		{"empty NO_COLOR is ignored (standard)", map[string]string{"NO_COLOR": "", "TERMDICT_THEME": "paper"}, paperTheme, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Select(func(key string) string { return tc.env[key] })
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got theme %q", got.Name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != tc.want.Name {
				t.Errorf("got theme %q, want %q", got.Name, tc.want.Name)
			}
		})
	}
}

func TestMonoThemeIsColorFree(t *testing.T) {
	for _, c := range []tcell.Color{
		monoTheme.Border, monoTheme.BorderFocus, monoTheme.Muted,
		monoTheme.PrimaryText, monoTheme.Accent, monoTheme.AccentText,
		monoTheme.Header, monoTheme.Success, monoTheme.Warning,
		monoTheme.Danger, monoTheme.BarFilled, monoTheme.BarEmpty,
		monoTheme.Background,
	} {
		if c != tcell.ColorDefault {
			t.Fatalf("mono theme must use only tcell.ColorDefault")
		}
	}
	if tag := monoTheme.Tag(monoTheme.Accent); tag != "[::b]" {
		t.Errorf("mono Tag should degrade to plain bold, got %q", tag)
	}
}

// --- WCAG 2.1 contrast mathematics ---------------------------------------

func linearize(c byte) float64 {
	v := float64(c) / 255
	if v <= 0.03928 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func luminance(c tcell.Color) float64 {
	r, g, b := c.RGB()
	return 0.2126*linearize(byte(r)) + 0.7152*linearize(byte(g)) + 0.0722*linearize(byte(b))
}

func contrastRatio(a, b tcell.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func TestThemeContrast(t *testing.T) {
	// WCAG 2.1 AA: text 4.5:1, UI components (borders, bar glyphs) 3:1.
	// Backgrounds are each palette's documented assumption (dark for
	// ocean/catppuccin, light for paper).
	textTokens := map[string]func(Theme) tcell.Color{
		"PrimaryText": func(th Theme) tcell.Color { return th.PrimaryText },
		"AccentText":  func(th Theme) tcell.Color { return th.AccentText },
		"Muted":       func(th Theme) tcell.Color { return th.Muted },
		"Header":      func(th Theme) tcell.Color { return th.Header },
		"Success":     func(th Theme) tcell.Color { return th.Success },
		"Warning":     func(th Theme) tcell.Color { return th.Warning },
		"Danger":      func(th Theme) tcell.Color { return th.Danger },
	}
	uiTokens := map[string]func(Theme) tcell.Color{
		"Border":      func(th Theme) tcell.Color { return th.Border },
		"BorderFocus": func(th Theme) tcell.Color { return th.BorderFocus },
		"Accent":      func(th Theme) tcell.Color { return th.Accent },
		"BarFilled":   func(th Theme) tcell.Color { return th.BarFilled },
		"BarEmpty":    func(th Theme) tcell.Color { return th.BarEmpty },
	}

	for _, theme := range []Theme{oceanTheme, catppuccinTheme, paperTheme} {
		for name, token := range textTokens {
			if ratio := contrastRatio(token(theme), theme.Background); ratio < 4.5 {
				t.Errorf("%s/%s: contrast %.2f < 4.5", theme.Name, name, ratio)
			}
		}
		for name, token := range uiTokens {
			if ratio := contrastRatio(token(theme), theme.Background); ratio < 3.0 {
				t.Errorf("%s/%s: contrast %.2f < 3.0", theme.Name, name, ratio)
			}
		}
	}
}

func TestThemeTag(t *testing.T) {
	ocean, _ := Select(func(string) string { return "" })
	tag := ocean.Tag(ocean.Accent)
	want := "[#74b3ff::b]"
	if tag != want {
		t.Errorf("Tag = %q, want %q", tag, want)
	}
}

// The contrast suite above validates the tokens; this validates that
// the definition renderer actually APPLIES them (QA v2 issue 1,
// action 3): themed options must embed the header, accent and muted
// tags in the output, and box headers must use the spelled-out label.
func TestRenderOptionsAppliedToOutput(t *testing.T) {
	ocean, _ := Select(func(string) string { return "" })
	ro := dict.RenderOptions{
		HeaderTag:      ocean.TagStyle(ocean.Header, "b"),
		AccentTag:      ocean.TagStyle(ocean.Accent, "b"),
		MutedTag:       ocean.TagStyle(ocean.Muted, ""),
		MutedItalicTag: ocean.TagStyle(ocean.Muted, "i"),
		ResetTag:       "[-:-:-]",
		Boxed:          true,
		Width:          80,
	}
	if ro.HeaderTag == ro.AccentTag {
		t.Fatal("header and accent tags must differ for a visible hierarchy")
	}

	var b strings.Builder
	if err := dict.RenderTUI(&b, dict.Entity{
		Word: "test",
		WordDefinitions: []dict.Definition{
			{PartOfSpeech: "n.", WordDefinition: "A trial."},
		},
	}, ro); err != nil {
		t.Fatal(err)
	}
	out := b.String()

	for name, tag := range map[string]string{
		"header": ro.HeaderTag,
		"accent": ro.AccentTag,
		"muted":  ro.MutedTag,
	} {
		if !strings.Contains(out, tag) {
			t.Errorf("rendered output never applies the %s tag (%q);\ngot:\n%s", name, tag, out)
		}
	}
	if !strings.Contains(out, "Noun") {
		t.Errorf("box header should use the spelled-out label:\n%s", out)
	}
	if strings.Contains(out, "n.") {
		t.Errorf("abbreviation leaked into the rendered header:\n%s", out)
	}
}
