//go:build unix

package e2etest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// Cell is one terminal grid position with its resolved colors.
type Cell struct {
	Ch   rune
	FG   RGB
	BG   RGB
	Bold bool
}

// RGB is a 24-bit color.
type RGB [3]int

// Screen is a minimal VT emulator — enough for tcell's output stream
// (CUP, SGR, ED/EL, OSC hyperlinks, charset designators) — tracking
// per-cell colors so tests can assert on rendered styling, not just
// text.
type Screen struct {
	cols, rows int
	cells      [][]Cell
	x, y       int
	fg, bg     RGB
	bold       bool
}

func defaultFG() RGB { return RGB{208, 215, 239} }
func defaultBG() RGB { return RGB{30, 30, 46} }

// NewScreen returns an empty screen with ocean-ish default colors.
func NewScreen(cols, rows int) *Screen {
	s := &Screen{cols: cols, rows: rows}
	s.Clear()
	return s
}

// Clear resets every cell and the cursor.
func (s *Screen) Clear() {
	blank := Cell{Ch: ' ', FG: defaultFG(), BG: defaultBG()}
	s.cells = make([][]Cell, s.rows)
	for y := range s.cells {
		s.cells[y] = make([]Cell, s.cols)
		for x := range s.cells[y] {
			s.cells[y][x] = blank
		}
	}
	s.x, s.y = 0, 0
}

// Text returns the visible screen contents, right-trimmed per row.
func (s *Screen) Text() string {
	lines := make([]string, s.rows)
	for y, row := range s.cells {
		var b strings.Builder
		for _, cell := range row {
			b.WriteRune(cell.Ch)
		}
		lines[y] = strings.TrimRight(b.String(), " ")
	}
	return strings.Join(lines, "\n")
}

// FocusedCorner returns the position of tview's double-line top-left
// border corner (drawn only for the focused widget), or ok=false.
func (s *Screen) FocusedCorner() (x, y int, ok bool) {
	for y := range s.cells {
		for x := range s.cells[y] {
			if s.cells[y][x].Ch == '╔' {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// Feed applies a raw terminal byte stream to the screen.
func (s *Screen) Feed(data string) {
	runes := []rune(data)
	for i := 0; i < len(runes); {
		r := runes[i]
		switch {
		case r == '\x1b':
			i += s.escape(runes[i:])
		case r == '\r':
			s.x = 0
			i++
		case r == '\n':
			s.y = min(s.y+1, s.rows-1)
			i++
		case r == '\b':
			s.x = max(s.x-1, 0)
			i++
		case r >= ' ':
			if s.x < s.cols && s.y < s.rows {
				s.cells[s.y][s.x] = Cell{Ch: r, FG: s.fg, BG: s.bg, Bold: s.bold}
			}
			s.x = min(s.x+1, s.cols)
			i++
		default:
			i++
		}
	}
}

// escape consumes one escape sequence, returning its length in runes.
func (s *Screen) escape(runes []rune) int {
	if len(runes) < 2 {
		return len(runes)
	}
	switch runes[1] {
	case '[':
		return s.csi(runes)
	case ']':
		// OSC: terminated by BEL or ST (ESC \).
		for i := 2; i < len(runes); i++ {
			if runes[i] == '\x07' {
				return i + 1
			}
			if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '\\' {
				return i + 2
			}
		}
		return len(runes)
	case '(':
		if len(runes) >= 3 {
			return 3 // charset designator, e.g. ESC ( B
		}
		return len(runes)
	default:
		return 2
	}
}

// csi applies a control sequence; only the ones tcell emits matter.
// Returns the number of runes consumed.
func (s *Screen) csi(runes []rune) int {
	params := ""
	for i := 2; i < len(runes); i++ {
		r := runes[i]
		if r >= '0' && r <= '9' || r == ';' || r == '?' {
			params += string(r)
			continue
		}
		consumed := i + 1
		switch r {
		case 'H', 'f': // cursor position
			p := splitInts(params)
			row, col := 1, 1
			if len(p) > 0 {
				row = p[0]
			}
			if len(p) > 1 {
				col = p[1]
			}
			s.y = clamp(row-1, 0, s.rows-1)
			s.x = clamp(col-1, 0, s.cols-1)
		case 'J':
			mode := splitInts(params)
			m := 0
			if len(mode) > 0 {
				m = mode[0]
			}
			switch m {
			case 2:
				s.Clear()
			case 0:
				for y := s.y; y < s.rows; y++ {
					start := s.x
					if y != s.y {
						start = 0
					}
					for x := start; x < s.cols; x++ {
						s.cells[y][x] = Cell{Ch: ' ', FG: s.fg, BG: s.bg}
					}
				}
			}
		case 'K':
			for x := s.x; x < s.cols; x++ {
				s.cells[s.y][x] = Cell{Ch: ' ', FG: s.fg, BG: s.bg}
			}
		case 'm':
			s.sgr(splitInts(params))
		}
		return consumed
	}
	return len(runes)
}

// sgr applies select graphic rendition parameters (colors and attrs).
func (s *Screen) sgr(p []int) {
	if len(p) == 0 {
		p = []int{0}
	}
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case 0:
			s.fg, s.bg = defaultFG(), defaultBG()
			s.bold = false
		case 1:
			s.bold = true
		case 22:
			s.bold = false
		case 39:
			s.fg = defaultFG()
		case 49:
			s.bg = defaultBG()
		case 38, 48:
			color, consumed := extendedColor(p[i:])
			if consumed == 0 {
				return
			}
			if p[i] == 38 {
				s.fg = color
			} else {
				s.bg = color
			}
			i += consumed
		}
	}
}

// extendedColor parses 38;2;r;g;b / 38;5;n forms, returning the color
// and how many parameters it consumed (0 = malformed).
func extendedColor(p []int) (RGB, int) {
	if len(p) < 2 {
		return RGB{}, 0
	}
	switch p[1] {
	case 2:
		if len(p) < 5 {
			return RGB{}, 0
		}
		return RGB{p[2], p[3], p[4]}, 4
	case 5:
		if len(p) < 3 {
			return RGB{}, 0
		}
		return palette(p[2]), 2
	}
	return RGB{}, 0
}

func palette(idx int) RGB {
	if idx < 16 {
		base := [16]RGB{
			{0, 0, 0}, {205, 49, 49}, {13, 188, 121}, {229, 229, 16},
			{36, 114, 200}, {188, 63, 188}, {17, 168, 205}, {229, 229, 229},
			{102, 102, 102}, {241, 76, 76}, {35, 209, 139}, {245, 245, 67},
			{59, 142, 234}, {214, 112, 214}, {41, 184, 219}, {229, 229, 229},
		}
		return base[idx]
	}
	if idx < 232 {
		i := idx - 16
		steps := [6]int{0, 95, 135, 175, 215, 255}
		return RGB{steps[i/36], steps[(i/6)%6], steps[i%6]}
	}
	g := 8 + (idx-232)*10
	return RGB{g, g, g}
}

func splitInts(params string) []int {
	if params == "" {
		return nil
	}
	parts := strings.Split(params, ";")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			out = append(out, 0)
			continue
		}
		if n, err := strconv.Atoi(part); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// PTYSession runs the termdict binary under a pseudo-terminal and
// maintains a live parsed Screen of its output.
type PTYSession struct {
	mu     sync.Mutex
	raw    []byte
	eaten  int
	screen *Screen
	tty    *os.File
	cmd    *exec.Cmd
	Cols   int
	Rows   int
}

// NewPTYSession starts the binary under a pty with the given size and
// environment overrides applied on top of the parent environment.
func NewPTYSession(binary string, cols, rows int, envOverrides map[string]string) (*PTYSession, error) {
	cmd := exec.CommandContext(context.Background(), binary)
	cmd.Env = os.Environ()
	for k, v := range envOverrides {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Dir = "."

	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, fmt.Errorf("starting under pty: %w", err)
	}

	s := &PTYSession{
		screen: NewScreen(cols, rows),
		tty:    tty,
		cmd:    cmd,
		Cols:   cols,
		Rows:   rows,
	}
	go s.readLoop()
	return s, nil
}

func (s *PTYSession) readLoop() {
	buf := make([]byte, 65536)
	for {
		n, err := s.tty.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.raw = append(s.raw, buf[:n]...)
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// Send writes raw bytes to the terminal (keystrokes).
func (s *PTYSession) Send(data ...[]byte) {
	for _, d := range data {
		if _, err := s.tty.Write(d); err != nil {
			return
		}
	}
}

// Screen returns a snapshot of the parsed screen state.
func (s *PTYSession) Screen() *Screen {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.raw) > s.eaten {
		s.screen.Feed(string(s.raw[s.eaten:]))
		s.eaten = len(s.raw)
	}
	return s.screen
}

// Raw returns everything the child has written so far.
func (s *PTYSession) Raw() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.raw...)
}

// WaitFor polls the parsed screen until pred holds or the deadline
// passes. Returns true when the predicate was satisfied.
func (s *PTYSession) WaitFor(d time.Duration, pred func(*Screen) bool) bool {
	deadline := time.Now().Add(d)
	for {
		if pred(s.Screen()) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// SettleDrain gives queued draws time to land and folds all output
// into the screen state. Use before absence-of-change assertions.
func (s *PTYSession) SettleDrain(d time.Duration) {
	time.Sleep(d)
	s.Screen()
}

// Resize changes the pty window size (tcell full-repaints in response).
func (s *PTYSession) Resize(cols, rows int) {
	_ = pty.Setsize(s.tty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Close kills the child and releases the terminal.
func (s *PTYSession) Close() {
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	_ = s.tty.Close()
	_ = s.cmd.Wait()
}
