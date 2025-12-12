package arkhamdb

import (
	"arkhamdb-mcp/tools"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
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
	url := fmt.Sprintf("%s/api/public/cards/", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch cards: %w", err)
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

	// Parse all cards
	var allCards []map[string]interface{}
	if err := json.Unmarshal(body, &allCards); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Filter cards by name (case-insensitive partial match)
	nameLower := strings.ToLower(name)
	var matchingCards []map[string]interface{}

	for _, card := range allCards {
		// Check both name and real_name fields for matches (either can match)
		matches := false

		// Check name field
		if nameVal, ok := card["name"].(string); ok && nameVal != "" {
			if strings.Contains(strings.ToLower(nameVal), nameLower) {
				matches = true
			}
		}

		// Check real_name field (check independently, not just if name didn't match)
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

	// Pretty print matching cards
	prettyJSON, err := json.MarshalIndent(matchingCards, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return fmt.Sprintf("Found %d card(s) matching '%s':\n%s", len(matchingCards), name, string(prettyJSON)), nil
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
	deckSize, _ := investigatorCard["deck_size"].(float64)
	deckbuildingOptions, _ := investigatorCard["deckbuilding_options"].(string)
	_ = investigatorCard["deckbuilding_requirements"] // Reserved for future use

	// Parse deckbuilding options (e.g., "Rogue 0-5, Guardian 0-2, Neutral 0-5")
	allowedFactions, allowedLevels := parseDeckbuildingOptions(deckbuildingOptions)

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
		cardFaction, _ := candidateCard["faction_code"].(string)
		cardLevel, _ := candidateCard["xp"].(float64)
		cardLevelInt := int(cardLevel)

		if !isCardAllowed(cardFaction, cardLevelInt, allowedFactions, allowedLevels) {
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
		"deckbuildingOptions": deckbuildingOptions,
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

// parseDeckbuildingOptions parses deckbuilding options string
// Returns maps of faction -> allowed level ranges
func parseDeckbuildingOptions(options string) (map[string][]int, map[string]int) {
	allowedFactions := make(map[string][]int) // faction -> [minLevel, maxLevel]
	allowedLevels := make(map[string]int)     // faction -> maxLevel (for backward compatibility)

	if options == "" {
		return allowedFactions, allowedLevels
	}

	// Parse format like "Rogue 0-5, Guardian 0-2, Neutral 0-5"
	parts := strings.Split(options, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try to match "Faction min-max" or "Faction max"
		re := regexp.MustCompile(`(\w+)\s+(\d+)(?:-(\d+))?`)
		matches := re.FindStringSubmatch(part)
		if len(matches) >= 3 {
			faction := strings.ToLower(matches[1])
			minLevel, _ := strconv.Atoi(matches[2])
			maxLevel := minLevel
			if len(matches) >= 4 && matches[3] != "" {
				maxLevel, _ = strconv.Atoi(matches[3])
			}
			allowedFactions[faction] = []int{minLevel, maxLevel}
			allowedLevels[faction] = maxLevel
		}
	}

	return allowedFactions, allowedLevels
}

// isCardAllowed checks if a card matches deckbuilding requirements
func isCardAllowed(cardFaction string, cardLevel int, allowedFactions map[string][]int, allowedLevels map[string]int) bool {
	if cardFaction == "" {
		return false
	}

	cardFactionLower := strings.ToLower(cardFaction)
	levelRange, ok := allowedFactions[cardFactionLower]
	if !ok {
		return false
	}

	return cardLevel >= levelRange[0] && cardLevel <= levelRange[1]
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

var _ tools.ArkhamDBTool = &ArkhamDBClient{}
