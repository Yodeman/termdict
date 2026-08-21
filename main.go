// Command termdict is a dictionary for the terminal: an interactive
// TUI by default, plus non-interactive lookup and database-maintenance
// verbs. See internal/cli for the command-line contract (exit codes,
// stream discipline) and internal/config for the data layout.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/yodeman/termdict/internal/cli"
	"github.com/yodeman/termdict/internal/config"
	"github.com/yodeman/termdict/internal/data"
	"github.com/yodeman/termdict/internal/data/embedded"
	"github.com/yodeman/termdict/internal/dict"
	"github.com/yodeman/termdict/internal/tui"
)

func main() {
	log.SetFlags(0)
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	action, err := cli.ParseArgs(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "termdict: %v\n\n%s", err, cli.Usage())
		return cli.ExitUsage
	}
	switch action.Kind {
	case cli.ActionHelp:
		_, _ = fmt.Fprint(stdout, cli.Usage())
		return cli.ExitOK
	case cli.ActionVersion:
		_, _ = fmt.Fprint(stdout, cli.VersionLine(config.AppVersion, config.Commit))
		return cli.ExitOK
	}

	cfg, err := config.Default()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "termdict: resolving application directories: %v\n", err)
		return cli.ExitRuntime
	}
	if err := config.Prepare(cfg); err != nil {
		_, _ = fmt.Fprintf(stderr, "termdict: preparing %s: %v\n", cfg.DataDir, err)
		return cli.ExitRuntime
	}

	ctx := context.Background()
	switch action.Kind {
	case cli.ActionLookup:
		svc, err := buildService(cfg)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "termdict: loading words database: %v\n", err)
			return cli.ExitRuntime
		}
		return (&cli.Runner{Stdout: stdout, Stderr: stderr}).RunLookup(svc, action.Query)

	case cli.ActionUpdate, cli.ActionDownload:
		client := data.NewClient(cfg.DbaseDir, cfg.TrackerPath, config.AppVersion)
		runner := &cli.Runner{Stdout: stdout, Stderr: stderr}
		if action.Kind == cli.ActionUpdate {
			return runner.RunUpdate(ctx, client)
		}
		return runner.RunDownload(ctx, client)

	default: // ActionTUI
		svc, err := buildService(cfg)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "termdict: loading words database: %v\n", err)
			return cli.ExitRuntime
		}
		if svc.Len() == 0 {
			_, _ = fmt.Fprintln(stderr, "termdict: the words database is empty; "+
				"this is a build error. Please report it at "+
				"https://github.com/yodeman/termdict/issues.")
			return cli.ExitRuntime
		}
		client := data.NewClient(cfg.DbaseDir, cfg.TrackerPath, config.AppVersion)
		if err := tui.New(cfg, svc, client).Run(); err != nil {
			_, _ = fmt.Fprintf(stderr, "termdict: %v\n", err)
			return cli.ExitRuntime
		}
		return cli.ExitOK
	}
}

// buildService merges the embedded core with the local letter files;
// local entries win on collisions.
func buildService(cfg config.Paths) (*dict.Service, error) {
	return dict.NewMulti(
		embedded.New(),
		data.FileStore{Dir: cfg.DbaseDir},
	)
}
