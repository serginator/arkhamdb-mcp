package arkhamdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SearchReferenceDecks fetches popular community decklists from ArkhamDB.
// investigatorCode: filter by investigator (empty = any).
// xpMin/xpMax: filter by xp_spent (-1 = unset).
// tags: comma-separated tags to look for in description or tags field (empty = any).
// daysBack: unused (kept for backwards compatibility).
// maxResults: max decklists to return (default 10, max 50).
func (c *ArkhamDBClient) SearchReferenceDecks(investigatorCode string, xpMin int, xpMax int, tags string, daysBack int, maxResults int) (string, error) {
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 50 {
		maxResults = 50
	}

	tagList := []string{}
	if tags != "" {
		for _, t := range strings.Split(tags, ",") {
			if t = strings.TrimSpace(strings.ToLower(t)); t != "" {
				tagList = append(tagList, t)
			}
		}
	}

	url := fmt.Sprintf("%s/api/public/decklists/popular.json", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch popular decklists: %w", err)
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

	var decklists []map[string]interface{}
	if err := json.Unmarshal(body, &decklists); err != nil {
		return "", fmt.Errorf("failed to parse decklists: %w", err)
	}

	var results []map[string]interface{}
	for _, dl := range decklists {
		if len(results) >= maxResults {
			break
		}

		if investigatorCode != "" {
			ic, _ := dl["investigator_code"].(string)
			if !strings.EqualFold(ic, investigatorCode) {
				continue
			}
		}

		xpSpent := 0
		if xp, ok := dl["xp_spent"].(float64); ok {
			xpSpent = int(xp)
		}
		if xpMin >= 0 && xpSpent < xpMin {
			continue
		}
		if xpMax >= 0 && xpSpent > xpMax {
			continue
		}

		if len(tagList) > 0 {
			deckTags, _ := dl["tags"].(string)
			desc, _ := dl["description_md"].(string)
			combined := strings.ToLower(deckTags + " " + desc)
			anyMatch := false
			for _, tag := range tagList {
				if strings.Contains(combined, tag) {
					anyMatch = true
					break
				}
			}
			if !anyMatch {
				continue
			}
		}

		likes := 0
		if l, ok := dl["nb_favorites"].(float64); ok {
			likes = int(l)
		}

		results = append(results, map[string]interface{}{
			"id":               dl["id"],
			"name":             dl["name"],
			"investigatorCode": dl["investigator_code"],
			"investigatorName": dl["investigator_name"],
			"xpSpent":          xpSpent,
			"likes":            likes,
			"tags":             dl["tags"],
			"url":              fmt.Sprintf("https://arkhamdb.com/decklist/view/%v", dl["id"]),
		})
	}

	out, err := json.MarshalIndent(map[string]interface{}{
		"count":     len(results),
		"decklists": results,
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return string(out), nil
}
