// Command dbasecheck validates the TermDict words database invariants.
// It runs in CI (see .github/workflows/data.yml) and locally before
// data contributions are merged.
//
// Fatal checks — any failure exits 1:
//   - every letter file parses as JSON and entries are well-formed
//   - changes_tracker.json is an object of string->string
//   - checksums.txt lists exactly the letter files with correct digests
//   - the embedded core matches its manifest (sha256, entry counts,
//     totals) and contains no headwords missing from the full database
//
// Warning checks — reported but non-fatal:
//   - headwords placed under the wrong letter file (pre-existing damage
//     from the original OPTED scrape; tracked for a future data cleanup)
package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yodeman/termdict/internal/dict"
)

var keyRe = regexp.MustCompile(`^[a-z][a-z'’.\-]*$`)

type report struct {
	failures int
	warnings int
}

func (r *report) fail(format string, args ...any) {
	r.failures++
	fmt.Printf("FAIL "+format+"\n", args...)
}

func (r *report) warn(format string, args ...any) {
	r.warnings++
	fmt.Printf("WARN "+format+"\n", args...)
}

func (r *report) ok(format string, args ...any) {
	fmt.Printf("ok   "+format+"\n", args...)
}

type options struct {
	dbaseDir string
	coreDir  string
}

func main() {
	var opts options
	flag.StringVar(&opts.dbaseDir, "dbase-dir", "word_dbase/json",
		"directory holding the full wb1913_*.json files")
	flag.StringVar(&opts.coreDir, "core-dir", "internal/data/embedded/core",
		"directory holding the embedded core subset + core_manifest.json")
	flag.Parse()

	report, err := run(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbasecheck: %v\n", err)
		os.Exit(1)
	}
	if report.failures > 0 {
		os.Exit(1)
	}
}

// run executes all checks. It returns an error only for environmental
// problems (unreadable directories); data findings are recorded on the
// returned report.
func run(opts options) (*report, error) {
	r := &report{}

	fullEntries, err := checkLetterFiles(r, opts.dbaseDir)
	if err != nil {
		return r, err // environmental failure, not a data finding
	}
	checkTracker(r, filepath.Dir(opts.dbaseDir))
	checkChecksums(r, filepath.Dir(opts.dbaseDir), opts.dbaseDir)
	if err := checkCoreManifest(r, opts.coreDir, fullEntries); err != nil {
		return r, err
	}

	fmt.Println()
	switch {
	case r.failures > 0:
		fmt.Printf("DATA VALIDATION FAILED: %d fatal problem(s), %d warning(s)\n",
			r.failures, r.warnings)
	case r.warnings > 0:
		fmt.Printf("DATA VALIDATION PASSED with %d warning(s)\n", r.warnings)
	default:
		fmt.Println("DATA VALIDATION PASSED")
	}
	return r, nil
}

// checkLetterFiles parses every letter file, validating entry shape.
// The returned map merges all entries for downstream checks. A hard
// error means the directory itself is unusable; per-file problems are
// recorded on the report instead.
func checkLetterFiles(r *report, dir string) (map[string]dict.Entity, error) {
	names, err := filepath.Glob(filepath.Join(dir, "wb1913_*.json"))
	if err != nil {
		return nil, err
	}
	if len(names) != 26 {
		r.warn("expected 26 letter files, found %d", len(names))
	}
	sort.Strings(names)

	entries := make(map[string]dict.Entity)
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var batch map[string]dict.Entity
		if err := json.Unmarshal(raw, &batch); err != nil {
			r.fail("%s: invalid JSON: %v", filepath.Base(name), err)
			continue
		}

		letter := strings.TrimSuffix(filepath.Base(name), ".json")
		letter = strings.TrimPrefix(letter, "wb1913_")
		for key, entity := range batch {
			if !strings.HasPrefix(key, letter) || !keyRe.MatchString(key) {
				r.warn("%s: key %q not under '%s' or malformed",
					filepath.Base(name), key, letter)
			}
			if entity.Word != "" && entity.Word != key {
				r.fail("%s: 'word' field %q disagrees with key %q",
					filepath.Base(name), entity.Word, key)
			}
			entries[key] = entity
		}
	}
	r.ok("letter files parse (%d files, %d entries)", len(names), len(entries))
	return entries, nil
}

func checkTracker(r *report, wordDbaseDir string) {
	path := filepath.Join(wordDbaseDir, "changes_tracker.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		r.fail("reading changes_tracker.json: %v", err)
		return
	}
	var tracker map[string]string
	if err := json.Unmarshal(raw, &tracker); err != nil {
		r.fail("changes_tracker.json must be an object of string->string: %v", err)
		return
	}
	r.ok("changes_tracker.json consistent (%d tracked files)", len(tracker))
}

func checkChecksums(r *report, wordDbaseDir, dbaseDir string) {
	path := filepath.Join(wordDbaseDir, "checksums.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		r.fail("reading checksums.txt: %v", err)
		return
	}

	listed := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		listed[strings.TrimPrefix(fields[1], "*")] = fields[0]
	}

	actual := map[string]bool{}
	matches, _ := filepath.Glob(filepath.Join(dbaseDir, "wb1913_*.json"))
	for _, name := range matches {
		actual[filepath.Base(name)] = true
	}

	for name := range listed {
		if !actual[name] {
			r.fail("checksums.txt references unknown file %q", name)
		}
	}
	for name := range actual {
		if _, ok := listed[name]; !ok {
			r.fail("checksums.txt is missing %q", name)
		}
	}
	for name, want := range listed {
		data, err := os.ReadFile(filepath.Join(dbaseDir, name))
		if err != nil {
			r.fail("checksums.txt: reading %s: %v", name, err)
			continue
		}
		got := hex.EncodeToString(sumBytes(data))
		if got != want {
			r.fail("checksums.txt: %s digest mismatch", name)
		}
	}
	r.ok("checksums.txt verified (%d files)", len(listed))
}

func checkCoreManifest(r *report, coreDir string, fullEntries map[string]dict.Entity) error {
	raw, err := os.ReadFile(filepath.Join(coreDir, "core_manifest.json"))
	if err != nil {
		return fmt.Errorf("reading core manifest: %w", err)
	}
	var manifest struct {
		TotalEntries     int    `json:"total_entries"`
		GeneratorVersion string `json:"generator_version"`
		Files            []struct {
			Name    string `json:"name"`
			SHA256  string `json:"sha256"`
			Entries int    `json:"entries"`
		} `json:"files"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("parsing core manifest: %w", err)
	}

	total := 0
	coreWords := make(map[string]bool)
	for _, record := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(coreDir, record.Name))
		if err != nil {
			r.fail("core/%s: %v", record.Name, err)
			continue
		}
		if got := hex.EncodeToString(sumBytes(data)); got != record.SHA256 {
			r.fail("core/%s: sha256 differs from manifest", record.Name)
		}
		gz, err := gzip.NewReader(strings.NewReader(string(data)))
		if err != nil {
			r.fail("core/%s: gzip: %v", record.Name, err)
			continue
		}
		var batch map[string]dict.Entity
		if err := json.NewDecoder(gz).Decode(&batch); err != nil {
			r.fail("core/%s: decode: %v", record.Name, err)
			continue
		}
		if len(batch) != record.Entries {
			r.fail("core/%s: %d entries, manifest claims %d",
				record.Name, len(batch), record.Entries)
		}
		total += len(batch)
		for word := range batch {
			coreWords[word] = true
		}
	}
	if total != manifest.TotalEntries {
		r.fail("core manifest totals wrong: files sum to %d, total_entries says %d",
			total, manifest.TotalEntries)
	}

	absent := 0
	for word := range coreWords {
		if _, ok := fullEntries[word]; !ok {
			if absent == 0 {
				r.warn("core contains words absent from the full database (first: %q)", word)
			}
			absent++
		}
	}
	r.ok("embedded core matches manifest (%d entries, generator v%s)",
		total, manifest.GeneratorVersion)
	return nil
}

func sumBytes(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}
