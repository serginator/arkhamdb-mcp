package arkhamdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TabooEntry represents a card restriction from the taboo list.
type TabooEntry struct {
	Code   string // card code
	XPCost int    // additional XP cost
	Banned bool   // true if card is effectively banned (deck_limit = 0)
}

// fetchTabooList fetches the latest taboo list from ArkhamDB.
func (c *ArkhamDBClient) fetchTabooList() (map[string]*TabooEntry, error) {
	url := fmt.Sprintf("%s/api/public/taboos/", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch taboo list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// ArkhamDB returns an array of taboo versions; the last one is the most recent.
	var versions []struct {
		Cards json.RawMessage `json:"cards"`
	}
	if err := json.Unmarshal(body, &versions); err != nil {
		return nil, fmt.Errorf("failed to parse taboo list: %w", err)
	}
	if len(versions) == 0 {
		return nil, nil
	}

	// Parse the latest version's cards.
	var cards []struct {
		Code      string      `json:"code"`
		DeckLimit interface{} `json:"deck_limit"`
		XP        interface{} `json:"xp"`
	}
	latest := versions[len(versions)-1]
	if err := json.Unmarshal(latest.Cards, &cards); err != nil {
		return nil, fmt.Errorf("failed to parse taboo cards: %w", err)
	}

	result := make(map[string]*TabooEntry, len(cards))
	for _, card := range cards {
		entry := &TabooEntry{Code: card.Code}
		// deck_limit: 0 means banned
		if dl, ok := card.DeckLimit.(float64); ok && dl == 0 {
			entry.Banned = true
		}
		// xp: additional XP cost
		if xp, ok := card.XP.(float64); ok {
			entry.XPCost = int(xp)
		}
		result[card.Code] = entry
	}
	return result, nil
}

// shouldUseTaboo returns true if taboo should be applied.
// If override is non-nil, its value is used. Otherwise falls back to the
// collection's UseTaboo setting (default false).
func (c *ArkhamDBClient) shouldUseTaboo(override *bool) bool {
	if override != nil {
		return *override
	}
	if c.collection != nil {
		return c.collection.UseTaboo
	}
	return false
}
