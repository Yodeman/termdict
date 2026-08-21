package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yodeman/termdict/internal/dict"
)

// compile-time interface conformance check
var _ dict.Store = FileStore{}

func writeTestLetterFile(t *testing.T, dir, name string, entries map[string]dict.Entity) {
	t.Helper()
	raw, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestFileStoreLoad(t *testing.T) {
	dir := t.TempDir()
	writeTestLetterFile(t, dir, "wb1913_a.json", map[string]dict.Entity{
		"apple": {Word: "apple"},
	})
	writeTestLetterFile(t, dir, "wb1913_b.json", map[string]dict.Entity{
		"banana": {Word: "banana"},
	})
	// Corrupt file must be skipped, not fatal.
	if err := os.WriteFile(filepath.Join(dir, "wb1913_c.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}
	// Non-matching files are ignored.
	writeTestLetterFile(t, dir, "other.json", map[string]dict.Entity{
		"ignored": {Word: "ignored"},
	})

	store := FileStore{Dir: dir}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("loaded %d entities, want 2: %+v", len(got), got)
	}
	if _, ok := got["apple"]; !ok {
		t.Error("missing apple")
	}
	if _, ok := got["banana"]; !ok {
		t.Error("missing banana")
	}
}

func TestFileStoreMissingDir(t *testing.T) {
	store := FileStore{Dir: filepath.Join(t.TempDir(), "does-not-exist")}
	if _, err := store.Load(); err == nil {
		t.Error("expected error for missing directory")
	}
}

func TestFileStoreEmptyDir(t *testing.T) {
	store := FileStore{Dir: t.TempDir()}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load on empty dir: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no entities, got %d", len(got))
	}
}

func TestFileStoreClosesFiles(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("open-fd accounting via /proc is linux-only")
	}

	dir := t.TempDir()
	for _, letter := range []string{"a", "b", "c"} {
		writeTestLetterFile(t, dir, "wb1913_"+letter+".json",
			map[string]dict.Entity{letter: {Word: letter}})
	}

	fdCount := func() int {
		entries, err := os.ReadDir("/proc/self/fd")
		if err != nil {
			t.Fatalf("reading /proc/self/fd: %v", err)
		}
		return len(entries)
	}

	before := fdCount()
	if _, err := (FileStore{Dir: dir}).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if after := fdCount(); after > before {
		t.Errorf("file descriptors leaked: before=%d after=%d", before, after)
	}
}

func TestLetterFiles(t *testing.T) {
	files := LetterFiles()
	if len(files) != 26 {
		t.Fatalf("len(LetterFiles()) = %d, want 26", len(files))
	}
	if files[0] != "wb1913_a.json" || files[25] != "wb1913_z.json" {
		t.Errorf("unexpected first/last: %q / %q", files[0], files[25])
	}
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f] {
			t.Errorf("duplicate file %q", f)
		}
		seen[f] = true
	}
}
