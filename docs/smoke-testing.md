# Manual Smoke-Testing Checklist

Run this checklist per OS before cutting a release. CI compiles all
targets, but only a human with the OS in front of them can confirm the
TUI behaves. Minimum viable terminal: **60×24**; comfortable: 100×30.

## Setup

```sh
go test ./... -race && go build -o bin/termdict .
```

Use an isolated HOME so real data is never touched:

| OS | Isolated data dir |
|---|---|
| Linux/BSD | `HOME=/tmp/td-smoke ./bin/termdict` |
| macOS | `HOME=/tmp/td-smoke ./bin/termdict` |
| Windows (PowerShell) | `$env:USERPROFILE='C:\temp\td-smoke'; .\termdict.exe` |

## 1. First boot — offline by default

- [ ] App starts instantly (no download prompt, no visible network wait).
- [ ] Search box + suggestions list render; typing `time` shows a
      definition (served from the embedded core).
- [ ] Typing a rare word (e.g. `zymurgy`) shows the not-found message.
- [ ] Mouse clicks focus widgets; Tab / Shift+Tab cycles
      search → suggestions → definition.

## 2. Popups & keys

- [ ] F1 opens help; Escape closes it and refocuses search.
- [ ] F2 shows About with correct version (`dev` for local builds).
- [ ] Ctrl+U opens the update popup with **Update Dbase** and
      **Download Full Dictionary** buttons.
- [ ] With network: Download Full completes with progress lines
      (`n/26 — file`), then "Done updating database."
- [ ] Without network (disable Wi-Fi first): the action fails with a
      readable message, no crash; Escape closes the popup.
- [ ] Pressing Ctrl+U repeatedly never stacks duplicate updates.
- [ ] Ctrl+Q quits cleanly (terminal restored, no garbage).

## 3. Persistence layout

Confirm data lands in the platform directory:

| OS | Expected `<data>/dbase/json/wb1913_*.json` under |
|---|---|
| Linux | `$XDG_DATA_HOME/termdict`, else `~/.local/share/termdict` |
| macOS | `~/Library/Application Support/termdict` |
| Windows | `%LOCALAPPDATA%\termdict` |

## 4. Migration from v0.1.0

Seed a legacy install before first launch of the new binary:

| OS | Create `<legacy>/dbase/json/wb1913_a.json` (+ any JSON) under |
|---|---|
| Linux/macOS | `~/.local/termdict` |
| Windows | `%LOCALAPPDATA%\..\..\termdict` i.e. `%LOCALAPPDATA%\termdict` |

- [ ] First launch migrates: files appear under the new location,
      `.migrated-v2` marker exists, legacy tree is removed.
- [ ] Second launch changes nothing (marker honored).
- [ ] If new-layout data already exists, migration is skipped and the
      legacy tree is left untouched.

## 5. Terminal robustness

- [ ] Resize the window during use — layout reflows without artifacts.
- [ ] At 60 columns the search/suggestion panels remain usable.
- [ ] Ctrl+C does not corrupt the terminal state (tview handles it;
      if it exits, relaunch works).

## 6. Windows-specific extras

- [ ] Windows Terminal and classic conhost both render borders/colors.
- [ ] Quick-edit mode does not freeze rendering while selecting text.
- [ ] Paths with spaces in `%LOCALAPPDATA%` are handled.

Report failures at <https://github.com/yodeman/termdict/issues> with OS,
terminal, terminal size, and steps to reproduce.
