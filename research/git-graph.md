# Git Object Graph: refs → commits → trees → blobs

Git can be modeled as a **typed directed graph** where each node is an object (or reference), and each edge is a named pointer.

## Node types

- **Ref nodes**: branch refs (for example, `refs/heads/main`), tags, `HEAD`
- **Commit nodes**: snapshots with metadata and parent links
- **Tree nodes**: directory objects that map names to child objects
- **Blob nodes**: file-content objects (opaque byte payloads)

## Edge types (relationships)

- **Ref → Commit** (`points-to`):
	A branch/tag ref is a mutable (or tag-specific) pointer to a commit node.

- **Commit → Commit** (`parent-of` / reverse `child-of`):
	Each commit has zero or more outgoing parent edges.
	- One parent: normal commit
	- Multiple parents: merge commit
	- Zero parents: root commit

- **Commit → Tree** (`has-root-tree`):
	Every commit has exactly one edge to its root tree object.

- **Tree → Tree** (`contains-subtree`):
	A tree can point to child trees (subdirectories), forming a directory hierarchy.

- **Tree → Blob** (`contains-blob`):
	A tree points to blobs (files) at particular path entries.

## Graph-theoretic view

- The **commit-parent subgraph** is a directed acyclic graph (DAG) under normal Git operation.
- The **tree/blob subgraph** is a rooted, ordered containment structure per commit snapshot.
- The full Git storage model is a **multi-relational directed graph**:
	refs select entry commits, commits select snapshots (trees), and trees resolve to subtrees/blobs.
- Objects are **content-addressed** (node identity = hash of content), so identical content reuses the same node, yielding a shared immutable graph.

## One-line summary

Git stores history and content as a typed directed graph where refs are entry pointers into a commit DAG, and each commit roots a tree structure that resolves recursively to blobs.
