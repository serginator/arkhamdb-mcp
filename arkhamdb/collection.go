package arkhamdb

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CollectionConfig stores the user's owned content and preferences.
type CollectionConfig struct {
	OwnedCycles []string `json:"ownedCycles"` // e.g. ["core","dwl","ptc"]
	Language    string   `json:"language"`    // "es" or "en" (default "en")
	UseTaboo    bool     `json:"useTaboo"`    // default false
}

func defaultCollection() *CollectionConfig {
	return &CollectionConfig{Language: "en"}
}

func collectionPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".arkhamdb-collection.json")
}

func loadCollection(path string) (*CollectionConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultCollection(), nil
	}
	if err != nil {
		return nil, err
	}
	cfg := defaultCollection()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	return cfg, nil
}

func saveCollection(path string, cfg *CollectionConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetCollection returns the current collection config as JSON.
func (c *ArkhamDBClient) GetCollection() (string, error) {
	data, err := json.MarshalIndent(c.collection, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SetCollection updates and persists the collection config.
func (c *ArkhamDBClient) SetCollection(ownedCycles []string, language string, useTaboo bool) (string, error) {
	if language == "" {
		language = "en"
	}
	c.collection = &CollectionConfig{
		OwnedCycles: ownedCycles,
		Language:    language,
		UseTaboo:    useTaboo,
	}
	if err := saveCollection(collectionPath(), c.collection); err != nil {
		return "", err
	}
	return `{"status":"saved","message":"Collection saved to ~/.arkhamdb-collection.json"}`, nil
}
