// Package main is the entry point of the terminal dictionary. It
// resolves configuration, builds the dictionary service (embedded core
// plus any locally downloaded database) and hands everything to the TUI.
//
// Since phase 2 the application is offline-by-default: an embedded core
// subset always loads, and downloading the full database is an explicit
// user action inside the app (ctrl+u), never a startup requirement.
package main

import (
	"log"
	"os"

	"github.com/yodeman/termdict/internal/config"
	"github.com/yodeman/termdict/internal/data"
	"github.com/yodeman/termdict/internal/data/embedded"
	"github.com/yodeman/termdict/internal/dict"
	"github.com/yodeman/termdict/internal/tui"
)

func main() {
	log.SetFlags(0)

	cfg, err := config.Default()
	if err != nil {
		log.Fatalf("Error resolving application directories.\n%v\n", err)
	}
	if err := os.MkdirAll(cfg.DbaseDir, 0o755); err != nil {
		log.Fatalf("Error creating directory %s.\n%v\n", cfg.DbaseDir, err)
	}

	svc, err := dict.NewMulti(
		embedded.New(),
		data.FileStore{Dir: cfg.DbaseDir},
	)
	if err != nil {
		log.Fatalf("Error loading words database.\n%v\n", err)
	}
	if svc.Len() == 0 {
		log.Fatalf("The words database is empty; this is a build error.\n" +
			"Please report it at https://github.com/yodeman/termdict/issues.\n")
	}

	client := data.NewClient(cfg.DbaseDir, cfg.TrackerPath, config.AppVersion)
	if err := tui.New(cfg, svc, client).Run(); err != nil {
		log.Fatalf("Error running terminal dictionary.\n%v\n", err)
	}
}
