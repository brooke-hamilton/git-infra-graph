.PHONY: help build lint test

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the grif CLI binary
	go build -o grif ./src/cmd/grif

lint: ## Run golangci-lint
	golangci-lint run ./...

test: ## Run tests
	go test ./...
