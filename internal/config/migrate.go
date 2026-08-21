package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// migrateLegacy moves a v0.1.0-era data directory into the new
// platform-standard location, exactly once.
//
// Safety rules:
//   - runs only when the marker is absent and the target dbase dir has
//     no letter files yet (never overwrites existing new-layout data)
//   - copies first, verifies file counts and sizes, writes the marker,
//     and only then removes the legacy tree
//   - any failure aborts with a warning; legacy data is never deleted
//     unless every file was copied and verified
func migrateLegacy(env envSource, goos string, paths Paths) {
	marker := filepath.Join(paths.DataDir, MarkerName)
	if _, err := os.Stat(marker); err == nil {
		return // already migrated
	}

	home, err := env.home()
	if err != nil {
		return // cannot locate legacy layout; nothing to do
	}
	legacyDir, ok := LegacyDataDirFor(goos, home)
	if !ok || legacyDir == paths.DataDir {
		return
	}

	legacyDbase := filepath.Join(legacyDir, "dbase", "json")
	if !hasLetterFiles(legacyDbase) {
		return // nothing worth migrating (fresh machine or empty install)
	}
	if hasLetterFiles(paths.DbaseDir) {
		// New-layout data already exists; keep it and mark migration done.
		writeMarker(marker)
		return
	}

	if err := copyDirVerified(legacyDir, paths.DataDir); err != nil {
		log.Printf("Warning: could not migrate data from %s: %v. "+
			"Continuing with an empty database; your old files were left in place.", legacyDir, err)
		return
	}
	if !writeMarker(marker) {
		log.Printf("Warning: migrated data from %s but could not write the "+
			"migration marker %s; the migration may run again next launch.", legacyDir, marker)
		return
	}
	if err := os.RemoveAll(legacyDir); err != nil {
		log.Printf("Warning: migrated data to %s but could not remove the old "+
			"directory %s (%v); you can delete it manually.", paths.DataDir, legacyDir, err)
		return
	}
	log.Printf("Migrated words database from %s to %s.", legacyDir, paths.DataDir)
}

// hasLetterFiles reports whether dir contains at least one regular
// letter file (directories named like letter files don't count).
func hasLetterFiles(dir string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, "wb1913_*.json"))
	if err != nil {
		return false
	}
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}

func writeMarker(path string) bool {
	return os.WriteFile(path, []byte("migrated\n"), 0o644) == nil
}

// copyDirVerified recursively copies src under dst and verifies that
// every regular file arrived with its size intact.
func copyDirVerified(src, dst string) error {
	inventory := map[string]int64{}
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return nil // skip symlinks etc.; database content is plain files
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := copyFile(path, target, info.Size()); err != nil {
			return err
		}
		inventory[rel] = info.Size()
		return nil
	})
	if err != nil {
		return err
	}

	// Verify: every copied file must exist at the expected size.
	for rel, size := range inventory {
		info, err := os.Stat(filepath.Join(dst, rel))
		if err != nil {
			return fmt.Errorf("copied file missing: %s: %w", rel, err)
		}
		if info.Size() != size {
			return fmt.Errorf("size mismatch after copy: %s", rel)
		}
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was copied")
	}
	return nil
}

func copyFile(src, dst string, size int64) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		in.Close() //nolint:errcheck // best effort on error path
		return err
	}
	written, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	inErr := in.Close()
	switch {
	case copyErr != nil:
		return fmt.Errorf("copying %s: %w", src, copyErr)
	case closeErr != nil:
		return fmt.Errorf("closing %s: %w", dst, closeErr)
	case inErr != nil:
		return fmt.Errorf("closing %s: %w", src, inErr)
	case written != size:
		return fmt.Errorf("short copy of %s: got %d bytes, want %d", src, written, size)
	}
	return nil
}
