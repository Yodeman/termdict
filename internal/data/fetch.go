package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/yodeman/termdict/internal/dict"
)

const (
	// DefaultBaseURL is the directory-style URL the letter files are
	// fetched from. Phase 2 makes this version-aware.
	DefaultBaseURL = "https://raw.githubusercontent.com/yodeman/termdict/main/word_dbase/json/"

	// DefaultTrackerURL is the remote changes tracker of the default
	// update channel.
	DefaultTrackerURL = "https://raw.githubusercontent.com/yodeman/termdict/main/word_dbase/changes_tracker.json"

	defaultWorkers     = 6
	defaultRetries     = 3
	requestTimeout     = 30 * time.Second
	baseBackoff        = 500 * time.Millisecond
	maxBackoff         = 4 * time.Second
	maxDataFileSize    = 64 << 20 // 64 MiB safety bound per letter file
	maxTrackerFileSize = 1 << 20  // 1 MiB safety bound for the tracker
)

// ProgressFn receives periodic progress during Fetch runs.
type ProgressFn func(done, total int, current string)

// Client downloads words-database files and keeps the local changes
// tracker in sync. The zero value is usable in tests once the URL
// fields are set; HTTP, Workers and Retries fall back to defaults.
type Client struct {
	HTTP        *http.Client
	BaseURL     string // must end in "/"
	TrackerURL  string
	Dir         string // local directory the letter files are saved to
	TrackerPath string // local changes_tracker.json path
	Workers     int
	Retries     int // attempts per file beyond the first

	backoff func(attempt int) time.Duration // overridable in tests
}

// NewClient returns a Client targeting the default update channel.
func NewClient(dir, trackerPath string) *Client {
	return &Client{
		BaseURL:     DefaultBaseURL,
		TrackerURL:  DefaultTrackerURL,
		Dir:         dir,
		TrackerPath: trackerPath,
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: requestTimeout}
}

func (c *Client) workerCount() int {
	if c.Workers > 0 {
		return c.Workers
	}
	return defaultWorkers
}

func (c *Client) retryCount() int {
	if c.Retries > 0 {
		return c.Retries
	}
	return defaultRetries
}

func (c *Client) backoffFor(attempt int) time.Duration {
	if c.backoff != nil {
		return c.backoff(attempt)
	}
	d := baseBackoff << (attempt - 1)
	if d < baseBackoff || d > maxBackoff {
		d = maxBackoff
	}
	return d + rand.N(time.Duration(d)/5) // deterministic-free jitter up to +20%
}

// CheckChanges diffs the local changes tracker against the remote one
// and returns the names of files that need downloading. A missing local
// tracker counts every remote file as changed.
func (c *Client) CheckChanges(ctx context.Context) ([]string, error) {
	local, err := ReadTracker(c.TrackerPath)
	if err != nil {
		return nil, err
	}

	var remote Tracker
	if err := c.fetchJSON(ctx, c.TrackerURL, &remote, maxTrackerFileSize); err != nil {
		return nil, fmt.Errorf("checking remote changes: %w", err)
	}

	var changes []string
	for file, version := range remote {
		if localVersion, ok := local[file]; !ok || version != localVersion {
			changes = append(changes, file)
		}
	}
	slices.Sort(changes)
	return changes, nil
}

// Update checks for changed files, downloads them and refreshes the
// local tracker. It returns the number of files updated; zero means the
// database was already current. The tracker is only refreshed when
// every download succeeded, so failed files are retried on the next run.
func (c *Client) Update(ctx context.Context, progress ProgressFn) (int, error) {
	changes, err := c.CheckChanges(ctx)
	if err != nil {
		return 0, err
	}
	if len(changes) == 0 {
		return 0, nil
	}

	if err := c.Fetch(ctx, changes, progress); err != nil {
		return 0, err
	}
	if err := c.RefreshTracker(ctx); err != nil {
		return len(changes), fmt.Errorf(
			"database updated, but recording the change log failed: %w", err)
	}
	return len(changes), nil
}

// RefreshTracker replaces the local changes tracker with the remote one.
// Callers must only invoke it after all relevant files downloaded
// successfully; otherwise failed files would be marked as current.
func (c *Client) RefreshTracker(ctx context.Context) error {
	var remote Tracker
	if err := c.fetchJSON(ctx, c.TrackerURL, &remote, maxTrackerFileSize); err != nil {
		return fmt.Errorf("obtaining changes: %w", err)
	}
	return WriteTracker(c.TrackerPath, remote)
}

// FetchError aggregates the per-file failures of a Fetch run.
type FetchError struct {
	Failures map[string]error
}

func (e *FetchError) Error() string {
	files := make([]string, 0, len(e.Failures))
	for file := range e.Failures {
		files = append(files, file)
	}
	slices.Sort(files)

	var b strings.Builder
	fmt.Fprintf(&b, "%d file(s) failed to download:", len(files))
	for _, file := range files {
		fmt.Fprintf(&b, "\n  %s: %v", file, e.Failures[file])
	}
	return b.String()
}

// Unwrap returns a joined error of every failure for errors.Is/As users.
func (e *FetchError) Unwrap() error {
	errs := make([]error, 0, len(e.Failures))
	for _, file := range slices.Sorted(maps.Keys(e.Failures)) {
		errs = append(errs, fmt.Errorf("%s: %w", file, e.Failures[file]))
	}
	return errors.Join(errs...)
}

// fetchJSON GETs url and decodes the JSON body into dst, enforcing a
// response-size bound. The body is always closed.
func (c *Client) fetchJSON(ctx context.Context, url string, dst any, maxSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	response, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }() //nolint:errcheck // read-only body

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", response.Status)
	}

	body := io.LimitReader(response.Body, maxSize+1)
	if err := json.NewDecoder(body).Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("empty response")
		}
		return err
	}
	return nil
}

type fileResult struct {
	file string
	err  error
}

// Fetch downloads every file into the client's Dir using a bounded
// worker pool, reporting progress as files complete. A partially failed
// run leaves successfully downloaded files in place and returns a
// *FetchError describing the rest; nothing is silently marked as synced.
func (c *Client) Fetch(ctx context.Context, files []string, progress ProgressFn) error {
	if len(files) == 0 {
		return nil
	}

	jobs := make(chan string)
	results := make(chan fileResult, len(files))

	var wg sync.WaitGroup
	workers := min(c.workerCount(), len(files))
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range jobs {
				savePath := filepath.Join(c.Dir, file)
				results <- fileResult{file: file, err: c.downloadFile(ctx, c.BaseURL+file, savePath)}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, file := range files {
			select {
			case jobs <- file:
			case <-ctx.Done():
				return
			}
		}
	}()

	failures := make(map[string]error)
	completed := 0
collect:
	for completed < len(files) {
		select {
		case <-ctx.Done():
			break collect
		case result := <-results:
			completed++
			if result.err != nil {
				failures[result.file] = result.err
			}
			if progress != nil {
				progress(completed, len(files), result.file)
			}
		}
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("download cancelled after %d/%d files (%d failed): %w",
			completed, len(files), len(failures), err)
	}
	if len(failures) > 0 {
		return &FetchError{Failures: failures}
	}
	return nil
}

// downloadFile downloads url to savePath with retries and exponential
// backoff. Each file is streamed to "<savePath>.part", validated as a
// letter file, then atomically renamed into place.
func (c *Client) downloadFile(ctx context.Context, url, savePath string) error {
	var lastErr error
	attempts := c.retryCount() + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			if err := sleepContext(ctx, c.backoffFor(attempt-1)); err != nil {
				return err
			}
		}

		lastErr = c.downloadOnce(ctx, url, savePath)
		if lastErr == nil {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return fmt.Errorf("%s: %w (after %d attempts)", url, lastErr, attempts)
}

func (c *Client) downloadOnce(ctx context.Context, url, savePath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	response, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }() //nolint:errcheck // read-only body

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", response.Status)
	}

	partPath := savePath + ".part"
	partFile, err := os.OpenFile(partPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", partPath, err)
	}

	written, copyErr := io.Copy(partFile, io.LimitReader(response.Body, maxDataFileSize+1))
	closeErr := partFile.Close()
	switch {
	case copyErr != nil:
		_ = os.Remove(partPath)
		return fmt.Errorf("saving %s: %w", partPath, copyErr)
	case closeErr != nil:
		_ = os.Remove(partPath)
		return fmt.Errorf("closing %s: %w", partPath, closeErr)
	case written > maxDataFileSize:
		_ = os.Remove(partPath)
		return fmt.Errorf("%s exceeds the %d byte size limit", url, maxDataFileSize)
	}

	if err := validateLetterFile(partPath); err != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("invalid content from %s: %w", url, err)
	}
	if err := os.Rename(partPath, savePath); err != nil {
		_ = os.Remove(partPath)
		return fmt.Errorf("publishing %s: %w", savePath, err)
	}
	return nil
}

func validateLetterFile(path string) error {
	openedFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer openedFile.Close() //nolint:errcheck // read-only file

	var batch map[string]dict.Entity
	return json.NewDecoder(openedFile).Decode(&batch)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
