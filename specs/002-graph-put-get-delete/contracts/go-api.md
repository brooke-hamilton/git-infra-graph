# Go Module API Contract: Graph Node Put, Get, and Delete

**Feature**: 002-graph-put-get-delete
**Date**: 2026-02-13
**Package**: `graph`

This project is a Go module with a CLI interface — not a REST API. The
"contracts" defined here are the public Go function signatures and types
exposed by the `graph` package, plus the CLI command interface.

## Go Public API

### Types

```go
package graph

// NodeType represents the kind of node in the graph.
type NodeType string

const (
    // BlobNode is a leaf node containing binary content.
    BlobNode NodeType = "blob"
    // TreeNode is a container node with named children.
    TreeNode NodeType = "tree"
)

// NodeResult is the return value from a successful Put operation.
type NodeResult struct {
    ID   string   `json:"id"`   // Content-addressable hash (SHA-1)
    Path string   `json:"path"` // Full path including graph name
    Type NodeType `json:"type"` // BlobNode or TreeNode
}

// NodeContent is the return value from a successful Get operation.
type NodeContent struct {
    ID       string      `json:"id"`                 // Content-addressable hash
    Path     string      `json:"path"`               // Full path including graph name
    Type     NodeType    `json:"type"`                // BlobNode or TreeNode
    Blob     []byte      `json:"content,omitempty"`   // Raw content (blobs only)
    Children []NodeEntry `json:"children,omitempty"`  // Immediate children (trees only)
}

// NodeEntry is a child descriptor within a NodeContent children listing.
type NodeEntry struct {
    Name string   `json:"name"` // Single path segment name
    Type NodeType `json:"type"` // BlobNode or TreeNode
    ID   string   `json:"id"`   // Content-addressable hash
}

// StatusChange represents a single change in the Status response.
type StatusChange struct {
    Path   string `json:"path"`   // Node path relative to graph root
    Status string `json:"status"` // "added", "modified", or "deleted"
}

// StatusResult is the return value from a successful Status operation.
type StatusResult struct {
    Graph   string         `json:"graph"`   // Graph name
    Changes []StatusChange `json:"changes"` // List of changes since last commit
}

// CommitResult is the return value from a successful Commit operation.
type CommitResult struct {
    Graph  string `json:"graph"`  // Graph name
    Commit string `json:"commit"` // Hash of the new commit
    Ref    string `json:"ref"`    // Full ref path
}
```

### Functions

#### Put

```go
// Put creates or replaces a node at the specified path within the graph.
// The first segment of path identifies the graph name (mapping to
// refs/infra/<name>); remaining segments address nodes within that graph's
// tree.
//
// When blob is non-nil, a blob node is created or replaced with the given
// content. When blob is nil, a tree node is created; if the tree already
// exists, this is a no-op that preserves existing children.
//
// Put automatically creates any missing intermediate tree nodes along the
// path. Changes are staged in a per-graph staging ref and are not committed
// until Commit is called.
//
// Returns an error if:
//   - repoPath is not a valid Git repository
//   - path is invalid (empty, empty segments, or invalid graph name)
//   - path has fewer than 2 segments (graph name + at least one node segment)
//   - an intermediate path segment resolves to an existing blob (blobs cannot
//     be parents)
//   - the target node exists as a tree and blob is non-nil (tree-to-blob
//     conversion is forbidden)
//   - the target node exists as a blob and blob is nil (blob-to-tree
//     conversion is forbidden)
func Put(repoPath string, path string, blob []byte) (*NodeResult, error)
```

**Preconditions**:

- `repoPath` is a path to a Git repository
- Graph identified by the first path segment has been initialized via `Init`
- `path` contains at least 2 segments after normalization

**Postconditions**:

- The node exists in the staged tree at the specified path
- Any missing intermediate trees have been created
- The staging ref (`refs/infra-stage/<name>`) is created or updated
- No commit is created
- A `NodeResult` with the node's hash, full path, and type is returned

**Errors**:

| Condition | Error message pattern |
| --------- | --------------------- |
| Not a git repo | "not a git repository: ..." |
| Invalid path | "invalid path: ..." |
| Path too short | "path must contain at least a graph name and node name" |
| Graph not found | "graph '...' not found" |
| Blob as parent | "blob at \"...\" cannot have children" |
| Tree-to-blob | "cannot convert tree to blob at \"...\"" |
| Blob-to-tree | "cannot convert blob to tree at \"...\"" |

#### Get

```go
// Get reads the node at the specified path within the graph. The first
// segment of path identifies the graph name; remaining segments address
// nodes within that graph's tree.
//
// Get reads from the current staged (uncommitted) tree if a staging ref
// exists, otherwise from the last committed tree.
//
// For blob nodes, returns NodeContent with Blob populated and Children nil.
// For tree nodes, returns NodeContent with Children populated and Blob nil.
//
// Returns an error if:
//   - repoPath is not a valid Git repository
//   - path is invalid (empty, empty segments, or invalid graph name)
//   - path has fewer than 2 segments
//   - the node at the specified path does not exist
//   - an intermediate path segment resolves to a blob (cannot traverse blobs)
func Get(repoPath string, path string) (*NodeContent, error)
```

**Preconditions**:

- `repoPath` is a path to a Git repository
- Graph identified by the first path segment has been initialized via `Init`
- `path` contains at least 2 segments after normalization

**Postconditions**:

- No state is modified
- Returns a `NodeContent` with the node's hash, path, type, and either blob
  content or children listing

**Errors**:

| Condition | Error message pattern |
| --------- | --------------------- |
| Not a git repo | "not a git repository: ..." |
| Invalid path | "invalid path: ..." |
| Path too short | "path must contain at least a graph name and node name" |
| Graph not found | "graph '...' not found" |
| Node not found | "node not found at \"...\"" |
| Blob traversal | "blob at \"...\" cannot be traversed" |

#### Delete (Node-Level)

```go
// Delete removes the node at the specified path within the graph. The first
// segment of path identifies the graph name; remaining segments address
// nodes within that graph's tree.
//
// When the path resolves to a blob, the blob is removed. When the path
// resolves to a tree, the entire subtree is recursively removed.
//
// Changes are staged in the per-graph staging ref and are not committed
// until Commit is called. Parent trees are not automatically pruned when
// all their children are deleted.
//
// Note: This is the node-level Delete, distinct from the graph-level Delete
// in graph.go which removes an entire graph ref.
//
// Returns an error if:
//   - repoPath is not a valid Git repository
//   - path is invalid (empty, empty segments, or invalid graph name)
//   - path has fewer than 2 segments
//   - the node at the specified path does not exist
//   - an intermediate path segment resolves to a blob (cannot traverse blobs)
func DeleteNode(repoPath string, path string) error
```

**Note on naming**: The function is named `DeleteNode` to avoid collision with
the existing `Delete(repoPath, name)` function that deletes an entire graph.
The CLI command uses `rm` for the same reason.

**Preconditions**:

- `repoPath` is a path to a Git repository
- Graph identified by the first path segment has been initialized via `Init`
- `path` contains at least 2 segments after normalization
- The node at the specified path exists

**Postconditions**:

- The node and all its descendants (if tree) are removed from the staged tree
- The staging ref is updated
- Parent trees are preserved even if empty

**Errors**:

| Condition | Error message pattern |
| --------- | --------------------- |
| Not a git repo | "not a git repository: ..." |
| Invalid path | "invalid path: ..." |
| Path too short | "path must contain at least a graph name and node name" |
| Graph not found | "graph '...' not found" |
| Node not found | "node not found at \"...\"" |
| Blob traversal | "blob at \"...\" cannot be traversed" |

#### Commit

```go
// Commit persists all staged (uncommitted) changes for the specified graph
// as a new commit on the graph ref's lineage. The commit becomes the new
// tip of the graph ref.
//
// If message is empty, a default message is generated in the format:
//   Update graph "<name>"
//
//   Source-Commit: <HEAD>
//
// If message is provided, it replaces the default message but the
// Source-Commit trailer is still appended.
//
// Returns an error if:
//   - repoPath is not a valid Git repository
//   - the graph does not exist
//   - no staging ref exists (no staged changes)
func Commit(repoPath string, graphName string, message string) (*CommitResult, error)
```

**Preconditions**:

- `repoPath` is a path to a Git repository
- Graph `graphName` has been initialized via `Init`
- A staging ref (`refs/infra-stage/<graphName>`) exists (changes have been staged)

**Postconditions**:

- A new commit is created with the staged tree as its tree and the previous
  graph commit as its parent
- `refs/infra/<graphName>` is updated to point to the new commit
- `refs/infra-stage/<graphName>` is deleted
- Returns a `CommitResult` with the graph name, commit hash, and ref path

**Errors**:

| Condition | Error message pattern |
| --------- | --------------------- |
| Not a git repo | "not a git repository: ..." |
| Graph not found | "graph '...' not found" |
| No staged changes | "no staged changes for graph \"...\"" |

#### Status

```go
// Status returns the uncommitted changes for the specified graph by
// comparing the last committed tree with the current staged tree.
//
// Returns a StatusResult with an empty Changes slice (not an error) if
// there are no uncommitted changes.
//
// Returns an error if:
//   - repoPath is not a valid Git repository
//   - the graph does not exist
func Status(repoPath string, graphName string) (*StatusResult, error)
```

**Preconditions**:

- `repoPath` is a path to a Git repository
- Graph `graphName` has been initialized via `Init`

**Postconditions**:

- No state is modified
- Returns a `StatusResult` with the graph name and a list of changes
  (added, modified, deleted)

**Errors**:

| Condition | Error message pattern |
| --------- | --------------------- |
| Not a git repo | "not a git repository: ..." |
| Graph not found | "graph '...' not found" |

#### ParseNodePath (Internal Helper)

```go
// ParseNodePath splits a path into a graph name and node path segments.
// It trims leading/trailing slashes, validates the path is non-empty and
// has no empty segments, and validates the graph name.
//
// Returns the graph name, remaining path segments, and any validation error.
func ParseNodePath(path string) (graphName string, segments []string, err error)
```

**Note**: This may be exported or unexported depending on whether external
consumers need path parsing. If unexported, use `parseNodePath`.

## CLI Interface

### Command Structure

```text
grif <command> [flags] [args]
```

### Path Convention

For node-level commands (`put`, `get`, `rm`), the `<path>` argument uses the
format `<graph>/<node-path>`. The first segment identifies the graph name;
remaining segments address nodes within that graph's tree.

### Commands

#### `grif put <path> [--data <content>] [--file <file>] [--json]`

Create or replace a node at the specified path.

| Argument/Flag | Required | Description |
| ------------- | -------- | ----------- |
| `<path>` | Yes | Node path in `<graph>/<node-path>` format |
| `--data <content>` | No | Blob content as a string |
| `--file <file>` | No | Read blob content from a file path |
| `--json` | No | Output in JSON format |

**Blob content sources** (mutually exclusive, checked in this order):

1. `--data <content>` — inline string content
2. `--file <file>` — read content from a file
3. Piped stdin — if stdin is not a terminal, read from stdin
4. None of the above — creates a tree node (nil blob)

**Success output (human)**:

```text
Put blob at "default/network/vpc" (id: a1b2c3d4)
```

```text
Put tree at "default/network" (id: e5f6a7b8)
```

**Success output (JSON)**:

```json
{"id": "a1b2c3d4...", "path": "default/network/vpc", "type": "blob"}
```

**Exit codes**: 0 on success, 1 on error.

#### `grif get <path> [--json]`

Read the node at the specified path.

| Argument/Flag | Required | Description |
| ------------- | -------- | ----------- |
| `<path>` | Yes | Node path in `<graph>/<node-path>` format |
| `--json` | No | Output in JSON format |

**Success output for a blob (human)** — raw content to stdout:

```text
10.0.0.0/16
```

**Success output for a blob (JSON)**:

```json
{"id": "a1b2c3d4...", "path": "default/network/vpc", "type": "blob", "content": "10.0.0.0/16"}
```

**Success output for a tree (human)** — tabular listing:

```text
TYPE  NAME    ID
blob  vpc     a1b2c3d4
tree  subnet  e5f6a7b8
```

**Success output for a tree (JSON)**:

```json
{
  "id": "f9e8d7c6...",
  "path": "default/network",
  "type": "tree",
  "children": [
    {"name": "subnet", "type": "tree", "id": "e5f6a7b8..."},
    {"name": "vpc", "type": "blob", "id": "a1b2c3d4..."}
  ]
}
```

**Exit codes**: 0 on success, 1 on error.

#### `grif rm <path> [--json]`

Delete the node at the specified path.

**Note**: To produce the `type` and `descendants` fields in the output, the CLI must call `Get` on the target path before calling `DeleteNode`. For tree nodes, the CLI recursively counts descendants via `Get` calls. This two-step strategy (Get then DeleteNode) is required because `DeleteNode` returns only an error.

| Argument/Flag | Required | Description |
| ------------- | -------- | ----------- |
| `<path>` | Yes | Node path in `<graph>/<node-path>` format |
| `--json` | No | Output in JSON format |

**Success output (human)**:

```text
Removed "default/network/vpc"
```

```text
Removed "default/network" (tree, 3 descendants)
```

**Success output (JSON)**:

```json
{"path": "default/network/vpc", "removed": true}
```

```json
{"path": "default/network", "removed": true, "type": "tree", "descendants": 3}
```

**Exit codes**: 0 on success, 1 on error.

#### `grif commit <graph> [--message <msg>] [--json]`

Commit all staged changes for the specified graph.

| Argument/Flag | Required | Description |
| ------------- | -------- | ----------- |
| `<graph>` | Yes | Graph name |
| `--message <msg>` | No | Commit message (default generated if omitted) |
| `--json` | No | Output in JSON format |

**Success output (human)**:

```text
Committed graph "default" (commit: c3d4e5f6)
```

**Success output (JSON)**:

```json
{"graph": "default", "commit": "c3d4e5f6...", "ref": "refs/infra/default"}
```

**Exit codes**: 0 on success, 1 on error.

#### `grif status <graph> [--json]`

Show uncommitted changes for the specified graph.

| Argument/Flag | Required | Description |
| ------------- | -------- | ----------- |
| `<graph>` | Yes | Graph name |
| `--json` | No | Output in JSON format |

**Success output (human)**:

```text
Changes for graph "default":
  added:    network/vpc
  added:    network/subnet
  modified: compute/instance
  deleted:  storage/old-bucket
```

**No changes (human)**:

```text
No uncommitted changes for graph "default"
```

**Success output (JSON)**:

```json
{
  "graph": "default",
  "changes": [
    {"path": "network/vpc", "status": "added"},
    {"path": "network/subnet", "status": "added"},
    {"path": "compute/instance", "status": "modified"},
    {"path": "storage/old-bucket", "status": "deleted"}
  ]
}
```

**No changes (JSON)**:

```json
{"graph": "default", "changes": []}
```

**Exit codes**: 0 on success (including no changes), 1 on error.

### Error Output

All errors go to stderr. In JSON mode, errors are formatted as:

```json
{"error": "node not found at \"default/network/missing\""}
```

In human-readable mode:

```text
Error: node not found at "default/network/missing"
```

### CLI-to-API Mapping

| CLI Command | Go API Function | Notes |
| ----------- | --------------- | ----- |
| `grif put <path> --data <content>` | `Put(repoPath, path, []byte(content))` | Non-nil blob |
| `grif put <path>` (no data) | `Put(repoPath, path, nil)` | Nil blob creates tree |
| `grif get <path>` | `Get(repoPath, path)` | Returns `NodeContent` |
| `grif rm <path>` | `DeleteNode(repoPath, path)` | Node-level delete |
| `grif commit <graph>` | `Commit(repoPath, graphName, message)` | Persists staged changes |
| `grif status <graph>` | `Status(repoPath, graphName)` | Reports uncommitted state |
