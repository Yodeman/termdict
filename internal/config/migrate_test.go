package config

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeEnv returns an envSource resolving home to the given directory.
func fakeEnv(home string) envSource {
	return envSource{
		home:   func() (string, error) { return home, nil },
		lookup: os.LookupEnv,
	}
}

// seedLegacy creates a v0.1.0-style data tree under <home>/.local/termdict
// and returns its path.
func seedLegacy(t *testing.T, home string) string {
	t.Helper()
	legacy := filepath.Join(home, ".local", AppName)
	dbase := filepath.Join(legacy, "dbase", "json")
	if err := os.MkdirAll(dbase, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"wb1913_a.json": `{"ant":{"word":"ant","definitions":[]}}`,
		"wb1913_b.json": `{"bee":{"word":"bee","definitions":[]}}`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dbase, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(legacy, "dbase", "changes_tracker.json"),
		[]byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return legacy
}

func newPaths(_ *testing.T, root string) Paths {
	return Paths{
		DataDir:     filepath.Join(root, "data"),
		DbaseDir:    filepath.Join(root, "data", "dbase", "json"),
		TrackerPath: filepath.Join(root, "data", "dbase", "changes_tracker.json"),
	}
}

func TestMigrateLegacyCopiesAndRemovesOldTree(t *testing.T) {
	home := t.TempDir()
	legacy := seedLegacy(t, home)
	paths := newPaths(t, filepath.Join(home, "termdict-data"))

	migrateLegacy(fakeEnv(home), "linux", paths)

	// Data arrived in the new location.
	raw, err := os.ReadFile(filepath.Join(paths.DbaseDir, "wb1913_a.json"))
	if err != nil || string(raw) != `{"ant":{"word":"ant","definitions":[]}}` {
		t.Fatalf("migrated file wrong/missing: %q err=%v", raw, err)
	}
	if _, err := os.Stat(filepath.Join(paths.DataDir, "dbase", "changes_tracker.json")); err != nil {
		t.Fatalf("tracker not migrated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(paths.DataDir, MarkerName)); err != nil {
		t.Fatalf("marker not written: %v", err)
	}
	// Legacy tree removed after verified copy.
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy tree still present (err=%v)", err)
	}
}

func TestMigrateIdempotentViaMarker(t *testing.T) {
	home := t.TempDir()
	legacy := seedLegacy(t, home)
	paths := newPaths(t, filepath.Join(home, "termdict-data"))

	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.DataDir, MarkerName), []byte("migrated\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	migrateLegacy(fakeEnv(home), "linux", paths)

	if _, err := os.Stat(filepath.Join(paths.DbaseDir, "wb1913_a.json")); !os.IsNotExist(err) {
		t.Error("migration must not run when marker exists")
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Error("legacy tree must be untouched when marker exists")
	}
}

func TestMigrateSkippedWhenNoLegacyData(t *testing.T) {
	home := t.TempDir()
	paths := newPaths(t, filepath.Join(home, "termdict-data"))

	migrateLegacy(fakeEnv(home), "linux", paths)

	if _, err := os.Stat(filepath.Join(paths.DataDir, MarkerName)); !os.IsNotExist(err) {
		t.Error("no migration should be recorded without legacy data")
	}
}

func TestMigrateKeepsExistingNewLayoutData(t *testing.T) {
	home := t.TempDir()
	legacy := seedLegacy(t, home)
	paths := newPaths(t, filepath.Join(home, "termdict-data"))
	if err := os.MkdirAll(paths.DbaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newer := []byte(`{"new":{"word":"new"}}`)
	if err := os.WriteFile(filepath.Join(paths.DbaseDir, "wb1913_z.json"), newer, 0o644); err != nil {
		t.Fatal(err)
	}

	migrateLegacy(fakeEnv(home), "linux", paths)

	raw, err := os.ReadFile(filepath.Join(paths.DbaseDir, "wb1913_z.json"))
	if err != nil || string(raw) != string(newer) {
		t.Fatalf("existing new-layout data was disturbed: %q err=%v", raw, err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Error("legacy tree should be preserved when target already has data")
	}
}

func TestMigrateFailureLeavesLegacyIntact(t *testing.T) {
	home := t.TempDir()
	legacy := seedLegacy(t, home)
	paths := newPaths(t, filepath.Join(home, "termdict-data"))

	// Sabotage: a directory where a copied file needs to land makes the
	// copy fail mid-way.
	if err := os.MkdirAll(filepath.Join(paths.DataDir, "dbase", "json", "wb1913_b.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	migrateLegacy(fakeEnv(home), "linux", paths)

	if _, err := os.Stat(legacy); err != nil {
		t.Fatal("legacy tree must survive failed migrations")
	}
	if _, err := os.Stat(filepath.Join(paths.DataDir, MarkerName)); !os.IsNotExist(err) {
		t.Error("marker must not be written for failed migrations")
	}
}

func TestPrepareCreatesLayoutAndMigrates(t *testing.T) {
	home := t.TempDir()
	seedLegacy(t, home)

	env := systemEnv()
	env.home = func() (string, error) { return home, nil }

	dataDir, err := DataDirFor("linux", home, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	paths := Paths{
		DataDir:     dataDir,
		DbaseDir:    filepath.Join(dataDir, "dbase", "json"),
		TrackerPath: filepath.Join(dataDir, "dbase", "changes_tracker.json"),
	}

	if err := prepareWith(env, "linux", paths); err != nil {
		t.Fatalf("prepareWith: %v", err)
	}
	if _, err := os.Stat(filepath.Join(paths.DbaseDir, "wb1913_b.json")); err != nil {
		t.Fatalf("expected migrated letter file: %v", err)
	}
}
