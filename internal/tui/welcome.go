package tui

import (
	"fmt"
	"strings"
)

// showWelcome renders the initial definition-pane content: a wordmark,
// a starting hint and the offline-core status (plan v2 phase 3 M2).
// It is replaced by the first lookup or suggestion selection.
func (u *UI) showWelcome() {
	opts := u.renderOptions()

	var b strings.Builder
	fmt.Fprintf(&b, "%sTermDict%s\n", opts.HeaderTag, opts.ResetTag)
	fmt.Fprintf(&b, "%s%s%s\n\n", opts.MutedTag, strings.Repeat("─", 32), opts.ResetTag)
	fmt.Fprintf(&b, "Type a word to begin — try %sserendipity%s.\n\n",
		opts.AccentTag, opts.ResetTag)
	fmt.Fprintf(&b, "%s%d words ready · offline — no internet needed.%s\n",
		opts.MutedTag, u.svc.Len(), opts.ResetTag)
	fmt.Fprintf(&b, "%sPress ctrl+u and choose \"Download Full Dictionary\" to add the complete word list.%s\n",
		opts.MutedTag, opts.ResetTag)

	u.definitionBox.SetText(b.String())
}
