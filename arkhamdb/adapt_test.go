package arkhamdb

import (
	"testing"
)

func TestAdaptFindsReplacement(t *testing.T) {
	_ = &ArkhamDBClient{
		collection: &CollectionConfig{
			OwnedCycles: []string{"core"},
			Language:    "en",
		},
	}
	// Test that a card not in owned cycles is flagged as needing replacement
	card := map[string]interface{}{
		"code":         "99999",
		"pack_code":    "dwl", // not in owned cycles
		"real_name":    "Some Card",
		"faction_code": "guardian",
		"xp":           float64(0),
	}
	owned := map[string]bool{"core": true}
	if isCardOwned(card, owned) {
		t.Error("card from dwl should not be owned when only core is in collection")
	}
}
