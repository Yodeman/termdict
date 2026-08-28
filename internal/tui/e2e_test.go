//go:build unix

package tui

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/yodeman/termdict/internal/e2etest"
)

// End-to-end regression tests that drive the REAL binary under a pty
// and assert on the reconstructed terminal screen (QA v2 issues 2/3:
// these guard app-level focus routing, which unit tests cannot see).
// Skipped under -short; run via `go test ./...` or `make test-e2e`.

const (
	cols = 100
	rows = 30

	tabKey   = "\t"
	stabKey  = "\x1b[Z"
	escKey   = "\x1b"
	f1Key    = "\x1bOP"
	f2Key    = "\x1bOQ"
	ctrlUKey = "\x15"
)

func skipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("e2e: skipped in -short mode")
	}
}

func newSession(t *testing.T, env map[string]string) *e2etest.PTYSession {
	t.Helper()
	binary, err := e2etest.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	home := t.TempDir()
	overrides := map[string]string{
		"HOME": home,
		"TERM": "xterm-256color",
	}
	for k, v := range env {
		overrides[k] = v
	}
	session, err := e2etest.NewPTYSession(binary, cols, rows, overrides)
	if err != nil {
		t.Fatalf("pty session: %v", err)
	}
	t.Cleanup(session.Close)

	// Boot: wait for the welcome screen to render.
	if !session.WaitFor(5*time.Second, func(s *e2etest.Screen) bool {
		return strings.Contains(s.Text(), "Type a word to begin")
	}) {
		t.Fatalf("app did not render the welcome screen within 5s:\n%s", session.Screen().Text())
	}
	return session
}

// focusedPane maps tview's focused-border corner (╔) to a pane name.
func focusedPane(s *e2etest.Screen) string {
	x, y, ok := s.FocusedCorner()
	if !ok {
		return "(none)"
	}
	switch {
	case y >= rows-4:
		return "commands"
	case x >= cols-45:
		return "definition"
	case y <= 2:
		return "search"
	default:
		return "suggestions"
	}
}

func waitFocused(t *testing.T, s *e2etest.PTYSession, want string) {
	t.Helper()
	if !s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
		return focusedPane(s) == want
	}) {
		t.Fatalf("focus never reached %q (currently %q)",
			want, focusedPane(s.Screen()))
	}
}

func TestFocusE2E(t *testing.T) {
	skipIfShort(t)

	t.Run("tabs", func(t *testing.T) {
		s := newSession(t, nil)

		steps := []struct {
			label string
			keys  string
			want  string
		}{
			{"tab1", tabKey, "suggestions"},
			{"tab2", tabKey, "definition"},
			{"tab3", tabKey, "search"},
			{"tab4 (wraps to suggestions)", tabKey, "suggestions"},
			{"shift+tab (suggestions -> search)", stabKey, "search"},
			{"shift+tab (search -> definition)", stabKey, "definition"},
		}
		for _, step := range steps {
			s.Send([]byte(step.keys))
			waitFocused(t, s, step.want)
			t.Logf("%s: focus on %s", step.label, step.want)
		}
	})

	t.Run("modal", func(t *testing.T) {
		s := newSession(t, nil)

		s.Send([]byte(f1Key))
		if !s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
			return strings.Contains(s.Text(), "KEYS")
		}) {
			t.Fatal("F1 did not open the help popup")
		}
		if _, _, ok := s.Screen().FocusedCorner(); !ok {
			t.Error("help popup did not receive keyboard focus")
		}

		// Tab inside the popup is a trapped no-op: it must neither
		// close the popup nor move focus to the main panes.
		s.Send([]byte(tabKey), []byte(tabKey), []byte(tabKey))
		time.Sleep(400 * time.Millisecond) // absence-of-change window
		s.SettleDrain(300 * time.Millisecond)
		if !strings.Contains(s.Screen().Text(), "KEYS") {
			t.Error("Tab inside help closed the popup")
		}

		s.Send([]byte(escKey))
		if !s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
			return !strings.Contains(s.Text(), "KEYS")
		}) {
			t.Fatal("Esc did not close the help popup")
		}
		waitFocused(t, s, "search")

		s.Send([]byte(f2Key))
		if !s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
			return strings.Contains(s.Text(), "Built with")
		}) {
			t.Fatal("F2 did not open the about modal")
		}
		s.Send([]byte(tabKey), []byte(tabKey))
		time.Sleep(400 * time.Millisecond)
		s.SettleDrain(300 * time.Millisecond)
		if !strings.Contains(s.Screen().Text(), "Built with") {
			t.Error("Tab inside about closed the modal")
		}
		s.Send([]byte(escKey))
		if !s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
			return !strings.Contains(s.Text(), "Built with")
		}) {
			t.Fatal("Esc did not close the about modal")
		}
		waitFocused(t, s, "search")
	})

	t.Run("update", func(t *testing.T) {
		// Black-holed network: the triggered update fails instantly and
		// deterministically, keeping the test offline and fast. The
		// "close / cancel" title fragment is unique to the popup (the
		// bottom button bar also contains "Update Dbase").
		s := newSession(t, map[string]string{
			"HTTP_PROXY": "http://127.0.0.1:9", "HTTPS_PROXY": "http://127.0.0.1:9",
		})

		s.Send([]byte(ctrlUKey))
		if !s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
			return strings.Contains(s.Text(), "close / cancel")
		}) {
			t.Fatal("ctrl+u did not open the update popup")
		}
		s.Send([]byte(escKey)) // cancels the in-flight check
		s.Send([]byte(escKey)) // closes the popup
		if !s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
			return !strings.Contains(s.Text(), "close / cancel")
		}) {
			t.Fatal("Esc did not close the update popup")
		}
		waitFocused(t, s, "search")
	})
}

// TestTerminalMatrix boots every palette on every terminal color
// profile and asserts the interface renders; the mono theme must
// additionally produce a completely color-free output stream
// (planning report v2 B.3.9 / R6 Unix half; the Windows half is the
// manual checklist in docs/smoke-testing.md).
func TestTerminalMatrix(t *testing.T) {
	skipIfShort(t)

	themes := []struct {
		name string
		env  map[string]string
	}{
		{"ocean", nil},
		{"catppuccin", map[string]string{"TERMDICT_THEME": "catppuccin"}},
		{"paper", map[string]string{"TERMDICT_THEME": "paper"}},
		{"mono", map[string]string{"NO_COLOR": "1"}},
	}
	profiles := []struct {
		name string
		env  map[string]string
	}{
		{"8-color", map[string]string{"TERM": "xterm"}},
		{"256-color", map[string]string{"TERM": "xterm-256color"}},
		{"truecolor", map[string]string{"TERM": "xterm-256color", "COLORTERM": "truecolor"}},
	}

	var colorSGR = regexp.MustCompile(
		`\x1b\[(?:3[0-7]|9[0-7]|38;5;\d+|38;2;\d+;\d+;\d+|48;5;\d+|48;2;\d+;\d+;\d+)m`)

	for _, theme := range themes {
		for _, profile := range profiles {
			t.Run(theme.name+" on "+profile.name, func(t *testing.T) {
				env := map[string]string{}
				for k, v := range theme.env {
					env[k] = v
				}
				for k, v := range profile.env {
					env[k] = v
				}
				s := newSession(t, env)

				rendered := strings.Contains(s.Screen().Text(), "search") &&
					strings.Contains(s.Screen().Text(), "suggestions")
				if !rendered {
					t.Fatalf("interface did not render:\n%s", s.Screen().Text())
				}
				if theme.name == "mono" {
					if matches := colorSGR.FindAll(s.Raw(), -1); len(matches) != 0 {
						t.Errorf("NO_COLOR stream contains %d color sequence(s), first: %q",
							len(matches), matches[0])
					}
				}
			})
		}
	}
}

// TestResizeRerender reproduces the QA freeze: resizing the terminal
// must re-render the definition at the new width and leave the app
// responsive. Before the fix, the width-change hook called
// QueueUpdateDraw from inside tview's draw (which holds the app lock
// while BeforeDraw hooks run) — QueueUpdate blocks on an unbuffered
// channel the event loop can no longer service, deadlocking the app.
func TestResizeRerender(t *testing.T) {
	skipIfShort(t)
	s := newSession(t, nil)

	// Define a word so the definition pane has wrapped content.
	s.Send([]byte("time"))
	s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
		return strings.Contains(s.Text(), "part of speech") ||
			strings.Contains(s.Text(), "Noun")
	})
	s.Send([]byte("\r"))
	s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
		return !strings.Contains(s.Text(), "Type a word to begin")
	})

	// Widen the terminal.
	s.Resize(160, rows-1)

	// Re-layout + re-render proof: rows longer than the old 100-column
	// layout only exist once the app redraws at the new size (a frozen
	// or deadlocked app keeps the old layout forever).
	reLaidOut := s.WaitFor(5*time.Second, func(s *e2etest.Screen) bool {
		longest := 0
		for _, line := range strings.Split(s.Text(), "\n") {
			if len([]rune(line)) > longest {
				longest = len([]rune(line))
			}
		}
		return longest > 100
	})
	if !reLaidOut {
		_ = os.WriteFile("/tmp/opencode/live-raw-after-resize.bin", s.Raw(), 0o644)
		text := s.Screen().Text()
		longest := 0
		var longestLine string
		for _, line := range strings.Split(text, "\n") {
			if len([]rune(line)) > longest {
				longest = len([]rune(line))
				longestLine = line
			}
		}
		t.Errorf("app did not re-layout after resize; maxRow=%d rawBytes=%d longestLine=%q",
			longest, len(s.Raw()), longestLine)
	}
	if text := s.Screen().Text(); !strings.Contains(text, "Noun") && !strings.Contains(text, "Verb (transitive)") {
		t.Errorf("definition content missing after resize:\n%s", text)
	}

	// Responsiveness probe: a deadlocked app cannot move focus.
	s.Send([]byte(tabKey))
	if !s.WaitFor(3*time.Second, func(s *e2etest.Screen) bool {
		return focusedPane(s) == "suggestions" ||
			focusedPane(s) == "definition"
	}) {
		t.Fatal("app froze after resize (no focus response to Tab)")
	}
}
