// Package config resolves TermDict's runtime locations and holds
// build metadata injected at link time.
//
// Data directory layout (phase 3):
//
//	linux (and other unix)  $XDG_DATA_HOME/termdict, else ~/.local/share/termdict
//	darwin                  ~/Library/Application Support/termdict
//	windows                 %LOCALAPPDATA%\termdict
//
// Installs upgrading from v0.1.0 (which stored data under
// ~/.local/termdict or %LOCALAPPDATA%\..\Local\termdict without the
// share segment) are migrated once on first run; see migrate.go.
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

// Commit is overridden at build time (see Makefile / .goreleaser.yaml)
// and shown by --version when available.
var Commit = ""

// MarkerName flags that the one-time legacy migration already ran.
const MarkerName = ".migrated-v2"

// Paths describes the on-disk locations used by the application.
type Paths struct {
	DataDir     string // root data directory
	DbaseDir    string // directory holding the wb1913_*.json letter files
	TrackerPath string // path of the changes_tracker.json file
}

// envSource abstracts environment access so resolution is testable
// across GOOS branches on any host.
type envSource struct {
	home   func() (string, error)
	lookup func(key string) (string, bool)
}

func systemEnv() envSource {
	return envSource{home: os.UserHomeDir, lookup: os.LookupEnv}
}

// DataDirFor resolves the platform-standard data directory for goos
// using the provided home path and environment lookup.
func DataDirFor(goos, homeDir string, lookupEnv func(string) (string, bool)) (string, error) {
	if homeDir == "" {
		return "", fmt.Errorf("home directory is not set")
	}

	switch goos {
	case "windows":
		base := filepath.Join(homeDir, "AppData", "Local")
		if local, ok := lookupEnv("LOCALAPPDATA"); ok && local != "" {
			base = local
		}
		return filepath.Join(base, AppName), nil

	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", AppName), nil

	case "linux", "freebsd", "openbsd", "netbsd", "dragonfly", "solaris", "illumos":
		// XDG Base Directory specification: $XDG_DATA_HOME defines the
		// base for user-specific data; default is ~/.local/share.
		base := filepath.Join(homeDir, ".local", "share")
		if xdg, ok := lookupEnv("XDG_DATA_HOME"); ok && xdg != "" && filepath.IsAbs(xdg) {
			base = xdg
		}
		return filepath.Join(base, AppName), nil

	default:
		return "", fmt.Errorf("unsupported platform %q", goos)
	}
}

// LegacyDataDirFor returns the v0.1.0-era data location for goos, used
// by the one-time migration. It returns ok=false when no legacy layout
// exists for the platform.
func LegacyDataDirFor(goos, homeDir string) (legacyPath string, ok bool) {
	switch goos {
	case "windows":
		if homeDir == "" {
			return "", false
		}
		return filepath.Join(homeDir, "AppData", "Local", AppName), true
	case "linux", "darwin", "freebsd", "openbsd", "netbsd", "dragonfly", "solaris", "illumos":
		if homeDir == "" {
			return "", false
		}
		return filepath.Join(homeDir, ".local", AppName), true
	default:
		return "", false
	}
}

// Default resolves the runtime Paths for the current platform. It does
// not touch the filesystem; call Prepare for that.
func Default() (Paths, error) {
	env := systemEnv()
	home, err := env.home()
	if err != nil {
		return Paths{}, fmt.Errorf("resolving home directory: %w", err)
	}

	dataDir, err := DataDirFor(runtime.GOOS, home, env.lookup)
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

// Prepare creates the directory layout and runs the one-time legacy
// migration when a previous installation is detected. Failures to
// migrate are non-fatal: the app continues with a fresh directory and
// legacy data is left untouched.
func Prepare(paths Paths) error {
	return prepareWith(systemEnv(), runtime.GOOS, paths)
}

func prepareWith(env envSource, goos string, paths Paths) error {
	if err := os.MkdirAll(paths.DbaseDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", paths.DbaseDir, err)
	}
	migrateLegacy(env, goos, paths)
	return nil
}
