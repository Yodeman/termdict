package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Theme holds the color tokens used across the interface. Meaning is
// never conveyed by color alone — every state that uses these tokens
// also carries it in text (see docs/smoke-testing.md).
type Theme struct {
	Name string

	// Structural colors.
	Border      tcell.Color // widget borders at rest
	BorderFocus tcell.Color // border of the focused widget (accent-on-focus)
	Muted       tcell.Color // secondary text (hints, spellings, "… N more")

	// Content colors.
	PrimaryText tcell.Color // main text
	Accent      tcell.Color // highlights (pressed buttons, progress bar)
	AccentText  tcell.Color // accent as text (passes stricter contrast)
	Header      tcell.Color // headwords, popup titles

	// Semantic states (always paired with explanatory text).
	Success tcell.Color
	Warning tcell.Color
	Danger  tcell.Color

	// Progress bar.
	BarFilled tcell.Color
	BarEmpty  tcell.Color

	// Background the palette assumes; used by the contrast test suite
	// and applied to tview.Styles.
	Background tcell.Color
}

func hexColor(h int32) tcell.Color { return tcell.NewHexColor(h) }

// ocean is the default palette: a refined evolution of v0.1.0's blue,
// tuned for dark terminals.
var oceanTheme = Theme{
	Name:        "ocean",
	Border:      hexColor(0x76829E),
	BorderFocus: hexColor(0x74B3FF),
	Muted:       hexColor(0x8A93A8),
	PrimaryText: hexColor(0xD6E1F5),
	Accent:      hexColor(0x74B3FF),
	AccentText:  hexColor(0x8FC7FF),
	Header:      hexColor(0x9ECFFF),
	Success:     hexColor(0x9EE6B0),
	Warning:     hexColor(0xF5DFA9),
	Danger:      hexColor(0xF793AE),
	BarFilled:   hexColor(0x74B3FF),
	BarEmpty:    hexColor(0x76829E),
	Background:  hexColor(0x1E1E2E),
}

// catppuccin mirrors the Catppuccin Mocha palette
// (https://github.com/catppuccin/catppuccin) for users who theme their
// whole environment with it.
var catppuccinTheme = Theme{
	Name:        "catppuccin",
	Border:      hexColor(0x6C7086), // overlay0
	BorderFocus: hexColor(0x89B4FA), // blue
	Muted:       hexColor(0x9399B2), // overlay2
	PrimaryText: hexColor(0xCDD6F4), // text
	Accent:      hexColor(0x89B4FA),
	AccentText:  hexColor(0x89B4FA),
	Header:      hexColor(0xCBA6F7), // mauve
	Success:     hexColor(0xA6E3A1), // green
	Warning:     hexColor(0xF9E2AF), // yellow
	Danger:      hexColor(0xF38BA8), // red
	BarFilled:   hexColor(0x89B4FA),
	BarEmpty:    hexColor(0x6C7086),
	Background:  hexColor(0x11111B), // crust
}

// paper is the light palette for light terminal backgrounds.
var paperTheme = Theme{
	Name:        "paper",
	Border:      hexColor(0x828BA0),
	BorderFocus: hexColor(0x1E66F5),
	Muted:       hexColor(0x5C5F77),
	PrimaryText: hexColor(0x3A3D4D),
	Accent:      hexColor(0x1E66F5),
	AccentText:  hexColor(0x04338C),
	Header:      hexColor(0x04338C),
	Success:     hexColor(0x1E7E34),
	Warning:     hexColor(0x9A6A00),
	Danger:      hexColor(0xC01C3E),
	BarFilled:   hexColor(0x1E66F5),
	BarEmpty:    hexColor(0x828BA0),
	Background:  hexColor(0xFFFFFF),
}

// mono honors the NO_COLOR standard (https://no-color.org): when the
// variable is present and non-empty, no ANSI color is added; styling is
// limited to attributes such as bold and underline.
var monoTheme = Theme{
	Name:        "mono",
	Border:      tcell.ColorDefault,
	BorderFocus: tcell.ColorDefault,
	Muted:       tcell.ColorDefault,
	PrimaryText: tcell.ColorDefault,
	Accent:      tcell.ColorDefault,
	AccentText:  tcell.ColorDefault,
	Header:      tcell.ColorDefault,
	Success:     tcell.ColorDefault,
	Warning:     tcell.ColorDefault,
	Danger:      tcell.ColorDefault,
	BarFilled:   tcell.ColorDefault,
	BarEmpty:    tcell.ColorDefault,
	Background:  tcell.ColorDefault,
}

// Select resolves the active theme from the environment:
//
//	NO_COLOR (present and non-empty) always wins -> mono
//	TERMDICT_THEME=ocean|catppuccin|paper (case-insensitive)
//	unset/unknown -> ocean; an unknown value yields an error the caller
//	should surface as a one-time warning.
func Select(getenv func(string) string) (Theme, error) {
	if getenv("NO_COLOR") != "" {
		return monoTheme, nil
	}

	switch value := strings.ToLower(getenv("TERMDICT_THEME")); value {
	case "", "ocean":
		return oceanTheme, nil
	case "catppuccin", "mocha":
		return catppuccinTheme, nil
	case "paper", "light":
		return paperTheme, nil
	default:
		return oceanTheme, fmt.Errorf(
			"unknown TERMDICT_THEME %q (want ocean, catppuccin or paper)", value)
	}
}

// applyStyles feeds the theme into tview's global style table. It must
// run BEFORE any widget is constructed: tview primitives capture the
// border/title colors from tview.Styles at construction time.
func (t Theme) applyStyles() {
	tview.Styles.PrimitiveBackgroundColor = t.Background
	tview.Styles.ContrastBackgroundColor = t.Background
	tview.Styles.MoreContrastBackgroundColor = t.Border
	tview.Styles.BorderColor = t.Border
	tview.Styles.TitleColor = t.Header
	tview.Styles.PrimaryTextColor = t.PrimaryText
	tview.Styles.SecondaryTextColor = t.Muted
	tview.Styles.TertiaryTextColor = t.Accent
	tview.Styles.InverseTextColor = t.Background
	tview.Styles.ContrastSecondaryTextColor = t.Muted
}

// Tag renders c as a bold tview markup tag. Under mono (NO_COLOR) it
// degrades to a plain bold attribute so markup stays color-free.
func (t Theme) Tag(c tcell.Color) string {
	return t.TagStyle(c, "b")
}

// TagStyle renders c as a tview markup tag with the given attribute
// flags (e.g. "b", "i", ""). Under mono (NO_COLOR) it degrades to an
// attribute-only tag so markup stays color-free.
func (t Theme) TagStyle(c tcell.Color, flags string) string {
	if c == tcell.ColorDefault {
		return "[::" + flags + "]"
	}
	r, g, b := c.RGB()
	return fmt.Sprintf("[#%02x%02x%02x::%s]", r, g, b, flags)
}

// applyRoundedBorders switches tview's box-drawing corners to the
// rounded style; focused widgets keep their double-line glyphs.
func applyRoundedBorders() {
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'
}

// applyFocusAccent paints the border of whichever widget holds focus in
// the accent color (the lazygit "active panel" pattern). It relies on
// tview firing Box focus/blur callbacks: subclasses such as InputField
// and TextView delegate to Box.Focus/Box.Blur, and focus landing on an
// InputField's inner text area is forwarded to the Box callback.
func (u *UI) applyFocusAccent(boxes ...*tview.Box) {
	for _, box := range boxes {
		b := box
		b.SetFocusFunc(func() { b.SetBorderColor(u.theme.BorderFocus) })
		b.SetBlurFunc(func() { b.SetBorderColor(u.theme.Border) })
	}
}
