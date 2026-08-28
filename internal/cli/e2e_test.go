package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yodeman/termdict/internal/e2etest"
)

// TestCLIEndToEnd ports the shell-based CLI contract checks (formerly
// scripts/cli_e2e.sh): exit codes, stream discipline, piping behavior.
// Lookups run against the embedded core, so no network or database is
// needed. Skipped under -short.
func TestCLIEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: skipped in -short mode")
	}

	binary, err := e2etest.Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// Fresh offline environment: embedded core only, no proxies.
	baseEnv := func(extra map[string]string) []string {
		env := os.Environ()
		var out []string
		for _, kv := range env {
			if strings.HasPrefix(kv, "HTTP_PROXY=") ||
				strings.HasPrefix(kv, "HTTPS_PROXY=") ||
				strings.HasPrefix(kv, "NO_PROXY=") {
				continue
			}
			out = append(out, kv)
		}
		home, err := os.MkdirTemp("", "termdict-e2e-home")
		if err != nil {
			t.Fatalf("temp home: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(home) })
		out = append(out, "HOME="+home, "USERPROFILE="+home)
		for k, v := range extra {
			out = append(out, k+"="+v)
		}
		return out
	}

	run := func(t *testing.T, env []string, args ...string) e2etest.CLIResult {
		t.Helper()
		result, err := e2etest.RunCLI(binary, env, 30*time.Second, args...)
		if err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
		return result
	}

	t.Run("lookup", func(t *testing.T) {
		hit := run(t, baseEnv(nil), "time")
		if hit.Code != 0 {
			t.Errorf("exit = %d, want 0", hit.Code)
		}
		if !strings.Contains(hit.Stdout, "part of speech") {
			t.Errorf("stdout missing definition payload:\n%q", hit.Stdout)
		}
		if hit.Stderr != "" {
			t.Errorf("stderr should be empty on success, got %q", hit.Stderr)
		}
		if lines := strings.Split(strings.TrimRight(hit.Stdout, "\n"), "\n"); lines[0] != "time" {
			t.Errorf("first stdout line = %q, want headword", lines[0])
		}

		miss := run(t, baseEnv(nil), "zzzzqqqxx")
		if miss.Code != 1 {
			t.Errorf("exit = %d, want 1", miss.Code)
		}
		if miss.Stdout != "" {
			t.Errorf("stdout must stay empty on miss, got %q", miss.Stdout)
		}
		if !strings.Contains(miss.Stderr, `No results for "zzzzqqqxx"`) {
			t.Errorf("stderr missing not-found notice: %q", miss.Stderr)
		}

		mixed := run(t, baseEnv(nil), "ZZZZQQQXX")
		if mixed.Code != 1 {
			t.Errorf("case-mixed miss exit = %d, want 1", mixed.Code)
		}

		if got := strings.Count(hit.Stdout, "part of speech"); got == 0 {
			t.Error("output is not greppable (zero part-of-speech lines)")
		}
	})

	t.Run("version and help", func(t *testing.T) {
		version := run(t, baseEnv(nil), "--version")
		if version.Code != 0 || !strings.Contains(version.Stdout, "termdict") {
			t.Errorf("--version: exit=%d stdout=%q", version.Code, version.Stdout)
		}
		help := run(t, baseEnv(nil), "help")
		if help.Code != 0 || !strings.Contains(help.Stdout, "Exit codes:") {
			t.Errorf("help: exit=%d stdout=%q", help.Code, help.Stdout)
		}
	})

	t.Run("usage errors", func(t *testing.T) {
		bogus := run(t, baseEnv(nil), "--bogus")
		if bogus.Code != 2 {
			t.Errorf("unknown flag exit = %d, want 2", bogus.Code)
		}
		if !strings.Contains(bogus.Stderr, "unknown flag") {
			t.Errorf("stderr missing usage error: %q", bogus.Stderr)
		}

		multi := run(t, baseEnv(nil), "ice", "cream")
		if multi.Code != 2 {
			t.Errorf("unquoted multi-word exit = %d, want 2", multi.Code)
		}
	})

	t.Run("double dash protects reserved verbs", func(t *testing.T) {
		// "-- update" must be a WORD LOOKUP (exit 0 or 1), never the
		// update verb (which would print updater output or fail with 3).
		result := run(t, baseEnv(nil), "--", "update")
		if result.Code != 0 && result.Code != 1 {
			t.Fatalf("exit = %d, want 0 or 1", result.Code)
		}
		if strings.Contains(result.Stdout, "Updated") ||
			strings.Contains(result.Stdout, "Already up to date.") {
			t.Errorf("verb ran despite -- protection: stdout=%q", result.Stdout)
		}
	})

	t.Run("update failure exits 3", func(t *testing.T) {
		proxy := "http://127.0.0.1:9" // black hole: instant connection refusal
		result := run(t, baseEnv(map[string]string{
			"HTTP_PROXY": proxy, "HTTPS_PROXY": proxy,
		}), "update")
		if result.Code != 3 {
			t.Errorf("exit = %d, want 3", result.Code)
		}
		if result.Stdout != "" {
			t.Errorf("stdout must stay empty on failure, got %q", result.Stdout)
		}
	})
}
