# Go API Contract: Recursive Tree Listing (`grif tree`)

<!-- markdownlint-disable MD013 -->

## Module API (`graph` package)

### Types

```go
// TreeItem represents a single node in the recursive tree output.
type TreeItem struct {
    Name     string     `json:"name"`               // Single path segment name
    Type     NodeType   `json:"type"`               // "tree" or "blob"
    ID       string     `json:"id"`                 // First 8 chars of SHA hash
    Children []TreeItem `json:"children,omitempty"` // Recursive children; nil for blobs
}

// TreeOptions controls the behavior of the Tree and TreeAll functions.
type TreeOptions struct {
    Depth    int  // Maximum recursion depth (0 = root only)
    HasDepth bool // Whether Depth was explicitly set
}

// TreeResult holds the recursive tree for a single graph or subtree.
// Structurally identical to TreeItem for JSON serialization.
type TreeResult = TreeItem

// TreeAllResult holds the results of listing trees for all graphs.
type TreeAllResult struct {
    Graphs   []TreeResult `json:"graphs"`             // One per graph, sorted alphabetically
    Warnings []string     `json:"warnings,omitempty"` // Non-empty if graphs failed to resolve
}
```

### Functions

#### `Tree` — Single Graph or Subtree

```go
// Tree returns the recursive tree structure for a single graph or subtree.
//
// The path argument follows the same format as Get: "graph" for the full graph,
// or "graph/path/to/subtree" for a subtree. When the path resolves to a blob,
// the result contains a single node with no children.
//
// The root tree is resolved from the staging ref (if present) or the committed
// tree, consistent with Get behavior.
func Tree(repoPath string, path string, opts TreeOptions) (*TreeResult, error)
```

**Parameters**:

| Param | Type | Description |
|-------|------|-------------|
| `repoPath` | `string` | Path to the Git repository (typically `"."`) |
| `path` | `string` | Graph name or `graph/subtree/path` |
| `opts` | `TreeOptions` | Options controlling depth limiting |

**Returns**:

| Return | Type | Description |
|--------|------|-------------|
| result | `*TreeResult` | Recursive tree structure; nil on error |
| err | `error` | Non-nil on failure |

**Error conditions**:

| Condition | Error message |
|-----------|---------------|
| Not a Git repository | `"could not open repository: ..."` |
| Empty path | `"path must not be empty"` |
| Invalid graph name | Validation error from `ValidateGraphName` |
| Graph not found | `"graph '<name>' not found"` |
| Path not found | `"path '<path>' not found in graph '<name>'"` |
| Negative depth | `"depth must be a non-negative integer"` |

**Behavior**:

1. Parse the path: single segment = graph name (full tree); multiple segments = subtree
2. Validate graph name via `ValidateGraphName`
3. Resolve root tree via `gitops.ResolveRootTree`
4. If path has segments beyond graph name, walk to the target subtree/blob
5. If target is a blob, return `TreeResult` with blob info and no children
6. If target is a tree, recursively build `TreeItem` children respecting depth limit
7. Sort children alphabetically at each level

#### `TreeAll` — All Graphs

```go
// TreeAll returns the recursive tree structure for all graphs in the repository.
//
// Graphs are returned in alphabetical order. If a graph fails to resolve
// (e.g., corrupted tree), it is skipped with a warning rather than failing
// the entire operation (partial success, consistent with Log broken-chain
// handling).
func TreeAll(repoPath string, opts TreeOptions) (*TreeAllResult, error)
```

**Parameters**:

| Param | Type | Description |
|-------|------|-------------|
| `repoPath` | `string` | Path to the Git repository (typically `"."`) |
| `opts` | `TreeOptions` | Options controlling depth limiting |

**Returns**:

| Return | Type | Description |
|--------|------|-------------|
| result | `*TreeAllResult` | All graph trees + warnings; nil on hard error |
| err | `error` | Non-nil only for hard failures (not a repo, no graphs) |

**Error conditions**:

| Condition | Error message |
|-----------|---------------|
| Not a Git repository | `"could not open repository: ..."` |
| No graphs found | `"no graphs found"` |
| Negative depth | `"depth must be a non-negative integer"` |

**Partial success**: If one or more graphs fail to resolve, the function returns
all successfully resolved graphs in `Graphs` and a warning string per failed
graph in `Warnings`. This is not an error — `err` is nil. The CLI emits warnings
to stderr and exits 0.

---

## CLI Command

### Synopsis

```text
grif tree [<graph>[/<path>]] [--depth N] [--json]
```

### Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `<graph>[/<path>]` | No | Graph name, optionally followed by a subtree path. Omit to show all graphs. |

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--depth N` | int | unlimited | Limit recursion to N levels below root |
| `--json` | bool | false | Output JSON instead of box-drawing tree |
| `-h`, `--help` | bool | false | Show usage help |

### Output Formats

#### Human-Readable (default)

Single graph:

```text
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    ├── subnet  (blob, e5f6a7b8)
    └── vpc  (blob, a1b2c3d4)
```

All graphs (separated by blank line):

```text
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    └── vpc  (blob, a1b2c3d4)

staging
└── compute
    └── instance  (blob, c3d4e5f6)
```

Blob target:

```text
vpc  (blob, a1b2c3d4)
```

#### JSON (`--json`)

Single graph:

```json
{
  "name": "default",
  "type": "tree",
  "id": "d4e5f6a7",
  "children": [
    {
      "name": "compute",
      "type": "tree",
      "id": "b8c9d0e1",
      "children": [
        {
          "name": "instance",
          "type": "blob",
          "id": "c3d4e5f6"
        }
      ]
    }
  ]
}
```

All graphs:

```json
{
  "graphs": [
    {
      "name": "default",
      "type": "tree",
      "id": "d4e5f6a7",
      "children": [...]
    },
    {
      "name": "staging",
      "type": "tree",
      "id": "f6a7b8c9",
      "children": [...]
    }
  ]
}
```

All graphs with warnings:

```json
{
  "graphs": [...],
  "warnings": ["failed to resolve graph 'broken': object not found"]
}
```

**Note**: The all-graphs JSON output always uses the wrapper object with `graphs`
and optional `warnings` fields. This provides a consistent schema for consumers
regardless of whether warnings are present.

### Error Output

Human-readable errors go to stderr:

```text
Error: graph 'nonexistent' not found
Error: no graphs found
Error: depth must be a non-negative integer
```

JSON errors go to stderr:

```json
{"error": "graph 'nonexistent' not found"}
```

Warnings (partial success) go to stderr in human mode:

```text
Warning: failed to resolve graph 'broken': object not found
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success (including partial success with warnings) |
| 1 | Failure (graph not found, no graphs, invalid depth, not a repo) |

### CLI Handler Routing

The `tree` command is added to the `main()` switch statement:

```go
case "tree":
    runTree(jsonMode)
```

The `runTree` handler:

1. Parses `--depth N` flag (optional)
2. Checks for positional argument
3. If no argument: calls `graph.TreeAll(repoPath, opts)`
4. If argument present: calls `graph.Tree(repoPath, arg, opts)`
5. Formats output (box-drawing or JSON) and writes to stdout
6. Emits warnings to stderr if present
