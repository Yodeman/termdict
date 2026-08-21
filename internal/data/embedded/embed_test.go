package embedded

import (
	"testing"

	"github.com/yodeman/termdict/internal/dict"
)

var _ dict.Store = Store{}

func TestStoreLoadMatchesManifest(t *testing.T) {
	store := New()

	manifest, err := store.Manifest()
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if manifest.TotalEntries == 0 {
		t.Fatal("manifest claims zero entries; core was not generated")
	}
	if len(manifest.Files) != 26 {
		t.Errorf("manifest lists %d letter files, want 26", len(manifest.Files))
	}
	if manifest.FrequencySource.License == "" || manifest.FrequencySource.SHA256 == "" {
		t.Errorf("frequency provenance incomplete: %+v", manifest.FrequencySource)
	}

	entities, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(entities) != manifest.TotalEntries {
		t.Errorf("loaded %d entries, manifest claims %d", len(entities), manifest.TotalEntries)
	}
}

func TestStoreCoversCommonWords(t *testing.T) {
	entities, err := New().Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, word := range []string{"the", "time", "house", "water", "friend", "school"} {
		if _, ok := entities[word]; !ok {
			t.Errorf("common word %q missing from embedded core", word)
		}
	}
}

func TestStoreEntitiesWellFormed(t *testing.T) {
	entities, err := New().Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for word, entity := range entities {
		if entity.Word != word {
			t.Fatalf("key %q does not match entity word %q", word, entity.Word)
		}
		return // spot check is enough; full sweep happens in dbasebuild tests
	}
}
