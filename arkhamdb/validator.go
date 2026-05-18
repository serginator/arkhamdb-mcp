package arkhamdb

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// ValidateDeck validates a deck or decklist for legality
func (c *ArkhamDBClient) ValidateDeck(deckID *int, decklistID *int) (string, error) {
	var deckData map[string]interface{}
	if deckID != nil {
		raw, err := c.GetDeck(*deckID)
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal([]byte(raw), &deckData); err != nil {
			return "", err
		}
	} else if decklistID != nil {
		raw, err := c.GetDecklist(*decklistID)
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal([]byte(raw), &deckData); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("either deckID or decklistID must be provided")
	}

	invCode, _ := deckData["investigator_code"].(string)
	invJSON, err := c.GetCard(invCode)
	if err != nil {
		return "", fmt.Errorf("could not fetch investigator %s: %w", invCode, err)
	}
	var inv map[string]interface{}
	if err := json.Unmarshal([]byte(invJSON), &inv); err != nil {
		return "", err
	}

	deckReqs := parseDeckRequirements(inv["deck_requirements"])
	deckOpts := parseDeckOptions(inv["deck_options"])

	requiredSize := 30
	if deckReqs != nil && deckReqs.Size > 0 {
		requiredSize = deckReqs.Size
	}

	deckCards := extractDeckCards(deckData)
	allCards, err := c.getAllCards()
	if err != nil {
		return "", err
	}
	cardByCode := make(map[string]map[string]interface{}, len(allCards))
	for _, card := range allCards {
		if code, _ := card["code"].(string); code != "" {
			cardByCode[code] = card
		}
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

	errors := []string{}
	warnings := []string{}

	// Check deck size
	totalCards := 0
	for _, qty := range deckCards {
		totalCards += qty
	}
	if totalCards != requiredSize {
		errors = append(errors, fmt.Sprintf("Deck has %d cards but %s requires %d", totalCards, getCardName(inv), requiredSize))
	}

	// Check required signatures
	if deckReqs != nil {
		for reqCode, alts := range deckReqs.Card {
			found := deckCards[reqCode] > 0
			if !found {
				for altCode := range alts {
					if deckCards[altCode] > 0 {
						found = true
						break
					}
				}
			}
			if !found {
				sigName := reqCode
				if card, ok := cardByCode[reqCode]; ok {
					sigName = getCardName(card)
				}
				errors = append(errors, fmt.Sprintf("Missing required signature card: %s (%s)", sigName, reqCode))
			}
		}
	}

	// Track option limits
	optionCounts := make([]int, len(deckOpts))

	// Validate each card
	for code, qty := range deckCards {
		card, ok := cardByCode[code]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("Card %s not found in card database", code))
			continue
		}

		// Skip signature cards from option validation
		isSig := false
		if deckReqs != nil {
			for reqCode, alts := range deckReqs.Card {
				if code == reqCode {
					isSig = true
					break
				}
				for altCode := range alts {
					if code == altCode {
						isSig = true
						break
					}
				}
				if isSig {
					break
				}
			}
		}
		if isSig {
			continue
		}

		// Skip weakness cards
		if st, _ := card["subtype_code"].(string); st == "basicweakness" || st == "weakness" {
			continue
		}

		// Check deck_limit
		deckLimit := 2
		if dl, ok := card["deck_limit"].(float64); ok {
			deckLimit = int(dl)
		}
		if qty > deckLimit {
			errors = append(errors, fmt.Sprintf("%s (%s): has %d copies but deck limit is %d", getCardName(card), code, qty, deckLimit))
		}

		// Check deck_options
		if !isCardAllowedByOptions(card, deckOpts) {
			errors = append(errors, fmt.Sprintf("%s (%s): not allowed by investigator deck options", getCardName(card), code))
			continue
		}

		// Track option limits
		optIdx := firstMatchingOption(card, deckOpts, textRegexes)
		if optIdx >= 0 {
			optionCounts[optIdx] += qty
		}
	}

	// Check option limits
	for i, opt := range deckOpts {
		if opt.Limit > 0 && optionCounts[i] > opt.Limit {
			desc := deckOptionsDescription(opt)
			errors = append(errors, fmt.Sprintf("Option \"%s\": has %d cards but limit is %d. %s", desc, optionCounts[i], opt.Limit, opt.Error))
		}
	}

	valid := len(errors) == 0

	result := map[string]interface{}{
		"valid":            valid,
		"investigatorCode": invCode,
		"investigatorName": getCardName(inv),
		"requiredDeckSize": requiredSize,
		"actualDeckSize":   totalCards,
		"errors":           errors,
		"warnings":         warnings,
	}
	if valid {
		result["message"] = "Deck is legal."
	} else {
		result["message"] = fmt.Sprintf("Deck has %d error(s).", len(errors))
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	prefix := "VALID"
	if !valid {
		prefix = fmt.Sprintf("INVALID (%d errors)", len(errors))
	}
	return fmt.Sprintf("[%s]\n%s", prefix, string(out)), nil
}
