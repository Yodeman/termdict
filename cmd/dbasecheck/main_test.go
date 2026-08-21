package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yodeman/termdict/internal/dict"
)

var zeroTime = time.Time{}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func seedValidTree(t *testing.T) (opts options) {
	t.Helper()
	root := t.TempDir()
	opts.dbaseDir = filepath.Join(root, "word_dbase", "json")
	opts.coreDir = filepath.Join(root, "core")

	writeFile(t, filepath.Join(opts.dbaseDir, "wb1913_a.json"),
		`{"ant":{"word":"ant","definitions":[{"part_of_speech":"n.","definition":"Insect."}]}}`)
	writeFile(t, filepath.Join(opts.dbaseDir, "wb1913_b.json"),
		`{"bee":{"word":"bee","definitions":[]}}`)
	writeFile(t, filepath.Join(root, "word_dbase", "changes_tracker.json"), `{"wb1913_a.json":"v1"}`)

	letterDigests := map[string]string{}
	for _, letter := range []string{"a", "b"} {
		raw := map[string]dict.Entity{
			map[string]string{"a": "ant", "b": "bee"}[letter]: {Word: map[string]string{"a": "ant", "b": "bee"}[letter]},
		}
		plain, _ := json.MarshalIndent(raw, "", "    ")
		var buf bytes.Buffer
		gz, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		gz.ModTime = zeroTime
		gz.Write(plain) //nolint:errcheck // in-memory
		gz.Close()      //nolint:errcheck // in-memory
		name := "wb1913_" + letter + ".json.gz"
		writeFile(t, filepath.Join(opts.coreDir, name), buf.String())
		sum := sha256.Sum256(buf.Bytes())
		letterDigests[name] = hex.EncodeToString(sum[:])
	}
	manifest := map[string]any{
		"format":            1,
		"generator_version": "1",
		"total_entries":     2,
		"files": []map[string]any{
			{"name": "wb1913_a.json.gz", "sha256": letterDigests["wb1913_a.json.gz"], "entries": 1},
			{"name": "wb1913_b.json.gz", "sha256": letterDigests["wb1913_b.json.gz"], "entries": 1},
		},
	}
	raw, _ := json.Marshal(manifest)
	writeFile(t, filepath.Join(opts.coreDir, "core_manifest.json"), string(raw))

	// checksums.txt over the full-database letter files
	aRaw, _ := os.ReadFile(filepath.Join(opts.dbaseDir, "wb1913_a.json"))
	bRaw, _ := os.ReadFile(filepath.Join(opts.dbaseDir, "wb1913_b.json"))
	aSum := sha256.Sum256(aRaw)
	bSum := sha256.Sum256(bRaw)
	checksums := hex.EncodeToString(aSum[:]) + "  wb1913_a.json\n" +
		hex.EncodeToString(bSum[:]) + "  wb1913_b.json\n"
	writeFile(t, filepath.Join(root, "word_dbase", "checksums.txt"), checksums)

	return opts
}

// refreshChecksums regenerates checksums.txt after a test mutates the
// letter files, so digest drift does not drown out the finding under
// test.
func refreshChecksums(t *testing.T, dbaseDir string) {
	t.Helper()
	var lines []string
	matches, _ := filepath.Glob(filepath.Join(dbaseDir, "wb1913_*.json"))
	for _, name := range matches {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+filepath.Base(name))
	}
	writeFile(t, filepath.Join(filepath.Dir(dbaseDir), "checksums.txt"),
		strings.Join(lines, "\n")+"\n")
}

func captureRun(t *testing.T, opts options) (string, *report, error) {
	t.Helper()
	var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	report, err := run(opts)
	_ = w.Close()
	os.Stdout = stdout
	_, _ = buf.ReadFrom(r)
	return buf.String(), report, err
}

func TestRunValidTreePasses(t *testing.T) {
	opts := seedValidTree(t)
	out, report, err := captureRun(t, opts)
	if err != nil || report.failures > 0 {
		t.Fatalf("expected pass, got error=%v failures=%d\n%s", err, report.failures, out)
	}
	if !strings.Contains(out, "DATA VALIDATION PASSED") {
		t.Errorf("missing success banner:\n%s", out)
	}
}

func TestRunDetectsBrokenJSON(t *testing.T) {
	opts := seedValidTree(t)
	writeFile(t, filepath.Join(opts.dbaseDir, "wb1913_c.json"), "{broken")

	out, report, err := captureRun(t, opts)
	if err != nil {
		t.Fatalf("data findings must not be environmental errors: %v\n%s", err, out)
	}
	if !strings.Contains(out, "invalid JSON") || report.failures == 0 {
		t.Errorf("expected fatal JSON finding:\n%s", out)
	}
}

func TestRunDetectsWordFieldMismatch(t *testing.T) {
	opts := seedValidTree(t)
	writeFile(t, filepath.Join(opts.dbaseDir, "wb1913_a.json"),
		`{"ant":{"word":"anteater","definitions":[]}}`)

	out, _, err := captureRun(t, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "disagrees with key") {
		t.Errorf("expected word/key mismatch finding:\n%s", out)
	}
}

func TestRunWarnsOnMisplacedKeys(t *testing.T) {
	opts := seedValidTree(t)
	writeFile(t, filepath.Join(opts.dbaseDir, "wb1913_a.json"),
		`{"embassade":{"word":"embassade","definitions":[]},"ant":{"word":"ant","definitions":[]}}`)
	refreshChecksums(t, opts.dbaseDir)

	out, report, err := captureRun(t, opts)
	if err != nil {
		t.Fatalf("misplaced keys are warnings, not failures: %v", err)
	}
	if !strings.Contains(out, "WARN") || !strings.Contains(out, "embassade") {
		t.Errorf("expected warning for misplaced key:\n%s", out)
	}
	if report.failures != 0 || report.warnings == 0 {
		t.Errorf("misplaced keys must warn without failing: %+v", report)
	}
}

func TestRunDetectsChecksumDrift(t *testing.T) {
	opts := seedValidTree(t)
	writeFile(t, filepath.Join(opts.dbaseDir, "wb1913_a.json"),
		`{"ant":{"word":"ant","definitions":[]},"new":{"word":"new","definitions":[]}}`)

	out, _, err := captureRun(t, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "digest mismatch") {
		t.Errorf("expected checksum drift finding:\n%s", out)
	}
}

func TestRunDetectsCoreManifestMismatch(t *testing.T) {
	opts := seedValidTree(t)
	manifestPath := filepath.Join(opts.coreDir, "core_manifest.json")
	raw, _ := os.ReadFile(manifestPath)
	var manifest map[string]any
	_ = json.Unmarshal(raw, &manifest)
	manifest["total_entries"] = 99
	updated, _ := json.Marshal(manifest)
	writeFile(t, manifestPath, string(updated))

	out, _, err := captureRun(t, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "totals wrong") {
		t.Errorf("expected core total mismatch finding:\n%s", out)
	}
}

func TestRunMissingDbaseDirIsEnvironmentalError(t *testing.T) {
	opts := options{dbaseDir: filepath.Join(t.TempDir(), "absent"), coreDir: t.TempDir()}
	_, _, err := captureRun(t, opts)
	if err == nil {
		t.Fatal("missing dbase dir should be a hard error")
	}
}
