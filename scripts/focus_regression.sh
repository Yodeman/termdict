#!/bin/sh
# Shell wrapper for the scripted-pty focus regression suite
# (scripts/pty_focus.py). See that file for what is covered.
set -eu
cd "$(dirname "$0")/.."
exec python3 scripts/pty_focus.py all
