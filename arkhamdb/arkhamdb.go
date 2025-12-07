package arkhamdb

import (
	"encoding/json"
	"fmt"
	"io"
	"mcp-arkhamdb/tools"
	"net/http"
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

var _ tools.ArkhamDBTool = &ArkhamDBClient{}
