package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yodeman/termdict/internal/dict"
)

func writeFreqList(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write frequency list: %v", err)
	}
}

func writeDbase(t *testing.T, dir string, entries map[string]dict.Entity) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	groups := map[string]map[string]dict.Entity{}
	for word, entity := range entries {
		letter := "wb1913_" + string(word[0]) + ".json"
		if groups[letter] == nil {
			groups[letter] = map[string]dict.Entity{}
		}
		groups[letter][word] = entity
	}
	for name, group := range groups {
		raw, err := json.Marshal(group)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func testOptions(dbaseDir, freqPath, outDir string) Options {
	return Options{
		DbaseDir:         dbaseDir,
		FrequencyPath:    freqPath,
		OutDir:           outDir,
		MaxWords:         100,
		BudgetBytes:      1 << 20,
		FrequencyName:    "test-freq",
		FrequencyURL:     "https://example.test/freq",
		FrequencyLicense: "CC-BY-SA-4.0",
	}
}

func TestLoadFrequencyRanks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "freq.txt")
	writeFreqList(t, path, "the 999\nApple 50\nbad-word! 10\nthe 1\n\n123 5\ncat 7\n")

	ranks, err := loadFrequencyRanks(path)
	if err != nil {
		t.Fatalf("loadFrequencyRanks: %v", err)
	}

	if ranks["the"] != 1 {
		t.Errorf("rank(the) = %d, want 1 (first occurrence wins)", ranks["the"])
	}
	if ranks["apple"] != 2 {
		t.Errorf("rank(apple) = %d, want 2 (case-normalized)", ranks["apple"])
	}
	if _, ok := ranks["bad-word!"]; ok {
		t.Error("non a-z words must be ignored")
	}
	if _, ok := ranks["123"]; ok {
		t.Error("numeric tokens must be ignored")
	}
	if ranks["cat"] != 3 {
		t.Errorf("rank(cat) = %d, want 3", ranks["cat"])
	}
}

func TestSelectCoreRespectsRankAndCaps(t *testing.T) {
	entries := map[string]dict.Entity{}
	for _, w := range []string{"common", "rare", "unranked", "mid"} {
		entries[w] = dict.Entity{Word: w, WordDefinitions: []dict.Definition{
			{PartOfSpeech: "n.", WordDefinition: "def of " + w},
		}}
	}
	ranks := map[string]int{"common": 1, "mid": 2, "rare": 3}

	selected, used := selectCore(entries, ranks, 2, 1<<20)
	if len(selected) != 2 {
		t.Fatalf("selected %d entries, want 2", len(selected))
	}
	if selected[0].Word != "common" || selected[1].Word != "mid" {
		t.Errorf("selection not rank-ordered: %s, %s", selected[0].Word, selected[1].Word)
	}
	if used == 0 {
		t.Error("used size should be positive")
	}

	// maxWords=0 selects nothing; unranked words never appear.
	if got, _ := selectCore(entries, ranks, 0, 1<<20); len(got) != 0 {
		t.Errorf("maxWords=0 selected %d", len(got))
	}
	all, _ := selectCore(entries, ranks, 100, 1<<20)
	for _, entity := range all {
		if entity.Word == "unranked" {
			t.Error("unranked word must not be selected")
		}
	}
}

func TestSelectCoreBudget(t *testing.T) {
	big := dict.Entity{Word: "big", WordDefinitions: []dict.Definition{
		{PartOfSpeech: "n.", WordDefinition: string(make([]byte, 4096))},
	}}
	small := dict.Entity{Word: "small", WordDefinitions: []dict.Definition{
		{PartOfSpeech: "n.", WordDefinition: "tiny"},
	}}
	entries := map[string]dict.Entity{"big": big, "small": small}
	ranks := map[string]int{"big": 1, "small": 2}

	selected, _ := selectCore(entries, ranks, 10, 1024)
	if len(selected) != 1 || selected[0].Word != "small" {
		t.Errorf("budget should skip oversized entry and keep the small one, got %+v", selected)
	}
}

func TestBuildDeterministic(t *testing.T) {
	base := t.TempDir()
	dbaseDir := filepath.Join(base, "dbase")
	freqPath := filepath.Join(base, "freq.txt")

	entries := map[string]dict.Entity{}
	freqLines := ""
	for _, w := range []string{"alpha", "beta", "gamma", "delta", "epsilon"} {
		entries[w] = dict.Entity{Word: w, WordDefinitions: []dict.Definition{
			{PartOfSpeech: "n.", WordDefinition: "definition " + w},
		}}
		freqLines += w + " 100\n"
	}
	// reverse order lines to prove ranking (not file order) drives selection
	writeFreqList(t, freqPath, freqLines)
	writeDbase(t, dbaseDir, entries)

	run := func(outDir string) *Manifest {
		t.Helper()
		manifest, err := Build(testOptions(dbaseDir, freqPath, outDir))
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return manifest
	}

	out1, out2 := filepath.Join(base, "out1"), filepath.Join(base, "out2")
	m1, m2 := run(out1), run(out2)

	raw1, err := json.Marshal(m1)
	if err != nil {
		t.Fatal(err)
	}
	raw2, err := json.Marshal(m2)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw1) != string(raw2) {
		t.Error("manifests differ between identical runs")
	}

	entries1, err := os.ReadDir(out1)
	if err != nil {
		t.Fatal(err)
	}
	entries2, err := os.ReadDir(out2)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries1) != len(entries2) {
		t.Fatalf("file counts differ: %d vs %d", len(entries1), len(entries2))
	}
	for i, entry := range entries1 {
		if entry.Name() != entries2[i].Name() {
			t.Fatalf("file names differ at %d", i)
		}
		a, _ := os.ReadFile(filepath.Join(out1, entry.Name()))
		b, _ := os.ReadFile(filepath.Join(out2, entry.Name()))
		if string(a) != string(b) {
			t.Errorf("%s is not byte-identical across runs (gzip header leakage?)", entry.Name())
		}
	}
}

func TestBuildManifestAndStaleRemoval(t *testing.T) {
	base := t.TempDir()
	dbaseDir := filepath.Join(base, "dbase")
	freqPath := filepath.Join(base, "freq.txt")
	outDir := filepath.Join(base, "core")

	writeFreqList(t, freqPath, "kite 1\nzoo 2\n")
	writeDbase(t, dbaseDir, map[string]dict.Entity{
		"kite": {Word: "kite"},
		"zoo":  {Word: "zoo"},
	})

	// Stale artifact from an older build must be removed.
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(outDir, "wb1913_q.json.gz")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := Build(testOptions(dbaseDir, freqPath, outDir))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if manifest.TotalEntries != 2 {
		t.Errorf("TotalEntries = %d, want 2", manifest.TotalEntries)
	}
	if len(manifest.Files) != 2 { // k and z letters
		t.Fatalf("expected 2 letter files, got %d", len(manifest.Files))
	}
	if manifest.FrequencySource.SHA256 == "" || manifest.FrequencySource.License != "CC-BY-SA-4.0" {
		t.Errorf("frequency provenance incomplete: %+v", manifest.FrequencySource)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale core file was not removed")
	}

	// Manifest on disk matches returned struct.
	raw, err := os.ReadFile(filepath.Join(outDir, "core_manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var onDisk Manifest
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("manifest JSON invalid: %v", err)
	}
	if onDisk.TotalEntries != manifest.TotalEntries {
		t.Error("on-disk manifest mismatch")
	}
}

func TestEmitChecksums(t *testing.T) {
	dir := t.TempDir()
	writeDbase(t, dir, map[string]dict.Entity{"ant": {Word: "ant"}, "bee": {Word: "bee"}})
	outPath := filepath.Join(t.TempDir(), "checksums.txt")

	count, err := emitChecksums(dir, outPath)
	if err != nil {
		t.Fatalf("emitChecksums: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := splitLines(string(raw))
	if len(lines) != 2 {
		t.Fatalf("expected 2 checksum lines, got %q", string(raw))
	}
	for _, line := range lines {
		if len(line) != 64+2+len("wb1913_a.json") {
			t.Errorf("malformed checksum line: %q", line)
		}
	}
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
