package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yodeman/termdict/internal/dict"
)

// newTestServer spins up an httptest server exposing a fake update
// channel and returns a Client pointed at it. Mutations may replace
// individual routes before they are registered.
func newTestServer(t *testing.T, dir string, mutate func(routes map[string]http.HandlerFunc)) *Client {
	t.Helper()

	routes := map[string]http.HandlerFunc{
		"/changes_tracker.json": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"wb1913_a.json":"v1","wb1913_b.json":"v2"}`)
		},
		"/wb1913_a.json": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"apple":{"word":"apple","definitions":[{"part_of_speech":"n.","definition":"A fruit."}]}}`)
		},
		"/wb1913_b.json": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"banana":{"word":"banana","definitions":[{"part_of_speech":"n.","definition":"A berry."}]}}`)
		},
	}
	// Serve plausible content for every remaining letter so full-set
	// downloads succeed.
	for r := 'c'; r <= 'z'; r++ {
		name := fmt.Sprintf("/wb1913_%c.json", r)
		letter := string(r)
		routes[name] = func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprintf(w, `{"%s":{"word":"%s"}}`, letter, letter)
		}
	}
	if mutate != nil {
		mutate(routes)
	}

	mux := http.NewServeMux()
	for pattern, handler := range routes {
		mux.HandleFunc(pattern, handler)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return &Client{
		BaseURL:      server.URL + "/",
		TrackerURL:   server.URL + "/changes_tracker.json",
		ChecksumsURL: server.URL + "/checksums.txt",
		Dir:          dir,
		TrackerPath:  filepath.Join(dir, "changes_tracker.json"),
		Workers:      2,
		Retries:      2,
		backoff:      func(int) time.Duration { return time.Millisecond },
	}
}

// newDefaultTestServer is newTestServer without handler mutations.
func newDefaultTestServer(t *testing.T, dir string) *Client {
	return newTestServer(t, dir, nil)
}

func TestCheckChangesDiff(t *testing.T) {
	dir := t.TempDir()
	client := newDefaultTestServer(t, dir)

	// Local already has a.json at v1; b.json is missing entirely.
	if err := WriteTracker(client.TrackerPath, Tracker{"wb1913_a.json": "v1"}); err != nil {
		t.Fatalf("seed tracker: %v", err)
	}

	changes, err := client.CheckChanges(context.Background())
	if err != nil {
		t.Fatalf("CheckChanges: %v", err)
	}
	if len(changes) != 1 || changes[0] != "wb1913_b.json" {
		t.Errorf("changes = %v, want [wb1913_b.json]", changes)
	}
}

func TestCheckChangesMissingLocalTracker(t *testing.T) {
	dir := t.TempDir()
	client := newDefaultTestServer(t, dir)

	changes, err := client.CheckChanges(context.Background())
	if err != nil {
		t.Fatalf("CheckChanges: %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("missing local tracker should flag every remote file, got %v", changes)
	}
}

func TestFetchSuccess(t *testing.T) {
	dir := t.TempDir()
	client := newDefaultTestServer(t, dir)

	var progressCalls int
	err := client.Fetch(context.Background(),
		[]string{"wb1913_a.json", "wb1913_b.json"},
		func(done, total int, _ string) {
			progressCalls++
			if done < 1 || done > total {
				t.Errorf("progress out of range: %d/%d", done, total)
			}
		})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if progressCalls != 2 {
		t.Errorf("progress called %d times, want 2", progressCalls)
	}

	for _, name := range []string{"wb1913_a.json", "wb1913_b.json"} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("downloaded file %s missing: %v", name, err)
		}
		var batch map[string]dict.Entity
		if err := json.Unmarshal(raw, &batch); err != nil {
			t.Errorf("%s is not valid letter JSON: %v", name, err)
		}
	}
	// No .part leftovers.
	matches, _ := filepath.Glob(filepath.Join(dir, "*.part"))
	if len(matches) != 0 {
		t.Errorf("leftover temp files: %v", matches)
	}
}

func TestFetchPartialFailure(t *testing.T) {
	dir := t.TempDir()
	client := newTestServer(t, dir, func(routes map[string]http.HandlerFunc) {
		routes["/wb1913_b.json"] = func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	err := client.Fetch(context.Background(),
		[]string{"wb1913_a.json", "wb1913_b.json"}, nil)

	var fetchErr *FetchError
	if !errors.As(err, &fetchErr) {
		t.Fatalf("expected *FetchError, got %v", err)
	}
	if _, ok := fetchErr.Failures["wb1913_b.json"]; !ok {
		t.Errorf("failure not reported for wb1913_b.json: %v", fetchErr.Failures)
	}
	if len(fetchErr.Failures) != 1 {
		t.Errorf("unexpected failure set: %v", fetchErr.Failures)
	}

	// The healthy file must still have been written.
	if _, statErr := os.Stat(filepath.Join(dir, "wb1913_a.json")); statErr != nil {
		t.Errorf("successful download should be kept despite sibling failure: %v", statErr)
	}
}

func TestUpdateDoesNotOverwriteTrackerOnFailure(t *testing.T) {
	dir := t.TempDir()
	client := newTestServer(t, dir, func(routes map[string]http.HandlerFunc) {
		routes["/wb1913_b.json"] = func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "server error", http.StatusInternalServerError)
		}
	})

	// Local state claims both files are current at v0.
	local := Tracker{"wb1913_a.json": "v0", "wb1913_b.json": "v0"}
	if err := WriteTracker(client.TrackerPath, local); err != nil {
		t.Fatalf("seed tracker: %v", err)
	}

	if _, err := client.Update(context.Background(), nil); err == nil {
		t.Fatal("Update should fail when a file cannot be downloaded")
	}

	got, err := ReadTracker(client.TrackerPath)
	if err != nil {
		t.Fatalf("ReadTracker: %v", err)
	}
	for file, version := range local {
		if got[file] != version {
			t.Errorf("tracker[%q] was overwritten on failed update: got %q, want %q",
				file, got[file], version)
		}
	}
}

func TestUpdateSuccessRefreshesTracker(t *testing.T) {
	dir := t.TempDir()
	client := newDefaultTestServer(t, dir)

	updated, err := client.Update(context.Background(), nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated != 2 {
		t.Errorf("updated = %d, want 2", updated)
	}

	got, err := ReadTracker(client.TrackerPath)
	if err != nil {
		t.Fatalf("ReadTracker: %v", err)
	}
	if got["wb1913_a.json"] != "v1" || got["wb1913_b.json"] != "v2" {
		t.Errorf("tracker not refreshed: %v", got)
	}
}

func TestUpdateUpToDate(t *testing.T) {
	dir := t.TempDir()
	client := newDefaultTestServer(t, dir)

	if err := WriteTracker(client.TrackerPath,
		Tracker{"wb1913_a.json": "v1", "wb1913_b.json": "v2"}); err != nil {
		t.Fatalf("seed tracker: %v", err)
	}

	updated, err := client.Update(context.Background(), nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0", updated)
	}
}

func TestDownloadRetriesTransientFailure(t *testing.T) {
	dir := t.TempDir()
	var attempts atomic.Int32
	client := newTestServer(t, dir, func(routes map[string]http.HandlerFunc) {
		routes["/flaky.json"] = func(w http.ResponseWriter, _ *http.Request) {
			if attempts.Add(1) < 2 {
				http.Error(w, "transient", http.StatusBadGateway)
				return
			}
			_, _ = fmt.Fprint(w, `{"ok":{"word":"ok"}}`)
		}
	})
	client.BaseURL += "" // unchanged

	if err := client.downloadFile(context.Background(),
		client.BaseURL+"flaky.json", filepath.Join(dir, "flaky.json")); err != nil {
		t.Fatalf("downloadFile should recover after transient failure: %v", err)
	}
	if attempts.Load() != 2 {
		t.Errorf("attempts = %d, want 2", attempts.Load())
	}
}

func TestDownloadGivesUpAfterRetries(t *testing.T) {
	dir := t.TempDir()
	client := newTestServer(t, dir, func(routes map[string]http.HandlerFunc) {
		routes["/dead.json"] = func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "gone", http.StatusNotFound)
		}
	})

	err := client.downloadFile(context.Background(),
		client.BaseURL+"dead.json", filepath.Join(dir, "dead.json"))
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "after") || !strings.Contains(err.Error(), "attempts") {
		t.Errorf("error should mention attempts: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "dead.json")); statErr == nil {
		t.Error("failed download must not leave a final file behind")
	}
}

func TestFetchContextCancellation(t *testing.T) {
	dir := t.TempDir()
	release := make(chan struct{})
	client := newTestServer(t, dir, func(routes map[string]http.HandlerFunc) {
		routes["/slow.json"] = func(w http.ResponseWriter, _ *http.Request) {
			<-release
			_, _ = fmt.Fprint(w, `{}`)
		}
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := client.Fetch(ctx, []string{"slow.json"}, nil)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("error should mention cancellation: %v", err)
	}
}

func TestResolveChannel(t *testing.T) {
	dev := ResolveChannel("dev")
	if dev.BaseURL != DefaultBaseURL || dev.TrackerURL != DefaultTrackerURL {
		t.Errorf("dev channel should track main: %+v", dev)
	}
	if ResolveChannel("").BaseURL != DefaultBaseURL {
		t.Error("empty version should use the dev channel")
	}

	tagged := ResolveChannel("v0.2.0")
	want := "https://github.com/yodeman/termdict/releases/download/v0.2.0/termdict-data/"
	if tagged.BaseURL != want {
		t.Errorf("tagged BaseURL = %q, want %q", tagged.BaseURL, want)
	}
	if tagged.TrackerURL != want+"changes_tracker.json" {
		t.Errorf("tagged TrackerURL = %q", tagged.TrackerURL)
	}
	if tagged.ChecksumsURL != want+"checksums.txt" {
		t.Errorf("tagged ChecksumsURL = %q", tagged.ChecksumsURL)
	}
}

func TestFetchVerifiesChecksums(t *testing.T) {
	dir := t.TempDir()
	valid := `{"apple":{"word":"apple","definitions":[{"part_of_speech":"n.","definition":"A fruit."}]}}`
	sum := sha256Hex([]byte(valid))

	client := newTestServer(t, dir, func(routes map[string]http.HandlerFunc) {
		routes["/checksums.txt"] = func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprintf(w, "%s  wb1913_a.json\n", sum)
		}
		routes["/wb1913_a.json"] = func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, valid)
		}
	})

	if err := client.Fetch(context.Background(), []string{"wb1913_a.json"}, nil); err != nil {
		t.Fatalf("Fetch with matching checksum should succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "wb1913_a.json")); err != nil {
		t.Fatalf("verified file missing: %v", err)
	}
}

func TestFetchRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	client := newTestServer(t, dir, func(routes map[string]http.HandlerFunc) {
		routes["/checksums.txt"] = func(w http.ResponseWriter, _ *http.Request) {
			// Deliberately wrong digest for the served content.
			_, _ = fmt.Fprintf(w, "%s  wb1913_a.json\n", strings.Repeat("ab", 32))
		}
	})

	err := client.Fetch(context.Background(), []string{"wb1913_a.json"}, nil)
	var fetchErr *FetchError
	if !errors.As(err, &fetchErr) {
		t.Fatalf("expected FetchError on checksum mismatch, got %v", err)
	}
	if msg := fetchErr.Failures["wb1913_a.json"].Error(); !strings.Contains(msg, "checksum mismatch") {
		t.Errorf("failure should mention checksum mismatch: %v", msg)
	}
	// The tampered file must have been removed.
	if _, statErr := os.Stat(filepath.Join(dir, "wb1913_a.json")); statErr == nil {
		t.Error("checksum-mismatched file was not removed")
	}
}

func TestFetchSkipsVerificationWhenManifestMissing(t *testing.T) {
	dir := t.TempDir()
	client := newDefaultTestServer(t, dir) // no /checksums.txt route => 404

	if err := client.Fetch(context.Background(), []string{"wb1913_a.json"}, nil); err != nil {
		t.Fatalf("unreachable checksum manifest should degrade gracefully, got %v", err)
	}
}

func TestDownloadFull(t *testing.T) {
	dir := t.TempDir()
	client := newDefaultTestServer(t, dir)

	downloaded, err := client.DownloadFull(context.Background(), nil)
	if err != nil {
		t.Fatalf("DownloadFull: %v", err)
	}
	if downloaded != 26 {
		t.Errorf("downloaded = %d, want 26 letter files", downloaded)
	}

	got, err := ReadTracker(client.TrackerPath)
	if err != nil {
		t.Fatalf("ReadTracker: %v", err)
	}
	if got["wb1913_a.json"] != "v1" || got["wb1913_b.json"] != "v2" {
		t.Errorf("tracker not refreshed after full download: %v", got)
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
