# Quickstart: Graph Commit History (`grif log`)

**Feature**: 003-graph-log

## Prerequisites

- Go 1.26+ installed
- A Git repository with at least one commit
- A graph initialized via `grif init` with at least one commit (see features 001 and 002)

## Build

```bash
go build -o grif ./src/cmd/grif
```

## Usage

### View full commit history

```bash
# Initialize a graph and make some commits
./grif init default
./grif put default/network/vpc --data "10.0.0.0/16"
./grif commit default --message "Add network resources"
./grif put default/compute/instance --data '{"type": "t3.micro"}'
./grif commit default --message "Add compute instance"

# View the full log
./grif log default
# Output:
# commit a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
# Date:   2026-02-14 10:30:00 -0700
# Source: f9e8d7c6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0
#
#     Add compute instance
#
# commit c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2
# Date:   2026-02-14 10:15:00 -0700
# Source: d2c1b0a9e8f7d6c5b4a3c2d1e0f9a8b7c6d5e4f3
#
#     Add network resources
#
# commit e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4
# Date:   2026-02-14 10:00:00 -0700
# Source: b0a9e8f7d6c5b4a3c2d1e0f9a8b7c6d5e4f3a2b1
#
#     Initialize graph "default"
```

### Compact one-line view

```bash
./grif log --oneline default
# Output:
# a1b2c3d4 Add compute instance
# c3d4e5f6 Add network resources
# e5f6a7b8 Initialize graph "default"
```

### Limit number of commits

```bash
# Show only the 2 most recent commits
./grif log --max-count 2 default
# Output:
# commit a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
# Date:   2026-02-14 10:30:00 -0700
# Source: f9e8d7c6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0
#
#     Add compute instance
#
# commit c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2
# Date:   2026-02-14 10:15:00 -0700
# Source: d2c1b0a9e8f7d6c5b4a3c2d1e0f9a8b7c6d5e4f3
#
#     Add network resources

# Combine with oneline
./grif log --oneline --max-count 2 default
# Output:
# a1b2c3d4 Add compute instance
# c3d4e5f6 Add network resources
```

### JSON output

```bash
./grif log --json default
# Output:
# [
#   {
#     "hash": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
#     "date": "2026-02-14T10:30:00-07:00",
#     "sourceCommit": "f9e8d7c6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0",
#     "author": "git-infra-graph",
#     "message": "Add compute instance\n\nSource-Commit: f9e8d7c6...\n"
#   },
#   ...
# ]

# JSON with max-count
./grif log --json --max-count 1 default
# Output:
# [
#   {
#     "hash": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
#     "date": "2026-02-14T10:30:00-07:00",
#     "sourceCommit": "f9e8d7c6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0",
#     "author": "git-infra-graph",
#     "message": "Add compute instance\n\nSource-Commit: f9e8d7c6...\n"
#   }
# ]
```

### Default graph selection

```bash
# When only one graph exists, the name is optional
./grif log
# Output: (same as ./grif log default)

# When multiple graphs exist
./grif log
# Error: multiple graphs exist; specify a graph name
```

## Error Handling

All errors print to stderr and exit with code 1:

```bash
# Graph not found
./grif log nonexistent
# Error: graph 'nonexistent' not found

# No graphs exist
./grif log
# Error: no graphs found

# Negative max-count
./grif log --max-count -1 default
# Error: max-count must be a non-negative integer
```

Broken chain warnings print to stderr with exit code 0:

```bash
# Broken commit chain (partial result)
./grif log default
# (displays reachable commits to stdout)
# Warning: broken commit chain at abc123...: commit object not found
```

## Using as a Go Library

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/brooke-hamilton/git-infra-graph/src/graph"
)

func main() {
    repoPath := "."

    // Full log
    result, err := graph.Log(repoPath, "default", graph.LogOptions{})
    if err != nil {
        log.Fatal(err)
    }

    for _, entry := range result.Entries {
        fmt.Printf("%s %s\n", entry.Hash[:8], firstLine(entry.Message))
    }

    if result.Warning != "" {
        fmt.Fprintf(os.Stderr, "Warning: %s\n", result.Warning)
    }

    // Limited log
    result, err = graph.Log(repoPath, "default", graph.LogOptions{
        MaxCount:    5,
        HasMaxCount: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, entry := range result.Entries {
        fmt.Printf("%s %s %s\n",
            entry.Hash[:8],
            entry.Date.Format(time.RFC3339),
            firstLine(entry.Message),
        )
    }
}

func firstLine(s string) string {
    if i := strings.Index(s, "\n"); i >= 0 {
        return s[:i]
    }
    return s
}
```
