#!/bin/bash

# Test script for MCP ArkhamDB server
echo "Testing MCP ArkhamDB Server"
echo "============================"

# Test initialize
echo "1. Testing initialize..."
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}' | ./arkhamdb-mcp | jq '.result.serverInfo'

echo -e "\n2. Testing tools/list..."
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./arkhamdb-mcp | tail -1 | jq '.result.tools | length'

echo -e "\n3. Testing ArkhamDB get card..."
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"arkhamdb_get_card","arguments":{"cardCode":"01001"}}}' | timeout 10s ./arkhamdb-mcp | tail -1 | jq '.result.content[0].text' | head -10

echo -e "\n4. Testing ArkhamDB search cards by name..."
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"arkhamdb_search_cards_by_name","arguments":{"name":"Research Librarian"}}}' | timeout 30s ./arkhamdb-mcp | tail -1 | jq '.result.content[0].text' | head -20

echo -e "\nMCP ArkhamDB Server tests completed!"

