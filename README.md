# Git Infrastructure Graph (grif)

Infrastructure graph management backed by Git-native storage.

`git-infra-graph` stores infrastructure graphs as Git objects (trees, commits, refs)
under a custom `refs/infra/` namespace. No external database required — all
data lives in your Git repository.

## Installation

```bash
go install github.com/brooke-hamilton/git-infra-graph/src/cmd/grif@latest
```

Or build from source:

```bash
make build
```

## Prerequisites

- A Git repository with at least one commit

## Commands

### Initialize a graph

```bash
grif init <name> [--json]
```

Creates a new infrastructure graph. The command creates an empty root tree, an
orphan commit with a `Source-Commit` trailer referencing the current HEAD, and a
ref at `refs/infra/<name>`.

```bash
# Human-readable output
grif init my-infra
# Output: Initialized graph "my-infra"

# JSON output
grif init staging --json
# Output: {"name":"staging","ref":"refs/infra/staging","sourceCommit":"a1b2c3..."}
```

### List graphs

```bash
grif list [--json]
```

Lists all infrastructure graphs in alphabetical order.

```bash
# Human-readable output (one name per line)
grif list
# Output:
# my-infra
# staging

# JSON output
grif list --json
# Output: ["my-infra","staging"]

# Empty repository (no output, exit code 0)
grif list
```

### Delete a graph

```bash
grif delete <name> [--json]
```

Removes a graph by deleting its ref. Unreferenced Git objects are left for
`git gc`.

```bash
# Human-readable output
grif delete staging
# Output: Deleted graph "staging"

# JSON output
grif delete my-infra --json
# Output: {"deleted":true,"name":"my-infra"}
```

## Error Handling

All errors print to stderr and exit with code 1.

In human-readable mode:

```text
Error: graph 'my-infra' already exists
```

In JSON mode:

```json
{"error": "graph 'my-infra' already exists"}
```

## Using as a Go Library

```go
package main

import (
    "fmt"
    "log"

    "github.com/brooke-hamilton/git-infra-graph/src/graph"
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

## Development

Run `make` to see all available targets:

```bash
make
```

Common targets:

```bash
# Build all packages
make build

# Run all tests
make test

# Run linter
make lint
```

## License

See [LICENSE](LICENSE) for details.
