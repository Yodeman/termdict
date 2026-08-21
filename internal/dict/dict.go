// Package dict holds the dictionary data model and the pure lookup,
// suggestion and rendering logic. It deliberately knows nothing about
// tview, HTTP or the filesystem beyond the pluggable Store interface,
// which keeps it fully unit-testable.
package dict

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Entity represents a single entry in the words database.
type Entity struct {
	Word            string       `json:"word"`
	Spellings       []string     `json:"alternate_spellings,omitempty"`
	WordDefinitions []Definition `json:"definitions"`
}

// Definition is one part-of-speech/definition pair of an Entity.
type Definition struct {
	PartOfSpeech   string `json:"part_of_speech"`
	WordDefinition string `json:"definition"`
}

// Store is a named source of dictionary entities (embedded core subset,
// local letter files, future remote caches...). Stores are merged in the
// order given to NewMulti; later stores win on key collisions.
type Store interface {
	Name() string
	Load() (map[string]Entity, error)
}

// Service provides lookups over a merged set of dictionary entities.
type Service struct {
	entities map[string]Entity
	words    []string // lowercase, sorted
}

// NewMulti builds a Service by merging every store in order. Later
// stores override earlier ones on key collisions.
func NewMulti(stores ...Store) (*Service, error) {
	svc := &Service{entities: make(map[string]Entity)}
	for _, store := range stores {
		batch, err := store.Load()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", store.Name(), err)
		}
		maps.Copy(svc.entities, batch)
	}

	svc.words = slices.Sorted(maps.Keys(svc.entities))
	return svc, nil
}

// Lookup returns the entity for word, matching case-insensitively.
func (s *Service) Lookup(word string) (Entity, bool) {
	entity, ok := s.entities[strings.ToLower(strings.TrimSpace(word))]
	return entity, ok
}

// Words returns the sorted list of all known words. The returned slice
// must not be modified.
func (s *Service) Words() []string { return s.words }

// Len returns the number of entries in the service.
func (s *Service) Len() int { return len(s.entities) }
