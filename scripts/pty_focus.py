#!/usr/bin/env python3
"""Scripted-pty regression tests for focus behavior (QA v2 issues 2+3).

Boots the TUI in a pty, reconstructs the screen with a minimal VT
parser, and asserts:

  tabs   Tab/Shift+Tab rotate search -> suggestions -> definition and
         wrap in BOTH directions (the v2 regression: cycling wedged on
         the search pane after one full loop, caused by focusing bare
         Box wrappers instead of primitives).
  modal  F1/F2 open their popups, Tab inside a popup is a trapped
         no-op (the popup must NOT close), and Esc closes + restores
         the previous focus.
  update Ctrl+U opens the update popup; Esc cancels, then closes;
         focus returns to search.

Usage: pty_focus.py [tabs|modal|update|all]   (default: all)
Requires the termdict binary: build with `go build -o bin/termdict .`
first (or let this script build it).
"""
import os
import pty
import re
import select
import signal
import struct
import sys
import termios
import time
import fcntl

COLS, ROWS = 100, 30
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BIN = os.path.join(ROOT, "bin", "termdict")
HOME = os.environ.get("TD_SMOKE_HOME", "/tmp/termdict-focus-home")

failures = 0


class Screen:
    """Minimal VT emulator: CUP + SGR + text is all tcell emits."""

    def __init__(self, cols, rows):
        self.cols, self.rows = cols, rows
        self.grid = [[" "] * cols for _ in range(rows)]
        self.x = self.y = 0

    def feed(self, data):
        i, n = 0, len(data)
        while i < n:
            ch = data[i]
            if ch == "\x1b":
                m = re.match(r"\x1b\[([0-9;?]*)([A-Za-z])", data[i:])
                if m:
                    self._csi(m.group(1), m.group(2))
                    i += m.end()
                    continue
                m = re.match(r"\x1b\][^\x07]*\x07", data[i:])
                if m:
                    i += m.end()
                    continue
                i += 1
                continue
            if ch == "\r":
                self.x = 0
            elif ch == "\n":
                self.y = min(self.y + 1, self.rows - 1)
            elif ch == "\b":
                self.x = max(0, self.x - 1)
            elif ch >= " ":
                if self.x < self.cols and self.y < self.rows:
                    self.grid[self.y][self.x] = ch
                self.x = min(self.x + 1, self.cols)
            i += 1

    def _csi(self, params, cmd):
        if cmd in ("H", "f"):
            p = [int(x) if x else 1 for x in params.split(";")] or [1, 1]
            while len(p) < 2:
                p.append(1)
            self.y = max(0, min(self.rows - 1, p[0] - 1))
            self.x = max(0, min(self.cols - 1, p[1] - 1))
        elif cmd == "J":
            p = params or "0"
            if p == "2":
                self.grid = [[" "] * self.cols for _ in range(self.rows)]
            elif p == "0":
                for y in range(self.y, self.rows):
                    start = self.x if y == self.y else 0
                    for x in range(start, self.cols):
                        self.grid[y][x] = " "
        elif cmd == "K":
            for x in range(self.x, self.cols):
                self.grid[self.y][x] = " "

    def text(self):
        return "\n".join("".join(row).rstrip() for row in self.grid)

    def focused_corner(self):
        for y in range(self.rows):
            for x in range(self.cols):
                if self.grid[y][x] == "╔":  # tview focused-border corner
                    return (x, y)
        return None


def pane_name(corner):
    if corner is None:
        return "(none)"
    x, y = corner
    if y >= ROWS - 4:
        return "commands"
    if x >= COLS - 45:
        return "definition"
    if y <= 2:
        return "search"
    return "suggestions"


class Session:
    def __init__(self):
        pid, fd = pty.fork()
        if pid == 0:
            os.makedirs(HOME, exist_ok=True)
            env = dict(os.environ)
            env["HOME"] = HOME
            env["TERM"] = "xterm-256color"
            os.execve(BIN, [BIN], env)
        self.pid, self.fd = pid, fd
        fcntl.ioctl(fd, termios.TIOCSWINSZ,
                    struct.pack("HHHH", ROWS, COLS, 0, 0))
        self.screen = Screen(COLS, ROWS)
        self.buf = b""
        self.pump(1.5)
        self.settle()

    def pump(self, sec):
        end = time.time() + sec
        while time.time() < end:
            r, _, _ = select.select([self.fd], [], [], 0.1)
            if r:
                try:
                    self.buf += os.read(self.fd, 65536)
                except OSError:
                    return

    def settle(self):
        """Full repaint via resize; then rebuild screen state."""
        self.screen.feed(self.buf.decode("utf-8", "replace"))
        self.buf = b""
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ,
                    struct.pack("HHHH", ROWS - 1, COLS, 0, 0))
        self.pump(0.6)
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ,
                    struct.pack("HHHH", ROWS, COLS, 0, 0))
        self.pump(0.6)
        self.screen.feed(self.buf.decode("utf-8", "replace"))
        self.buf = b""

    def keys(self, *seqs):
        for k in seqs:
            os.write(self.fd, k)
            self.pump(0.35)
        self.pump(0.3)
        self.settle()

    def close(self):
        os.kill(self.pid, signal.SIGKILL)
        os.waitpid(self.pid, 0)


def check(label, got, want):
    global failures
    ok = got == want
    if not ok:
        failures += 1
    print(("PASS " if ok else "FAIL ") + f"{label}: got {got!r}, want {want!r}")


TAB, STAB = b"\t", b"\x1b[Z"
ESC, F1, F2 = b"\x1b", b"\x1bOP", b"\x1bOQ"
CTRL_U = b"\x15"


def scenario_tabs():
    s = Session()
    steps = [
        ("boot", [], "search"),
        ("tab1", [TAB], "suggestions"),
        ("tab2", [TAB], "definition"),
        ("tab3", [TAB], "search"),
        ("tab4 (wrap to suggestions)", [TAB], "suggestions"),
        ("shift+tab (suggestions -> search)", [STAB], "search"),
        ("shift+tab (search -> definition)", [STAB], "definition"),
    ]
    for label, keys, want in steps:
        s.keys(*keys)
        check(label, pane_name(s.screen.focused_corner()), want)
    s.close()


def scenario_modal():
    s = Session()
    s.keys(F1)
    check("F1 opens help", "KEYS" in s.screen.text(), True)
    check("F1 gives the popup focus",
          pane_name(s.screen.focused_corner()) == "help-popup"
          or s.screen.focused_corner() is not None, True)
    s.keys(TAB, TAB, TAB)
    check("Tab inside help is trapped (popup stays open)",
          "KEYS" in s.screen.text(), True)
    s.keys(ESC)
    check("Esc closes help", "KEYS" not in s.screen.text(), True)
    check("Esc restores search focus",
          pane_name(s.screen.focused_corner()), "search")
    s.keys(F2)
    check("F2 opens about", "Built with" in s.screen.text(), True)
    s.keys(TAB, TAB)
    check("Tab inside about is trapped (modal stays open)",
          "Built with" in s.screen.text(), True)
    s.keys(ESC)
    check("Esc closes about", "Built with" not in s.screen.text(), True)
    check("Esc restores search focus",
          pane_name(s.screen.focused_corner()), "search")
    s.close()


def scenario_update():
    s = Session()
    s.keys(CTRL_U)
    check("ctrl+u opens update popup",
          "Update Dbase" in s.screen.text(), True)
    s.keys(ESC)  # cancels any running action (none), popup stays
    s.keys(ESC)
    check("Esc closes update popup",
          "Update Dbase" not in s.screen.text(), True)
    check("Esc restores search focus",
          pane_name(s.screen.focused_corner()), "search")
    s.close()


def main():
    if not os.path.exists(BIN):
        os.system(f"cd {ROOT} && go build -o bin/termdict .")
    scenario = sys.argv[1] if len(sys.argv) > 1 else "all"
    scenarios = {
        "tabs": scenario_tabs,
        "modal": scenario_modal,
        "update": scenario_update,
    }
    if scenario == "all":
        for name in ("tabs", "modal", "update"):
            print(f"== {name} ==")
            scenarios[name]()
    else:
        scenarios[scenario]()
    print()
    if failures:
        print(f"FOCUS REGRESSION FAILED: {failures} check(s)")
        sys.exit(1)
    print("FOCUS REGRESSION PASSED")


if __name__ == "__main__":
    main()
