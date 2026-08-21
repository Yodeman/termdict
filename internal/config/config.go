// Package config resolves TermDict's runtime locations and holds
// build metadata injected at link time.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// AppName is the canonical project name used in paths and user-facing text.
const AppName = "termdict"

// AppVersion is overridden at build time via
// -ldflags "-X github.com/yodeman/termdict/internal/config.AppVersion=vX.Y.Z".
// Unset builds report "dev".
var AppVersion = "dev"

// Paths describes the on-disk locations used by the application.
type Paths struct {
	DataDir     string // root data directory
	DbaseDir    string // directory holding the wb1913_*.json letter files
	TrackerPath string // path of the changes_tracker.json file
}

// DataDirFor resolves the platform data directory.
//
// NOTE (phase 1): this intentionally reproduces the legacy v0.1.0 layout
// (~/.local on unix, %LOCALAPPDATA% on Windows) so existing installs keep
// working. Phase 3 replaces it with per-platform standard directories and
// a one-time migration.
func DataDirFor(goos string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}

	switch goos {
	case "windows":
		return filepath.Join(home, "AppData", "Local", AppName), nil
	case "linux", "darwin", "freebsd", "openbsd", "netbsd", "dragonfly", "solaris", "illumos":
		return filepath.Join(home, ".local", AppName), nil
	default:
		return "", fmt.Errorf("unsupported platform %q", goos)
	}
}

// Default resolves the runtime Paths for the current platform.
func Default() (Paths, error) {
	dataDir, err := DataDirFor(runtime.GOOS)
	if err != nil {
		return Paths{}, err
	}

	dbaseDir := filepath.Join(dataDir, "dbase", "json")
	return Paths{
		DataDir:     dataDir,
		DbaseDir:    dbaseDir,
		TrackerPath: filepath.Join(dataDir, "dbase", "changes_tracker.json"),
	}, nil
}
