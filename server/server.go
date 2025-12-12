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
				},
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

		return s.ArkhamDB.SuggestDeckImprovements(deckID, decklistID, maxResults)

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
