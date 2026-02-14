.PHONY: help build lint test report example example-report

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the grif CLI binary
	go build -o grif ./src/cmd/grif

lint: ## Run golangci-lint
	golangci-lint run ./...

test: ## Run tests
	go test ./...

example: build ## Set up example repo in testdata/example
	rm -rf testdata/example
	mkdir -p testdata/example
	cp ./grif testdata/example/
	cd testdata/example && \
		git init && \
		touch readme.md && \
		git add readme.md && \
		git commit -m "Add README" && \
		./grif init default

example-report: ## Show the infra ref, its commit, and root tree
	@cd testdata/example && \
		git cat-file -p refs/infra/default^{commit}