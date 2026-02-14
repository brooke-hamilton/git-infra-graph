# Feature Specification: Graph Node Put, Get, and Delete

<!-- markdownlint-disable MD013 -->

**Feature Branch**: `002-graph-put-get-delete`
**Created**: 2026-02-13
**Status**: Draft
**Input**: User description: "Implement the path-addressed tree/blob node API with Put, Get, and Delete operations for graph nodes"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Put a Blob Node at a Path (Priority: P1)

A user has an initialized infrastructure graph and wants to store data (a blob) at a specific path within it. They call `Put` with a path like `network/vpc` and non-nil blob content. The system creates any missing intermediate tree nodes along the path and stores the blob at the final segment. The user receives a `NodeResult` confirming the node's ID, path, and type.

**Why this priority**: Put is the foundational write operation. Without the ability to create nodes, no graph content exists to read or delete. It also establishes the auto-creation of parent trees, which is the core path-traversal behavior.

**Independent Test**: Can be fully tested by initializing a graph, calling `Put` with a path and blob content, and verifying the returned `NodeResult` contains the correct path, type (`BlobNode`), and a valid hash ID.

**Acceptance Scenarios**:

1. **Given** an initialized graph with an empty root, **When** the user calls `Put("network/vpc", []byte("10.0.0.0/16"))`, **Then** the system creates a tree node for `network`, a blob node for `vpc` with the given content, and returns a `NodeResult` with `Type` = `BlobNode`, `Path` = `network/vpc`, and a non-empty `ID`.
2. **Given** an initialized graph with an existing blob at `network/vpc`, **When** the user calls `Put("network/vpc", []byte("172.16.0.0/12"))`, **Then** the blob content is replaced and a `NodeResult` is returned with the updated hash ID.
3. **Given** an initialized graph with an existing tree at `network`, **When** the user calls `Put("network", nil)`, **Then** the operation is a no-op; the tree and all its children are preserved, and a `NodeResult` with `Type` = `TreeNode` is returned.
4. **Given** an initialized graph, **When** the user calls `Put("network", nil)` and then `Put("network/vpc", []byte("data"))`, **Then** both operations succeed because `Put` with nil blob creates a tree that can have children.

---

### User Story 2 - Get a Node at a Path (Priority: P2)

A user wants to read the contents of a node in the graph. They call `Get` with a path. If the path points to a blob, the raw content is returned. If it points to a tree, the immediate children are listed with their names, types, and IDs.

**Why this priority**: Get is the primary read operation and the natural complement to Put. Users need to retrieve what they've stored. It also enables verification of graph structure.

**Independent Test**: Can be fully tested by putting one or more nodes into a graph, then calling `Get` on a blob path (verifying content) and on a tree path (verifying children listing).

**Acceptance Scenarios**:

1. **Given** a graph with a blob at `network/vpc` containing `[]byte("10.0.0.0/16")`, **When** the user calls `Get("network/vpc")`, **Then** a `NodeContent` is returned with `Type` = `BlobNode`, `Blob` = `[]byte("10.0.0.0/16")`, and `Children` = nil.
2. **Given** a graph with children `vpc` (blob) and `subnet` (tree) under `network`, **When** the user calls `Get("network")`, **Then** a `NodeContent` is returned with `Type` = `TreeNode`, `Children` listing both child entries (with their names, types, and IDs), and `Blob` = nil.
3. **Given** a graph with no node at `network/missing`, **When** the user calls `Get("network/missing")`, **Then** an error is returned indicating the node does not exist.
4. **Given** a graph with a blob at `network/vpc`, **When** the user calls `Get("network/vpc/child")`, **Then** an error is returned because `vpc` is a blob and cannot be traversed.

---

### User Story 3 - Delete a Node at a Path (Priority: P3)

A user wants to remove a node from the graph. They call `Delete` with the node's path. If the path points to a blob, it is removed. If it points to a tree, the entire subtree (all descendants) is recursively deleted.

**Why this priority**: Delete completes the CRUD lifecycle. It enables cleanup of nodes no longer needed and is essential for graph maintenance, but it depends on content existing first.

**Independent Test**: Can be fully tested by putting nodes into a graph, calling `Delete`, and then verifying with `Get` that the nodes no longer exist.

**Acceptance Scenarios**:

1. **Given** a graph with a blob at `network/vpc`, **When** the user calls `Delete("network/vpc")`, **Then** the blob is removed (subsequent `Get("network/vpc")` returns an error), the parent tree `network` still exists, and the operation returns nil error.
2. **Given** a graph with a tree at `network` containing children `vpc` and `subnet`, **When** the user calls `Delete("network")`, **Then** the entire `network` subtree is recursively removed including all children, and subsequent `Get("network")` returns an error.
3. **Given** a graph with no node at `nonexistent`, **When** the user calls `Delete("nonexistent")`, **Then** an error is returned indicating the node does not exist.
4. **Given** a graph with a blob at `network/vpc`, **When** the user calls `Delete("network/vpc/child")`, **Then** an error is returned because `vpc` is a blob and cannot be traversed.

---

### User Story 4 - Put a Tree Node Explicitly (Priority: P2)

A user wants to create an empty tree node (directory-like container) at a path without storing blob content. They call `Put` with a path and nil blob. This is useful for pre-creating organizational structure before populating it with blobs.

**Why this priority**: Tree creation is integral to the Put operation and enables organizational graph structures. It shares priority with Get because it rounds out the core write semantics.

**Independent Test**: Can be fully tested by calling `Put` with nil blob, then calling `Get` on that path and verifying a `TreeNode` with an empty children list is returned.

**Acceptance Scenarios**:

1. **Given** an initialized graph with an empty root, **When** the user calls `Put("network", nil)`, **Then** a tree node is created and a `NodeResult` with `Type` = `TreeNode` is returned.
2. **Given** a graph with a tree at `network`, **When** the user calls `Put("network/security", nil)`, **Then** a new tree node `security` is created under `network`.

---

### Edge Cases

- What happens when the path is empty or consists only of slashes? The operation MUST return an error indicating invalid path syntax.
- What happens when path segments contain empty components (e.g., `network//vpc`)? The operation MUST return an error indicating invalid path syntax.
- What happens when the user calls `Put` with a non-nil blob on an existing tree node? The operation MUST return an error (tree-to-blob conversion is forbidden).
- What happens when the user calls `Put` with nil blob on an existing blob node? The operation MUST return an error (blob-to-tree conversion is forbidden).
- What happens when an intermediate path segment is a blob (e.g., `Put("a/b", data)` then `Put("a/b/c", data)`)? The second operation MUST return an error because blob `b` cannot have children.
- What happens when `Put` is called with a single-segment path (e.g., `Put("node", data)`)? The node is created directly under the root tree with no intermediate trees needed.
- What happens when `Delete` removes all children of a tree? The parent tree still exists as an empty tree; it is not automatically pruned.
- What happens when the blob content is empty (`[]byte{}`)? This is a valid non-nil blob; the node is stored as a blob with zero-length content.
- What happens when a user Deletes a blob and then Puts a tree at the same path (or vice versa)? This is the valid workflow for changing a node's type. After `Delete`, the path is fully cleared and can be reused with any node type.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `Put` function MUST accept a `repoPath` string, a `path` string, and a `blob` byte slice as parameters and return a `*NodeResult` and an `error`. The first segment of `path` identifies the graph name (mapping to `refs/infra/<name>`); remaining segments address nodes within that graph's tree.
- **FR-002**: When `blob` is non-nil, `Put` MUST create or replace a blob node at the specified path with the given content.
- **FR-003**: When `blob` is nil, `Put` MUST create a tree node at the specified path. If the tree already exists, this is a no-op that preserves the existing tree and all its children.
- **FR-004**: `Put` MUST automatically create any missing intermediate tree nodes along the path (auto-create parent trees).
- **FR-005**: `Put` MUST return an error if any intermediate path segment resolves to an existing blob node (blobs cannot be parents).
- **FR-006**: `Put` MUST return an error if the target node exists as a tree and the call passes a non-nil blob (tree-to-blob conversion is forbidden).
- **FR-007**: `Put` MUST return an error if the target node exists as a blob and the call passes nil blob (blob-to-tree conversion is forbidden).
- **FR-008**: `Put` MUST return an error for invalid path syntax, including empty paths, paths with empty segments, and paths with only slashes.
- **FR-009**: On success, `Put` MUST return a `NodeResult` containing the hash ID, full path, and node type of the written node.
- **FR-010**: The `Get` function MUST accept a `repoPath` string and a `path` string and return a `*NodeContent` and an `error`. The first segment of `path` identifies the graph name; remaining segments address nodes within that graph's tree.
- **FR-011**: When the path resolves to a blob node, `Get` MUST return a `NodeContent` with `Blob` populated and `Children` = nil.
- **FR-012**: When the path resolves to a tree node, `Get` MUST return a `NodeContent` with `Children` populated (listing immediate children with name, type, and ID) and `Blob` = nil.
- **FR-012a**: `Get` MUST read from the current staged (uncommitted) tree, reflecting any pending `Put` or `Delete` changes that have not yet been committed.
- **FR-013**: `Get` MUST return an error if any intermediate path segment resolves to a blob node (cannot traverse blobs).
- **FR-014**: `Get` MUST return an error if the node at the specified path does not exist.
- **FR-015**: The `Delete` function MUST accept a `repoPath` string and a `path` string and return an `error`. The first segment of `path` identifies the graph name; remaining segments address nodes within that graph's tree.
- **FR-016**: When the path resolves to a blob node, `Delete` MUST remove the blob.
- **FR-017**: When the path resolves to a tree node, `Delete` MUST recursively remove the entire subtree, including all descendant trees and blobs.
- **FR-018**: `Delete` MUST return an error if any intermediate path segment resolves to a blob node (cannot traverse blobs).
- **FR-019**: `Delete` MUST return an error if the node at the specified path does not exist.
- **FR-020**: `Delete` MUST NOT automatically remove empty parent trees after deleting a child; parent trees persist even when empty.
- **FR-021**: Node names MUST be unique per parent across both node types. A parent cannot contain both a tree and a blob with the same name.
- **FR-022**: Each node MUST have a unique ID (hash) generated by the underlying storage system.
- **FR-023**: All operations MUST operate within the context of an existing initialized graph (created via `Init` from the earlier feature). The graph is identified by the first segment of the path, which maps to the ref `refs/infra/<name>`.
- **FR-024**: A `Commit` function MUST accept a `repoPath` string and a `graphName` string (or the graph name extracted from a path) and persist all staged (uncommitted) tree changes as a new commit on the graph ref's lineage. The commit MUST become the new tip of the graph ref. If no changes are staged, `Commit` MUST return an error.
- **FR-025**: A `Status` function MUST accept a `repoPath` string and a `graphName` string and return a representation of the uncommitted tree state, showing which nodes have been added, modified, or deleted since the last commit.
- **FR-026**: `Put` and `Delete` MUST stage their changes in the Git index (uncommitted tree) without creating a commit. Changes are only persisted when `Commit` is called.

### Key Entities

- **Tree Node**: A container node that has a name, children (trees and/or blobs), and no blob content. Represented as a Git tree object.
- **Blob Node**: A leaf node that has a name, blob content (`[]byte`), and no children. Represented as a Git blob object.
- **Path**: A `/`-separated string where the first segment is the graph name (mapping to `refs/infra/<name>`) and remaining segments identify a node's location in the graph hierarchy. For example, `default/network/vpc` refers to node `network/vpc` within the graph `default` (ref `refs/infra/default`).
- **NodeResult**: The return value from a successful `Put`, containing the node's hash ID, full path, and type.
- **NodeContent**: The return value from a successful `Get`, containing the node's hash ID, full path, type, and either blob content (for blob nodes) or a children listing (for tree nodes).
- **NodeEntry**: A child descriptor within a `NodeContent`, containing the child's name, type, and hash ID.
- **Session (Git Index)**: The on-disk staging area (Git index) that accumulates `Put` and `Delete` changes before they are persisted via `Commit`. Managed implicitly—no session object is exposed in the API; stateless functions with `repoPath` read/write the index directly.
- **Commit**: The operation that takes all staged changes for a graph and writes them as a new commit on the graph ref's lineage. Accepts `repoPath` and `graphName`.
- **Status**: The operation that reports the diff between the last committed tree and the current staged (uncommitted) tree for a graph. Accepts `repoPath` and `graphName`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can store a blob at any valid multi-segment path and retrieve the exact same content back via `Get`, round-tripping with 100% fidelity.
- **SC-002**: A user can navigate the graph hierarchy by calling `Get` on tree nodes and receiving a complete listing of immediate children.
- **SC-003**: All invalid operations (type conversion, blob traversal, missing nodes, invalid paths) produce clear, descriptive error messages that identify the specific problem without exposing raw storage-layer errors.
- **SC-004**: Auto-creation of parent trees succeeds for paths of any reasonable depth (at least 10 levels deep) in a single `Put` call.
- **SC-005**: Recursive `Delete` of a tree node removes all descendants, verified by subsequent `Get` calls on every previously existing descendant returning "not found" errors.
- **SC-006**: After any `Put` or `Delete` operation, the graph remains in a consistent state where all tree nodes have valid children references and all blob nodes have valid content.
- **SC-007**: Operations complete within 2 seconds for graphs containing up to 1,000 nodes on a local repository.

## Clarifications

### Session 2026-02-13

- Q: Does each Put/Delete create a commit immediately, or do operations accumulate and require an explicit commit? → A: Operations accumulate in a session backed by the Git index; an explicit Commit operation persists them. A Status operation exposes the uncommitted tree.
- Q: Does Get read from the last committed tree or the staged (uncommitted) tree? → A: Get reads from the staged (uncommitted) tree, reflecting pending Put/Delete changes.
- Q: Should the API use stateless functions (with repoPath) or a stateful session/receiver object? → A: Stateless functions with repoPath. No session object. The graph name from Init is the first segment of the path (e.g., "default/node1/node2" maps to ref "refs/infra/default" with node path "node1/node2").
- Q: Can a node's type be changed from blob to tree (or vice versa) via Delete then Put? → A: Yes. Delete + Put is the valid workflow for type changes; a path is fully reusable after deletion. Single-operation type conversion errors (FR-006, FR-007) guard against accidental overwrites only.

## Assumptions

- A graph has already been initialized using the `Init` operation (from feature 001) before any Put, Get, or Delete operations are performed.
- The operations work with the graph's Git tree structure stored under the graph's ref (`refs/infra/<name>`). `Put` and `Delete` stage changes in the Git index (uncommitted tree). An explicit `Commit` call persists all staged changes as a new commit on the graph's independent commit lineage. A `Status` operation allows inspecting the uncommitted tree before committing.
- Path segments follow the same character restrictions as Git tree entry names.
- Blob content can be of any size supported by the underlying Git object store. This spec does not address streaming for large blobs.
- Deleting a node does not trigger automatic pruning of empty parent trees; this behavior is intentional to preserve the organizational structure created by the user.
- The root of the graph (empty path) is itself a tree node and cannot be deleted via `Delete`; it is the graph's root created by `Init`.
