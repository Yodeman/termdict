package config

import (
	"path/filepath"
	"testing"
)

func lookup(pairs map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := pairs[key]
		return v, ok
	}
}

func TestDataDirFor(t *testing.T) {
	home := string(filepath.Separator) + filepath.Join("home", "u")

	cases := []struct {
		name    string
		goos    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "linux default",
			goos: "linux", env: map[string]string{},
			want: filepath.Join(home, ".local", "share", AppName),
		},
		{
			name: "linux xdg override",
			goos: "linux", env: map[string]string{"XDG_DATA_HOME": "/xdg/data"},
			want: filepath.Join("/xdg/data", AppName),
		},
		{
			name: "linux relative xdg ignored",
			goos: "linux", env: map[string]string{"XDG_DATA_HOME": "relative/path"},
			want: filepath.Join(home, ".local", "share", AppName),
		},
		{
			name: "darwin bundle style",
			goos: "darwin", env: map[string]string{},
			want: filepath.Join(home, "Library", "Application Support", AppName),
		},
		{
			name: "windows localappdata",
			goos: "windows", env: map[string]string{"LOCALAPPDATA": `C:\Users\u\AppData\Local`},
			want: filepath.Join(`C:\Users\u\AppData\Local`, AppName),
		},
		{
			name: "windows falls back to home layout",
			goos: "windows", env: map[string]string{},
			want: filepath.Join(home, "AppData", "Local", AppName),
		},
		{
			name: "freebsd follows xdg",
			goos: "freebsd", env: map[string]string{},
			want: filepath.Join(home, ".local", "share", AppName),
		},
		{
			name: "unsupported platform",
			goos: "plan9", env: map[string]string{},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DataDirFor(tc.goos, home, lookup(tc.env))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDataDirForEmptyHome(t *testing.T) {
	if _, err := DataDirFor("linux", "", lookup(nil)); err == nil {
		t.Error("expected error for empty home")
	}
}

func TestLegacyDataDirFor(t *testing.T) {
	home := string(filepath.Separator) + filepath.Join("home", "u")

	if got, ok := LegacyDataDirFor("linux", home); !ok ||
		got != filepath.Join(home, ".local", AppName) {
		t.Errorf("linux legacy = %q (ok=%v)", got, ok)
	}
	if got, ok := LegacyDataDirFor("darwin", home); !ok ||
		got != filepath.Join(home, ".local", AppName) {
		t.Errorf("darwin legacy should match v0.1.0 unix layout, got %q", got)
	}
	if got, ok := LegacyDataDirFor("windows", home); !ok ||
		got != filepath.Join(home, "AppData", "Local", AppName) {
		t.Errorf("windows legacy = %q (ok=%v)", got, ok)
	}
	if _, ok := LegacyDataDirFor("plan9", home); ok {
		t.Error("plan9 should have no legacy layout")
	}
}

func TestDefaultLayout(t *testing.T) {
	// Default() wires the real OS environment; on linux hosts the new
	// layout must land under XDG data home.
	paths, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if filepath.Base(paths.DataDir) != AppName {
		t.Errorf("DataDir base = %q, want %q", filepath.Base(paths.DataDir), AppName)
	}
	if paths.DbaseDir != filepath.Join(paths.DataDir, "dbase", "json") {
		t.Errorf("DbaseDir = %q", paths.DbaseDir)
	}
	if paths.TrackerPath != filepath.Join(paths.DataDir, "dbase", "changes_tracker.json") {
		t.Errorf("TrackerPath = %q", paths.TrackerPath)
	}
}
