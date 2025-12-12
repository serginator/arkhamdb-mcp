# MCP ArkhamDB Server

An MCP (Model Context Protocol) server for interacting with the ArkhamDB public API. This server provides tools to query cards and decks from Arkham Horror: The Card Game.

## Features

- **Get Card**: Retrieve details of a specific card by its code
- **Search Cards by Name**: Search for cards by name (case-insensitive partial match)
- **Get Deck**: Retrieve a deck by its ID (may require authentication)
- **Get Decklist**: Retrieve a decklist by its ID

## Example of promts using the MCP server

### In Spanish

```
Utiliza el mcp de arkhamdb para bajar información sobre el siguiente mazo: 5570650

Quiero que analices todas sus cartas, sus sinergias, y me expliques sus puntos fuertes y sus debilidades, y cómo utilizarlo y jugar este mazo, incluyendo información sobre qué mulligan debería hacer. Puedes buscar en internet información sobre Arkham Horror LCG. Los nombres de las cartas y la explicación han de estar en español.
```

Result: [examples/darrell-spanish.md](examples/darrell-spanish.md)

### In English

```
Use the arkhamdb mcp to get information about the following deck: 5570650

I want you to analyze all its cards, its synergies, and explain its strengths and weaknesses, and how to use it and play this deck, including information about what mulligan should be made. You can search the internet for information about Arkham Horror LCG.
```

Result: [examples/darrell-english.md](examples/darrell-english.md)

## Installation

1. Clone this repository
2. Install dependencies:

```bash
make dependencies
```

Or manually:

```bash
go mod download
```

3. Build the server:

```bash
make build
```

Or manually:

```bash
go build -o arkhamdb-mcp
```

## Running with Docker

1. Build and start the server:

```bash
docker-compose up --build
```

2. The server will listen on stdin/stdout using the MCP protocol.

For Cursor integration, you can use the `run-docker-mcp.sh` script:

```json
{
  "mcpServers": {
    "arkhamdb": {
      "command": "<path-to>/arkhamdb-mcp/run-docker-mcp.sh",
      "args": []
    }
  }
}
```

## Usage

The server communicates via stdin/stdout using the MCP protocol. It connects to the ArkhamDB API at `https://es.arkhamdb.com`.

### Testing

Use the provided test script:

```bash
./test-mcp.sh
```

Or test manually using JSON-RPC over stdin:

#### Initialize the server:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}' | ./arkhamdb-mcp
```

#### List available tools:

```bash
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./arkhamdb-mcp
```

#### Get a card:

```bash
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"arkhamdb_get_card","arguments":{"cardCode":"01001"}}}' | ./arkhamdb-mcp
```

#### Search cards by name:

```bash
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"arkhamdb_search_cards_by_name","arguments":{"name":"Roland"}}}' | ./arkhamdb-mcp
```

#### Get a deck:

```bash
echo -e '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"arkhamdb_get_deck","arguments":{"deckID":12345}}}' | ./arkhamdb-mcp
```

## Integration with AI Models

This MCP server can be integrated with:

- Cursor
- Claude Desktop
- ChatGPT with MCP support
- Any MCP-compatible AI client

Configure your AI client to connect to this server using the MCP protocol over stdio. For Cursor, add to your `mcp.json`:

```json
{
  "mcpServers": {
    "arkhamdb": {
      "command": "<path-to>/arkhamdb-mcp/arkhamdb-mcp",
      "args": []
    }
  }
}
```

## API Reference

The server uses the public ArkhamDB API endpoints:

- `GET /api/public/card/{card_code}.json` - Get a single card
- `GET /api/public/cards/` - Get all cards (used for searching by name)
- `GET /api/public/deck/{deck_id}.json` - Get a deck (may require authentication)
- `GET /api/public/decklist/{decklist_id}.json` - Get a decklist

For more information, see the [ArkhamDB API documentation](https://es.arkhamdb.com/api/doc).

## Development

### Code Quality

- **Lint**: Run `make lint` to check code quality
- **Format**: Run `make format` to format code
- **Inspect**: Run `make inspect` to inspect MCP protocol compliance

### Makefile Targets

- `make dependencies` - Install all dependencies
- `make lint` - Run linter
- `make format` - Format code
- `make inspect` - Inspect MCP protocol compliance
- `make build` - Build Docker image

## Architecture

The server follows a modular architecture:

- `main.go` - Entry point and initialization
- `server/` - MCP protocol implementation
- `tools/` - Tool interface definitions
- `arkhamdb/` - ArkhamDB API client implementation

## License

MIT License
