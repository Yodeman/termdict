package data

import (
	"encoding/json"
	"fmt"
	"os"
)

// Tracker maps a database file name to the version string of its last
// synced content.
type Tracker map[string]string

// ReadTracker loads the local changes tracker. A missing file is not an
// error: it means nothing has been downloaded yet, so an empty tracker
// (forcing a full sync) is returned.
func ReadTracker(path string) (Tracker, error) {
	openedFile, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Tracker{}, nil
		}
		return nil, fmt.Errorf("reading tracker %s: %w", path, err)
	}
	defer openedFile.Close() //nolint:errcheck // read-only file

	var tracker Tracker
	if err := json.NewDecoder(openedFile).Decode(&tracker); err != nil {
		return nil, fmt.Errorf("decoding tracker %s: %w", path, err)
	}
	return tracker, nil
}

// WriteTracker atomically replaces the local changes tracker via a temp
// file and rename, so a crash mid-write cannot corrupt it.
func WriteTracker(path string, tracker Tracker) error {
	encoding, err := json.MarshalIndent(tracker, "", "    ")
	if err != nil {
		return fmt.Errorf("encoding tracker %s: %w", path, err)
	}

	tmpPath := path + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("opening tracker %s: %w", tmpPath, err)
	}

	if _, err = tmpFile.Write(append(encoding, '\n')); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing tracker %s: %w", tmpPath, err)
	}
	if err = tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing tracker %s: %w", tmpPath, err)
	}
	if err = os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replacing tracker %s: %w", path, err)
	}
	return nil
}
