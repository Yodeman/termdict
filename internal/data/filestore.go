// Package data handles persistence and transfer of the words database:
// local letter-file stores, the changes tracker, and the HTTP client
// used for updates. The Store interface itself lives in internal/dict to
// keep the dependency direction data -> dict (avoids an import cycle
// while letting dict merge arbitrary stores).
package data

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/yodeman/termdict/internal/dict"
)

// FilePattern is the glob (relative to DbaseDir) of the dictionary
// letter files.
const FilePattern = "wb1913_*.json"

// FileStore loads dictionary entities from a directory of JSON letter
// files. Files that fail to decode are skipped with a warning; only
// environmental failures (unreadable directory) are returned as errors.
type FileStore struct {
	Dir string
}

// Name implements dict.Store.
func (s FileStore) Name() string { return "files:" + s.Dir }

// Load reads every matching letter file in s.Dir.
func (s FileStore) Load() (map[string]dict.Entity, error) {
	if info, err := os.Stat(s.Dir); err != nil {
		return nil, fmt.Errorf("accessing %s: %w", s.Dir, err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", s.Dir)
	}

	names, err := filepath.Glob(filepath.Join(s.Dir, FilePattern))
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", s.Dir, err)
	}

	entities := make(map[string]dict.Entity)
	for _, name := range names {
		batch, err := loadLetterFile(name)
		if err != nil {
			log.Printf("Warning: skipping %s: %v", name, err)
			continue
		}
		for word, entity := range batch {
			entities[word] = entity
		}
	}
	return entities, nil
}

func loadLetterFile(path string) (map[string]dict.Entity, error) {
	openedFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer openedFile.Close() //nolint:errcheck // read-only file

	var batch map[string]dict.Entity
	if err := json.NewDecoder(openedFile).Decode(&batch); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return batch, nil
}

// LetterFiles returns the canonical list of database letter file names,
// "wb1913_a.json" through "wb1913_z.json".
func LetterFiles() []string {
	files := make([]string, 0, 26)
	for r := 'a'; r <= 'z'; r++ {
		files = append(files, fmt.Sprintf("wb1913_%c.json", r))
	}
	return files
}
