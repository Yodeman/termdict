#!/bin/sh
# Terminal color-profile matrix (plan v2 phase 4 M3):
#   4 themes (ocean, catppuccin, paper, mono/NO_COLOR)
#   x 3 profiles (8-color xterm, 256-color, truecolor via COLORTERM)
#
# Each combination boots the TUI in a pty and must render the search
# pane. Mono additionally asserts a color-free output stream (no color
# SGR sequences). Requires python3 (pty) — CI: ubuntu runner.
set -u

cd "$(dirname "$0")/.."
BIN=$(mktemp /tmp/termdict-matrix.XXXXXX)
trap 'rm -f "$BIN"' EXIT
go build -o "$BIN" . || exit 1

python3 - "$BIN" <<'EOF'
import os, pty, re, select, signal, struct, sys, termios, time, fcntl

binpath = sys.argv[1]
themes = [("ocean", {}), ("catppuccin", {"TERMDICT_THEME": "catppuccin"}),
          ("paper", {"TERMDICT_THEME": "paper"}),
          ("mono", {"NO_COLOR": "1"})]
profiles = [("8-color", {"TERM": "xterm"}),
            ("256-color", {"TERM": "xterm-256color"}),
            ("truecolor", {"TERM": "xterm-256color", "COLORTERM": "truecolor"})]

failures = 0
for theme_name, theme_env in themes:
    for profile_name, profile_env in profiles:
        env = dict(os.environ)
        env.update({"HOME": "/tmp", "TERM": "xterm"})
        env.pop("NO_COLOR", None); env.pop("TERMDICT_THEME", None); env.pop("COLORTERM", None)
        env.update(theme_env); env.update(profile_env)

        pid, fd = pty.fork()
        if pid == 0:
            os.execve(binpath, [binpath], env)
        fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", 30, 100, 0, 0))
        out = b""
        end = time.time() + 2.0
        while time.time() < end:
            r, _, _ = select.select([fd], [], [], 0.2)
            if r:
                try:
                    out += os.read(fd, 65536)
                except OSError:
                    break
        os.kill(pid, signal.SIGKILL)
        os.waitpid(pid, 0)

        label = f"{theme_name:11} on {profile_name:9}"
        rendered = b"search" in out and b"suggestions" in out
        clean = True
        if theme_name == "mono":
            color_sgrs = re.findall(
                rb"\x1b\[(?:3[0-7]|9[0-7]|38;5;\d+|38;2;\d+;\d+;\d+"
                rb"|48;5;\d+|48;2;\d+;\d+;\d+)m", out)
            clean = not color_sgrs
        if rendered and clean:
            print(f"ok   {label}")
        else:
            print(f"FAIL {label} rendered={rendered} color-free={clean}")
            failures += 1

print()
if failures:
    print(f"PTY MATRIX FAILED: {failures} combination(s)")
    sys.exit(1)
print("PTY MATRIX PASSED")
EOF
