package dict

import (
	"slices"
	"strings"
)

// CountPrefix returns the total number of headwords beginning with
// prefix (case-insensitive). Used to report truncation when the
// suggestion list is capped; an empty prefix returns 0.
func (s *Service) CountPrefix(prefix string) int {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return 0
	}

	idx, _ := slices.BinarySearch(s.words, prefix)
	count := 0
	for i := idx; i < len(s.words); i++ {
		if strings.HasPrefix(s.words[i], prefix) {
			count++
			continue
		}
		if s.words[i] > prefix {
			break
		}
	}
	return count
}

// Suggest returns up to max words beginning with prefix, in
// lexicographic order. Matching is case-insensitive. An empty prefix or
// a non-positive max returns nil.
//
// NOTE: returning nothing for an empty input is a deliberate change from
// v0.1.0 (which listed the first max words of the dictionary); it will be
// recorded in the changelog during the docs phase.
func (s *Service) Suggest(prefix string, limit int) []string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" || limit <= 0 {
		return nil
	}

	wordsLen := len(s.words)
	idx, _ := slices.BinarySearch(s.words, prefix)

	suggestions := make([]string, 0, limit)
	for i := idx; i < wordsLen && len(suggestions) < limit; i++ {
		word := s.words[i]
		if strings.HasPrefix(word, prefix) {
			suggestions = append(suggestions, word)
			continue
		}
		if word > prefix {
			break
		}
	}
	return suggestions
}
