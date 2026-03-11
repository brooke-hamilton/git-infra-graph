# Git Infrastructure Graph (grif)

Infrastructure graph management backed by Git-native storage.

Git Infrastructure Graph is a versioned graph database that stores infrastructure topology directly in Git's object database — blobs hold node payloads, trees encode containment hierarchy, commits capture immutable point-in-time snapshots, and refs under a dedicated `refs/infra/` namespace track graph lineage — so that infrastructure entities and their relationships live alongside IaC source code with full history, structural sharing via content-addressable deduplication, and distributed collaboration through standard Git push/pull mechanics, all without provisioning an external database, running a daemon, or managing additional credentials.

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

### Store a node

```bash
grif put <path> [--data <content>] [--file <file>] [--json]
```

Creates or replaces a node at the specified path. The first segment of `<path>`
is the graph name; remaining segments address nodes within that graph's tree.

Blob content sources (mutually exclusive, checked in this order):

1. `--data <content>` — inline string content
2. `--file <file>` — read content from a file
3. Piped stdin — if stdin is not a terminal, read from stdin
4. None of the above — creates a tree node (nil blob)

```bash
# Put a blob with inline data
grif put default/network/vpc --data "10.0.0.0/16"
# Output: Put blob at "default/network/vpc" (id: a1b2c3d4)

# Put a blob from a file
grif put default/network/vpc --file vpc.json
# Output: Put blob at "default/network/vpc" (id: e5f6a7b8)

# Put a blob from stdin
echo "172.16.0.0/12" | grif put default/network/vpc
# Output: Put blob at "default/network/vpc" (id: c3d4e5f6)

# Create a tree node (no data)
grif put default/network
# Output: Put tree at "default/network" (id: f9e8d7c6)

# JSON output
grif put default/network/vpc --data "10.0.0.0/16" --json
# Output: {"id":"a1b2c3d4...","path":"default/network/vpc","type":"blob"}
```

### Read a node

```bash
grif get <path> [--json]
```

Reads the node at the specified path. For blobs, prints raw content. For trees,
lists children in a tabular format.

```bash
# Get a blob — prints raw content
grif get default/network/vpc
# Output: 10.0.0.0/16

# Get a tree — lists children
grif get default/network
# Output:
# TYPE  NAME    ID
# blob  vpc     a1b2c3d4
# tree  subnet  e5f6a7b8

# JSON output
grif get default/network/vpc --json
# Output: {"id":"a1b2c3d4...","path":"default/network/vpc","type":"blob","content":"..."}
```

### Remove a node

```bash
grif rm <path> [--json]
```

Removes a node at the specified path. If the node is a tree, all descendants
are recursively removed.

```bash
# Remove a blob
grif rm default/network/vpc
# Output: Removed "default/network/vpc"

# Remove a tree (recursively)
grif rm default/network
# Output: Removed "default/network" (tree, 3 descendants)

# JSON output
grif rm default/network/vpc --json
# Output: {"path":"default/network/vpc","removed":true}
```

### Commit changes

```bash
grif commit <graph> [--message <msg>] [--json]
```

Commits all staged changes for the specified graph. If no message is provided,
a default message is generated with a `Source-Commit` trailer referencing HEAD.

```bash
# Commit with default message
grif commit default
# Output: Committed graph "default" (commit: c3d4e5f6)

# Commit with custom message
grif commit default --message "Add network configuration"
# Output: Committed graph "default" (commit: d4e5f6a7)

# JSON output
grif commit default --json
# Output: {"graph":"default","commit":"c3d4e5f6...","ref":"refs/infra/default"}
```

### Check status

```bash
grif status <graph> [--json]
```

Shows uncommitted changes for the specified graph.

```bash
# Show uncommitted changes
grif status default
# Output:
# Changes for graph "default":
#   added:    network/vpc
#   added:    network/subnet

# No changes
grif status default
# Output: No uncommitted changes for graph "default"

# JSON output
grif status default --json
# Output: {"graph":"default","changes":[{"path":"network/vpc","status":"added"}]}
```

### View commit history

```bash
grif log [graph] [--oneline] [--max-count N] [--json]
```

Displays the commit history for a graph in reverse chronological order.

If only one graph exists, the graph name is optional.

```bash
# Full log
grif log default
# Output:
# commit a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
# Date:   2026-02-14 10:30:00 -0700
# Source: f9e8d7c6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0
#
#     Add network resources
#
# commit e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4
# Date:   2026-02-14 10:00:00 -0700
# Source: b0a9e8f7d6c5b4a3c2d1e0f9a8b7c6d5e4f3a2b1
#
#     Initialize graph "default"

# Compact one-line format
grif log --oneline default
# Output:
# a1b2c3d4 Add network resources
# e5f6a7b8 Initialize graph "default"

# Limit to N most recent commits
grif log --max-count 1 default

# JSON output
grif log --json default
# Output: [{"hash":"a1b2...","date":"2026-02-14T10:30:00-07:00","sourceCommit":"f9e8...","author":"git-infra-graph","message":"..."}]

# Combine flags
grif log --oneline --max-count 5 default
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

    // Store a blob node
    result, err := graph.Put(repoPath, "my-infra/network/vpc", []byte("10.0.0.0/16"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created %s at %s (id: %s)\n", result.Type, result.Path, result.ID)

    // Read it back
    content, err := graph.Get(repoPath, "my-infra/network/vpc")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Content: %s\n", string(content.Blob))

    // Check what changed
    status, err := graph.Status(repoPath, "my-infra")
    if err != nil {
        log.Fatal(err)
    }
    for _, c := range status.Changes {
        fmt.Printf("  %s: %s\n", c.Status, c.Path)
    }

    // Commit changes
    commitResult, err := graph.Commit(repoPath, "my-infra", "Add VPC")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Committed: %s\n", commitResult.Commit)

    // Delete a node and commit
    if err := graph.DeleteNode(repoPath, "my-infra/network/vpc"); err != nil {
        log.Fatal(err)
    }
    if _, err := graph.Commit(repoPath, "my-infra", "Remove VPC"); err != nil {
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
