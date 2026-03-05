.PHONY: help build install lint test report scratch example-report

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the grif CLI binary
	go build -o grif ./src/cmd/grif

install: ## Install grif to GOPATH/bin for use anywhere
	go install ./src/cmd/grif

lint: ## Run golangci-lint
	golangci-lint run ./...

test: ## Run tests
	go test ./...

scratch: install ## Create a scratch Git repo for interactive testing
	@if [ -d testdata/scratch/.git ]; then \
		echo "Scratch repo already exists at testdata/scratch"; \
	else \
		mkdir -p testdata/scratch && \
		cd testdata/scratch && \
		git init && \
		touch readme.md && \
		git add readme.md && \
		git commit -m "Initial commit" && \
		echo "Scratch repo created at testdata/scratch"; \
	fi
	@echo "Run 'make install' then 'cd testdata/scratch' to start using grif"

example-report: ## Show the infra ref, its commit, and root tree
	@cd testdata/example && \
		echo "=== Graph Ref ===" && \
		git cat-file -p refs/infra/default^{commit} && \
		echo "" && \
		echo "=== Root Tree ===" && \
		git cat-file -p refs/infra/default^{tree}