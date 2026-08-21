package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yodeman/termdict/internal/data"
	"github.com/yodeman/termdict/internal/dict"
)

// GeneratorVersion is bumped whenever selection or encoding behavior
// changes, so manifests record which logic produced them.
const GeneratorVersion = "1"

var wordRe = regexp.MustCompile(`^[a-z]+$`)

// Options configures a core-subset build.
type Options struct {
	DbaseDir      string // directory holding the full wb1913_*.json files
	FrequencyPath string // vendored frequency list (word count per line)
	OutDir        string // destination for *.json.gz + core_manifest.json
	MaxWords      int    // hard cap on selected headwords
	BudgetBytes   int64  // cap on uncompressed JSON size of the subset
	Timestamp     bool   // embed generation time (breaks determinism)

	FrequencyName    string // provenance metadata recorded in the manifest
	FrequencyURL     string
	FrequencyLicense string
}

// FileRecord describes one generated letter file.
type FileRecord struct {
	Name         string `json:"name"`
	SHA256       string `json:"sha256"`
	Entries      int    `json:"entries"`
	Uncompressed int64  `json:"bytes_uncompressed"`
	Compressed   int64  `json:"bytes_compressed"`
}

// FrequencySource records where the ranking came from.
type FrequencySource struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	License string `json:"license"`
	File    string `json:"file"`
	SHA256  string `json:"sha256"`
}

// Manifest is the integrity/provenance document embedded alongside the
// core subset.
type Manifest struct {
	Format            int             `json:"format"`
	GeneratorVersion  string          `json:"generator_version"`
	GeneratedAt       string          `json:"generated_at,omitempty"`
	FrequencySource   FrequencySource `json:"frequency_source"`
	TotalEntries      int             `json:"total_entries"`
	TotalUncompressed int64           `json:"bytes_uncompressed"`
	TotalCompressed   int64           `json:"bytes_compressed"`
	Files             []FileRecord    `json:"files"`
}

// loadFrequencyRanks reads a "word count" frequency list and returns
// word -> rank (1-based; first occurrence wins). Words containing
// anything other than a-z are ignored.
func loadFrequencyRanks(path string) (map[string]int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ranks := make(map[string]int)
	rank := 0
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		word := strings.ToLower(fields[0])
		if !wordRe.MatchString(word) {
			continue
		}
		if _, seen := ranks[word]; !seen {
			rank++
			ranks[word] = rank
		}
	}
	return ranks, nil
}

// selectCore picks at most maxWords headwords by ascending frequency
// rank while staying within budgetBytes of uncompressed JSON. Headwords
// absent from the frequency list are never selected. Ties break on the
// word itself so output is deterministic.
func selectCore(entries map[string]dict.Entity, ranks map[string]int,
	maxWords int, budgetBytes int64,
) ([]dict.Entity, int64) {
	type rankedEntity struct {
		entity dict.Entity
		rank   int
	}

	candidates := make([]rankedEntity, 0, len(entries))
	for word, entity := range entries {
		if rank, ok := ranks[word]; ok {
			candidates = append(candidates, rankedEntity{entity: entity, rank: rank})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].rank != candidates[j].rank {
			return candidates[i].rank < candidates[j].rank
		}
		return candidates[i].entity.Word < candidates[j].entity.Word
	})

	selected := make([]dict.Entity, 0, min(maxWords, len(candidates)))
	var used int64
	for _, candidate := range candidates {
		if len(selected) >= maxWords {
			break
		}
		size := int64(len(entityFileJSON(candidate.entity)))
		if used+size > budgetBytes {
			continue // smaller entries later in the list may still fit
		}
		selected = append(selected, candidate.entity)
		used += size
	}
	return selected, used
}

// entityFileJSON marshals exactly as writeCoreFiles will store the
// entity, so budget accounting matches on-disk sizes.
func entityFileJSON(entity dict.Entity) []byte {
	raw, err := json.MarshalIndent(map[string]dict.Entity{
		strings.ToLower(entity.Word): entity,
	}, "", "    ")
	if err != nil {
		panic(fmt.Sprintf("dbasebuild: marshaling %s: %v", entity.Word, err))
	}
	return raw
}

// writeCoreFiles groups entities by initial letter and writes one
// gzipped JSON file per letter. Output is byte-deterministic: fixed
// compression level, no gzip timestamp, sorted keys.
func writeCoreFiles(outDir string, entities []dict.Entity) ([]FileRecord, error) {
	groups := map[byte]map[string]dict.Entity{}
	for _, entity := range entities {
		word := strings.ToLower(entity.Word)
		if word == "" {
			continue
		}
		letter := word[0]
		if groups[letter] == nil {
			groups[letter] = map[string]dict.Entity{}
		}
		groups[letter][word] = entity
	}

	letters := make([]byte, 0, len(groups))
	for letter := range groups {
		letters = append(letters, letter)
	}
	sort.Slice(letters, func(i, j int) bool { return letters[i] < letters[j] })

	records := make([]FileRecord, 0, len(letters))
	for _, letter := range letters {
		name := fmt.Sprintf("wb1913_%c.json.gz", letter)
		uncompressed, err := json.MarshalIndent(groups[letter], "", "    ")
		if err != nil {
			return nil, err
		}

		compressed, err := gzipBytes(uncompressed)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(outDir, name), compressed, 0o644); err != nil {
			return nil, err
		}

		records = append(records, FileRecord{
			Name:         name,
			SHA256:       hashHex(compressed),
			Entries:      len(groups[letter]),
			Uncompressed: int64(len(uncompressed)),
			Compressed:   int64(len(compressed)),
		})
	}
	return records, nil
}

func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	writer.ModTime = time.Time{} // omit MTIME for reproducible output
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// removeStaleCore deletes leftover *.json.gz files from previous builds
// so the out directory exactly matches the new manifest.
func removeStaleCore(outDir string, keep map[string]bool) error {
	matches, err := filepath.Glob(filepath.Join(outDir, "wb1913_*.json.gz"))
	if err != nil {
		return err
	}
	for _, path := range matches {
		if !keep[filepath.Base(path)] {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	return nil
}

// emitChecksums writes a sha256sum-compatible manifest for every
// wb1913_*.json in dir (the downloadable full database).
func emitChecksums(dir, outPath string) (int, error) {
	names, err := filepath.Glob(filepath.Join(dir, "wb1913_*.json"))
	if err != nil {
		return 0, err
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			return 0, err
		}
		fmt.Fprintf(&b, "%s  %s\n", hashHex(raw), filepath.Base(name))
	}
	return len(names), os.WriteFile(outPath, []byte(b.String()), 0o644)
}

func fileSHA(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashHex(raw), nil
}

func hashHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Build runs the whole pipeline and returns the written manifest.
func Build(opts Options) (*Manifest, error) {
	ranks, err := loadFrequencyRanks(opts.FrequencyPath)
	if err != nil {
		return nil, fmt.Errorf("loading frequency list: %w", err)
	}
	freqSHA, err := fileSHA(opts.FrequencyPath)
	if err != nil {
		return nil, err
	}

	entries, err := data.FileStore{Dir: opts.DbaseDir}.Load()
	if err != nil {
		return nil, fmt.Errorf("loading database: %w", err)
	}

	selected, _ := selectCore(entries, ranks, opts.MaxWords, opts.BudgetBytes)
	if len(selected) == 0 {
		return nil, fmt.Errorf("no headwords selected; check frequency list overlap")
	}

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return nil, err
	}
	records, err := writeCoreFiles(opts.OutDir, selected)
	if err != nil {
		return nil, err
	}
	keep := make(map[string]bool, len(records))
	for _, record := range records {
		keep[record.Name] = true
	}
	if err := removeStaleCore(opts.OutDir, keep); err != nil {
		return nil, err
	}

	manifest := &Manifest{
		Format:           1,
		GeneratorVersion: GeneratorVersion,
		FrequencySource: FrequencySource{
			Name:    opts.FrequencyName,
			URL:     opts.FrequencyURL,
			License: opts.FrequencyLicense,
			File:    filepath.Base(opts.FrequencyPath),
			SHA256:  freqSHA,
		},
		TotalEntries: len(selected),
		Files:        records,
	}
	for _, record := range records {
		manifest.TotalUncompressed += record.Uncompressed
		manifest.TotalCompressed += record.Compressed
	}
	if opts.Timestamp {
		manifest.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}

	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(opts.OutDir, "core_manifest.json"), append(raw, '\n'), 0o644); err != nil {
		return nil, err
	}
	return manifest, nil
}
