// Package main is the entry point of the terminal dictionary. It
// resolves configuration, ensures a words database exists, builds the
// dictionary service and hands everything to the TUI.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yodeman/termdict/internal/config"
	"github.com/yodeman/termdict/internal/data"
	"github.com/yodeman/termdict/internal/dict"
	"github.com/yodeman/termdict/internal/tui"
)

// message shown when dictionary database files are missing in the
// expected application directory.
const dbaseCheckMsg = `
Words database not found in the expected directory.

Would you like to download the words database [Y|y]es or [N|n]o: `

func main() {
	log.SetFlags(0)

	cfg, err := config.Default()
	if err != nil {
		log.Fatalf("Error resolving application directories.\n%v\n", err)
	}
	if err := os.MkdirAll(cfg.DbaseDir, 0o755); err != nil {
		log.Fatalf("Error creating directory %s.\n%v\n", cfg.DbaseDir, err)
	}

	client := data.NewClient(cfg.DbaseDir, cfg.TrackerPath)
	if err := ensureDatabase(cfg, client); err != nil {
		log.Fatalf("%v\n", err)
	}

	svc, err := dict.NewMulti(data.FileStore{Dir: cfg.DbaseDir})
	if err != nil {
		log.Fatalf("Error loading words database.\n%v\n", err)
	}
	if svc.Len() == 0 {
		log.Fatalf("The words database at %s appears to be empty.\n"+
			"Run termdict again and answer yes to download it.\n", cfg.DbaseDir)
	}

	if err := tui.New(cfg, svc, client).Run(); err != nil {
		log.Fatalf("Error running terminal dictionary.\n%v\n", err)
	}
}

// ensureDatabase offers to download any missing letter files. Unlike
// v0.1.0 it only downloads what is missing, so an interrupted first run
// resumes instead of starting over.
func ensureDatabase(cfg config.Paths, client *data.Client) error {
	var missing []string
	for _, file := range data.LetterFiles() {
		if _, err := os.Stat(filepath.Join(cfg.DbaseDir, file)); os.IsNotExist(err) {
			missing = append(missing, file)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	fmt.Print(dbaseCheckMsg)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "n") {
		fmt.Println("Exiting...")
		os.Exit(0)
	}

	fmt.Printf("Downloading words database (%d files)...\n", len(missing))
	ctx := context.Background()
	err := client.Fetch(ctx, missing, printProgress)
	var fetchErr *data.FetchError
	switch {
	case errors.As(err, &fetchErr):
		return fmt.Errorf("downloading words database: %w", err)
	case err != nil:
		return fmt.Errorf("downloading words database: %w", err)
	}

	// Record versions only now that every missing file succeeded.
	if err := client.RefreshTracker(ctx); err != nil {
		log.Printf("Warning: could not save the update log: %v", err)
	}
	fmt.Println("Done.")
	return nil
}

func printProgress(done, total int, current string) {
	fmt.Printf("\rDownloading %d/%d — %-24s", done, total, current)
	if done == total {
		fmt.Println()
	}
}
