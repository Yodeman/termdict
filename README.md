<div align="center">

<h1><b>TermDict</b></h1>

<p>
<b>A dictionary that lives in your terminal — offline by default.</b>
</p>

[![CI](https://github.com/yodeman/termdict/actions/workflows/ci.yml/badge.svg)](https://github.com/yodeman/termdict/actions/workflows/ci.yml)
[![Data integrity](https://github.com/yodeman/termdict/actions/workflows/data.yml/badge.svg)](https://github.com/yodeman/termdict/actions/workflows/data.yml)
[![Latest release](https://img.shields.io/github/v/release/yodeman/termdict?include_prereleases)](https://github.com/yodeman/termdict/releases)

<img src="https://github.com/Yodeman/termdict/assets/59335237/01b8da72-58ce-48de-8dea-45cf169dee74" alt="TermDict running in a terminal: a search box with suggestions on the left and the definition of a word on the right, with Help/About/Update/Quit buttons along the bottom." width="720">

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

### Linux & macOS

```sh
curl -fsSLO https://raw.githubusercontent.com/yodeman/termdict/main/install.sh
sh install.sh                 # review the script, then run it
```

Options: `--version vX.Y.Z` pin a release · `--system` install to
`/usr/local/bin` · `--prefix DIR` choose another directory ·
`uninstall` remove it (your data is kept).

### Windows (PowerShell)

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/yodeman/termdict/main/install.ps1 -OutFile install.ps1
.\install.ps1                 # adds %LOCALAPPDATA%\Programs\termdict\bin to your user PATH
```

Remove again any time with `.\install.ps1 -Uninstall`.

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
