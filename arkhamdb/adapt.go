package arkhamdb

import (
	"encoding/json"
	"fmt"
	"strings"
)

// isCardOwned returns true if the card's pack is in the owned set.
func isCardOwned(card map[string]interface{}, ownedPacks map[string]bool) bool {
	packCode, _ := card["pack_code"].(string)
	if packCode == "" {
		return true // unknown pack = assume owned
	}
	// Check exact pack code
	if ownedPacks[packCode] {
		return true
	}
	// Check cycle prefix (e.g. "dwl" matches "dwl01", "dwlc" etc.)
	for owned := range ownedPacks {
		if strings.HasPrefix(packCode, owned) || strings.HasPrefix(owned, packCode) {
			return true
		}
	}
	return false
}

// AdaptDeckToCollection takes a public decklist and returns a modified version
// where cards not in the user's collection are replaced by legal owned equivalents.
func (c *ArkhamDBClient) AdaptDeckToCollection(decklistID int) (string, error) {
	if c.collection == nil || len(c.collection.OwnedCycles) == 0 {
		return "", fmt.Errorf("no collection configured — run arkhamdb_set_collection first")
	}

	// Build owned pack set
	owned := make(map[string]bool)
	for _, cycle := range c.collection.OwnedCycles {
		owned[cycle] = true
	}

	decklistJSON, err := c.GetDecklist(decklistID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch decklist: %w", err)
	}

	var decklist map[string]interface{}
	if err := json.Unmarshal([]byte(decklistJSON), &decklist); err != nil {
		return "", err
	}

	slots, _ := decklist["slots"].(map[string]interface{})
	if slots == nil {
		return "", fmt.Errorf("decklist has no slots")
	}

	allCards, err := c.getAllCards()
	if err != nil {
		return "", err
	}

	// Build card lookup by code
	cardByCode := make(map[string]map[string]interface{}, len(allCards))
	for _, card := range allCards {
		if code, _ := card["code"].(string); code != "" {
			cardByCode[code] = card
		}
	}

	var missing []map[string]interface{}
	var kept []string

	for code := range slots {
		card, ok := cardByCode[code]
		if !ok {
			continue
		}
		if !isCardOwned(card, owned) {
			missing = append(missing, card)
		} else {
			kept = append(kept, code)
		}
	}

	// For missing cards, find replacements: same faction, same XP level, same type, owned
	replacements := make([]map[string]interface{}, 0, len(missing))
	for _, missingCard := range missing {
		faction, _ := missingCard["faction_code"].(string)
		xp := 0
		if x, ok := missingCard["xp"].(float64); ok {
			xp = int(x)
		}
		typeCode, _ := missingCard["type_code"].(string)

		var candidates []map[string]interface{}
		for _, card := range allCards {
			if !isCardOwned(card, owned) {
				continue
			}
			if card["faction_code"] != faction {
				continue
			}
			if card["type_code"] != typeCode {
				continue
			}
			cardXP := 0
			if x, ok := card["xp"].(float64); ok {
				cardXP = int(x)
			}
			if cardXP != xp {
				continue
			}
			candidates = append(candidates, card)
		}

		names := make([]string, 0, len(candidates))
		for _, cand := range candidates {
			names = append(names, fmt.Sprintf("%s (%s)", c.cardName(cand), cand["code"]))
		}
		if len(names) > 5 {
			names = names[:5]
		}

		replacements = append(replacements, map[string]interface{}{
			"missing":      c.cardName(missingCard),
			"missingCode":  missingCard["code"],
			"packCode":     missingCard["pack_code"],
			"replacements": names,
		})
	}

	out, err := json.MarshalIndent(map[string]interface{}{
		"decklistID":   decklistID,
		"deckName":     decklist["name"],
		"totalCards":   len(slots),
		"keptCards":    len(kept),
		"missingCards": len(missing),
		"replacements": replacements,
		"note":         "Review replacements and pick the best fit for your strategy. Use arkhamdb_validate_deck after adapting.",
	}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
