# Data Model: Graph Node Put, Get, and Delete

**Feature**: 002-graph-put-get-delete
**Date**: 2026-02-13

## Entities

### NodeResult

The return value from a successful `Put` operation.

| Field | Type | Description |
| ----- | ---- | ----------- |
| ID | string | Content-addressable hash (SHA-1) of the created/updated node |
| Path | string | Full path including graph name (e.g., `default/network/vpc`) |
| Type | NodeType | `BlobNode` or `TreeNode` |

**Constraints**:

- `ID` is always a valid 40-character hex SHA-1 hash
- `Path` is the original input path after normalization (trimming slashes)
- `Type` is determined by whether `blob` was nil (`TreeNode`) or non-nil (`BlobNode`)

### NodeContent

The return value from a successful `Get` operation.

| Field | Type | Description |
| ----- | ---- | ----------- |
| ID | string | Content-addressable hash of the node |
| Path | string | Full path including graph name |
| Type | NodeType | `BlobNode` or `TreeNode` |
| Blob | []byte | Raw content (non-nil for blobs, nil for trees) |
| Children | []NodeEntry | Immediate children (non-nil for trees, nil for blobs) |

**Constraints**:

- When `Type` = `BlobNode`: `Blob` is populated (may be empty `[]byte{}`), `Children` is nil
- When `Type` = `TreeNode`: `Children` is populated (may be empty slice), `Blob` is nil
- `Children` are sorted alphabetically by `Name` (Git tree entry order)

### NodeEntry

A child descriptor within a `NodeContent` children listing.

| Field | Type | Description |
| ----- | ---- | ----------- |
| Name | string | Single path segment name of the child |
| Type | NodeType | `BlobNode` or `TreeNode` |
| ID | string | Content-addressable hash of the child |

**Constraints**:

- `Name` is a single path segment (no slashes)
- `Type` is determined by the Git tree entry's file mode (`filemode.Dir` → `TreeNode`, `filemode.Regular` → `BlobNode`)

### NodeType

An enumeration of node types.

| Value | Description |
| ----- | ----------- |
| `BlobNode` | A leaf node containing binary content |
| `TreeNode` | A container node with named children |

**Representation**: String constant. Used in API return types and JSON output (serialized as `"blob"` or `"tree"`).

### StatusChange

A single change entry in the `Status` response.

| Field | Type | Description |
| ----- | ---- | ----------- |
| Path | string | Node path relative to graph root (e.g., `network/vpc`) |
| Status | string | `"added"`, `"modified"`, or `"deleted"` |

### ChangeStatus

An enumeration of change statuses, represented as `string` values.

| Value | Description |
| ----- | ----------- |
| `"added"` | Node exists in staged tree but not in committed tree |
| `"modified"` | Node exists in both trees but with different content/hash |
| `"deleted"` | Node exists in committed tree but not in staged tree |

### StatusResult

The return value from a successful `Status` operation.

| Field | Type | Description |
| ----- | ---- | ----------- |
| Graph | string | Graph name |
| Changes | []StatusChange | List of changes since last commit |

### CommitResult

The return value from a successful `Commit` operation.

| Field | Type | Description |
| ----- | ---- | ----------- |
| Graph | string | Graph name |
| Commit | string | Hash of the newly created commit |
| Ref | string | Full ref path (e.g., `refs/infra/default`) |

## Git Object Mapping

### How Entities Map to Git Objects

| Entity | Git Object Type | Storage Location |
| ------ | --------------- | ---------------- |
| Blob Node | Git blob | Object store (`objects/`) |
| Tree Node | Git tree | Object store (`objects/`) |
| Graph Root | Git tree | Referenced by the graph commit's `TreeHash` |
| Graph Snapshot | Git commit | Object store, referenced by `refs/infra/<name>` |
| Staging State | Git tree | Object store, referenced by `refs/infra-stage/<name>` |

### Ref Namespace

| Ref Pattern | Purpose |
| ----------- | ------- |
| `refs/infra/<name>` | Points to the latest committed graph snapshot (commit object) |
| `refs/infra-stage/<name>` | Points to the current staged root tree (tree object). Exists only when uncommitted changes are present. Deleted on `Commit`. |

### Tree Entry Modes

| Node Type | Git filemode | Constant |
| --------- | ------------ | -------- |
| TreeNode | `0040000` | `filemode.Dir` |
| BlobNode | `0100644` | `filemode.Regular` |

## Relationships

```text
Graph Ref (refs/infra/<name>)
  └── Commit
        ├── TreeHash → Root Tree
        │     ├── TreeEntry (Dir)  → Sub-Tree
        │     │     ├── TreeEntry (Regular) → Blob
        │     │     └── TreeEntry (Dir)     → Sub-Tree → ...
        │     └── TreeEntry (Regular) → Blob
        └── ParentHashes → [Previous Commit]

Staging Ref (refs/infra-stage/<name>)
  └── Root Tree (staged, uncommitted)
        └── (same tree/blob structure as above)
```

## State Transitions

### Node Lifecycle

```text
(does not exist) --[Put blob]--> Blob Node --[Delete]--> (does not exist)
(does not exist) --[Put nil]---> Tree Node --[Delete]--> (does not exist)

Blob Node --[Put blob]--> Blob Node (content replaced, new hash)
Tree Node --[Put nil]---> Tree Node (no-op, preserves children)

Blob Node --[Put nil]---> ERROR (blob-to-tree conversion forbidden)
Tree Node --[Put blob]--> ERROR (tree-to-blob conversion forbidden)
```

### Staging Lifecycle

```text
(no staging ref)
  --[Put or Delete]--> Staging ref exists (points to modified root tree)
  --[Put or Delete]--> Staging ref updated (points to further modified root tree)
  --[Commit]---------> Staging ref deleted; graph ref updated to new commit

(staging ref exists)
  --[Get]------------> Reads from staged tree (staging ref)
  --[Status]---------> Diffs committed tree vs staged tree

(no staging ref)
  --[Get]------------> Reads from committed tree (graph ref)
  --[Status]---------> Reports no changes
  --[Commit]---------> ERROR (no staged changes)
```

## Validation Rules

### Path Validation

| Rule | Example Invalid Input | Error |
| ---- | --------------------- | ----- |
| Empty after trim | `""`, `"/"`, `"//"` | "invalid path: empty" |
| Empty segment | `"network//vpc"` | "invalid path: empty segment in ..." |
| Graph name invalid | `"a..b/node"` | (from `ValidateGraphName`) |
| Minimum segments | `"default"` (for put/get/rm) | "path must contain at least a graph name and node name" |

### Operation Constraints

| Constraint | Operations | Error |
| ---------- | ---------- | ----- |
| Blob cannot be parent | Put, Get, Delete | "blob at ... cannot have children" / "cannot be traversed" |
| Tree-to-blob conversion | Put | "cannot convert tree to blob at ..." |
| Blob-to-tree conversion | Put | "cannot convert blob to tree at ..." |
| Node not found | Get, Delete | "node not found at ..." |
| No staged changes | Commit | "no staged changes for graph ..." |
| Graph not found | All | "graph ... not found" |
