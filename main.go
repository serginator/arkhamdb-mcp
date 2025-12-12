package main

import (
	"arkhamdb-mcp/arkhamdb"
	"arkhamdb-mcp/server"
	"log"
	"os"
)

func main() {
	// Ensure all logs go to stderr, not stdout
	log.SetOutput(os.Stderr)

	arkhamdbClient := arkhamdb.NewArkhamDBClient("https://es.arkhamdb.com")

	log.Println("Starting MCP ArkhamDB server...")

	srv := &server.MCPServer{
		ArkhamDB: arkhamdbClient,
	}
	srv.Start()
}
