package arkhamdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
)

// getAllPacks fetches all packs from the API with 24h caching
func (c *ArkhamDBClient) getAllPacks() ([]map[string]interface{}, error) {
	packsCache.mu.RLock()
	if !packsCache.cachedAt.IsZero() && timeNow().Sub(packsCache.cachedAt) < cacheTTL {
		data := packsCache.data
		packsCache.mu.RUnlock()
		return data, nil
	}
	packsCache.mu.RUnlock()

	url := fmt.Sprintf("%s/api/public/packs/", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch packs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read packs response: %w", err)
	}
	var packs []map[string]interface{}
	if err := json.Unmarshal(body, &packs); err != nil {
		return nil, fmt.Errorf("failed to parse packs JSON: %w", err)
	}

	packsCache.mu.Lock()
	if packsCache.cachedAt.IsZero() || timeNow().Sub(packsCache.cachedAt) >= cacheTTL {
		packsCache.data = packs
		packsCache.cachedAt = timeNow()
	}
	packsCache.mu.Unlock()

	return packs, nil
}

// buildPackLookup returns a map from pack_code to pack object
func buildPackLookup(packs []map[string]interface{}) map[string]map[string]interface{} {
	m := make(map[string]map[string]interface{}, len(packs))
	for _, p := range packs {
		if code, ok := p["code"].(string); ok {
			m[code] = p
		}
	}
	return m
}

// GetPacksAndCycles returns all packs grouped by cycle with chapter information
func (c *ArkhamDBClient) GetPacksAndCycles() (string, error) {
	allPacks, err := c.getAllPacks()
	if err != nil {
		return "", err
	}

	type packSummary struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		Position  int    `json:"position"`
		Available string `json:"available,omitempty"`
		Replaced  bool   `json:"replaced,omitempty"`
	}
	type cycleSummary struct {
		CyclePosition int           `json:"cyclePosition"`
		Chapter       int           `json:"chapter"`
		Packs         []packSummary `json:"packs"`
	}

	cycleMap := make(map[int]*cycleSummary)
	for _, p := range allPacks {
		cyclePos := int(floatVal(p["cycle_position"]))
		chapter := int(floatVal(p["chapter"]))
		code, _ := p["code"].(string)
		name, _ := p["name"].(string)
		pos := int(floatVal(p["position"]))
		available, _ := p["available"].(string)
		replaced, _ := p["replaced"].(bool)

		if _, ok := cycleMap[cyclePos]; !ok {
			cycleMap[cyclePos] = &cycleSummary{
				CyclePosition: cyclePos,
				Chapter:       chapter,
				Packs:         []packSummary{},
			}
		}
		cycleMap[cyclePos].Packs = append(cycleMap[cyclePos].Packs, packSummary{
			Code:      code,
			Name:      name,
			Position:  pos,
			Available: available,
			Replaced:  replaced,
		})
	}

	cycles := make([]*cycleSummary, 0, len(cycleMap))
	for _, c := range cycleMap {
		cycles = append(cycles, c)
	}
	sort.Slice(cycles, func(i, j int) bool {
		return cycles[i].CyclePosition < cycles[j].CyclePosition
	})

	result := map[string]interface{}{
		"cycles":     cycles,
		"totalPacks": len(allPacks),
	}
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return string(out), nil
}

// floatVal extracts a float64 from interface{}, returning 0 if not present
func floatVal(v interface{}) float64 {
	f, _ := v.(float64)
	return f
}
