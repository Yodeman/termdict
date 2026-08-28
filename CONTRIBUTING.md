# Contributing to termdict

First of all, thank you for taking the time to contribute.

The following provides you with some guidance on how to contribute to
this project. Mainly, it is meant to save us all some time so please
read it.

Please note that this document is work in progress so I might add to it
in the future.

## Issues

- Please include enough information so everybody understands your request.
- Screenshots or code that illustrates your point always helps.
- It's fine to ask for help.
- If you request a new feature, state your motivation. It should be something that others will also need.

## Pull Requests

If you have a feature request, open an issue first before sending a
pull request, and allow for some discussion.

Before opening a PR, make sure the local gates pass:

```sh
make test   # go test ./... -race (includes the end-to-end tests)
make lint   # golangci-lint (0 issues expected)
make vet
```

CI runs the same checks on Linux, macOS and Windows with the two
currently supported Go releases.

## Development environment

- A currently supported Go release (see `go.mod` for the floor).
- `make build` produces `bin/termdict` with version metadata injected;
  `make install` puts it on `$(go env GOPATH)/bin`.
- Optional: [golangci-lint](https://golangci-lint.run/welcome/install/)
  for `make lint`, and GoReleaser for `goreleaser release --snapshot`
  dry-runs.

### Repository layout

| Path | Purpose |
|---|---|
| `internal/dict` | data model + pure lookup/suggest/fuzzy/render logic |
| `internal/data` | letter-file store, changes tracker, HTTP update client |
| `internal/data/embedded` | generated offline core subset (`go:embed`) |
| `internal/tui` | tview interface wiring |
| `internal/cli` | command-line verbs, exit codes, stream contract |
| `cmd/dbasebuild` | generates the embedded core subset + checksums |
| `cmd/dbasecheck` | validates database invariants (run in CI) |
| `cmd/getwords`, `cmd/htmltojson` | OPTED scrape/converter dev tools |

### Testing the installers locally (maintainers)

Both installers accept a `--from-dir` / `-FromDir` flag that installs from a
local directory instead of downloading — this flag is deliberately **not**
part of the public README instructions:

```sh
goreleaser release --snapshot --clean     # produces dist/ archives
cat install.sh | sh -s -- --from-dir dist --version <ver> --prefix /tmp/prefix
```

```powershell
goreleaser release --snapshot --clean
.\install.ps1 -FromDir .\dist -Version <ver>
.\install.ps1 -Uninstall                  # clean up
```

## More on contributing to dictionary words database

Each file in the [words database](https://github.com/Yodeman/termdict/tree/main/word_dbase/json)
is named after the english alphabet, each containing words starting
with that particular alphabet.

When contributing to the database, locate the appropriate file and the
appropriate position (words are sorted in lexicographical order) to
place the new word. The format for each word is shown below:

```json
"new word in lower case" : {
    "word" : "new word in lower case",
    "alternate_spellings": ["list", "of", "alternative", "spellings"],
    "definitions": [
        {
            "part_of_speech": "",
            "definition": ""
        }
    ]
}
```

After editing, validate your change:

```sh
go run ./cmd/dbasecheck          # full invariant check
go test ./...                    # embedded-core tests still pass
```

Note: the embedded core subset under `internal/data/embedded/core/` is
generated — never edit those files by hand. After changing
`word_dbase/json`, regenerate with:

```sh
go run ./cmd/dbasebuild
```

and commit both the source files and the regenerated core together.
