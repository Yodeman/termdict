package dict

import (
	"testing"
)

func testFuzzyService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewMulti(fakeStore{name: "fuzzy", entries: map[string]Entity{
		"receive":  {Word: "receive"},
		"recieve":  {Word: "recieve"}, //nolint:misspell // deliberate misspelling fixture
		"deceive":  {Word: "deceive"},
		"perceive": {Word: "perceive"},
		"relieve":  {Word: "relieve"},
		"retrieve": {Word: "retrieve"},
		"cat":      {Word: "cat"},
		"dog":      {Word: "dog"},
		"house":    {Word: "house"},
		"hose":     {Word: "hose"},
		"horse":    {Word: "horse"},
	}})
	if err != nil {
		t.Fatalf("NewMulti: %v", err)
	}
	return svc
}

func TestFuzzyBasicSuggestions(t *testing.T) {
	svc := testFuzzyService(t)

	got := svc.Fuzzy("receve", 5)
	if len(got) == 0 {
		t.Fatal("expected suggestions for 'receve'")
	}
	if got[0] != "receive" {
		t.Errorf("top suggestion = %q, want receive", got[0])
	}
	for _, word := range got {
		if word == "receve" {
			t.Error("query itself must not be suggested")
		}
	}
}

func TestFuzzyExactHitReturnsNil(t *testing.T) {
	svc := testFuzzyService(t)
	if got := svc.Fuzzy("receive", 5); got != nil {
		t.Errorf("exact hit should return nil, got %v", got)
	}
	if got := svc.Fuzzy("RECEIVE", 5); got != nil {
		t.Errorf("case-insensitive exact hit should return nil, got %v", got)
	}
}

func TestFuzzyEmptyAndDegenerate(t *testing.T) {
	svc := testFuzzyService(t)
	cases := []struct {
		name  string
		query string
		max   int
	}{
		{"empty query", "", 5},
		{"whitespace query", "   ", 5},
		{"zero max", "receve", 0},
		{"negative max", "receve", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.Fuzzy(tc.query, tc.max); got != nil {
				t.Errorf("Fuzzy(%q,%d) = %v, want nil", tc.query, tc.max, got)
			}
		})
	}
}

func TestFuzzyRespectsMax(t *testing.T) {
	svc := testFuzzyService(t)
	if got := svc.Fuzzy("receve", 2); len(got) > 2 {
		t.Errorf("max=2 yielded %d suggestions: %v", len(got), got)
	}
}

func TestFuzzyOrderingDistanceThenAlpha(t *testing.T) {
	svc := testFuzzyService(t)
	got := svc.Fuzzy("hous", 5)
	if len(got) != 1 || got[0] != "house" {
		// budget for a 4-letter query is 1; only "house" is within it.
		t.Fatalf("Fuzzy(hous) = %v, want [house]", got)
	}
}

func TestFuzzyAlphaTieBreak(t *testing.T) {
	svc, err := NewMulti(fakeStore{name: "tie", entries: map[string]Entity{
		"aac": {Word: "aac"},
		"aaa": {Word: "aaa"},
		"aab": {Word: "aab"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := svc.Fuzzy("aad", 5) // every entry is distance 1
	want := []string{"aaa", "aab", "aac"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q (alphabetical tie-break)", i, got[i], want[i])
		}
	}
}

func TestBoundedEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		cap  int
		want int // cap+1 means "exceeds budget"
	}{
		{"kitten", "sitting", 3, 3}, // classic distance-3 pair, exactly at cap
		{"kitten", "sitting", 2, 3}, // exceeds cap 2
		{"abc", "abc", 1, 0},
		{"abc", "abd", 1, 1},
		{"abc", "xyz", 1, 2}, // distance 3 > cap 1
		{"a", "abcdef", 1, 2},
		{"", "", 1, 0},
	}

	for _, tc := range cases {
		if got := boundedEditDistance(tc.a, tc.b, tc.cap); got != tc.want {
			t.Errorf("boundedEditDistance(%q,%q,%d) = %d, want %d",
				tc.a, tc.b, tc.cap, got, tc.want)
		}
	}
}

func TestFuzzyUnicodeQuery(t *testing.T) {
	svc, err := NewMulti(fakeStore{name: "uni", entries: map[string]Entity{
		"café": {Word: "café"},
		"cafe": {Word: "cafe"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got := svc.Fuzzy("cafeé", 5)
	if len(got) == 0 || (!contains(got, "café") && !contains(got, "cafe")) {
		t.Errorf("unicode query should match rune-wise, got %v", got)
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
