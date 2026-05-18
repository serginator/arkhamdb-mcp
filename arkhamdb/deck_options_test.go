package arkhamdb

import "testing"

func TestParseDeckOptions_StandardGuardian(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"faction": []interface{}{"guardian"},
			"level":   map[string]interface{}{"min": float64(0), "max": float64(5)},
		},
		map[string]interface{}{
			"faction": []interface{}{"neutral"},
			"level":   map[string]interface{}{"min": float64(0), "max": float64(5)},
		},
	}
	opts := parseDeckOptions(raw)
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
	if opts[0].Faction[0] != "guardian" {
		t.Errorf("expected guardian, got %s", opts[0].Faction[0])
	}
	if opts[0].Level.Max != 5 {
		t.Errorf("expected max 5, got %d", opts[0].Level.Max)
	}
}

func TestIsCardAllowedByOptions_GuardianCard(t *testing.T) {
	opts := []DeckOption{
		{Faction: []string{"guardian"}, Level: &LevelRange{Min: 0, Max: 5}},
		{Faction: []string{"neutral"}, Level: &LevelRange{Min: 0, Max: 5}},
	}
	guardianCard := map[string]interface{}{
		"faction_code": "guardian",
		"xp":           float64(0),
		"type_code":    "asset",
	}
	if !isCardAllowedByOptions(guardianCard, opts) {
		t.Error("guardian level-0 card should be allowed")
	}
}

func TestIsCardAllowedByOptions_RogueCardRejected(t *testing.T) {
	opts := []DeckOption{
		{Faction: []string{"guardian"}, Level: &LevelRange{Min: 0, Max: 5}},
		{Faction: []string{"neutral"}, Level: &LevelRange{Min: 0, Max: 5}},
	}
	rogueCard := map[string]interface{}{
		"faction_code": "rogue",
		"xp":           float64(0),
		"type_code":    "event",
	}
	if isCardAllowedByOptions(rogueCard, opts) {
		t.Error("rogue card should be rejected for a guardian investigator")
	}
}

func TestIsCardAllowedByOptions_LimitedLevel(t *testing.T) {
	opts := []DeckOption{
		{Faction: []string{"guardian"}, Level: &LevelRange{Min: 0, Max: 5}},
		{Faction: []string{"survivor"}, Level: &LevelRange{Min: 0, Max: 0}, Limit: 5},
	}
	survivorL2 := map[string]interface{}{
		"faction_code": "survivor",
		"xp":           float64(2),
		"type_code":    "asset",
	}
	if isCardAllowedByOptions(survivorL2, opts) {
		t.Error("survivor level-2 card should be rejected (limit is level 0 only)")
	}
	survivorL0 := map[string]interface{}{
		"faction_code": "survivor",
		"xp":           float64(0),
		"type_code":    "asset",
	}
	if !isCardAllowedByOptions(survivorL0, opts) {
		t.Error("survivor level-0 card should be allowed")
	}
}

func TestIsCardAllowedByOptions_TraitOption(t *testing.T) {
	opts := []DeckOption{
		{Trait: []string{"Blessed"}, Level: &LevelRange{Min: 0, Max: 5}},
	}
	blessedCard := map[string]interface{}{
		"faction_code": "neutral",
		"xp":           float64(0),
		"type_code":    "event",
		"real_traits":  "Blessed.",
	}
	if !isCardAllowedByOptions(blessedCard, opts) {
		t.Error("Blessed card should be allowed by trait option")
	}
}
