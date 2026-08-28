<div align="center">

<h1><b>TermDict</b></h1>

<p>
<b>A dictionary that lives in your terminal — offline by default.</b>
</p>

[![CI](https://github.com/yodeman/termdict/actions/workflows/ci.yml/badge.svg)](https://github.com/yodeman/termdict/actions/workflows/ci.yml)
[![Data integrity](https://github.com/yodeman/termdict/actions/workflows/data.yml/badge.svg)](https://github.com/yodeman/termdict/actions/workflows/data.yml)
[![Latest release](https://img.shields.io/github/v/release/yodeman/termdict?include_prereleases)](https://github.com/yodeman/termdict/releases)

<img src="https://github.com/Yodeman/termdict/assets/59335237/01b8da72-58ce-48de-8dea-45cf169dee74" alt="TermDict running in a terminal: a search box with suggestions on the left and the definition of a word on the right, with Help/About/Update/Quit buttons along the bottom. Current releases add a persistent footer with key hints, color themes, and numbered-sense definitions." width="720">

<!-- TODO(maintainer): re-capture the screenshot on the current theme
     (ocean) after publishing v0.2.1 — the image above predates themes,
     the footer and the numbered-sense definitions. -->

</div>

## Why TermDict?

- **Works instantly, works offline.** Every binary ships with an
  embedded core of ~22,000 of the most common English words. No
  accounts, no first-run downloads, no network required.
- **The full dictionary is one key away.** 106,000+ words (Webster's
  1913) download incrementally inside the app whenever you want them.
- **Two interfaces.** A friendly TUI with type-ahead suggestions and
  mouse support, plus a scriptable CLI for pipes and scripts.
- **Cross-platform.** Linux, macOS and Windows, amd64 and arm64.

## Installation

### Linux & macOS — one command

```sh
curl -fsSL https://raw.githubusercontent.com/yodeman/termdict/main/install.sh | sh
```

With options (still a single invocation):

```sh
curl -fsSL https://raw.githubusercontent.com/yodeman/termdict/main/install.sh | sh -s -- --version v0.2.1
curl -fsSL https://raw.githubusercontent.com/yodeman/termdict/main/install.sh | sh -s -- --system   # /usr/local/bin
curl -fsSL https://raw.githubusercontent.com/yodeman/termdict/main/install.sh | sh -s -- uninstall  # remove (data kept)
```

`--prefix DIR` chooses another directory. The script never edits your shell
rc files — it prints a PATH hint instead.

### Windows (PowerShell) — one command

```powershell
irm https://raw.githubusercontent.com/yodeman/termdict/main/install.ps1 | iex
```

With options:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/yodeman/termdict/main/install.ps1))) -Version v0.2.1
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/yodeman/termdict/main/install.ps1))) -Uninstall
```

Installs to `%LOCALAPPDATA%\Programs\termdict\bin` and adds it to your user
PATH (restart the terminal afterwards).

> **About the one-liners & security.** The installer verifies the downloaded
> archive against the SHA256 checksum published **alongside it in the same
> release**. That protects you from corrupted or truncated downloads — it
> does **not** protect against a compromised release (whoever can replace
> the archive can replace the checksum beside it). If you'd rather review
> before running, download the script and the archive from the
> [releases page](https://github.com/yodeman/termdict/releases) manually —
> the scripts work the same way from a local file.

### Via Go

```sh
go install github.com/yodeman/termdict@latest
```

Two caveats: `--version` will report `dev` (version metadata is injected at
release builds only), and database updates will follow the development data
channel (tracking `main`) instead of the data pinned to a release. Updates
themselves work normally — `termdict update` only refreshes database files,
never the binary. Prefer the one-liners above if you want release-pinned
data.

### From source

```sh
git clone https://github.com/yodeman/termdict && cd termdict
make install                  # installs to $(go env GOPATH)/bin
```

Prebuilt archives are also on the
[releases page](https://github.com/yodeman/termdict/releases) if you
prefer to copy a binary to your `PATH` yourself.

## Using the TUI

Run `termdict` with no arguments. Start typing; suggestions appear as
you go, Enter shows the definition.

| Key | Action |
|---|---|
| type | live search suggestions |
| Enter | define the word in the search box |
| Tab / Shift+Tab | move between search, suggestions, definition |
| F1 | help (incl. part-of-speech legend) |
| F2 | about |
| Ctrl+U | update database / download full dictionary |
| Ctrl+Q or Ctrl+C | quit |

Mouse clicks work everywhere too.

### Color themes

| Environment | Result |
|---|---|
| `TERMDICT_THEME=ocean` | default — blue accents on dark (no config needed) |
| `TERMDICT_THEME=catppuccin` | [Catppuccin Mocha](https://github.com/catppuccin/catppuccin) palette |
| `TERMDICT_THEME=paper` | light palette for light terminal schemes |
| `NO_COLOR` (set to anything) | completely color-free interface, per [no-color.org](https://no-color.org) |

Every state is conveyed in text as well as color, and all palettes pass
WCAG AA contrast checks automatically (enforced by the test suite).

## Using the CLI

```console
$ termdict time            # plain-text definition, pipe-friendly
$ termdict recieve         # misspelled? get hints
No results for "recieve". Run 'termdict download' to get the full dictionary.
Did you mean: relieve, believe, recede, receive, recipe?
$ termdict update          # pull changed dictionary files
$ termdict download        # fetch the complete word list
$ termdict --version
termdict v0.2.0 (commit abc1234) (go1.26.0)
```

Exit codes: `0` success · `1` word not found · `2` usage error ·
`3` runtime error. Stdout carries only definitions, so
`termdict time | grep -c part` just works.

## The offline core & the full database

TermDict embeds the ~22k most frequent headwords so the app is useful
the moment it starts. Words outside the core (rare, archaic) show a
hint instead of a definition — press <kbd>Ctrl+U</kbd> and choose
**Download Full Dictionary**, or run `termdict download`, to add all
106k entries. Updates after that are incremental: only files that
changed upstream are re-downloaded, and every file is verified against
SHA256 checksums.

Data lives under your platform's data directory:

| OS | Location |
|---|---|
| Linux | `$XDG_DATA_HOME/termdict` (default `~/.local/share/termdict`) |
| macOS | `~/Library/Application Support/termdict` |
| Windows | `%LOCALAPPDATA%\termdict` |

Upgrading from v0.1.0? Your existing database migrates automatically
on first launch.

## Data license & attribution

The definitions come from **Webster's Unabridged Dictionary (1913
edition)** via [Project Gutenberg](https://www.gutenberg.org/) and
[OPTED — The Online Plain Text English Dictionary](https://www.mso.anu.edu.au/~ralph/OPTED/),
which distributes them as public domain. Per OPTED's terms we keep the
content free, plain-text and at no cost, and gratefully acknowledge
OPTED, Project Gutenberg and the 1913 Webster's Unabridged Dictionary.
(Contact Project Gutenberg before including this data in commercial
products.)

The embedded-core *ranking* uses the
[FrequencyWords](https://github.com/hermitdave/FrequencyWords) `en_50k`
list (derived from the OpenSubtitles 2018 corpus via OPUS,
[CC-BY-SA-4.0](./word_dbase/frequency/DATA_LICENSE.md)) — used solely
to decide which words ship offline.

## Development

```sh
make test      # go test ./... -race
make lint      # golangci-lint
make build     # bin/termdict with version metadata
sh scripts/cli_e2e.sh
go run ./cmd/dbasecheck   # validate database invariants
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the repository layout, the
database entry format and PR expectations, and
[docs/smoke-testing.md](docs/smoke-testing.md) for the per-OS manual
test checklist. Changes are described in the
[CHANGELOG](CHANGELOG.md).

## License

Code: [MIT](LICENSE) © Paul Oyelabi.
Dictionary data: public domain (see attribution above); frequency
ranking: CC-BY-SA-4.0.
