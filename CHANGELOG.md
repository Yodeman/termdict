# Changelog

All notable changes to TermDict are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning follows [Semantic Versioning](https://semver.org/).

## [v0.2.0] - 2026-08-21

### Added

- **Offline by default**: an embedded core dictionary of ~22,000 of the
  most frequent headwords ships inside every binary — lookups work with
  zero setup and zero network.
- **CLI mode**: `termdict <word>` prints plain-text definitions you can
  pipe and grep; `termdict update` refreshes changed dictionary files;
  `termdict download` fetches the complete 106k-word database.
  Documented exit codes (`0` ok, `1` not found, `2` usage, `3` error).
- **Did-you-mean suggestions** after a missed lookup, in both the TUI
  and the CLI.
- **Download progress bar** with live counts and Esc-to-cancel during
  database updates/downloads.
- One-command installers: `install.sh` (Linux/macOS) and `install.ps1`
  (Windows), plus automated release archives for
  linux/darwin/windows × amd64/arm64 via GoReleaser.
- Downloaded files are verified against SHA256 checksums; release
  builds pin their update channel to their own release assets.
- `termdict --version` reports build version, commit and Go runtime.

### Changed

- Data now lives in platform-standard directories
  (`~/.local/share/termdict`, `~/Library/Application Support/termdict`,
  `%LOCALAPPDATA%\termdict`). Existing v0.1.0 installs are migrated
  automatically on first run.
- Suggestions are case-insensitive; an empty search box no longer
  lists the first 50 words of the dictionary.
- Unknown words show an explicit not-found message instead of an empty
  definition pane.
- Search column shrinks gracefully on narrow terminals (~60 columns).

### Fixed

- Interrupted or partially failed database downloads no longer mark
  failed files as synced; they resume on the next run.
- Uppercase input no longer breaks the suggestion list.
- HTTP layer: request timeouts, bounded retries with backoff, response
  bodies closed on every path.
- The dev tools that build the database compile again and capture
  hyphenated/apostrophized headwords that were silently dropped before.

## [v0.1.0] - 2023

Initial release: tview-based terminal dictionary over the OPTED
(Webster's 1913) word list, with type-ahead suggestions, incremental
database updates and mouse support.
