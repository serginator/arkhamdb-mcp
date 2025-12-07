APPLICATION_NAME=mcp-arkhamdb

.DEFAULT_GOAL := help

bootstrap: ## Bootstrap local development
.PHONY: bootstrap

dependencies: bootstrap ## Install all deps
	go mod download
	go mod verify
.PHONY: dependencies

lint: ## Lint code
	golangci-lint run --config golanci.config.yml
.PHONY: lint

format: ## Format code
	go mod tidy
	go fmt ./...
.PHONY: format

inspect: ## Inspect code
	npx @modelcontextprotocol/inspector
.PHONY: inspect

build: ## Build with docker
	docker build -t mcp-arkhamdb:latest .
	docker-compose build
.PHONY: build

