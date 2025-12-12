#!/bin/bash

# Wrapper script to run MCP server in Docker for Cursor integration
# This allows Cursor to spawn the Docker container as if it were a local process

cd "$(dirname "$0")"
docker run --rm -i \
  arkhamdb-mcp:latest

