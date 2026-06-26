package arkhamdb

import (
	"path/filepath"
	"testing"
)

func TestCollectionRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "collection.json")

	cfg := &CollectionConfig{
		OwnedCycles: []string{"core", "dwl", "ptc"},
		Language:    "es",
		UseTaboo:    false,
	}

	if err := saveCollection(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := loadCollection(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got.Language != "es" {
		t.Errorf("language: got %q, want %q", got.Language, "es")
	}
	if len(got.OwnedCycles) != 3 {
		t.Errorf("cycles: got %d, want 3", len(got.OwnedCycles))
	}
}

func TestLoadCollectionMissing(t *testing.T) {
	cfg, err := loadCollection("/nonexistent/path/collection.json")
	if err != nil {
		t.Fatalf("missing file should return defaults, got error: %v", err)
	}
	if cfg.Language != "en" {
		t.Errorf("default language should be 'en', got %q", cfg.Language)
	}
}
