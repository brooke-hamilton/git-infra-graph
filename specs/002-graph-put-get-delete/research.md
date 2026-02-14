# Research Report: Graph Node Put, Get, and Delete

<!-- markdownlint-disable MD024 -->

**Feature Branch**: `002-graph-put-get-delete`
**Date**: 2026-02-13
**Input**: Plan from [plan.md](plan.md)

## R1: Staging Mechanism — Per-Graph Index vs Direct Tree Composition

### Context

The spec requires that `Put` and `Delete` stage changes without creating a commit. Changes accumulate and are persisted only when `Commit` is called. Each graph must have isolated staging (no cross-graph contamination). The spec suggests per-graph index files (`.git/infra-index-<name>`).

### Option A: Git Index Files

Git's index (`.git/index`) is a flat file listing tracked file paths with working-tree metadata (timestamps, inodes, device IDs). The native `git` CLI supports alternate index files via the `GIT_INDEX_FILE` environment variable, which would make per-graph index files straightforward. However, this project uses go-git (a pure-Go Git reimplementation that does not shell out to `git`) as decided in [feature 001 research](../001-graph-init-list/research.md). go-git's `IndexStorer` interface (`SetIndex`/`Index`) operates on a single hardcoded path (`.git/index`), does not read `GIT_INDEX_FILE`, and provides no built-in mechanism to specify an alternate index path.

To use alternate index files with go-git:

- Manually use `index.Encoder`/`index.Decoder` (which accept generic `io.Writer`/`io.Reader`) to read/write from `.git/infra-index-<name>`
- Bypass the `IndexStorer` interface entirely
- Build a custom abstraction over the low-level index format

**Drawbacks**: Even if the go-git index limitation were worked around, the Git index is fundamentally designed for tracking working tree state — it stores file timestamps, inodes, device IDs, and other metadata that is irrelevant here since there is no working tree. Using the index format adds complexity with no benefit, and couples the implementation to an internal file format that is a poor fit for the hierarchical tree model.

### Option B: Direct Tree Composition with Staging Refs

Since the codebase already operates directly on the object store (`store.NewEncodedObject()`, `store.SetEncodedObject()`, `object.GetTree()`), the staged state can be represented as a tree hash stored in a lightweight staging ref:

1. Use `refs/infra-stage/<name>` to store the current staged root tree hash
2. On each `Put`/`Delete`, read the current staged tree, walk to the relevant path, rebuild the modified tree chain bottom-up, and store the new root tree hash
3. On `Commit`, take the staged tree hash, create a commit, update the graph ref, and delete the staging ref
4. On `Get`, read from the staging ref if it exists, else from the graph ref's latest commit tree (satisfies FR-012a)

**Staging ref lifecycle**:

- `Put`/`Delete` with no existing staging ref: Read tree from the graph ref's latest commit, apply modification, create staging ref with new tree hash
- `Put`/`Delete` with existing staging ref: Read tree from staging ref, apply modification, update staging ref
- `Commit`: Create commit from staged tree, update graph ref, delete staging ref
- `Get`: Read from staging ref if it exists, else from graph ref's committed tree

### Decision

**Use direct tree composition with staging refs** (`refs/infra-stage/<name>`).

### Rationale

1. **Consistent with existing patterns**: The codebase already creates trees and commits directly via the object store. No new abstraction layer is needed.
2. **No impedance mismatch**: Avoids mapping between the flat index format and the hierarchical tree model.
3. **Simpler implementation**: No custom index encoder/decoder. Tree manipulation uses the same `object.Tree`, `Encode`, `SetEncodedObject` pattern already established.
4. **Atomic**: Storing the staging tree hash in a ref is atomic (filesystem rename on POSIX, lock files). If the process crashes, either the old ref or the new ref exists — no corruption.
5. **Per-graph isolation is trivial**: Each graph gets its own staging ref.

### Alternatives Considered

- **Git index files**: Rejected. Adds complexity for no benefit. The index format includes working-tree metadata irrelevant to this use case.
- **Side files (e.g., `.git/infra-stage-<name>`)**: Considered. Works but refs are a better fit — they are part of Git's native ref system, are discoverable via `git for-each-ref`, and go-git already has ref CRUD operations.

## R2: Blob and Tree Object Manipulation in go-git

### Creating Blob Objects

Use `EncodedObject` with `plumbing.BlobObject` type:

```go
store := repo.Storer
obj := store.NewEncodedObject()
obj.SetType(plumbing.BlobObject)
obj.SetSize(int64(len(data)))

w, err := obj.Writer()
if err != nil {
    return plumbing.ZeroHash, err
}
_, err = w.Write(data)
w.Close()

hash, err := store.SetEncodedObject(obj)
```

**Key details**:

- Must call `SetType(plumbing.BlobObject)` before writing
- Must close the writer before storing
- `SetSize` should be set explicitly for correctness
- Returns a `plumbing.Hash` — the content-addressable SHA-1

### Creating/Modifying Tree Objects

**`object.TreeEntry` fields**:

```go
type TreeEntry struct {
    Name string              // single path segment
    Mode filemode.FileMode   // file mode
    Hash plumbing.Hash       // SHA-1 of referenced object
}
```

**File mode constants** (from `plumbing/filemode`):

- `filemode.Dir` (`0040000`) — sub-tree (directory/tree node)
- `filemode.Regular` (`0100644`) — regular file (blob node)

Only `Dir` and `Regular` are needed for this project.

**Building a tree**:

```go
tree := &object.Tree{
    Entries: []object.TreeEntry{
        {Name: "file.txt", Mode: filemode.Regular, Hash: blobHash},
        {Name: "subdir",   Mode: filemode.Dir,     Hash: subtreeHash},
    },
}
sort.Sort(object.TreeEntrySorter(tree.Entries))

obj := store.NewEncodedObject()
err := tree.Encode(obj)
hash, err := store.SetEncodedObject(obj)
```

**Modifying an existing tree**:

1. Retrieve: `existingTree, err := object.GetTree(store, treeHash)`
2. Copy entries, add/remove/replace as needed
3. Sort: `sort.Sort(object.TreeEntrySorter(entries))`
4. Create new tree, encode, store

**Gotchas**:

- `Tree.Encode()` returns `ErrEntriesNotSorted` if not sorted. Use `object.TreeEntrySorter`.
- Entry names must not contain null bytes.
- Content-addressed: if nothing changes, the hash is the same (structural sharing).

### Reading Tree Entries and Blob Content

**Tree entries**: `object.GetTree(store, treeHash)` returns `*object.Tree` with populated `Entries` slice and storer reference for recursive lookups.

**Blob content**: `object.GetBlob(store, blobHash)` returns `*object.Blob`. Call `blob.Reader()` to get an `io.ReadCloser`, then `io.ReadAll`.

**Path traversal**: Walk segment-by-segment via `GetTree` at each level rather than using `tree.FindEntry(path)`, since `Put`/`Delete` need to rebuild trees bottom-up during mutations.

### Decision

Use `object.TreeEntrySorter`, `filemode.Dir`/`filemode.Regular`, and direct tree encoding for all tree operations. Walk paths manually segment-by-segment.

### Rationale

Manual segment-by-segment traversal enables collecting the parent tree chain needed for bottom-up rebuilds during `Put` and `Delete`. Using `FindEntry` would find the target but not preserve the ancestor context.

## R3: Tree Diffing for Status

### go-git Built-in Diff

`object.DiffTree(a, b *Tree) (Changes, error)` compares two tree objects recursively and returns a `Changes` collection (`[]*Change`). Each `Change` has:

- `From`/`To` fields of type `ChangeEntry` (containing `Name` with full path, `TreeEntry`)
- `Action()` method returning `merkletrie.Insert`, `merkletrie.Delete`, or `merkletrie.Modify`

### Usage for Status

```go
committedTree, _ := object.GetTree(store, committedTreeHash)
stagedTree, _ := object.GetTree(store, stagedTreeHash)

changes, err := object.DiffTree(committedTree, stagedTree)
for _, change := range changes {
    action, _ := change.Action()
    switch action {
    case merkletrie.Insert:   // added
    case merkletrie.Delete:   // deleted
    case merkletrie.Modify:   // modified
    }
}
```

**Key details**:

- Reports changes at the leaf (blob) level — paths like `network/vpc` rather than intermediate tree changes
- `Change.From.Name` / `Change.To.Name` gives the full path relative to root
- `DiffTree(nil, stagedTree)` works — reports everything as `Insert` (useful for first commit scenario)
- Both trees must have their storer field set (via `GetTree` from store). Trees created in memory without the storer fail when `DiffTree` tries to recurse into subtrees.

### Decision

Use `object.DiffTree` directly for `Status`. Map `merkletrie.Insert` to "added", `merkletrie.Modify` to "modified", `merkletrie.Delete` to "deleted".

### Rationale

Built-in, recursive, handles all edge cases (empty trees, nested changes). No need for manual tree walking. The output format maps directly to the `Status` API requirements.

## R4: Commit Creation with Parents

### Extending Existing Pattern

The existing `CreateOrphanCommit` uses `ParentHashes: []plumbing.Hash{}`. To add a parent:

```go
commit := &object.Commit{
    TreeHash:     treeHash,
    ParentHashes: []plumbing.Hash{parentCommitHash},
    Author:       sig,
    Committer:    sig,
    Message:      message,
}
```

### Ref Update for Commit

The existing `CreateRef` returns an error if the ref already exists. For `Commit`, the ref must be updated. Use `repo.Storer.SetReference(plumbing.NewHashReference(...))` directly — this is an upsert (creates or updates).

### Decision

Generalize `CreateOrphanCommit` into `CreateCommit(repo, treeHash, parentHashes, message, sig)` where `parentHashes` can be empty (orphan for `Init`) or contain one hash (linear history for `Commit`). Add `UpdateRef` that unconditionally sets the ref.

### Rationale

Reuses existing encoding/storing pattern. Linear parent chain is all that is needed — no merge commits. The generalized function replaces the orphan-specific variant without breaking existing callers.

## R5: Path Parsing and Validation

### Path Structure

Per the spec, paths use the format `<graph>/<node-path>`:

- First segment: graph name (maps to `refs/infra/<graph>`)
- Remaining segments: node location within the graph's tree
- Leading/trailing slashes are trimmed before processing (FR-008)

### Validation Rules

- Empty paths after trimming: error
- Empty segments (e.g., `network//vpc`): error
- Single-segment paths (just graph name with no node path): valid only for `Get`, invalid for `Put`/`Get`/`rm` per CLI spec

### Parsing Function

```go
func ParseNodePath(path string) (graphName string, segments []string, err error) {
    trimmed := strings.Trim(path, "/")
    if trimmed == "" {
        return "", nil, errors.New("invalid path: empty")
    }
    parts := strings.Split(trimmed, "/")
    // Check for empty segments
    for _, p := range parts {
        if p == "" {
            return "", nil, fmt.Errorf("invalid path: empty segment in %q", path)
        }
    }
    return parts[0], parts[1:], nil
}
```

### Decision

Implement `ParseNodePath` in a new `node.go` file in the `graph` package. Validate graph name with existing `ValidateGraphName`. Validate node-path segments for empty strings.

### Rationale

Centralized path parsing ensures consistent validation across `Put`, `Get`, `Delete`. The first segment extraction separates graph name resolution from node path traversal, keeping the API clean.

## R6: Bottom-Up Tree Rebuild Strategy for Put and Delete

### Problem

When `Put("network/firewall/rules", data)` is called, the system must:

1. Walk from root through `network` → `firewall` → `rules`
2. Create/replace the target node at `rules`
3. Rebuild `firewall`'s tree with the updated entry for `rules`
4. Rebuild `network`'s tree with the updated entry for `firewall`
5. Rebuild the root tree with the updated entry for `network`
6. Store the new root tree hash as the staging ref

### Algorithm

```text
Walk phase (top-down):
  Collect parent_stack = [(root_tree, "network"), (network_tree, "firewall"), ...]
  At each level:
    - If entry exists and is a tree → push and continue
    - If entry exists and is a blob → error (cannot traverse blob)
    - If entry doesn't exist → auto-create tree (for Put only; error for Get/Delete)

Mutate phase (target):
  Create/replace/remove the target entry

Rebuild phase (bottom-up):
  For each level in reverse(parent_stack):
    Replace the child entry with updated hash
    Encode and store the new tree
    Use the new tree's hash as the child hash for the next level up
```

### Decision

Implement a generic `walkAndRebuild` helper that handles the walk/mutate/rebuild cycle. `Put` and `Delete` supply different mutation functions.

### Rationale

The walk-mutate-rebuild pattern is the natural consequence of Git's content-addressed tree model. Each mutation at a leaf requires rehashing all ancestor trees up to the root. Factoring this into a shared helper avoids code duplication between `Put` and `Delete`.
