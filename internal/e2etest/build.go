// Package e2etest provides test-only infrastructure for TermDict's
// end-to-end regression tests: it builds the real binary and runs it,
// either with plain pipes (CLI surface) or under a pseudo-terminal
// (TUI surface, unix only).
//
// It is a normal package imported exclusively by _test.go files, so
// nothing here ships in the binaries.
package e2etest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	buildOnce sync.Once
	buildPath string
	buildErr  error
)

// Build compiles the termdict binary once per test process and returns
// its path. The module root is derived from this file's location so
// the build works from any working directory.
func Build() (string, error) {
	buildOnce.Do(func() {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			buildErr = fmt.Errorf("e2etest: cannot locate module root")
			return
		}
		root := filepath.Dir(filepath.Dir(filepath.Dir(file)))

		out := filepath.Join(os.TempDir(), fmt.Sprintf("termdict-e2e-%d", os.Getpid()))
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		cmd := exec.CommandContext(context.Background(), "go", "build", "-o", out, ".")
		cmd.Dir = root
		if combined, err := cmd.CombinedOutput(); err != nil {
			buildErr = fmt.Errorf("building binary: %v\n%s", err, combined)
			return
		}
		buildPath = out
	})
	return buildPath, buildErr
}

// CLIResult captures the three observable surfaces of a CLI run.
type CLIResult struct {
	Stdout string
	Stderr string
	Code   int
}

// RunCLI executes the binary with the given arguments and environment
// (nil inherits the parent's environment), enforcing a timeout, and
// returns the collected streams plus the process exit code. A missing
// or negative exit status is reported as an error containing the
// streams, so callers can still assert on them.
func RunCLI(binary string, env []string, timeout time.Duration, args ...string) (CLIResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	result := CLIResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			result.Code = exitErr.ExitCode()
			return result, nil
		}
		if ctx.Err() != nil {
			return result, fmt.Errorf("timed out after %s (stdout: %q, stderr: %q)",
				timeout, result.Stdout, result.Stderr)
		}
		return result, runErr
	}
	return result, nil
}
