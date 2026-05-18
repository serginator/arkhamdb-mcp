package arkhamdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SearchReferenceDecks searches published community decklists by iterating the
// /api/public/decklists/by_date/{date} endpoint backward from today.
// investigatorCode: filter by investigator (empty = any).
// xpMin/xpMax: filter by xp_spent (-1 = unset, nil deck xp counts as 0).
// tags: comma-separated tags to look for in description or tags field (empty = any).
// daysBack: how many days back to search (default/max 90).
// maxResults: stop after collecting this many matching decklists.
func (c *ArkhamDBClient) SearchReferenceDecks(investigatorCode string, xpMin int, xpMax int, tags string, daysBack int, maxResults int) (string, error) {
	if daysBack <= 0 || daysBack > 90 {
		daysBack = 90
	}
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 50 {
		maxResults = 50
	}

	tagList := []string{}
	if tags != "" {
		for _, t := range strings.Split(tags, ",") {
			t = strings.TrimSpace(strings.ToLower(t))
			if t != "" {
				tagList = append(tagList, t)
			}
		}
	}

	var results []map[string]interface{}
	today := timeNow()

	for day := 0; day < daysBack && len(results) < maxResults; day++ {
		date := today.AddDate(0, 0, -day).Format("2006-01-02")
		url := fmt.Sprintf("%s/api/public/decklists/by_date/%s.json", c.baseURL, date)

		resp, err := c.httpClient.Get(url)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var decklists []map[string]interface{}
		if err := json.Unmarshal(body, &decklists); err != nil {
			continue
		}

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

			results = append(results, map[string]interface{}{
				"id":               dl["id"],
				"name":             dl["name"],
				"investigatorCode": dl["investigator_code"],
				"investigatorName": dl["investigator_name"],
				"xpSpent":          xpSpent,
				"tags":             dl["tags"],
				"dateCreation":     dl["date_creation"],
				"url":              fmt.Sprintf("https://arkhamdb.com/decklist/view/%v", dl["id"]),
			})
		}
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
