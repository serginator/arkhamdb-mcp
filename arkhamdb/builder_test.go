package arkhamdb

import "testing"

func TestBuildAllowedPackCodes_ChapterFilter(t *testing.T) {
	packs := []map[string]interface{}{
		{"code": "core", "chapter": float64(1)},
		{"code": "dwl1", "chapter": float64(1)},
		{"code": "core_2026", "chapter": float64(2)},
	}
	allowed := buildAllowedPackCodes(packs, 2, nil)
	if !allowed["core_2026"] {
		t.Error("core_2026 should be allowed for chapter 2")
	}
	if allowed["core"] {
		t.Error("core should not be allowed for chapter 2")
	}
}

func TestBuildAllowedPackCodes_CycleFilter(t *testing.T) {
	packs := []map[string]interface{}{
		{"code": "core", "chapter": float64(1)},
		{"code": "dwl1", "chapter": float64(1)},
		{"code": "dwl2", "chapter": float64(1)},
	}
	allowed := buildAllowedPackCodes(packs, 0, []string{"dwl"})
	if !allowed["dwl1"] {
		t.Error("dwl1 should be allowed by 'dwl' cycle prefix")
	}
	if allowed["core"] {
		t.Error("core should not be allowed by 'dwl' cycle filter")
	}
}

func TestFirstMatchingOption(t *testing.T) {
	opts := []DeckOption{
		{Faction: []string{"guardian"}, Level: &LevelRange{Min: 0, Max: 5}},
		{Faction: []string{"neutral"}, Level: &LevelRange{Min: 0, Max: 5}},
	}
	neutralCard := map[string]interface{}{
		"faction_code": "neutral",
		"xp":           float64(0),
		"type_code":    "asset",
	}
	idx := firstMatchingOption(neutralCard, opts, nil)
	if idx != 1 {
		t.Errorf("expected option index 1 for neutral card, got %d", idx)
	}
}
