# Data Model: Recursive Tree Listing (`grif tree`)

<!-- markdownlint-disable MD013 -->

## Entities

### TreeNode (output element)

Represents a single node in the recursive tree output — either a tree (container)
or blob (leaf). This is the core building block for both human-readable and JSON
output.

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Single path segment name (e.g., `vpc`, `network`) |
| `Type` | `NodeType` | `"tree"` or `"blob"` — reuses existing `NodeType` from `node.go` |
| `ID` | `string` | First 8 characters of the Git object SHA hash |
| `Children` | `[]TreeNode` | Recursive children (populated for trees; `nil` for blobs) |

**JSON serialization**:

- `name` (string, always present)
- `type` (string, always present: `"tree"` or `"blob"`)
- `id` (string, always present: 8-char abbreviated hash)
- `children` (array, omitted for blobs via `omitempty`)

**Validation rules**:

- `Name` is always non-empty (sourced from Git tree entry names)
- `Type` is always one of `BlobNode` or `TreeNode`
- `ID` is always exactly 8 characters (truncated from 40-char SHA)
- `Children` is `nil` for blobs; may be empty slice `[]` for empty trees

### TreeOptions

Options struct controlling the `Tree` function behavior.

| Field | Type | Description |
|-------|------|-------------|
| `Depth` | `int` | Maximum recursion depth (0 = root only, 1 = root + children) |
| `HasDepth` | `bool` | Whether `Depth` was explicitly set (distinguishes "not set" from 0) |

**Validation rules**:

- Negative `Depth` when `HasDepth` is true → error
- `HasDepth` false → unlimited recursion
- `HasDepth` true, `Depth` 0 → show root only, no children

### TreeResult (single graph)

Result for a single graph or subtree tree operation.

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Graph name or subtree root name |
| `Type` | `NodeType` | `"tree"` or `"blob"` (blob only when path resolves to a leaf) |
| `ID` | `string` | 8-char abbreviated hash of the root node |
| `Children` | `[]TreeNode` | Recursive children (nil for blobs) |

**JSON serialization**: Same fields as `TreeNode` — `name`, `type`, `id`, `children`.

This type is intentionally structurally identical to `TreeNode` so that the JSON
output for a single graph is a single nested object matching the spec examples.

### TreeListResult (all graphs)

Result for the no-argument (all graphs) mode.

| Field | Type | Description |
|-------|------|-------------|
| `Graphs` | `[]TreeResult` | One entry per graph, sorted alphabetically by name |
| `Warnings` | `[]string` | Non-empty if one or more graphs failed to resolve |

**JSON serialization**:

- All-graphs mode outputs `Graphs` as a JSON array (not the wrapper object)
- `Warnings` are emitted to stderr in human mode; included in JSON only if non-empty

## Git Object Mapping

| Data Model Field | Git Source |
|-----------------|-----------|
| `TreeNode.Name` | `object.TreeEntry.Name` |
| `TreeNode.Type` | Derived from `object.TreeEntry.Mode` (`filemode.Dir` → tree, else → blob) |
| `TreeNode.ID` | `object.TreeEntry.Hash.String()[:8]` |
| `TreeNode.Children` | Recursive: `gitops.GetTreeByHash(repo, entry.Hash).Entries` |
| `TreeResult.Name` | Graph name from ref (e.g., `refs/infra/default` → `default`) or last path segment |
| `TreeResult.ID` | Root tree hash from `gitops.ResolveRootTree` → `hash.String()[:8]` |

## State Diagram

The `tree` command is read-only. No state transitions occur.

```text
Repository State ──[grif tree]──> Repository State (unchanged)
```

Resolution order for root tree:

```text
refs/infra-stage/<name> exists?
  ├── Yes → Use staging ref tree hash
  └── No  → refs/infra/<name> exists?
              ├── Yes → Use latest commit's tree hash
              └── No  → Error: graph not found
```

## Traversal Algorithm

```text
Tree(graph, path, depth):
  1. Resolve root tree hash via ResolveRootTree(repo, graph)
  2. If path provided: walk tree to target subtree/blob
  3. If target is blob: return single TreeResult (no children)
  4. If target is tree:
     a. Read tree entries via GetTreeByHash
     b. Sort entries alphabetically by name
     c. For each entry:
        - If blob: create TreeNode{name, "blob", hash[:8], nil}
        - If tree and depth allows: recurse into subtree
        - If tree and depth exhausted: create TreeNode{name, "tree", hash[:8], nil}
     d. Return TreeResult with children

TreeAll(depth):
  1. List all graph names via ListRefsByPrefix("refs/infra/")
  2. If empty: error "no graphs found"
  3. Sort alphabetically
  4. For each graph: call Tree(graph, "", depth)
     - On error: record warning, continue to next graph
  5. Return TreeListResult with all results and any warnings
```

## Error Conditions

| Condition | Error Message Pattern |
|-----------|----------------------|
| Not a Git repository | `"could not open repository: ..."` (from `gitops.OpenRepo`) |
| Graph not found | `"graph '<name>' not found"` |
| Path not found in graph | `"path '<path>' not found in graph '<name>'"` |
| Negative depth value | `"depth must be a non-negative integer"` |
| No graphs found (all-graphs mode) | `"no graphs found"` |
| Corrupted tree object (partial) | Warning: `"failed to resolve graph '<name>': ..."` |
