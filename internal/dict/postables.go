package dict

import (
	"strings"
)

// The words database stores part-of-speech tags as Webster's 1913
// abbreviations ("n.", "v. t.", "p. pr. & vb. n.", ...). This file
// expands them into full, self-explanatory labels for the definition
// headers.
//
// Resolution order in POSLabel:
//  1. normalize the tag (case, spacing, "&."/".." artifacts)
//  2. exact phrase table (covers compound tags as one unit, in the
//     source's own order)
//  3. component split on "&", "," and "/": expand each component and
//     join; unknown/ambiguous components abort the expansion
//  4. passthrough: the abbreviation is returned unchanged and ok=false
//     — a wrong expansion is worse than the abbreviation it replaces
//     (flagged tags: "p. a.", "n. i.", "n. t.", "b. t." and the data-
//     corruption fragments noted in the QA report).

// posPhrases maps normalized full tags to labels. Keys are the output
// of normalizePOS.
var posPhrases = map[string]string{
	"n.":                "Noun",
	"n":                 "Noun",
	"a":                 "Adjective",
	"v":                 "Verb",
	"n. pl.":            "Noun (plural)",
	"n.pl.":             "Noun (plural)",
	"n pl":              "Noun (plural)",
	"n. pl":             "Noun (plural)",
	"n. sing.":          "Noun (singular)",
	"n. sing. & pl.":    "Noun (singular & plural)",
	"p. pr & vb. n.":    "Present participle & verbal noun",
	"p pr. & vb. n.":    "Present participle & verbal noun",
	"n.sing. & pl.":     "Noun (singular & plural)",
	"n. fem.":           "Noun (feminine)",
	"n. f.":             "Noun (feminine)",
	"n. m.":             "Noun (masculine)",
	"n. m":              "Noun (masculine)",
	"n. collect. & pl.": "Collective noun (plural)",

	"a.":   "Adjective",
	"adj.": "Adjective",
	"adj":  "Adjective",

	"v.":                        "Verb",
	"v. t.":                     "Verb (transitive)",
	"v.t.":                      "Verb (transitive)",
	"v. i.":                     "Verb (intransitive)",
	"v.i.":                      "Verb (intransitive)",
	"v. t. & i.":                "Verb (transitive & intransitive)",
	"v. i. & t.":                "Verb (transitive & intransitive)",
	"v. t. & n.":                "Transitive verb & noun",
	"n. & v. t.":                "Noun & transitive verb",
	"n. & v. i.":                "Noun & intransitive verb",
	"v. i. & n.":                "Intransitive verb & noun",
	"imp., p. p., or auxiliary": "Imperative, past participle, or auxiliary",
	"v. i. & auxiliary":         "Verb (intransitive) & auxiliary",
	"v. t. / auxiliary":         "Verb (transitive) & auxiliary",

	"adv.":           "Adverb",
	"adv":            "Adverb",
	"a. & adv.":      "Adjective & adverb",
	"adv. & a.":      "Adverb & adjective",
	"adv. & conj.":   "Adverb & conjunction",
	"adv. & n.":      "Adverb & noun",
	"adv. & prep.":   "Adverb & preposition",
	"adv. / interj.": "Adverb & interjection",
	"adv. / conj.":   "Adverb & conjunction",

	"prep.":        "Preposition",
	"prep":         "Preposition",
	"prep. & adv.": "Preposition & adverb",

	"conj.": "Conjunction",
	"conj":  "Conjunction",

	"interj.":      "Interjection",
	"interj":       "Interjection",
	"interj. & n.": "Interjection & noun",

	"pron.":                    "Pronoun",
	"pron":                     "Pronoun",
	"pron. pl.":                "Pronoun (plural)",
	"a. & pron.":               "Adjective & pronoun",
	"pron. & a.":               "Pronoun & adjective",
	"pron., a., & adv.":        "Pronoun, adjective & adverb",
	"pron., a., conj., & adv.": "Pronoun, adjective, conjunction & adverb",

	"p. p.":       "Past participle",
	"p.p.":        "Past participle",
	"p. pr.":      "Present participle",
	"p. p. & a.":  "Past participle & adjective",
	"p. p. / a.":  "Past participle & adjective",
	"p. p. & a":   "Past participle & adjective",
	"p. p & a.":   "Past participle & adjective",
	"p. pr. & a.": "Present participle & adjective",

	"imp.":          "Imperative",
	"imp. & p. p.":  "Imperative & past participle",
	"imp. p. p.":    "Imperative & past participle",
	"imp. &. p. p.": "Imperative & past participle",
	"imp. &  p. p.": "Imperative & past participle",
	"obs. imp.":     "Obsolete imperative",
	"obs. p. p.":    "Obsolete past participle",

	"vb. n.":           "Verbal noun",
	"vb.n.":            "Verbal noun",
	"p. pr. & vb. n.":  "Present participle & verbal noun",
	"p. pr  & vb. n.":  "Present participle & verbal noun",
	"p. pr. &  vb. n.": "Present participle & verbal noun",
	"p. pr. vb. n.":    "Present participle & verbal noun",
	"p. pr. &. vb. n.": "Present participle & verbal noun",
	"p. pr. & vb. n":   "Present participle & verbal noun",

	"pl.":      "Plural",
	"pl":       "Plural",
	"sing.":    "Singular",
	"superl.":  "Superlative",
	"superl":   "Superlative",
	"supperl.": "Superlative",
	"compar.":  "Comparative",
	"compar":   "Comparative",

	"pref.":     "Prefix",
	"obs.":      "Obsolete",
	"obj.":      "Objective",
	"m.":        "Masculine",
	"f.":        "Feminine",
	"auxiliary": "Auxiliary verb",
	"collect.":  "Collective",

	"a. & n.":              "Adjective & noun",
	"n. & a.":              "Noun & adjective",
	"n. & v.":              "Noun & verb",
	"v. & n.":              "Verb & noun",
	"v. & a.":              "Verb & adjective",
	"n. & adv.":            "Noun & adverb",
	"n. & interj.":         "Noun & interjection",
	"3d pers. sing. pres.": "Third person singular present",
}

// posComponents expands single abbreviations inside compound tags.
var posComponents = map[string]string{
	"n.": "Noun", "n": "Noun",
	"a.": "Adjective", "a": "Adjective", "adj.": "Adjective", "adj": "Adjective",
	"v.": "Verb", "v": "Verb",
	"v. t.": "Transitive verb", "v.t.": "Transitive verb",
	"v. i.": "Intransitive verb", "v.i.": "Intransitive verb",
	"adv.": "Adverb", "adv": "Adverb",
	"prep.": "Preposition", "prep": "Preposition",
	"pron.": "Pronoun", "pron": "Pronoun",
	"conj.": "Conjunction", "conj": "Conjunction",
	"interj.": "Interjection", "interj": "Interjection",
	"pl.": "Plural", "pl": "Plural",
	"sing.":   "Singular",
	"superl.": "Superlative", "superl": "Superlative",
	"compar.": "Comparative", "compar": "Comparative",
	"imp.":  "Imperative",
	"p. p.": "Past participle", "p.p.": "Past participle",
	"p. pr.": "Present participle", "p.pr.": "Present participle",
	"vb. n.": "Verbal noun",
	"pref.":  "Prefix",
	"obs.":   "Obsolete",
	"obj.":   "Objective",
	"m.":     "Masculine", "f.": "Feminine",
	"auxiliary": "Auxiliary verb",
	"collect.":  "Collective",
}

// normalizePOS canonicalizes a tag: lowercase, "&." -> "&", a space
// before a period removed, duplicated periods collapsed, whitespace
// collapsed. Token spelling like "v.t." or "3d" is preserved verbatim.
func normalizePOS(abbr string) string {
	s := strings.ToLower(strings.TrimSpace(abbr))
	s = strings.ReplaceAll(s, "&.", "&")
	s = strings.ReplaceAll(s, " .", ".")
	for strings.Contains(s, "..") {
		s = strings.ReplaceAll(s, "..", ".")
	}
	return strings.Join(strings.Fields(s), " ")
}

// POSLabel expands a part-of-speech abbreviation into a full label
// ("n." -> "Noun", "p. pr. & vb. n." -> "Present participle &
// verbal noun"). ok is false for unknown or genuinely ambiguous tags;
// label then holds the original abbreviation unchanged.
func POSLabel(abbr string) (string, bool) {
	normalized := normalizePOS(abbr)
	if normalized == "" {
		return "", false
	}
	if label, ok := posPhrases[normalized]; ok {
		return label, true
	}

	// Component split: every component must be known, else passthrough.
	components := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '&' || r == ',' || r == '/'
	})
	if len(components) > 1 {
		labels := make([]string, 0, len(components))
		for _, component := range components {
			component = strings.TrimSpace(component)
			label, ok := posComponents[component]
			if !ok {
				return abbr, false
			}
			labels = append(labels, label)
		}
		return strings.Join(labels, " & "), true
	}
	return abbr, false
}

// POSGroup holds the senses of one entity that share an original
// part-of-speech tag.
type POSGroup struct {
	POS    string // the original tag, verbatim from the source
	Senses []Definition
}

// GroupByPOS regroups an entity's senses by part of speech, preserving
// both the source's group order (first appearance) and the sense order
// within each group. Nothing is reordered beyond the grouping.
func GroupByPOS(entity Entity) []POSGroup {
	var groups []POSGroup
	index := map[string]int{}
	for _, def := range entity.WordDefinitions {
		pos := strings.TrimSpace(def.PartOfSpeech)
		i, seen := index[pos]
		if !seen {
			groups = append(groups, POSGroup{POS: pos})
			i = len(groups) - 1
			index[pos] = i
		}
		groups[i].Senses = append(groups[i].Senses, def)
	}
	return groups
}
