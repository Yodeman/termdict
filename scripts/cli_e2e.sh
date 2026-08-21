#!/bin/sh
# End-to-end checks for the termdict CLI surface (plan phase 4, M4).
# Uses an isolated HOME; lookups are served by the embedded core, so no
# network or pre-seeded database is required.
set -u

cd "$(dirname "$0")/.."
BIN=$(mktemp /tmp/termdict-e2e.XXXXXX)
SMOKE_HOME=$(mktemp -d /tmp/termdict-e2e-home.XXXXXX)
trap 'rm -f "$BIN"; rm -rf "$SMOKE_HOME"' EXIT

go build -o "$BIN" . || exit 1

export HOME="$SMOKE_HOME"
failures=0

expect_exit() { # expect_exit <want> <desc> <cmd...>
    want=$1; desc=$2; shift 2
    "$@" >/dev/null 2>&1
    got=$?
    if [ "$got" != "$want" ]; then
        echo "FAIL $desc: exit $got, want $want"
        failures=$((failures + 1))
    else
        echo "ok   $desc (exit $got)"
    fi
}

expect_stdout() { # expect_stdout <substr> <desc> <cmd...>
    substr=$1; desc=$2; shift 2
    out=$("$@" 2>/dev/null)
    case "$out" in
        *"$substr"*) echo "ok   $desc" ;;
        *) echo "FAIL $desc: stdout missing \"$substr\""; failures=$((failures + 1)) ;;
    esac
}

expect_stderr() { # expect_stderr <substr> <desc> <cmd...>
    substr=$1; desc=$2; shift 2
    err=$("$@" 2>&1 >/dev/null)
    case "$err" in
        *"$substr"*) echo "ok   $desc" ;;
        *) echo "FAIL $desc: stderr missing \"$substr\""; failures=$((failures + 1)) ;;
    esac
}

expect_empty() { # expect_empty <desc> <cmd...>
    desc=$1; shift
    out=$("$@" 2>/dev/null)
    if [ -z "$out" ]; then echo "ok   $desc"; else
        echo "FAIL $desc: stdout not empty: $out"; failures=$((failures + 1))
    fi
}

echo "== lookup =="
expect_exit 0 "hit exits 0"                    "$BIN" time
expect_exit 1 "miss exits 1"                   "$BIN" zzzzqqqxx
expect_exit 1 "case-mixed miss exits 1"        "$BIN" ZZZZQQQXX
expect_stdout "part of speech" "hit prints payload"      "$BIN" time
expect_stdout "time"           "first line is headword"  sh -c "\"$BIN\" time | head -1"
expect_empty   "miss keeps stdout clean"                 "$BIN" zzzzqqqxx
expect_stderr  "No results for" "miss explains on stderr" "$BIN" zzzzqqqxx

echo "== verbs & flags =="
expect_exit 0 "--version exits 0"              "$BIN" --version
expect_stdout "termdict"       "version line"  "$BIN" version
expect_exit 0 "--help exits 0"                 "$BIN" --help
expect_stdout "Exit codes:"     "help documents codes" "$BIN" help
expect_exit 2 "unknown flag exits 2"           "$BIN" --bogus
expect_exit 2 "multi-word unquoted exits 2"    "$BIN" ice cream
expect_stderr  "unknown flag"   "flag error on stderr"  "$BIN" --bogus
expect_exit 0 "-- protects reserved verb" sh -c "\"$BIN\" -- update >/dev/null; true"

echo "== piping contract =="
first=$(./bin/termdict time 2>/dev/null | head -1)
[ "$first" = "time" ] && echo "ok   pipe to head yields headword first" || {
    echo "FAIL pipe to head: got \"$first\""; failures=$((failures + 1)); }

grep_count=$("$BIN" time 2>/dev/null | grep -c "part of speech")
case "$grep_count" in
    ''|0) echo "FAIL greppable output: zero part-of-speech lines"; failures=$((failures + 1)) ;;
    *)    echo "ok   output greppable ($grep_count POS lines)" ;;
esac

echo "== update with blackholed network =="
HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 \
    expect_exit 3 "update failure exits 3" env HOME="$HOME" HTTP_PROXY=http://127.0.0.1:9 \
    HTTPS_PROXY=http://127.0.0.1:9 "$BIN" update

echo
if [ "$failures" -gt 0 ]; then
    echo "E2E FAILED: $failures check(s)"
    exit 1
fi
echo "E2E PASSED"
