// Package cli implements TermDict's non-interactive command-line
// surface: word lookups and database maintenance verbs with a strict
// stdout/stderr contract.
//
// Exit codes:
//
//	0  found / operation succeeded
//	1  word not found (lookup only)
//	2  usage error
//	3  runtime error (network, storage, unexpected failure)
//
// Stream discipline: stdout carries only the definition payload (safe
// to pipe); every diagnostic — progress, warnings, errors — goes to
// stderr. There is no tty detection: output is identical whether
// piped or printed to a terminal.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/yodeman/termdict/internal/data"
	"github.com/yodeman/termdict/internal/dict"
)

// Process exit codes, shared by the CLI verbs and the dispatcher.
const (
	ExitOK       = 0
	ExitNotFound = 1
	ExitUsage    = 2
	ExitRuntime  = 3
)

// ActionKind identifies what the dispatcher should do.
type ActionKind int

const (
	// ActionTUI launches the interactive interface (zero positionals).
	ActionTUI ActionKind = iota
	// ActionLookup defines a single word.
	ActionLookup
	// ActionUpdate downloads changed dictionary files incrementally.
	ActionUpdate
	// ActionDownload fetches the complete letter-file set.
	ActionDownload
	// ActionHelp prints usage on stdout.
	ActionHelp
	// ActionVersion prints version information on stdout.
	ActionVersion
)

// Action is the parsed result of the command line.
type Action struct {
	Kind  ActionKind
	Query string // lookup target; empty unless Kind == ActionLookup
}

// reservedVerbs are matched only as the first positional argument,
// lowercase-exact. Anything else (including "Update") is a lookup.
var reservedVerbs = map[string]ActionKind{
	"update":   ActionUpdate,
	"download": ActionDownload,
	"help":     ActionHelp,
	"version":  ActionVersion,
}

// ParseArgs turns raw arguments into an Action. A non-nil error is a
// usage error (exit code 2). Help/version requests win over any
// positional argument regardless of order; "--" makes the remaining
// arguments literal positionals.
func ParseArgs(args []string) (Action, error) {
	action := Action{Kind: ActionTUI}
	verb := ""
	literal := false // "--" seen: positionals never match reserved verbs
	positionalsStart := -1

scan:
	for i, arg := range args {
		switch {
		case arg == "--":
			literal = true
			positionalsStart = i + 1
			break scan
		case arg == "-h", arg == "--help":
			return Action{Kind: ActionHelp}, nil
		case arg == "-v", arg == "--version":
			return Action{Kind: ActionVersion}, nil
		case len(arg) > 1 && strings.HasPrefix(arg, "-"):
			return Action{}, fmt.Errorf("unknown flag %q", arg)
		default:
			positionalsStart = i
			break scan
		}
	}

	if positionalsStart < 0 {
		return action, nil // no positionals: TUI (or flag-only run)
	}
	positional := args[positionalsStart:]

	if !literal && action.Kind == ActionTUI && len(positional) > 0 {
		if kind, isVerb := reservedVerbs[positional[0]]; isVerb {
			action.Kind = kind
			action.Query = ""
			verb = positional[0]
			positional = positional[1:]
		}
	}

	switch action.Kind {
	case ActionHelp, ActionVersion:
		return action, nil // flags/verbs win over stray words
	case ActionUpdate, ActionDownload:
		if len(positional) > 0 {
			return Action{}, fmt.Errorf("%q takes no arguments", verb)
		}
		return action, nil
	default: // lookup or TUI
		switch {
		case len(positional) == 0:
			return action, nil
		case len(positional) > 1:
			return Action{}, fmt.Errorf(
				"expected a single word (quote multi-word phrases), got %q",
				strings.Join(positional, " "))
		case strings.TrimSpace(positional[0]) == "":
			return Action{}, fmt.Errorf("empty search term")
		default:
			action.Kind = ActionLookup
			action.Query = positional[0]
			return action, nil
		}
	}
}

// Usage returns the help text shown for --help, the help verb, and
// usage errors.
func Usage() string {
	return `TermDict — dictionary in the terminal.

Usage:
  termdict                     launch the interactive terminal UI
  termdict <word>              print the definition of <word>
  termdict update              download changed dictionary files
  termdict download            download the complete dictionary
  termdict help                show this help
  termdict version             print version information

Flags:
  -h, --help                   show this help
  -v, --version                print version information

Notes:
  Multi-word phrases must be quoted: termdict "ice cream".
  The first word of a phrase can be protected from the reserved
  verbs (update, download, help, version) with "--": termdict -- update.

Exit codes:
  0  success                 2  usage error
  1  word not found          3  runtime error

Data directory layout:
  linux    $XDG_DATA_HOME/termdict  (~/.local/share/termdict)
  macos    ~/Library/Application Support/termdict
  windows  %LOCALAPPDATA%\termdict
`
}

// VersionLine renders the --version output.
func VersionLine(version string) string {
	if version == "" || version == "dev" {
		version = "dev"
	}
	return fmt.Sprintf("termdict %s\n", version)
}

// LookupService is the subset of *dict.Service the CLI needs.
type LookupService interface {
	Lookup(word string) (dict.Entity, bool)
}

// Updater is the subset of *data.Client the CLI needs.
type Updater interface {
	Update(ctx context.Context, progress data.ProgressFn) (updated int, err error)
	DownloadFull(ctx context.Context, progress data.ProgressFn) (downloaded int, err error)
}

// ProgressFn aliases data.ProgressFn so callers can use either name.
type ProgressFn = data.ProgressFn

// Runner executes CLI verbs against injected dependencies.
type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

// RunLookup prints the definition of query. It returns ExitNotFound
// when the service has no entry; stdout stays empty in that case.
func (r *Runner) RunLookup(svc LookupService, query string) int {
	entity, found := svc.Lookup(query)
	if !found {
		_, _ = fmt.Fprint(r.Stderr, dict.PlainTextNotFound(query))
		return ExitNotFound
	}
	if err := dict.RenderPlainText(r.Stdout, entity); err != nil {
		_, _ = fmt.Fprintf(r.Stderr, "termdict: writing definition failed: %v\n", err)
		return ExitRuntime
	}
	return ExitOK
}

// RunUpdate runs an incremental update. Progress goes to stderr, the
// final summary to stdout.
func (r *Runner) RunUpdate(ctx context.Context, updater Updater) int {
	return r.runMaintenance(ctx, updater.Update, func(n int) string {
		return fmt.Sprintf("Updated %d file(s).\n", n)
	}, "update")
}

// RunDownload fetches the complete letter-file set.
func (r *Runner) RunDownload(ctx context.Context, updater Updater) int {
	return r.runMaintenance(ctx, updater.DownloadFull, func(n int) string {
		return fmt.Sprintf("Downloaded %d file(s).\n", n)
	}, "download")
}

func (r *Runner) runMaintenance(
	ctx context.Context,
	run func(context.Context, data.ProgressFn) (int, error),
	summary func(int) string,
	name string,
) int {
	_, _ = fmt.Fprintf(r.Stderr, "%s in progress…\n", name)
	updated, err := run(ctx, func(done, total int, current string) {
		_, _ = fmt.Fprintf(r.Stderr, "\r%s %d/%d — %-30s", name, done, total, current)
	})
	_, _ = fmt.Fprint(r.Stderr, "\n")
	if err != nil {
		_, _ = fmt.Fprintf(r.Stderr, "termdict: %s failed:\n%v\n", name, err)
		return ExitRuntime
	}
	if updated == 0 {
		_, _ = fmt.Fprint(r.Stdout, "Already up to date.\n")
		return ExitOK
	}
	_, _ = fmt.Fprint(r.Stdout, summary(updated))
	return ExitOK
}
