package server

import (
	"arkhamdb-mcp/tools"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// MCPServer implements the Model Context Protocol server
type MCPServer struct {
	ArkhamDB tools.ArkhamDBTool
}

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"` // Can be string, number, or null
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents content in a tool result
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Start starts the MCP server
func (s *MCPServer) Start() {
	// Ensure all logs go to stderr, not stdout
	log.SetOutput(os.Stderr)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var request MCPRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			// For parse errors, id MUST be null per JSON-RPC 2.0 spec
			s.sendError(nil, -32700, "Parse error", nil)
			continue
		}

		s.handleRequest(request)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Printf("Error reading from stdin: %v", err)
	}
}

// handleRequest processes an MCP request
func (s *MCPServer) handleRequest(request MCPRequest) {
	// Check if this is a notification (no ID) - notifications don't get responses
	isNotification := request.ID == nil

	// Validate request has a method
	if request.Method == "" {
		if !isNotification {
			s.sendError(request.ID, -32600, "Invalid Request", nil)
		}
		return
	}

	switch request.Method {
	case "initialize":
		if !isNotification {
			s.handleInitialize(request)
		}
	case "tools/list":
		if !isNotification {
			s.handleToolsList(request)
		}
	case "tools/call":
		if !isNotification {
			s.handleToolCall(request)
		}
	default:
		if !isNotification {
			s.sendError(request.ID, -32601, "Method not found", nil)
		}
	}
}

// handleInitialize handles the initialize request
func (s *MCPServer) handleInitialize(request MCPRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "arkhamdb-mcp",
			"version": "1.0.0",
		},
	}
	s.sendResponse(request.ID, result)
}

// handleToolsList handles the tools/list request
func (s *MCPServer) handleToolsList(request MCPRequest) {
	tools := s.getAvailableTools()
	result := map[string]interface{}{
		"tools": tools,
	}
	s.sendResponse(request.ID, result)
}

// handleToolCall handles the tools/call request
func (s *MCPServer) handleToolCall(request MCPRequest) {
	params, ok := request.Params.(map[string]interface{})
	if !ok {
		s.sendError(request.ID, -32602, "Invalid params", nil)
		return
	}

	name, ok := params["name"].(string)
	if !ok {
		s.sendError(request.ID, -32602, "Missing tool name", nil)
		return
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	result, err := s.executeTool(name, arguments)
	if err != nil {
		s.sendResponse(request.ID, ToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		})
		return
	}

	s.sendResponse(request.ID, ToolResult{
		Content: []ToolContent{{Type: "text", Text: result}},
		IsError: false,
	})
}

// sendResponse sends a JSON-RPC response
func (s *MCPServer) sendResponse(id interface{}, result interface{}) {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.sendJSON(response)
}

// sendError sends a JSON-RPC error response
func (s *MCPServer) sendError(id interface{}, code int, message string, data interface{}) {
	// For parse errors (code -32700), id MUST be null per JSON-RPC 2.0 spec
	responseID := id
	if code == -32700 {
		responseID = nil
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      responseID,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.sendJSON(response)
}

// sendJSON sends a JSON message to stdout
func (s *MCPServer) sendJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Error marshaling JSON: %v", err)
		return
	}
	fmt.Println(string(data))
}

// floatFromArg extracts a float64 from interface{}
func floatFromArg(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	}
	return 0
}

// intFromArgDefault extracts an int from interface{} with a default fallback
func intFromArgDefault(v interface{}, def int) int {
	if v == nil {
		return def
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		n, err := strconv.Atoi(x)
		if err != nil {
			return def
		}
		return n
	}
	return def
}

// stringSliceFromArg extracts a []string from interface{}
func stringSliceFromArg(v interface{}) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// getAvailableTools returns the list of available tools
func (s *MCPServer) getAvailableTools() []Tool {
	return []Tool{
		{
			Name:        "arkhamdb_get_card",
			Description: "Get details of a specific card by its code (e.g., '01001')",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cardCode": map[string]interface{}{
						"type":        "string",
						"description": "The code of the card to get, e.g. '01001'",
					},
				},
				"required": []string{"cardCode"},
			},
		},
		{
			Name:        "arkhamdb_search_cards_by_name",
			Description: "Search for cards by name (case-insensitive partial match)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The name (or partial name) to search for, e.g. 'Roland'",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "arkhamdb_get_deck",
			Description: "Get details of a specific deck by its ID. Note: This endpoint may require authentication",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"deckID": map[string]interface{}{
						"type":        "integer",
						"description": "The ID of the deck to get",
					},
				},
				"required": []string{"deckID"},
			},
		},
		{
			Name:        "arkhamdb_get_decklist",
			Description: "Get details of a specific decklist by its ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"decklistID": map[string]interface{}{
						"type":        "integer",
						"description": "The ID of the decklist to get",
					},
				},
				"required": []string{"decklistID"},
			},
		},
		{
			Name:        "arkhamdb_find_card_synergies",
			Description: "Find cards that synergize with a given card based on text analysis, trait matching, and mechanic keywords",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cardCode": map[string]interface{}{
						"type":        "string",
						"description": "The code of the card to find synergies for, e.g. '06332'",
					},
					"maxResults": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of synergistic cards to return (default: 10, max: 50)",
					},
				},
				"required": []string{"cardCode"},
			},
		},
		{
			Name:        "arkhamdb_suggest_deck_improvements",
			Description: "Suggest cards that would improve a deck, taking into account investigator requirements (deck size, class restrictions, level, experience)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"deckID": map[string]interface{}{
						"type":        "integer",
						"description": "The ID of the deck to analyze (either deckID or decklistID must be provided)",
					},
					"decklistID": map[string]interface{}{
						"type":        "integer",
						"description": "The ID of the decklist to analyze (either deckID or decklistID must be provided)",
					},
					"maxResults": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of card suggestions to return (default: 20, max: 50)",
					},
					"strategy": map[string]interface{}{
						"type":        "string",
						"description": "Freetext strategy hint, e.g. 'fast cluever, minimal combat'. Used alongside popular decks to guide card selection.",
					},
				},
			},
		},
		{
			Name:        "arkhamdb_get_packs_and_cycles",
			Description: "Get all available packs and cycles for Arkham Horror LCG, grouped by cycle, with chapter (1 or 2) and release information. Use this to understand which content belongs to Chapter 1 (legacy) vs Chapter 2 (2026 relaunch).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "arkhamdb_search_cards_advanced",
			Description: "Search cards with filters: chapter (1=Chapter 1 legacy, 2=Chapter 2 2026 relaunch), cycleCode (e.g. 'dwl' for Dunwich Legacy), factionCode (e.g. 'guardian'), typeCode (e.g. 'asset'), XP range, cost range, traits (ALL must match), tags (ANY must match, e.g. 'hd'=heals damage, 'hh'=heals horror). Skips investigators and weakness cards.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chapter":     map[string]interface{}{"type": "integer", "description": "1 for Chapter 1, 2 for Chapter 2 (2026). Omit for any."},
					"cycleCode":   map[string]interface{}{"type": "string", "description": "Pack code or cycle prefix, e.g. 'dwl', 'core', 'eoe'"},
					"factionCode": map[string]interface{}{"type": "string", "description": "e.g. 'guardian', 'rogue', 'neutral'"},
					"typeCode":    map[string]interface{}{"type": "string", "description": "e.g. 'asset', 'event', 'skill'"},
					"xpMin":       map[string]interface{}{"type": "integer", "description": "Minimum XP level (0-5). Omit for no minimum."},
					"xpMax":       map[string]interface{}{"type": "integer", "description": "Maximum XP level (0-5). Omit for no maximum."},
					"costMin":     map[string]interface{}{"type": "integer", "description": "Minimum resource cost. Omit for no minimum."},
					"costMax":     map[string]interface{}{"type": "integer", "description": "Maximum resource cost. Omit for no maximum."},
					"traits":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "All traits must be present, e.g. [\"Ally\", \"Blessed\"]"},
					"tags":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Any tag must be present, e.g. [\"hd\", \"hh\"]"},
					"textSearch":  map[string]interface{}{"type": "string", "description": "Search card text for keywords, e.g. 'free action' or 'after you successfully investigate'. Results ranked by relevance."},
					"maxResults":  map[string]interface{}{"type": "integer", "description": "Max results (default 50, max 200)"},
				},
			},
		},
		{
			Name:        "arkhamdb_get_investigator_constraints",
			Description: "Get full deck-building constraints for an investigator: deck size, required signature cards, random weakness count, and all deck_options rules (faction/level/trait/tag/limit restrictions). Use this before building or validating a deck.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"investigatorCode": map[string]interface{}{
						"type":        "string",
						"description": "The investigator's card code, e.g. '01001' for Roland Banks",
					},
				},
				"required": []string{"investigatorCode"},
			},
		},
		{
			Name:        "arkhamdb_build_starter_deck",
			Description: "Build a complete legal deck for an investigator. Respects deck_options (faction/level/trait restrictions), deck size, and required signature cards. Use chapter (1 or 2) to restrict to a content generation; use cycleCodes to restrict to specific campaigns (e.g. [\"core\",\"dwl\"]). xpBudget=0 produces a standard 0-XP starter deck; higher values allow upgraded cards.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"investigatorCode": map[string]interface{}{"type": "string", "description": "Investigator card code, e.g. '01001'"},
					"chapter":          map[string]interface{}{"type": "integer", "description": "1 or 2 to restrict card pool by chapter. 0 or omit for all chapters."},
					"cycleCodes":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Pack/cycle code prefixes to restrict card pool, e.g. [\"core\", \"dwl\"]. Empty = all packs."},
					"xpBudget":         map[string]interface{}{"type": "integer", "description": "XP budget (0 for starter, e.g. 15 for an upgraded deck). Default: 0."},
					"strategy":         map[string]interface{}{"type": "string", "description": "Freetext strategy hint, e.g. 'fast cluever, minimal combat'. Used alongside popular decks to guide card selection."},
					"useTaboo":         map[string]interface{}{"type": "boolean", "description": "Enforce taboo list rules. Defaults to your collection config setting (default: false)."},
				},
				"required": []string{"investigatorCode"},
			},
		},
		{
			Name:        "arkhamdb_search_reference_decks",
			Description: "Search popular community decklists on ArkhamDB, sorted by likes. Filter by investigator code, XP spent range, and tags. Returns community-validated decks — use these as archetype references or starting points for deck building.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"investigatorCode": map[string]interface{}{"type": "string", "description": "Filter by investigator, e.g. '01001'. Empty = any."},
					"xpMin":            map[string]interface{}{"type": "integer", "description": "Minimum XP spent (0 = starter decks included). -1 or omit = no minimum."},
					"xpMax":            map[string]interface{}{"type": "integer", "description": "Maximum XP spent. -1 or omit = no maximum."},
					"tags":             map[string]interface{}{"type": "string", "description": "Comma-separated tags to search for, e.g. 'solo,beginner'"},
					"daysBack":         map[string]interface{}{"type": "integer", "description": "Unused, kept for backwards compatibility"},
					"maxResults":       map[string]interface{}{"type": "integer", "description": "Max results to return (default 10, max 50)"},
				},
			},
		},
		{
			Name:        "arkhamdb_get_upgrade_path",
			Description: "Given an existing deck and an XP budget, identify which cards can be upgraded to higher-level versions and produce a prioritised spending plan. Upgrades are scored by synergy with existing cards and sorted by value. Either deckID or decklistID must be provided.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"deckID":     map[string]interface{}{"type": "integer", "description": "Private deck ID"},
					"decklistID": map[string]interface{}{"type": "integer", "description": "Public decklist ID"},
					"xpBudget":   map[string]interface{}{"type": "integer", "description": "XP available to spend on upgrades"},
				},
			},
		},
		{
			Name:        "arkhamdb_validate_deck",
			Description: "Validate a deck or decklist for legality: checks deck size, required signature cards, investigator deck_options compliance (faction/level/trait/tag/limit rules), and per-card deck limits. Returns a list of errors and warnings. Use before finalising any deck.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"deckID":     map[string]interface{}{"type": "integer", "description": "Private deck ID"},
					"decklistID": map[string]interface{}{"type": "integer", "description": "Public decklist ID"},
					"useTaboo":   map[string]interface{}{"type": "boolean", "description": "Enforce taboo list rules. Defaults to your collection config setting (default: false)."},
				},
			},
		},
		{
			Name:        "arkhamdb_get_collection",
			Description: "Get your current collection config: owned cycles, language preference, and taboo setting.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "arkhamdb_set_collection",
			Description: "Save your collection: which cycles/packs you own, language ('es' or 'en'), and whether to enforce taboo rules. OwnedCycles is a list of cycle codes e.g. [\"core\",\"dwl\",\"ptc\",\"tfa\",\"tcu\",\"tde\",\"tic\",\"eoe\",\"tsk\",\"fhv\"]. Chapter 1 = all of those. Miguel de la Cruz starter = add [\"mdc\"].",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ownedCycles": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "string"},
						"description": "Cycle/pack codes you own",
					},
					"language": map[string]interface{}{
						"type":        "string",
						"description": "'es' for Spanish card names, 'en' for English (default)",
					},
					"useTaboo": map[string]interface{}{
						"type":        "boolean",
						"description": "Enforce the taboo list (default false)",
					},
				},
				"required": []string{"ownedCycles"},
			},
		},
		{
			Name:        "arkhamdb_adapt_deck_to_collection",
			Description: "Take a public decklist and identify which cards you don't own, then suggest owned replacements of the same faction, type, and XP level. Run this after arkhamdb_search_reference_decks to adapt a popular deck to your collection.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"decklistID": map[string]interface{}{
						"type":        "integer",
						"description": "Public decklist ID from ArkhamDB",
					},
				},
				"required": []string{"decklistID"},
			},
		},
	}
}

// executeTool executes the specified tool with given arguments
func (s *MCPServer) executeTool(name string, args map[string]interface{}) (string, error) {
	switch name {
	case "arkhamdb_get_card":
		cardCode, ok := args["cardCode"].(string)
		if !ok {
			return "", fmt.Errorf("cardCode must be a string")
		}
		return s.ArkhamDB.GetCard(cardCode)

	case "arkhamdb_search_cards_by_name":
		cardName, ok := args["name"].(string)
		if !ok {
			return "", fmt.Errorf("name must be a string")
		}
		return s.ArkhamDB.SearchCardsByName(cardName)

	case "arkhamdb_get_deck":
		deckID, ok := args["deckID"]
		if !ok {
			return "", fmt.Errorf("deckID is required")
		}
		var id int
		switch v := deckID.(type) {
		case float64:
			id = int(v)
		case int:
			id = v
		case string:
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return "", fmt.Errorf("deckID must be a valid integer: %w", err)
			}
			id = parsed
		default:
			return "", fmt.Errorf("deckID must be an integer")
		}
		return s.ArkhamDB.GetDeck(id)

	case "arkhamdb_get_decklist":
		decklistID, ok := args["decklistID"]
		if !ok {
			return "", fmt.Errorf("decklistID is required")
		}
		var id int
		switch v := decklistID.(type) {
		case float64:
			id = int(v)
		case int:
			id = v
		case string:
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return "", fmt.Errorf("decklistID must be a valid integer: %w", err)
			}
			id = parsed
		default:
			return "", fmt.Errorf("decklistID must be an integer")
		}
		return s.ArkhamDB.GetDecklist(id)

	case "arkhamdb_find_card_synergies":
		cardCode, ok := args["cardCode"].(string)
		if !ok {
			return "", fmt.Errorf("cardCode must be a string")
		}
		maxResults := 10 // default
		if maxResultsVal, ok := args["maxResults"]; ok {
			switch v := maxResultsVal.(type) {
			case float64:
				maxResults = int(v)
			case int:
				maxResults = v
			case string:
				parsed, err := strconv.Atoi(v)
				if err != nil {
					return "", fmt.Errorf("maxResults must be a valid integer: %w", err)
				}
				maxResults = parsed
			}
		}
		return s.ArkhamDB.FindCardSynergies(cardCode, maxResults)

	case "arkhamdb_suggest_deck_improvements":
		var deckID *int
		var decklistID *int

		// Parse deckID if provided
		if deckIDVal, ok := args["deckID"]; ok && deckIDVal != nil {
			var id int
			switch v := deckIDVal.(type) {
			case float64:
				id = int(v)
			case int:
				id = v
			case string:
				parsed, err := strconv.Atoi(v)
				if err != nil {
					return "", fmt.Errorf("deckID must be a valid integer: %w", err)
				}
				id = parsed
			default:
				return "", fmt.Errorf("deckID must be an integer")
			}
			deckID = &id
		}

		// Parse decklistID if provided
		if decklistIDVal, ok := args["decklistID"]; ok && decklistIDVal != nil {
			var id int
			switch v := decklistIDVal.(type) {
			case float64:
				id = int(v)
			case int:
				id = v
			case string:
				parsed, err := strconv.Atoi(v)
				if err != nil {
					return "", fmt.Errorf("decklistID must be a valid integer: %w", err)
				}
				id = parsed
			default:
				return "", fmt.Errorf("decklistID must be an integer")
			}
			decklistID = &id
		}

		// At least one must be provided
		if deckID == nil && decklistID == nil {
			return "", fmt.Errorf("either deckID or decklistID must be provided")
		}

		maxResults := 20 // default
		if maxResultsVal, ok := args["maxResults"]; ok {
			switch v := maxResultsVal.(type) {
			case float64:
				maxResults = int(v)
			case int:
				maxResults = v
			case string:
				parsed, err := strconv.Atoi(v)
				if err != nil {
					return "", fmt.Errorf("maxResults must be a valid integer: %w", err)
				}
				maxResults = parsed
			}
		}

		strategy, _ := args["strategy"].(string)
		return s.ArkhamDB.SuggestDeckImprovements(deckID, decklistID, maxResults, strategy)

	case "arkhamdb_search_cards_advanced":
		chapter := 0
		if v, ok := args["chapter"]; ok && v != nil {
			chapter = int(floatFromArg(v))
		}
		cycleCode, _ := args["cycleCode"].(string)
		factionCode, _ := args["factionCode"].(string)
		typeCode, _ := args["typeCode"].(string)
		xpMin := intFromArgDefault(args["xpMin"], -1)
		xpMax := intFromArgDefault(args["xpMax"], -1)
		costMin := intFromArgDefault(args["costMin"], -1)
		costMax := intFromArgDefault(args["costMax"], -1)
		traits := stringSliceFromArg(args["traits"])
		tags := stringSliceFromArg(args["tags"])
		textSearch, _ := args["textSearch"].(string)
		maxResults := intFromArgDefault(args["maxResults"], 50)
		return s.ArkhamDB.SearchCardsAdvanced(chapter, cycleCode, factionCode, typeCode, xpMin, xpMax, costMin, costMax, traits, tags, textSearch, maxResults)

	case "arkhamdb_get_packs_and_cycles":
		return s.ArkhamDB.GetPacksAndCycles()

	case "arkhamdb_get_investigator_constraints":
		code, ok := args["investigatorCode"].(string)
		if !ok {
			return "", fmt.Errorf("investigatorCode must be a string")
		}
		return s.ArkhamDB.GetInvestigatorConstraints(code)

	case "arkhamdb_build_starter_deck":
		invCode, ok := args["investigatorCode"].(string)
		if !ok {
			return "", fmt.Errorf("investigatorCode must be a string")
		}
		chapter := 0
		if v, ok := args["chapter"]; ok && v != nil {
			chapter = int(floatFromArg(v))
		}
		cycleCodes := stringSliceFromArg(args["cycleCodes"])
		xpBudget := intFromArgDefault(args["xpBudget"], 0)
		strategy, _ := args["strategy"].(string)
		var tabooOverride *bool
		if v, ok := args["useTaboo"].(bool); ok {
			tabooOverride = &v
		}
		return s.ArkhamDB.BuildStarterDeck(invCode, chapter, cycleCodes, xpBudget, strategy, tabooOverride)

	case "arkhamdb_search_reference_decks":
		invCode, _ := args["investigatorCode"].(string)
		xpMin := intFromArgDefault(args["xpMin"], -1)
		xpMax := intFromArgDefault(args["xpMax"], -1)
		tags, _ := args["tags"].(string)
		daysBack := intFromArgDefault(args["daysBack"], 30)
		maxResults := intFromArgDefault(args["maxResults"], 10)
		return s.ArkhamDB.SearchReferenceDecks(invCode, xpMin, xpMax, tags, daysBack, maxResults)

	case "arkhamdb_get_upgrade_path":
		var deckID *int
		var decklistID *int
		if v, ok := args["deckID"]; ok && v != nil {
			id := int(floatFromArg(v))
			deckID = &id
		}
		if v, ok := args["decklistID"]; ok && v != nil {
			id := int(floatFromArg(v))
			decklistID = &id
		}
		if deckID == nil && decklistID == nil {
			return "", fmt.Errorf("either deckID or decklistID must be provided")
		}
		xpBudget := intFromArgDefault(args["xpBudget"], 0)
		return s.ArkhamDB.GetUpgradePath(deckID, decklistID, xpBudget)

	case "arkhamdb_validate_deck":
		var deckID *int
		var decklistID *int
		if v, ok := args["deckID"]; ok && v != nil {
			id := int(floatFromArg(v))
			deckID = &id
		}
		if v, ok := args["decklistID"]; ok && v != nil {
			id := int(floatFromArg(v))
			decklistID = &id
		}
		if deckID == nil && decklistID == nil {
			return "", fmt.Errorf("either deckID or decklistID must be provided")
		}
		var tabooOverride *bool
		if v, ok := args["useTaboo"].(bool); ok {
			tabooOverride = &v
		}
		return s.ArkhamDB.ValidateDeck(deckID, decklistID, tabooOverride)

	case "arkhamdb_get_collection":
		return s.ArkhamDB.GetCollection()

	case "arkhamdb_set_collection":
		cycles := stringSliceFromArg(args["ownedCycles"])
		lang, _ := args["language"].(string)
		useTaboo, _ := args["useTaboo"].(bool)
		return s.ArkhamDB.SetCollection(cycles, lang, useTaboo)

	case "arkhamdb_adapt_deck_to_collection":
		id := intFromArgDefault(args["decklistID"], 0)
		if id == 0 {
			return "", fmt.Errorf("decklistID is required")
		}
		return s.ArkhamDB.AdaptDeckToCollection(id)

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
