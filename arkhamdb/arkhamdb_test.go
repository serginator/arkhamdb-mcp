package arkhamdb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testCards provides a controlled set of cards for SearchCardsAdvanced tests
var testCards = []map[string]interface{}{
	// Guardian asset, level 2, cost 3, Ally trait
	{
		"code": "01001", "name": "Guard Dog", "faction_code": "guardian",
		"type_code": "asset", "xp": float64(2), "cost": float64(3),
		"traits": "Ally.", "pack_code": "core",
	},
	// Rogue event, level 0, cost 1
	{
		"code": "01002", "name": "Easy Mark", "faction_code": "rogue",
		"type_code": "event", "xp": float64(0), "cost": float64(1),
		"traits": "Trick.", "pack_code": "core",
	},
	// Investigator — should always be filtered out
	{
		"code": "01003", "name": "Roland Banks", "faction_code": "guardian",
		"type_code": "investigator", "xp": float64(0), "cost": float64(0),
		"traits": "Detective. Agency.", "pack_code": "core",
	},
	// Basic weakness — should always be filtered out
	{
		"code": "01004", "name": "Paranoia", "faction_code": "neutral",
		"type_code": "treachery", "subtype_code": "basicweakness",
		"traits": "Madness.", "pack_code": "core",
	},
	// Guardian skill, level 1, no cost (cost == nil)
	{
		"code": "01005", "name": "Overpower", "faction_code": "guardian",
		"type_code": "skill", "xp": float64(1),
		"traits": "Practiced.", "pack_code": "core",
	},
	// Mystic asset, level 3, cost 4, heals horror tag
	{
		"code": "01006", "name": "Holy Rosary", "faction_code": "mystic",
		"type_code": "asset", "xp": float64(3), "cost": float64(4),
		"traits": "Item. Charm.", "tags": "hh", "pack_code": "eoe1",
	},
}

var testPacks = []map[string]interface{}{
	{"code": "core", "name": "Core Set", "cycle_position": float64(1), "chapter": float64(1), "position": float64(1)},
	{"code": "eoe1", "name": "Edge of the Earth 1", "cycle_position": float64(8), "chapter": float64(1), "position": float64(1)},
}

// resetCaches clears the package-level card and pack caches so each test gets a fresh fetch.
func resetCaches() {
	cardsCache.mu.Lock()
	cardsCache.data = nil
	cardsCache.cachedAt = time.Time{}
	cardsCache.mu.Unlock()

	packsCache.mu.Lock()
	packsCache.data = nil
	packsCache.cachedAt = time.Time{}
	packsCache.mu.Unlock()
}

func newTestArkhamDBClient(cards []map[string]interface{}, packs []map[string]interface{}) (*ArkhamDBClient, *httptest.Server) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/public/cards/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cards)
	})
	mux.HandleFunc("/api/public/packs/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(packs)
	})
	srv := httptest.NewServer(mux)
	client := NewArkhamDBClient(srv.URL)
	return client, srv
}

func TestSearchCardsAdvanced_FactionFilter(t *testing.T) {
	resetCaches()
	client, srv := newTestArkhamDBClient(testCards, testPacks)
	defer srv.Close()

	result, err := client.SearchCardsAdvanced(0, "", "guardian", "", -1, -1, -1, -1, nil, nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	cards := out["cards"].([]interface{})
	for _, c := range cards {
		card := c.(map[string]interface{})
		if card["faction_code"] != "guardian" {
			t.Errorf("expected only guardian cards, got faction_code=%v (card %v)", card["faction_code"], card["code"])
		}
		// investigators and weaknesses must not appear
		if card["type_code"] == "investigator" {
			t.Errorf("investigators should be excluded, got %v", card["code"])
		}
	}
	// Guard Dog (guardian asset) and Overpower (guardian skill) should be present; Roland (investigator) must not
	count := int(out["count"].(float64))
	if count != 2 {
		t.Errorf("expected 2 guardian non-investigator cards, got %d", count)
	}
}

func TestSearchCardsAdvanced_XPRangeFilter(t *testing.T) {
	resetCaches()
	client, srv := newTestArkhamDBClient(testCards, testPacks)
	defer srv.Close()

	// xpMin=1, xpMax=2 — should return Guard Dog (xp=2) and Overpower (xp=1)
	result, err := client.SearchCardsAdvanced(0, "", "", "", 1, 2, -1, -1, nil, nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)
	count := int(out["count"].(float64))
	if count != 2 {
		t.Errorf("expected 2 cards with xp 1-2, got %d", count)
	}
}

func TestSearchCardsAdvanced_CostRangeFilter_ExcludesNilCost(t *testing.T) {
	resetCaches()
	client, srv := newTestArkhamDBClient(testCards, testPacks)
	defer srv.Close()

	// costMin=1, costMax=3 — Overpower has nil cost and should be excluded
	result, err := client.SearchCardsAdvanced(0, "", "", "", -1, -1, 1, 3, nil, nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)

	cards := out["cards"].([]interface{})
	for _, c := range cards {
		card := c.(map[string]interface{})
		if card["code"] == "01005" {
			t.Errorf("Overpower (nil cost) should be excluded when cost filter is active")
		}
	}
	// Easy Mark (cost=1) and Guard Dog (cost=3) should be included
	count := int(out["count"].(float64))
	if count != 2 {
		t.Errorf("expected 2 cards with cost 1-3 (excluding nil-cost), got %d", count)
	}
}

func TestSearchCardsAdvanced_TraitFilter(t *testing.T) {
	resetCaches()
	client, srv := newTestArkhamDBClient(testCards, testPacks)
	defer srv.Close()

	result, err := client.SearchCardsAdvanced(0, "", "", "", -1, -1, -1, -1, []string{"Ally"}, nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)
	count := int(out["count"].(float64))
	if count != 1 {
		t.Errorf("expected 1 Ally card (Guard Dog), got %d", count)
	}
}

func TestSearchCardsAdvanced_TagFilter(t *testing.T) {
	resetCaches()
	client, srv := newTestArkhamDBClient(testCards, testPacks)
	defer srv.Close()

	result, err := client.SearchCardsAdvanced(0, "", "", "", -1, -1, -1, -1, nil, []string{"hh"}, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)
	count := int(out["count"].(float64))
	if count != 1 {
		t.Errorf("expected 1 card with hh tag (Holy Rosary), got %d", count)
	}
}

func TestSearchCardsAdvanced_CyclePrefixFilter(t *testing.T) {
	resetCaches()
	client, srv := newTestArkhamDBClient(testCards, testPacks)
	defer srv.Close()

	// cycleCode "eoe" should match pack code "eoe1" via prefix
	result, err := client.SearchCardsAdvanced(0, "eoe", "", "", -1, -1, -1, -1, nil, nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)
	count := int(out["count"].(float64))
	if count != 1 {
		t.Errorf("expected 1 card from eoe cycle (Holy Rosary), got %d", count)
	}
}

func TestSearchCardsAdvanced_SkipsInvestigatorsAndWeaknesses(t *testing.T) {
	resetCaches()
	client, srv := newTestArkhamDBClient(testCards, testPacks)
	defer srv.Close()

	result, err := client.SearchCardsAdvanced(0, "", "", "", -1, -1, -1, -1, nil, nil, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)

	cards := out["cards"].([]interface{})
	for _, c := range cards {
		card := c.(map[string]interface{})
		if card["type_code"] == "investigator" {
			t.Errorf("investigator should be excluded: %v", card["code"])
		}
		if st, _ := card["subtype_code"].(string); st == "basicweakness" || st == "weakness" {
			t.Errorf("weakness should be excluded: %v", card["code"])
		}
	}
	// Should have: Guard Dog, Easy Mark, Overpower, Holy Rosary = 4
	count := int(out["count"].(float64))
	if count != 4 {
		t.Errorf("expected 4 non-investigator non-weakness cards, got %d", count)
	}
}
