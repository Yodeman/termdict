package dict

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestPOSLabel(t *testing.T) {
	cases := []struct {
		abbr string
		want string
		ok   bool
	}{
		// core single tags
		{"n.", "Noun", true},
		{"a.", "Adjective", true},
		{"v. t.", "Verb (transitive)", true},
		{"v. i.", "Verb (intransitive)", true},
		{"adv.", "Adverb", true},
		{"prep.", "Preposition", true},
		{"conj.", "Conjunction", true},
		{"interj.", "Interjection", true},
		{"pron.", "Pronoun", true},
		{"p. p.", "Past participle", true},
		{"imp.", "Imperative", true},
		{"pl.", "Plural", true},
		{"superl.", "Superlative", true},

		// normalization variants found in the data
		{"n..", "Noun", true},
		{"n .", "Noun", true},
		{"n", "Noun", true},
		{"N.", "Noun", true},
		{"v.t.", "Verb (transitive)", true},
		{"v. t. ", "Verb (transitive)", true},
		{"Compar.", "Comparative", true},
		{"supperl.", "Superlative", true},
		{"n.pl.", "Noun (plural)", true},
		{"n. pl", "Noun (plural)", true},
		{"p. pr.  & vb. n.", "Present participle & verbal noun", true},
		{"imp. &. p. p.", "Imperative & past participle", true},

		// compound phrases keep source order
		{"imp. & p. p.", "Imperative & past participle", true},
		{"p. pr. & vb. n.", "Present participle & verbal noun", true},
		{"a. & n.", "Adjective & noun", true},
		{"n. & v. t.", "Noun & transitive verb", true},
		{"v. t. / auxiliary", "Verb (transitive) & auxiliary", true},
		{"pron., a., conj., & adv.", "Pronoun, adjective, conjunction & adverb", true},
		{"3d pers. sing. pres.", "Third person singular present", true},

		// unknown or ambiguous: passthrough, not guesses
		{"p. a.", "p. a.", false},
		{"p.a.", "p.a.", false},
		{"n. i.", "n. i.", false},
		{"n. t.", "n. t.", false},
		{"b. t.", "b. t.", false},
		{"", "", false},
		{"ambassade.", "ambassade.", false},
	}
	for _, tc := range cases {
		t.Run(tc.abbr, func(t *testing.T) {
			label, ok := POSLabel(tc.abbr)
			if ok != tc.ok || label != tc.want {
				t.Errorf("POSLabel(%q) = (%q, %v), want (%q, %v)",
					tc.abbr, label, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestGroupByPOS(t *testing.T) {
	entity := Entity{Word: "subject", WordDefinitions: []Definition{
		{PartOfSpeech: "a.", WordDefinition: "sense a1"},
		{PartOfSpeech: "a.", WordDefinition: "sense a2"},
		{PartOfSpeech: "n.", WordDefinition: "sense n1"},
		{PartOfSpeech: "a.", WordDefinition: "sense a3"}, // interleaved: stays in the a. group
		{PartOfSpeech: "v. t.", WordDefinition: "sense vt1"},
		{PartOfSpeech: "", WordDefinition: "sense unlabeled"},
	}}

	groups := GroupByPOS(entity)
	if len(groups) != 4 {
		t.Fatalf("got %d groups, want 4: %+v", len(groups), groups)
	}
	wantOrder := []string{"a.", "n.", "v. t.", ""}
	for i, want := range wantOrder {
		if groups[i].POS != want {
			t.Errorf("group[%d].POS = %q, want %q (source order)", i, groups[i].POS, want)
		}
	}
	if len(groups[0].Senses) != 3 || groups[0].Senses[0].WordDefinition != "sense a1" ||
		groups[0].Senses[2].WordDefinition != "sense a3" {
		t.Errorf("adjective group lost order/content: %+v", groups[0].Senses)
	}
	if len(groups[1].Senses) != 1 || len(groups[2].Senses) != 1 {
		t.Errorf("unexpected group sizes: %+v", groups)
	}
}

func TestGroupByPOSEmpty(t *testing.T) {
	if groups := GroupByPOS(Entity{Word: "x"}); len(groups) != 0 {
		t.Errorf("empty entity should yield no groups, got %d", len(groups))
	}
}

// flaggedAmbiguous lists tags whose expansion is genuinely ambiguous;
// they are exempt from the must-map threshold (rendered as the
// original abbreviation) and await maintainer confirmation. Do not add
// mappings here without verifying against the source.
var flaggedAmbiguous = map[string]string{
	"p. a.": "likely participial adjective, unconfirmed",
	"p.a.":  "likely participial adjective, unconfirmed",
	"n. i.": "unknown (noun intransitive?), unconfirmed",
	"b. t.": "unknown, unconfirmed",
}

// TestDataCoverage guards the POS label table against the shipped
// database: at least 99% of all senses with a part-of-speech tag must
// resolve to a full label, and every tag with more than a handful of
// senses must be mapped unless it is flagged ambiguous. Skips when the
// database files are not present.
func TestDataCoverage(t *testing.T) {
	dir := filepath.Join("..", "..", "word_dbase", "json")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("word database not present")
	}

	files, err := filepath.Glob(filepath.Join(dir, "wb1913_*.json"))
	if err != nil || len(files) == 0 {
		t.Skip("no letter files found")
	}

	total, unmapped := 0, map[string]int{}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading %s: %v", file, err)
		}
		var batch map[string]struct {
			Definitions []Definition `json:"definitions"`
		}
		if err := json.Unmarshal(raw, &batch); err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		for _, entity := range batch {
			for _, def := range entity.Definitions {
				pos := strings.TrimSpace(def.PartOfSpeech)
				if pos == "" {
					continue // unlabeled senses render without a header
				}
				total++
				if _, ok := POSLabel(pos); !ok {
					unmapped[pos]++
				}
			}
		}
	}
	if total == 0 {
		t.Skip("no senses found")
	}

	// Tags with more than this many senses must be mapped; rarer ones
	// may pass through as abbreviations pending confirmation.
	const mustMapThreshold = 10
	var flagged []string
	unmappedSenses := 0
	for tag, count := range unmapped {
		unmappedSenses += count
		_, knownAmbiguous := flaggedAmbiguous[tag]
		if count > mustMapThreshold && !knownAmbiguous {
			flagged = append(flagged, fmt.Sprintf("%q(%d)", tag, count))
		}
	}
	sort.Slice(flagged, func(i, j int) bool { return flagged[i] < flagged[j] })
	if len(flagged) > 0 {
		t.Errorf("frequent part-of-speech tags lack labels: %s", strings.Join(flagged, ", "))
	}

	ratio := 100 * float64(total-unmappedSenses) / float64(total)
	t.Logf("POS label coverage: %.2f%% of %d senses (%d unmapped across %d tags)",
		ratio, total, unmappedSenses, len(unmapped))
	if ratio < 99.0 {
		t.Errorf("POS label coverage %.2f%% below 99%%", ratio)
	}
}
