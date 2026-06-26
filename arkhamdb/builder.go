package arkhamdb

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type deckEntry struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Quantity    int    `json:"quantity"`
	XP          int    `json:"xp"`
	Faction     string `json:"faction"`
	IsSignature bool   `json:"isSignature"`
}

// BuildStarterDeck builds a complete legal deck for an investigator.
// chapter: 1 or 2 (0 = no chapter filter).
// cycleCodes: list of pack/cycle codes to restrict card pool (empty = all packs for chapter).
// xpBudget: 0 for a standard starter deck; >0 allows higher-level cards.
func (c *ArkhamDBClient) BuildStarterDeck(investigatorCode string, chapter int, cycleCodes []string, xpBudget int, strategy string) (string, error) {
	invJSON, err := c.GetCard(investigatorCode)
	if err != nil {
		return "", fmt.Errorf("investigator not found: %w", err)
	}
	var inv map[string]interface{}
	if err := json.Unmarshal([]byte(invJSON), &inv); err != nil {
		return "", err
	}
	if tc, _ := inv["type_code"].(string); tc != "investigator" {
		return "", fmt.Errorf("%s is not an investigator", investigatorCode)
	}

	deckReqs := parseDeckRequirements(inv["deck_requirements"])
	deckOpts := parseDeckOptions(inv["deck_options"])

	deckSize := 30
	if deckReqs != nil && deckReqs.Size > 0 {
		deckSize = deckReqs.Size
	}

	allCards, err := c.getAllCards()
	if err != nil {
		return "", err
	}
	allPacks, err := c.getAllPacks()
	if err != nil {
		return "", err
	}

	var tabooList map[string]*TabooEntry
	if c.shouldUseTaboo(nil) {
		tabooList, _ = c.fetchTabooList()
	}

	sigCodes := map[string]bool{}
	var entries []deckEntry
	sigSlotsUsed := 0

	if deckReqs != nil {
		for code := range deckReqs.Card {
			sigCodes[code] = true
			for _, card := range allCards {
				if c2, _ := card["code"].(string); c2 == code {
					entries = append(entries, deckEntry{
						Code:        code,
						Name:        getCardName(card),
						Quantity:    1,
						XP:          int(floatVal(card["xp"])),
						Faction:     getStr(card, "faction_code"),
						IsSignature: true,
					})
					sigSlotsUsed++
					break
				}
			}
		}
	}

	// If collection has owned cycles, restrict to those (user's cycleCodes param takes priority)
	if len(cycleCodes) == 0 && c.collection != nil && len(c.collection.OwnedCycles) > 0 {
		cycleCodes = c.collection.OwnedCycles
	}

	allowedPackCodes := buildAllowedPackCodes(allPacks, chapter, cycleCodes)

	// Fetch popular decks to use as archetype signal
	popularJSON, _ := c.SearchReferenceDecks(investigatorCode, -1, -1, "", 0, 5)
	popularContext := ""
	if popularJSON != "" {
		popularContext = fmt.Sprintf("Top popular decks for this investigator:\n%s\n\nPlayer strategy hint: %s", popularJSON, strategy)
	}

	// Pre-compile text regexes for deck options
	textRegexes := make(map[string]*regexp.Regexp)
	for _, opt := range deckOpts {
		if opt.Text != "" {
			if _, ok := textRegexes[opt.Text]; !ok {
				if re, err := regexp.Compile("(?i)" + opt.Text); err == nil {
					textRegexes[opt.Text] = re
				}
			}
		}
	}

	type scored struct {
		card  map[string]interface{}
		score int
	}
	var candidates []scored

	for _, card := range allCards {
		code, _ := card["code"].(string)
		if sigCodes[code] {
			continue
		}
		tc, _ := card["type_code"].(string)
		if tc == "investigator" {
			continue
		}
		st, _ := card["subtype_code"].(string)
		if st == "basicweakness" || st == "weakness" {
			continue
		}

		// Skip taboo-banned cards
		if tabooList != nil {
			if entry, ok := tabooList[code]; ok && entry.Banned {
				continue
			}
		}

		if len(allowedPackCodes) > 0 {
			packCode, _ := card["pack_code"].(string)
			if !allowedPackCodes[packCode] {
				continue
			}
		}

		if !isCardAllowedByOptions(card, deckOpts) {
			continue
		}

		cardXP := int(floatVal(card["xp"]))
		if xpBudget == 0 && cardXP > 0 {
			continue
		}
		if xpBudget > 0 && cardXP > xpBudget {
			continue
		}

		score, _ := scoreSynergy(inv, card)
		candidates = append(candidates, scored{card: card, score: score})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	optionCounts := make([]int, len(deckOpts))
	added := map[string]int{}

	slotsRemaining := deckSize - sigSlotsUsed
	for _, cand := range candidates {
		if slotsRemaining <= 0 {
			break
		}
		card := cand.card
		code, _ := card["code"].(string)

		deckLimit := 2
		if dl, ok := card["deck_limit"].(float64); ok && dl > 0 {
			deckLimit = int(dl)
		}
		isUnique, _ := card["is_unique"].(bool)
		if isUnique {
			deckLimit = 1
		}

		alreadyAdded := added[code]
		canAdd := deckLimit - alreadyAdded
		if canAdd <= 0 {
			continue
		}

		optIdx := firstMatchingOption(card, deckOpts, textRegexes)
		if optIdx >= 0 && deckOpts[optIdx].Limit > 0 {
			if optionCounts[optIdx] >= deckOpts[optIdx].Limit {
				continue
			}
			limitRemaining := deckOpts[optIdx].Limit - optionCounts[optIdx]
			if canAdd > limitRemaining {
				canAdd = limitRemaining
			}
		}

		qty := canAdd
		if qty > slotsRemaining {
			qty = slotsRemaining
		}

		added[code] += qty
		slotsRemaining -= qty
		if optIdx >= 0 {
			optionCounts[optIdx] += qty
		}

		entries = append(entries, deckEntry{
			Code:        code,
			Name:        getCardName(card),
			Quantity:    qty,
			XP:          int(floatVal(card["xp"])),
			Faction:     getStr(card, "faction_code"),
			IsSignature: false,
		})
	}

	warnings := []string{}
	extraWeaknesses := 0
	if xpBudget >= 30 {
		extraWeaknesses = 3
	} else if xpBudget >= 20 {
		extraWeaknesses = 2
	} else if xpBudget >= 10 {
		extraWeaknesses = 1
	}
	if deckReqs != nil {
		extraWeaknesses += len(deckReqs.Random)
	}
	if extraWeaknesses > 0 {
		warnings = append(warnings, fmt.Sprintf("Add %d random basic weakness card(s) to the deck as required by deck rules.", extraWeaknesses))
	}

	totalCards := 0
	for _, e := range entries {
		totalCards += e.Quantity
	}
	if totalCards < deckSize {
		warnings = append(warnings, fmt.Sprintf("Only %d/%d slots filled — not enough legal cards in the filtered pool. Expand cycle filters or remove chapter restriction.", totalCards, deckSize))
	}

	result := map[string]interface{}{
		"investigatorCode": investigatorCode,
		"investigatorName": getCardName(inv),
		"deckSize":         deckSize,
		"xpBudget":         xpBudget,
		"chapter":          chapter,
		"cycleCodes":       cycleCodes,
		"cards":            entries,
		"totalCards":       totalCards,
		"warnings":         warnings,
		"strategyContext":  popularContext,
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return string(out), nil
}

// buildAllowedPackCodes returns a set of pack codes allowed by chapter/cycle filters.
// Returns nil (meaning no filter) when both chapter and cycleCodes are unset.
func buildAllowedPackCodes(packs []map[string]interface{}, chapter int, cycleCodes []string) map[string]bool {
	if chapter == 0 && len(cycleCodes) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, p := range packs {
		code, _ := p["code"].(string)
		packChapter := int(floatVal(p["chapter"]))

		chapterOK := chapter == 0 || packChapter == chapter
		cycleOK := len(cycleCodes) == 0
		if !cycleOK {
			for _, cc := range cycleCodes {
				if strings.EqualFold(code, cc) || strings.HasPrefix(strings.ToLower(code), strings.ToLower(cc)) {
					cycleOK = true
					break
				}
			}
		}
		if chapterOK && cycleOK {
			allowed[code] = true
		}
	}
	return allowed
}

// firstMatchingOption returns the index of the first deck_option that allows this card, or -1.
// Size-only options (options that set deck size without faction/trait constraints) are skipped.
func firstMatchingOption(card map[string]interface{}, opts []DeckOption, textRegexes map[string]*regexp.Regexp) int {
	faction, _ := card["faction_code"].(string)
	xp := int(floatVal(card["xp"]))
	tc, _ := card["type_code"].(string)
	text := getCardText(card)
	for i, opt := range opts {
		if opt.Size > 0 && len(opt.Faction) == 0 {
			continue
		}
		if cardMatchesDeckOption(card, opt, faction, xp, tc, text, textRegexes) {
			return i
		}
	}
	return -1
}

// getStr safely extracts a string value from a card map
func getStr(card map[string]interface{}, key string) string {
	v, _ := card[key].(string)
	return v
}
