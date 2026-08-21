package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/yodeman/termdict/internal/data"
	"github.com/yodeman/termdict/internal/dict"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		want      Action
		wantErr   bool
		errSubstr string
	}{
		{"no args launches TUI", nil, Action{Kind: ActionTUI}, false, ""},
		{"single word", []string{"time"}, Action{Kind: ActionLookup, Query: "time"}, false, ""},
		{"update verb", []string{"update"}, Action{Kind: ActionUpdate}, false, ""},
		{"download verb", []string{"download"}, Action{Kind: ActionDownload}, false, ""},
		{"help verb", []string{"help"}, Action{Kind: ActionHelp}, false, ""},
		{"version verb", []string{"version"}, Action{Kind: ActionVersion}, false, ""},
		{"uppercase verb is a word", []string{"Update"},
			Action{Kind: ActionLookup, Query: "Update"}, false, ""},
		{"verb must be first positional", []string{"time", "update"},
			Action{}, true, "single word"},
		{"long help flag", []string{"--help"}, Action{Kind: ActionHelp}, false, ""},
		{"short version flag", []string{"-v"}, Action{Kind: ActionVersion}, false, ""},
		{"flag beats positional", []string{"-h", "time"}, Action{Kind: ActionHelp}, false, ""},
		{"args after the word are positionals", []string{"time", "-h"},
			Action{}, true, "single word"},
		{"unknown long flag", []string{"--bogus"}, Action{}, true, `unknown flag "--bogus"`},
		{"unknown short flag", []string{"-x"}, Action{}, true, `unknown flag "-x"`},
		{"dash alone is a word", []string{"-"}, Action{Kind: ActionLookup, Query: "-"}, false, ""},
		{"double dash protects verbs", []string{"--", "update"},
			Action{Kind: ActionLookup, Query: "update"}, false, ""},
		{"empty term after double dash", []string{"--", ""}, Action{}, true, "empty search term"},
		{"multi-word needs quoting", []string{"ice", "cream"}, Action{}, true, "quote multi-word phrases"},
		{"verb rejects extra args", []string{"update", "now"}, Action{}, true, "takes no arguments"},
		{"download rejects extra args", []string{"download", "x"}, Action{}, true, "takes no arguments"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("error %q missing %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Kind != tc.want.Kind || got.Query != tc.want.Query {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

type fakeService struct {
	entries map[string]dict.Entity
	fuzzy   map[string][]string
}

func (s fakeService) Lookup(word string) (dict.Entity, bool) {
	e, ok := s.entries[strings.ToLower(strings.TrimSpace(word))]
	return e, ok
}

func (s fakeService) Fuzzy(query string, _ int) []string {
	if s.fuzzy == nil {
		return nil
	}
	return s.fuzzy[query]
}

type fakeUpdater struct {
	updateResult  int
	err           error
	progressCalls int
	lastCtx       context.Context
}

func (f *fakeUpdater) Update(ctx context.Context, progress data.ProgressFn) (int, error) {
	f.lastCtx = ctx
	f.progressCalls++
	if progress != nil {
		progress(1, 2, "wb1913_a.json")
	}
	return f.updateResult, f.err
}

func (f *fakeUpdater) DownloadFull(ctx context.Context, progress data.ProgressFn) (int, error) {
	return f.Update(ctx, progress)
}

func newRunner() (*Runner, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	return &Runner{Stdout: &out, Stderr: &errBuf}, &out, &errBuf
}

func TestRunLookupHit(t *testing.T) {
	svc := fakeService{entries: map[string]dict.Entity{"time": {Word: "time", WordDefinitions: []dict.Definition{
		{PartOfSpeech: "n.", WordDefinition: "A duration."},
	}}}}
	runner, out, errOut := newRunner()

	code := runner.RunLookup(svc, "Time") // case-insensitive

	if code != ExitOK {
		t.Errorf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out.String(), "part of speech: n.") ||
		!strings.Contains(out.String(), "└A duration.") {
		t.Errorf("stdout missing definition payload:\n%s", out.String())
	}
	if strings.Contains(out.String(), "[::b]") || strings.Contains(out.String(), "[-:-:-]") {
		t.Error("tview markup leaked into CLI output")
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr should be empty on success, got %q", errOut.String())
	}
}

func TestRunLookupMissStreamDiscipline(t *testing.T) {
	svc := fakeService{entries: map[string]dict.Entity{"time": {Word: "time"}}}
	runner, out, errOut := newRunner()

	code := runner.RunLookup(svc, "zzzzqqq")

	if code != ExitNotFound {
		t.Errorf("exit = %d, want %d", code, ExitNotFound)
	}
	if out.Len() != 0 {
		t.Errorf("stdout must stay empty on miss, got %q", out.String())
	}
	if !strings.Contains(errOut.String(), `No results for "zzzzqqq"`) ||
		!strings.Contains(errOut.String(), "termdict download") {
		t.Errorf("stderr message incomplete: %q", errOut.String())
	}
}

func TestRunLookupEmptyDefinitionEntity(t *testing.T) {
	runner, out, _ := newRunner()
	code := runner.RunLookup(fakeService{entries: map[string]dict.Entity{"void": {Word: "void"}}}, "void")
	if code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if got := out.String(); got != "void\n" {
		t.Errorf("entity without definitions should print just the headword, got %q", got)
	}
}

func TestRunUpdate(t *testing.T) {
	t.Run("already up to date", func(t *testing.T) {
		updater := &fakeUpdater{}
		runner, out, errOut := newRunner()
		if code := runner.RunUpdate(context.Background(), updater); code != ExitOK {
			t.Fatalf("exit = %d", code)
		}
		if out.String() != "Already up to date.\n" {
			t.Errorf("stdout = %q", out.String())
		}
		if errOut.Len() == 0 {
			t.Error("progress diagnostics expected on stderr")
		}
	})

	t.Run("updated files", func(t *testing.T) {
		updater := &fakeUpdater{updateResult: 3}
		runner, out, _ := newRunner()
		if code := runner.RunUpdate(context.Background(), updater); code != ExitOK {
			t.Fatalf("exit = %d", code)
		}
		if out.String() != "Updated 3 file(s).\n" {
			t.Errorf("stdout = %q", out.String())
		}
	})

	t.Run("failure exit code", func(t *testing.T) {
		updater := &fakeUpdater{err: errors.New("network unreachable")}
		runner, out, errOut := newRunner()
		if code := runner.RunUpdate(context.Background(), updater); code != ExitRuntime {
			t.Fatalf("exit = %d, want %d", code, ExitRuntime)
		}
		if out.Len() != 0 {
			t.Errorf("stdout must stay empty on failure, got %q", out.String())
		}
		if !strings.Contains(errOut.String(), "network unreachable") {
			t.Errorf("stderr missing cause: %q", errOut.String())
		}
	})
}

func TestRunDownload(t *testing.T) {
	updater := &fakeUpdater{updateResult: 26}
	runner, out, _ := newRunner()
	if code := runner.RunDownload(context.Background(), updater); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if out.String() != "Downloaded 26 file(s).\n" {
		t.Errorf("stdout = %q", out.String())
	}
}

func TestUsageAndVersionLine(t *testing.T) {
	usage := Usage()
	for _, want := range []string{
		"termdict <word>", "termdict update", "termdict download",
		"-h, --help", "-v, --version", "Exit codes:", "XDG_DATA_HOME",
	} {
		if !strings.Contains(usage, want) {
			t.Errorf("usage missing %q", want)
		}
	}

	if v := VersionLine("", ""); !strings.HasPrefix(v, "termdict dev (go") {
		t.Errorf("VersionLine(\"\") = %q", v)
	}
	if v := VersionLine("v0.2.0", "abc1234"); v != fmt.Sprintf("termdict v0.2.0 (commit abc1234) (%s)\n", runtime.Version()) {
		t.Errorf("VersionLine(v0.2.0, abc1234) = %q", v)
	}
	if v := VersionLine("dev", ""); strings.Contains(v, "commit") {
		t.Errorf("empty commit must be omitted: %q", v)
	}
}

func TestRunLookupSuggestsAlternatives(t *testing.T) {
	svc := fakeService{
		entries: map[string]dict.Entity{"receive": {Word: "receive"}},
		fuzzy:   map[string][]string{"receve": {"receive", "recieve"}}, //nolint:misspell // deliberate misspelling
	}
	runner, out, errOut := newRunner()

	code := runner.RunLookup(svc, "receve")

	if code != ExitNotFound {
		t.Fatalf("exit = %d, want %d", code, ExitNotFound)
	}
	if out.Len() != 0 {
		t.Errorf("stdout must stay empty, got %q", out.String())
	}
	if !strings.Contains(errOut.String(), "Did you mean:") ||
		!strings.Contains(errOut.String(), "receive") {
		t.Errorf("stderr missing did-you-mean hint: %q", errOut.String())
	}
}
