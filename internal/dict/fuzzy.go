package dict

import (
	"slices"
	"strings"
)

// MaxFuzzySuggestions caps the did-you-mean list by default.
const MaxFuzzySuggestions = 5

// Fuzzy returns up to maxSuggestions headwords close to query,
// ordered by ascending edit distance then alphabetically. It is meant
// for "did you mean" hints after a lookup miss; an exact hit, an empty
// query or a non-positive max returns nil.
//
// The distance budget is len(query)/3 (minimum 1): longer queries
// tolerate proportionally more edits. Candidates whose length differs
// from the query by more than that budget are skipped without scoring.
func (s *Service) Fuzzy(query string, maxSuggestions int) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || maxSuggestions <= 0 {
		return nil
	}
	if _, exact := s.entities[query]; exact {
		return nil
	}

	budget := len(query) / 3
	if budget < 1 {
		budget = 1
	}

	type candidate struct {
		word     string
		distance int
	}
	best := make([]candidate, 0, maxSuggestions)

	for _, word := range s.words {
		if abs(len(word)-len(query)) > budget {
			continue
		}
		distance := boundedEditDistance(query, word, budget)
		if distance > budget {
			continue
		}
		best = append(best, candidate{word: word, distance: distance})
	}

	slices.SortStableFunc(best, func(a, b candidate) int {
		if a.distance != b.distance {
			return a.distance - b.distance
		}
		return strings.Compare(a.word, b.word)
	})
	if len(best) > maxSuggestions {
		best = best[:maxSuggestions]
	}

	result := make([]string, 0, len(best))
	for _, c := range best {
		result = append(result, c.word)
	}
	return result
}

// boundedEditDistance computes the Levenshtein distance between a and
// b, abandoning early (returning cap+1) as soon as the distance must
// exceed cap. Uses the two-row dynamic-programming formulation.
func boundedEditDistance(a, b string, maxDist int) int {
	ar, br := []rune(a), []rune(b)
	if abs(len(ar)-len(br)) > maxDist {
		return maxDist + 1
	}

	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		rowMin := curr[0]
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
			if curr[j] < rowMin {
				rowMin = curr[j]
			}
		}
		if rowMin > maxDist {
			return maxDist + 1 // every path already exceeds the budget
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
