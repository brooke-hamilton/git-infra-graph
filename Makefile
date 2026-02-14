.PHONY: help build lint test report example example-report

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the grif CLI binary
	go build -o grif ./src/cmd/grif

lint: ## Run golangci-lint
	golangci-lint run ./...

test: ## Run tests
	go test ./...

example: build ## Set up example repo with put/get/commit workflow
	rm -rf testdata/example
	mkdir -p testdata/example
	cp ./grif testdata/example/
	cd testdata/example && \
		git init && \
		touch readme.md && \
		git add readme.md && \
		git commit -m "Add README" && \
		./grif init default && \
		./grif put default/network/vpc --data "10.0.0.0/16" && \
		./grif put default/network/subnet --data "10.0.1.0/24" && \
		./grif put default/compute/instance --data '{"type": "t3.micro"}' && \
		./grif status default && \
		./grif commit default --message "Set up initial infrastructure" && \
		./grif get default/network && \
		./grif get default/network/vpc

example-report: ## Show the infra ref, its commit, and root tree
	@cd testdata/example && \
		echo "=== Graph Ref ===" && \
		git cat-file -p refs/infra/default^{commit} && \
		echo "" && \
		echo "=== Root Tree ===" && \
		git cat-file -p refs/infra/default^{tree}