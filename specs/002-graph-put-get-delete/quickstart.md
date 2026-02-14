# Quickstart: Graph Node Put, Get, and Delete

**Feature**: 002-graph-put-get-delete

## Prerequisites

- Go 1.22+ installed
- A Git repository with at least one commit
- A graph initialized via `grif init` (see feature 001)

## Build

```bash
go build -o grif ./src/cmd/grif
```

## Usage

### Store a blob node

```bash
# Initialize a graph first
./grif init default

# Put a blob node with inline data
./grif put default/network/vpc --data "10.0.0.0/16"
# Output: Put blob at "default/network/vpc" (id: a1b2c3d4)

# Put a blob from a file
echo '{"cidr": "10.0.0.0/16"}' > vpc.json
./grif put default/network/vpc --file vpc.json
# Output: Put blob at "default/network/vpc" (id: e5f6a7b8)

# Put a blob from stdin
echo "172.16.0.0/12" | ./grif put default/network/vpc
# Output: Put blob at "default/network/vpc" (id: c3d4e5f6)

# JSON output
./grif put default/network/vpc --data "10.0.0.0/16" --json
# Output: {"id":"a1b2c3d4...","path":"default/network/vpc","type":"blob"}
```

### Create a tree node

```bash
# Create a tree node (no data = tree)
./grif put default/network
# Output: Put tree at "default/network" (id: f9e8d7c6)
```

### Read a node

```bash
# Get a blob — prints raw content
./grif get default/network/vpc
# Output: 10.0.0.0/16

# Get a tree — lists children
./grif get default/network
# Output:
# TYPE  NAME    ID
# blob  vpc     a1b2c3d4
# tree  subnet  e5f6a7b8

# JSON output for a blob
./grif get default/network/vpc --json
# Output: {"id":"a1b2c3d4...","path":"default/network/vpc","type":"blob","content":"10.0.0.0/16"}

# JSON output for a tree
./grif get default/network --json
# Output: {"id":"f9e8d7c6...","path":"default/network","type":"tree","children":[{"name":"vpc","type":"blob","id":"a1b2c3d4..."}]}
```

### Delete a node

```bash
# Remove a blob node
./grif rm default/network/vpc
# Output: Removed "default/network/vpc"

# Remove a tree node (recursively deletes all descendants)
./grif rm default/network
# Output: Removed "default/network" (tree, 3 descendants)

# JSON output
./grif rm default/network/vpc --json
# Output: {"path":"default/network/vpc","removed":true}
```

### Check status

```bash
# Show uncommitted changes
./grif status default
# Output:
# Changes for graph "default":
#   added:    network/vpc
#   added:    network/subnet

# No changes
./grif status default
# Output: No uncommitted changes for graph "default"

# JSON output
./grif status default --json
# Output: {"graph":"default","changes":[{"path":"network/vpc","status":"added"}]}
```

### Commit changes

```bash
# Commit with default message
./grif commit default
# Output: Committed graph "default" (commit: c3d4e5f6)

# Commit with custom message
./grif commit default --message "Add network configuration"
# Output: Committed graph "default" (commit: d4e5f6a7)

# JSON output
./grif commit default --json
# Output: {"graph":"default","commit":"c3d4e5f6...","ref":"refs/infra/default"}
```

## Complete Workflow Example

```bash
# 1. Initialize a git repo and a graph
git init my-project && cd my-project
git commit --allow-empty -m "Initial commit"
./grif init default

# 2. Build the graph structure
./grif put default/network/vpc --data "10.0.0.0/16"
./grif put default/network/subnet --data "10.0.1.0/24"
./grif put default/compute/instance --data '{"type": "t3.micro"}'

# 3. Check what has changed
./grif status default
# Changes for graph "default":
#   added:    compute/instance
#   added:    network/subnet
#   added:    network/vpc

# 4. Commit the changes
./grif commit default --message "Set up initial infrastructure"
# Committed graph "default" (commit: a1b2c3d4)

# 5. Read back the data
./grif get default/network
# TYPE  NAME    ID
# blob  subnet  e5f6a7b8
# blob  vpc     a1b2c3d4

./grif get default/network/vpc
# 10.0.0.0/16

# 6. Update a node
./grif put default/network/vpc --data "172.16.0.0/12"
./grif status default
# Changes for graph "default":
#   modified: network/vpc

./grif commit default
# Committed graph "default" (commit: b2c3d4e5)

# 7. Clean up
./grif rm default/compute/instance
./grif commit default --message "Remove compute instance"
```

## Error Handling

All errors print to stderr and exit with code 1:

```bash
# Node not found
./grif get default/network/missing
# Error: node not found at "default/network/missing"

# Blob traversal
./grif get default/network/vpc/child
# Error: blob at "default/network/vpc" cannot be traversed

# Type conversion forbidden
./grif put default/network --data "some data"  # network is a tree
# Error: cannot convert tree to blob at "default/network"

# Invalid path
./grif put default//vpc --data "data"
# Error: invalid path: empty segment in "default//vpc"

# No staged changes
./grif commit default
# Error: no staged changes for graph "default"
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
    if err := graph.Init(repoPath, "default"); err != nil {
        log.Fatal(err)
    }

    // Put a blob node
    result, err := graph.Put(repoPath, "default/network/vpc", []byte("10.0.0.0/16"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created %s node at %s (id: %s)\n", result.Type, result.Path, result.ID)

    // Put a tree node
    result, err = graph.Put(repoPath, "default/network/security", nil)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created %s node at %s (id: %s)\n", result.Type, result.Path, result.ID)

    // Get a blob
    content, err := graph.Get(repoPath, "default/network/vpc")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Content: %s\n", string(content.Blob))

    // Get a tree (list children)
    content, err = graph.Get(repoPath, "default/network")
    if err != nil {
        log.Fatal(err)
    }
    for _, child := range content.Children {
        fmt.Printf("  %s %s (%s)\n", child.Type, child.Name, child.ID)
    }

    // Check status
    status, err := graph.Status(repoPath, "default")
    if err != nil {
        log.Fatal(err)
    }
    for _, change := range status.Changes {
        fmt.Printf("  %s: %s\n", change.Status, change.Path)
    }

    // Commit changes
    commitResult, err := graph.Commit(repoPath, "default", "")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Committed: %s\n", commitResult.Commit)

    // Delete a node
    if err := graph.DeleteNode(repoPath, "default/network/vpc"); err != nil {
        log.Fatal(err)
    }

    // Commit the deletion
    _, err = graph.Commit(repoPath, "default", "Remove vpc")
    if err != nil {
        log.Fatal(err)
    }
}
```
