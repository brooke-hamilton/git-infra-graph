# Quickstart: Graph Init, List, and Delete

**Feature**: 001-graph-init-list

## Prerequisites

- Go 1.22+ installed
- A Git repository with at least one commit

## Build

```bash
go build -o grif ./cmd/grif
```

## Usage

### Initialize a graph

```bash
# Create a new graph called "my-infra"
./grif init my-infra
# Output: Initialized graph "my-infra"

# JSON output
./grif init staging --json
# Output: {"name": "staging", "ref": "refs/infra/staging", "sourceCommit": "a1b2c3..."}
```

### List graphs

```bash
# List all graphs (alphabetical order)
./grif list
# Output:
# my-infra
# staging

# JSON output
./grif list --json
# Output: ["my-infra", "staging"]

# Empty repository (no graphs)
./grif list
# (no output, exit code 0)
```

### Delete a graph

```bash
# Delete a graph
./grif delete staging
# Output: Deleted graph "staging"

# JSON output
./grif delete my-infra --json
# Output: {"name": "my-infra", "deleted": true}
```

## Error Handling

All errors print to stderr and exit with code 1:

```bash
# Not in a git repo
./grif init my-infra
# Error: not a git repository: /home/user/not-a-repo

# Duplicate graph name
./grif init my-infra
./grif init my-infra
# Error: graph 'my-infra' already exists

# Invalid graph name
./grif init "my graph"
# Error: graph name must not contain ' '

# Delete non-existent graph
./grif delete nonexistent
# Error: graph 'nonexistent' not found
```

## Using as a Go Library

```go
package main

import (
    "fmt"
    "log"

    "github.com/brooke-hamilton/grif/graph"
)

func main() {
    repoPath := "."

    // Initialize a graph
    if err := graph.Init(repoPath, "my-infra"); err != nil {
        log.Fatal(err)
    }

    // List all graphs
    graphs, err := graph.List(repoPath)
    if err != nil {
        log.Fatal(err)
    }
    for _, g := range graphs {
        fmt.Println(g.Name)
    }

    // Delete a graph
    if err := graph.Delete(repoPath, "my-infra"); err != nil {
        log.Fatal(err)
    }
}
```

## Running Tests

```bash
# Run all tests (including integration tests with live Git repos)
go test ./...

# Run tests in parallel
go test -parallel 4 ./...

# Verbose output (shows temp repo paths for debugging)
go test -v ./...
```
