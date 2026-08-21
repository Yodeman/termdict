package dict

import (
	"strings"
	"testing"
)

type fakeStore struct {
	name    string
	entries map[string]Entity
	err     error
}

func (s fakeStore) Name() string { return s.name }

func (s fakeStore) Load() (map[string]Entity, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.entries, nil
}

func testService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewMulti(fakeStore{name: "test", entries: map[string]Entity{
		"apple": {Word: "apple", WordDefinitions: []Definition{
			{PartOfSpeech: "n.", WordDefinition: "A round fruit."},
		}},
		"apply":   {Word: "apply"},
		"applied": {Word: "applied"},
		"banana":  {Word: "banana"},
		"zebra":   {Word: "zebra"},
	}})
	if err != nil {
		t.Fatalf("NewMulti: %v", err)
	}
	return svc
}

func TestNewMultiMergePrecedence(t *testing.T) {
	first := fakeStore{name: "first", entries: map[string]Entity{
		"apple": {Word: "apple", WordDefinitions: []Definition{{WordDefinition: "stale"}}},
		"kiwi":  {Word: "kiwi"},
	}}
	second := fakeStore{name: "second", entries: map[string]Entity{
		"apple": {Word: "apple", WordDefinitions: []Definition{{WordDefinition: "fresh"}}},
	}}

	svc, err := NewMulti(first, second)
	if err != nil {
		t.Fatalf("NewMulti: %v", err)
	}

	got, ok := svc.Lookup("apple")
	if !ok || len(got.WordDefinitions) != 1 || got.WordDefinitions[0].WordDefinition != "fresh" {
		t.Errorf("later store should win on collision, got %+v (found=%v)", got, ok)
	}
	if _, ok := svc.Lookup("kiwi"); !ok {
		t.Error("entry from earlier store should survive")
	}
	if svc.Len() != 2 {
		t.Errorf("Len = %d, want 2", svc.Len())
	}
}

func TestNewMultiStoreError(t *testing.T) {
	_, err := NewMulti(fakeStore{name: "broken", err: errTest{}})
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Errorf("expected store name in error, got %v", err)
	}
}

type errTest struct{}

func (errTest) Error() string { return "boom" }

func TestLookup(t *testing.T) {
	svc := testService(t)

	cases := []struct {
		query string
		want  bool
	}{
		{"apple", true},
		{"Apple", true},    // case-insensitive
		{"APPLE", true},    // case-insensitive
		{"  apple ", true}, // trimmed
		{"appl", false},    // no prefix matching in Lookup
		{"missing", false},
		{"", false},
	}
	for _, tc := range cases {
		if _, ok := svc.Lookup(tc.query); ok != tc.want {
			t.Errorf("Lookup(%q) found=%v, want %v", tc.query, ok, tc.want)
		}
	}
}

func TestWordsSorted(t *testing.T) {
	svc := testService(t)
	words := svc.Words()
	want := []string{"apple", "applied", "apply", "banana", "zebra"}
	if len(words) != len(want) {
		t.Fatalf("Words = %v, want %v", words, want)
	}
	for i := range want {
		if words[i] != want[i] {
			t.Errorf("Words[%d] = %q, want %q", i, words[i], want[i])
		}
	}
}

func TestSuggest(t *testing.T) {
	svc := testService(t)

	cases := []struct {
		name   string
		prefix string
		max    int
		want   []string
	}{
		{"empty prefix returns nothing", "", 50, nil},
		{"uppercase prefix is normalized", "AP", 50, []string{"apple", "applied", "apply"}},
		{"exact word included", "apple", 50, []string{"apple"}},
		{"max cutoff", "app", 2, []string{"apple", "applied"}},
		{"no match", "zzz", 50, nil},
		{"zero max", "a", 0, nil},
		{"negative max", "a", -1, nil},
		{"prefix beyond last word", "z", 50, []string{"zebra"}},
		{"prefix after last word", "zzzz", 50, nil},
		{"whitespace trimmed", " ap", 50, []string{"apple", "applied", "apply"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.Suggest(tc.prefix, tc.max)
			if len(got) != len(tc.want) {
				t.Fatalf("Suggest(%q,%d) = %v, want %v", tc.prefix, tc.max, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("Suggest(%q,%d)[%d] = %q, want %q",
						tc.prefix, tc.max, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestSuggestCutoffAtMax(t *testing.T) {
	entries := map[string]Entity{}
	for _, r := range "abcdefghijklmnop" {
		entries[string(r)] = Entity{Word: string(r)}
	}
	svc, err := NewMulti(fakeStore{name: "bulk", entries: entries})
	if err != nil {
		t.Fatalf("NewMulti: %v", err)
	}

	got := svc.Suggest("", 5)
	if got != nil {
		t.Errorf("empty prefix should return nil, got %v", got)
	}
	got = svc.Suggest("a", 5)
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("Suggest(a,5) = %v", got)
	}
}
