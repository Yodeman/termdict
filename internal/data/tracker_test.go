package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTrackerMissingFile(t *testing.T) {
	tracker, err := ReadTracker(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("missing tracker should not error, got %v", err)
	}
	if len(tracker) != 0 {
		t.Errorf("expected empty tracker, got %v", tracker)
	}
}

func TestTrackerRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "changes_tracker.json")
	want := Tracker{"wb1913_a.json": "v1", "wb1913_b.json": "v2"}

	if err := WriteTracker(path, want); err != nil {
		t.Fatalf("WriteTracker: %v", err)
	}
	got, err := ReadTracker(path)
	if err != nil {
		t.Fatalf("ReadTracker: %v", err)
	}
	for file, version := range want {
		if got[file] != version {
			t.Errorf("tracker[%q] = %q, want %q", file, got[file], version)
		}
	}
}

func TestWriteTrackerReplacesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "changes_tracker.json")
	if err := WriteTracker(path, Tracker{"old": "1"}); err != nil {
		t.Fatalf("initial write: %v", err)
	}
	if err := WriteTracker(path, Tracker{"new": "2"}); err != nil {
		t.Fatalf("second write: %v", err)
	}
	got, err := ReadTracker(path)
	if err != nil {
		t.Fatalf("ReadTracker: %v", err)
	}
	if _, ok := got["old"]; ok {
		t.Error("stale entry survived rewrite")
	}
	if got["new"] != "2" {
		t.Errorf("got %v, want new=2", got)
	}
}

func TestWriteTrackerLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "changes_tracker.json")
	if err := WriteTracker(path, Tracker{"a": "1"}); err != nil {
		t.Fatalf("WriteTracker: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "changes_tracker.json" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected only the tracker file, got %v", names)
	}
}

func TestReadTrackerInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "changes_tracker.json")
	if err := os.WriteFile(path, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := ReadTracker(path)
	if err == nil || !strings.Contains(err.Error(), "decoding") {
		t.Errorf("expected decode error, got %v", err)
	}
}
