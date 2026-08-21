package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDataDirFor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	if runtime.GOOS == "windows" {
		got, err := DataDirFor("windows")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := filepath.Join(home, "AppData", "Local", AppName)
		if got != want {
			t.Errorf("DataDirFor(windows) = %q, want %q", got, want)
		}
	}

	for _, goos := range []string{"linux", "darwin", "freebsd"} {
		got, err := DataDirFor(goos)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", goos, err)
		}
		want := filepath.Join(home, ".local", AppName)
		if got != want {
			t.Errorf("DataDirFor(%s) = %q, want %q", goos, got, want)
		}
	}

	if _, err := DataDirFor("plan9"); err == nil {
		t.Error("DataDirFor(plan9) should fail on an unsupported platform")
	}
}

func TestDataDirForHomeError(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("HOME manipulation only reliable on unix")
	}
	t.Setenv("HOME", "")
	if _, err := DataDirFor("linux"); err == nil {
		t.Error("expected error with empty HOME")
	}
}

func TestDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path expectations differ on windows; covered by TestDataDirFor")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	paths, err := Default()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, ".local", AppName)
	if paths.DataDir != want {
		t.Errorf("DataDir = %q, want %q", paths.DataDir, want)
	}
	if paths.DbaseDir != filepath.Join(want, "dbase", "json") {
		t.Errorf("DbaseDir = %q", paths.DbaseDir)
	}
	if paths.TrackerPath != filepath.Join(want, "dbase", "changes_tracker.json") {
		t.Errorf("TrackerPath = %q", paths.TrackerPath)
	}
}
