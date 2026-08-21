// Command dbasebuild generates the embedded offline core subset of the
// words database: the most frequent headwords, ranked by a vendored
// frequency list, written as deterministic per-letter gzip files plus a
// provenance manifest. It can also emit the sha256 checksums file for
// the downloadable full database.
package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	defaultMaxWords  = 25000
	defaultBudgetMB  = 16.0
	frequencyName    = "FrequencyWords en_50k (OpenSubtitles 2018 via OPUS)"
	frequencyURL     = "https://github.com/hermitdave/FrequencyWords"
	frequencyLicense = "CC-BY-SA-4.0"
)

func main() {
	var (
		dbaseDir      = flag.String("dbase-dir", "word_dbase/json", "directory holding the full wb1913_*.json files")
		frequencyPath = flag.String("frequency", "word_dbase/frequency/en_50k.txt", "vendored frequency list used for ranking")
		outDir        = flag.String("out", "internal/data/embedded/core", "destination directory for the core subset")
		checksumsOut  = flag.String("checksums-out", "word_dbase/checksums.txt", "path for the full-database checksums file (empty disables)")
		maxWords      = flag.Int("max-words", defaultMaxWords, "hard cap on embedded headwords")
		budgetMB      = flag.Float64("budget-mb", defaultBudgetMB, "cap on uncompressed JSON size of the subset, in MiB")
		timestamp     = flag.Bool("timestamp", false, "record generation time in the manifest (breaks byte determinism)")
	)
	flag.Parse()

	manifest, err := Build(Options{
		DbaseDir:         *dbaseDir,
		FrequencyPath:    *frequencyPath,
		OutDir:           *outDir,
		MaxWords:         *maxWords,
		BudgetBytes:      int64(*budgetMB * 1024 * 1024),
		Timestamp:        *timestamp,
		FrequencyName:    frequencyName,
		FrequencyURL:     frequencyURL,
		FrequencyLicense: frequencyLicense,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbasebuild: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("core subset: %d headwords (%d letter files)\n",
		manifest.TotalEntries, len(manifest.Files))
	fmt.Printf("size: %.2f MiB uncompressed -> %.2f MiB gzip\n",
		mib(manifest.TotalUncompressed), mib(manifest.TotalCompressed))
	for _, record := range manifest.Files {
		fmt.Printf("  %-22s %6d entries  %7.1f KiB gz\n",
			record.Name, record.Entries, kib(record.Compressed))
	}

	if *checksumsOut != "" {
		count, err := emitChecksums(*dbaseDir, *checksumsOut)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dbasebuild: writing checksums: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("checksums: %d files listed in %s\n", count, *checksumsOut)
	}
}

func mib(b int64) float64 { return float64(b) / (1024 * 1024) }
func kib(b int64) float64 { return float64(b) / 1024 }
