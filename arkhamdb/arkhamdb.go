package arkhamdb

import (
	"arkhamdb-mcp/tools"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// ArkhamDBClient is a client for the ArkhamDB API
// It implements the tools.ArkhamDBTool interface
type ArkhamDBClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewArkhamDBClient creates a new ArkhamDBClient
func NewArkhamDBClient(baseURL string) *ArkhamDBClient {
	return &ArkhamDBClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// GetCard gets a card by its code
func (c *ArkhamDBClient) GetCard(cardCode string) (string, error) {
	url := fmt.Sprintf("%s/api/public/card/%s.json", c.baseURL, cardCode)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Pretty print JSON
	var card interface{}
	if err := json.Unmarshal(body, &card); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// SearchCardsByName searches for cards by name (case-insensitive partial match)
func (c *ArkhamDBClient) SearchCardsByName(name string) (string, error) {
	allCards, err := c.getAllCards()
	if err != nil {
		return "", fmt.Errorf("failed to fetch cards: %w", err)
	}

	// Filter cards by name (case-insensitive partial match)
	nameLower := strings.ToLower(name)
	var matchingCards []map[string]interface{}

	for _, card := range allCards {
		matches := false
		if nameVal, ok := card["name"].(string); ok && nameVal != "" {
			if strings.Contains(strings.ToLower(nameVal), nameLower) {
				matches = true
			}
		}
		if realNameVal, ok := card["real_name"].(string); ok && realNameVal != "" {
			if strings.Contains(strings.ToLower(realNameVal), nameLower) {
				matches = true
			}
		}
		if matches {
			matchingCards = append(matchingCards, card)
		}
	}

	if len(matchingCards) == 0 {
		return fmt.Sprintf("No cards found matching '%s'", name), nil
	}
	prettyJSON, err := json.MarshalIndent(matchingCards, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return fmt.Sprintf("Found %d card(s) matching '%s':\n%s", len(matchingCards), name, string(prettyJSON)), nil
}

// SearchCardsAdvanced searches cards with multiple filters.
// chapter: 1 or 2 (0 = any). xpMin/xpMax/costMin/costMax: -1 = unset.
func (c *ArkhamDBClient) SearchCardsAdvanced(
	chapter int,
	cycleCode string,
	factionCode string,
	typeCode string,
	xpMin int, xpMax int,
	costMin int, costMax int,
	traits []string,
	tags []string,
	maxResults int,
) (string, error) {
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 200 {
		maxResults = 200
	}

	allCards, err := c.getAllCards()
	if err != nil {
		return "", err
	}

	var packLookup map[string]map[string]interface{}
	if chapter > 0 || cycleCode != "" {
		allPacks, err := c.getAllPacks()
		if err != nil {
			return "", err
		}
		packLookup = buildPackLookup(allPacks)
	}

	var matched []map[string]interface{}
	for _, card := range allCards {
		// Skip investigator and weakness cards from generic searches
		if tc, _ := card["type_code"].(string); tc == "investigator" {
			continue
		}
		if st, _ := card["subtype_code"].(string); st == "basicweakness" || st == "weakness" {
			continue
		}

		// Chapter filter
		if chapter > 0 && packLookup != nil {
			packCode, _ := card["pack_code"].(string)
			if pack, ok := packLookup[packCode]; ok {
				packChapter := int(floatVal(pack["chapter"]))
				if packChapter != chapter {
					continue
				}
			}
		}

		// Cycle filter — match pack_code equal to cycleCode or starting with cycleCode prefix
		if cycleCode != "" && packLookup != nil {
			packCode, _ := card["pack_code"].(string)
			pack, ok := packLookup[packCode]
			if !ok {
				continue
			}
			packCodeVal, _ := pack["code"].(string)
			if !strings.EqualFold(packCodeVal, cycleCode) &&
				!strings.HasPrefix(strings.ToLower(packCodeVal), strings.ToLower(cycleCode)) {
				continue
			}
		}

		// Faction filter
		if factionCode != "" {
			fc, _ := card["faction_code"].(string)
			if !strings.EqualFold(fc, factionCode) {
				continue
			}
		}

		// Type filter
		if typeCode != "" {
			tc, _ := card["type_code"].(string)
			if !strings.EqualFold(tc, typeCode) {
				continue
			}
		}

		// XP range filter
		xpVal := int(floatVal(card["xp"]))
		if xpMin >= 0 && xpVal < xpMin {
			continue
		}
		if xpMax >= 0 && xpVal > xpMax {
			continue
		}

		// Cost range filter — skip cards with no cost field when a cost filter is active
		if costMin >= 0 || costMax >= 0 {
			if card["cost"] == nil {
				continue
			}
			costVal := int(floatVal(card["cost"]))
			if costMin >= 0 && costVal < costMin {
				continue
			}
			if costMax >= 0 && costVal > costMax {
				continue
			}
		}

		// Traits filter (ALL must match)
		if len(traits) > 0 {
			allMatch := true
			for _, t := range traits {
				if !hasTrait(card, t) {
					allMatch = false
					break
				}
			}
			if !allMatch {
				continue
			}
		}

		// Tags filter (ANY must match)
		if len(tags) > 0 {
			tagVal, _ := card["tags"].(string)
			anyMatch := false
			for _, tag := range tags {
				if strings.Contains(tagVal, tag) {
					anyMatch = true
					break
				}
			}
			if !anyMatch {
				continue
			}
		}

		matched = append(matched, card)
		if len(matched) >= maxResults {
			break
		}
	}

	out, err := json.MarshalIndent(map[string]interface{}{
		"count": len(matched),
		"cards": matched,
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return string(out), nil
}

// GetDeck gets a deck by its ID (requires authentication, so this may not work for public API)
func (c *ArkhamDBClient) GetDeck(deckID int) (string, error) {
	url := fmt.Sprintf("%s/api/public/deck/%d.json", c.baseURL, deckID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch deck: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s. Note: Deck endpoints may require authentication", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Pretty print JSON
	var deck interface{}
	if err := json.Unmarshal(body, &deck); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(deck, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// GetDecklist gets a decklist by its ID
func (c *ArkhamDBClient) GetDecklist(decklistID int) (string, error) {
	url := fmt.Sprintf("%s/api/public/decklist/%d.json", c.baseURL, decklistID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch decklist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Pretty print JSON
	var decklist interface{}
	if err := json.Unmarshal(body, &decklist); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(decklist, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// getAllCards fetches all cards from the API (helper method)
func (c *ArkhamDBClient) getAllCards() ([]map[string]interface{}, error) {
	cardsCache.mu.RLock()
	if !cardsCache.cachedAt.IsZero() && timeNow().Sub(cardsCache.cachedAt) < cacheTTL {
		data := cardsCache.data
		cardsCache.mu.RUnlock()
		return data, nil
	}
	cardsCache.mu.RUnlock()

	url := fmt.Sprintf("%s/api/public/cards/", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cards: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	var allCards []map[string]interface{}
	if err := json.Unmarshal(body, &allCards); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	cardsCache.mu.Lock()
	if cardsCache.cachedAt.IsZero() || timeNow().Sub(cardsCache.cachedAt) >= cacheTTL {
		cardsCache.data = allCards
		cardsCache.cachedAt = timeNow()
	}
	cardsCache.mu.Unlock()

	return allCards, nil
}

// FindCardSynergies finds cards that synergize with the given card
func (c *ArkhamDBClient) FindCardSynergies(cardCode string, maxResults int) (string, error) {
	// Default maxResults to 10 if not specified or invalid
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 50 {
		maxResults = 50 // Cap at 50 for performance
	}

	// Fetch the target card
	url := fmt.Sprintf("%s/api/public/card/%s.json", c.baseURL, cardCode)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var targetCard map[string]interface{}
	if err := json.Unmarshal(body, &targetCard); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Fetch all cards
	allCards, err := c.getAllCards()
	if err != nil {
		return "", fmt.Errorf("failed to fetch all cards: %w", err)
	}

	// Score each card for synergy
	synergies := []SynergyScore{}
	for _, candidateCard := range allCards {
		score, reasons := scoreSynergy(targetCard, candidateCard)
		if score > 0 && len(reasons) > 0 {
			synergies = append(synergies, SynergyScore{
				Card:    candidateCard,
				Score:   score,
				Reasons: reasons,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(synergies, func(i, j int) bool {
		return synergies[i].Score > synergies[j].Score
	})

	// Take top N results
	if len(synergies) > maxResults {
		synergies = synergies[:maxResults]
	}

	// Build result structure
	targetCardName := getCardName(targetCard)
	if targetCardName == "" {
		targetCardName = cardCode
	}

	result := map[string]interface{}{
		"card":      cardCode,
		"cardName":  targetCardName,
		"synergies": []map[string]interface{}{},
	}

	synergyList := []map[string]interface{}{}
	for _, synergy := range synergies {
		candidateCode, _ := synergy.Card["code"].(string)
		candidateName := getCardName(synergy.Card)
		if candidateName == "" {
			candidateName = candidateCode
		}

		synergyList = append(synergyList, map[string]interface{}{
			"cardCode": candidateCode,
			"cardName": candidateName,
			"score":    synergy.Score,
			"reasons":  synergy.Reasons,
		})
	}

	result["synergies"] = synergyList

	// Pretty print JSON
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// SuggestDeckImprovements suggests cards that would improve a deck
// It considers investigator requirements (deck size, class, level, experience)
func (c *ArkhamDBClient) SuggestDeckImprovements(deckID *int, decklistID *int, maxResults int) (string, error) {
	// Default maxResults to 20 if not specified or invalid
	if maxResults <= 0 {
		maxResults = 20
	}
	if maxResults > 50 {
		maxResults = 50 // Cap at 50 for performance
	}

	// Fetch deck or decklist
	var deckData map[string]interface{}
	if deckID != nil {
		deckJSON, err := c.GetDeck(*deckID)
		if err != nil {
			return "", fmt.Errorf("failed to fetch deck: %w", err)
		}
		if err := json.Unmarshal([]byte(deckJSON), &deckData); err != nil {
			return "", fmt.Errorf("failed to parse deck JSON: %w", err)
		}
	} else if decklistID != nil {
		decklistJSON, err := c.GetDecklist(*decklistID)
		if err != nil {
			return "", fmt.Errorf("failed to fetch decklist: %w", err)
		}
		if err := json.Unmarshal([]byte(decklistJSON), &deckData); err != nil {
			return "", fmt.Errorf("failed to parse decklist JSON: %w", err)
		}
	} else {
		return "", fmt.Errorf("either deckID or decklistID must be provided")
	}

	// Extract investigator code
	investigatorCode, ok := deckData["investigator_code"].(string)
	if !ok {
		return "", fmt.Errorf("could not find investigator_code in deck/decklist")
	}

	// Get investigator card to understand deck requirements
	investigatorCardJSON, err := c.GetCard(investigatorCode)
	if err != nil {
		return "", fmt.Errorf("failed to fetch investigator card: %w", err)
	}
	var investigatorCard map[string]interface{}
	if err := json.Unmarshal([]byte(investigatorCardJSON), &investigatorCard); err != nil {
		return "", fmt.Errorf("failed to parse investigator card JSON: %w", err)
	}

	// Extract deck requirements from investigator
	deckReqs := parseDeckRequirements(investigatorCard["deck_requirements"])
	deckOptions := parseDeckOptions(investigatorCard["deck_options"])
	deckSize := float64(30)
	if deckReqs != nil && deckReqs.Size > 0 {
		deckSize = float64(deckReqs.Size)
	}

	// Get all cards in the current deck
	deckCards := extractDeckCards(deckData)
	deckCardCodes := make(map[string]bool)
	for code := range deckCards {
		deckCardCodes[code] = true
	}

	// Fetch all cards
	allCards, err := c.getAllCards()
	if err != nil {
		return "", fmt.Errorf("failed to fetch all cards: %w", err)
	}

	// Score cards for deck improvement
	type ImprovementScore struct {
		Card    map[string]interface{}
		Score   int
		Reasons []string
	}
	improvements := []ImprovementScore{}

	for _, candidateCard := range allCards {
		// Skip if already in deck
		cardCode, _ := candidateCard["code"].(string)
		if deckCardCodes[cardCode] {
			continue
		}

		// Skip if it's an investigator card
		if cardType, _ := candidateCard["type_code"].(string); cardType == "investigator" {
			continue
		}

		// Check if card matches deckbuilding requirements
		if !isCardAllowedByOptions(candidateCard, deckOptions) {
			continue
		}

		// Score the card based on synergies with deck
		score, reasons := scoreCardForDeck(candidateCard, deckCards, allCards)

		if score > 0 {
			improvements = append(improvements, ImprovementScore{
				Card:    candidateCard,
				Score:   score,
				Reasons: reasons,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(improvements, func(i, j int) bool {
		return improvements[i].Score > improvements[j].Score
	})

	// Take top N results
	if len(improvements) > maxResults {
		improvements = improvements[:maxResults]
	}

	// Build result structure
	investigatorName := getCardName(investigatorCard)
	if investigatorName == "" {
		investigatorName = investigatorCode
	}

	result := map[string]interface{}{
		"investigatorCode":    investigatorCode,
		"investigatorName":    investigatorName,
		"deckSize":            int(deckSize),
		"deckOptions": deckOptions,
		"suggestions":         []map[string]interface{}{},
	}

	suggestionList := []map[string]interface{}{}
	for _, improvement := range improvements {
		candidateCode, _ := improvement.Card["code"].(string)
		candidateName := getCardName(improvement.Card)
		if candidateName == "" {
			candidateName = candidateCode
		}
		candidateFaction, _ := improvement.Card["faction_code"].(string)
		candidateLevel, _ := improvement.Card["xp"].(float64)
		cost, _ := improvement.Card["cost"].(float64)

		suggestionList = append(suggestionList, map[string]interface{}{
			"cardCode": candidateCode,
			"cardName": candidateName,
			"faction":  candidateFaction,
			"level":    int(candidateLevel),
			"cost":     cost,
			"score":    improvement.Score,
			"reasons":  improvement.Reasons,
		})
	}

	result["suggestions"] = suggestionList

	// Pretty print JSON
	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(prettyJSON), nil
}

// extractDeckCards extracts card codes and counts from deck/decklist data
func extractDeckCards(deckData map[string]interface{}) map[string]int {
	deckCards := make(map[string]int)

	// Try slots field (common in deck/decklist format)
	if slots, ok := deckData["slots"].(map[string]interface{}); ok {
		for code, count := range slots {
			if countFloat, ok := count.(float64); ok {
				deckCards[code] = int(countFloat)
			} else if countInt, ok := count.(int); ok {
				deckCards[code] = countInt
			}
		}
	}

	return deckCards
}

// scoreCardForDeck scores a candidate card for how well it would improve the deck
func scoreCardForDeck(candidateCard map[string]interface{}, deckCards map[string]int, allCards []map[string]interface{}) (int, []string) {
	score := 0
	reasons := []string{}

	// Build a map of deck cards for quick lookup
	deckCardsMap := make(map[string]map[string]interface{})
	for code := range deckCards {
		// Find the card in allCards
		for _, card := range allCards {
			if cardCode, _ := card["code"].(string); cardCode == code {
				deckCardsMap[code] = card
				break
			}
		}
	}

	// Score synergies with each card in the deck
	totalSynergyScore := 0
	synergyCount := 0
	for _, deckCard := range deckCardsMap {
		synScore, synReasons := scoreSynergy(deckCard, candidateCard)
		if synScore > 0 {
			totalSynergyScore += synScore
			synergyCount++
			if len(reasons) < 5 { // Limit reasons to avoid clutter
				deckCardName := getCardName(deckCard)
				if deckCardName != "" {
					reasons = append(reasons, fmt.Sprintf("Synergizes with '%s' (%s)", deckCardName, strings.Join(synReasons[:min(len(synReasons), 1)], ", ")))
				}
			}
		}
	}

	if synergyCount > 0 {
		avgSynergy := totalSynergyScore / synergyCount
		score += avgSynergy
		if synergyCount > 1 {
			score += synergyCount * 5 // Bonus for multiple synergies
		}
	}

	// Bonus for low-cost cards (easier to include)
	if cost, ok := candidateCard["cost"].(float64); ok && cost <= 2 {
		score += 5
		if len(reasons) < 5 {
			reasons = append(reasons, "Low cost (easy to include)")
		}
	}

	// Bonus for level 0 cards (no XP required)
	if level, ok := candidateCard["xp"].(float64); ok && level == 0 {
		score += 10
		if len(reasons) < 5 {
			reasons = append(reasons, "Level 0 (no XP required)")
		}
	}

	return score, reasons
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetInvestigatorConstraints returns structured deck building constraints for an investigator
func (c *ArkhamDBClient) GetInvestigatorConstraints(investigatorCode string) (string, error) {
	cardJSON, err := c.GetCard(investigatorCode)
	if err != nil {
		return "", err
	}
	var inv map[string]interface{}
	if err := json.Unmarshal([]byte(cardJSON), &inv); err != nil {
		return "", fmt.Errorf("failed to parse investigator: %w", err)
	}

	if tc, _ := inv["type_code"].(string); tc != "investigator" {
		return "", fmt.Errorf("card %s is not an investigator (type: %s)", investigatorCode, tc)
	}

	allPacks, err := c.getAllPacks()
	if err != nil {
		return "", err
	}
	packLookup := buildPackLookup(allPacks)

	packCode, _ := inv["pack_code"].(string)
	chapter := 0
	if pack, ok := packLookup[packCode]; ok {
		chapter = int(floatVal(pack["chapter"]))
	}

	deckReqs := parseDeckRequirements(inv["deck_requirements"])
	deckOpts := parseDeckOptions(inv["deck_options"])

	deckSize := 30
	weaknessCount := 0
	requiredSignatures := []map[string]interface{}{}

	if deckReqs != nil {
		if deckReqs.Size > 0 {
			deckSize = deckReqs.Size
		}
		weaknessCount = len(deckReqs.Random)
		for code := range deckReqs.Card {
			sig := map[string]interface{}{"code": code}
			sigJSON, err := c.GetCard(code)
			if err == nil {
				var sigCard map[string]interface{}
				if json.Unmarshal([]byte(sigJSON), &sigCard) == nil {
					sig["name"] = getCardName(sigCard)
					sig["type"] = sigCard["type_code"]
				}
			}
			requiredSignatures = append(requiredSignatures, sig)
		}
	}

	optDescriptions := make([]map[string]interface{}, 0, len(deckOpts))
	for _, opt := range deckOpts {
		levelMin := 0
		levelMax := 5
		if opt.Level != nil {
			levelMin = opt.Level.Min
			levelMax = opt.Level.Max
		}
		optDescriptions = append(optDescriptions, map[string]interface{}{
			"description": deckOptionsDescription(opt),
			"faction":     opt.Faction,
			"levelMin":    levelMin,
			"levelMax":    levelMax,
			"limit":       opt.Limit,
			"trait":       opt.Trait,
			"tag":         opt.Tag,
			"type":        opt.Type,
			"not":         opt.Not,
			"size":        opt.Size,
		})
	}

	result := map[string]interface{}{
		"code":               investigatorCode,
		"name":               getCardName(inv),
		"subname":            inv["subname"],
		"health":             inv["health"],
		"sanity":             inv["sanity"],
		"willpower":          inv["skill_willpower"],
		"intellect":          inv["skill_intellect"],
		"combat":             inv["skill_combat"],
		"agility":            inv["skill_agility"],
		"deckSize":           deckSize,
		"randomWeaknesses":   weaknessCount,
		"requiredSignatures": requiredSignatures,
		"deckOptions":        optDescriptions,
		"packCode":           packCode,
		"chapter":            chapter,
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return string(out), nil
}

var _ tools.ArkhamDBTool = &ArkhamDBClient{}
