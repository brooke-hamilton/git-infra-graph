# Upcoming Features

This document tracks planned features for `git-infra-graph`, ordered by priority.

## Priority Summary

| #   | Feature                       | Priority | Complexity | Dependencies | Status  |
| --- | ----------------------------- | -------- | ---------- | ------------ | ------- |
| 1   | `grif log`                    | High     | Low        | None         | Planned |
| 2   | `grif show`                   | High     | Low        | None         | Planned |
| 3   | `grif diff`                   | High     | Medium     | None         | Planned |
| 4   | `grif push` / `grif pull`     | High     | Medium     | None         | Planned |
| 5   | `grif tree`                   | Medium   | Low        | None         | Planned |
| 6   | `grif mv`                     | Medium   | Low        | None         | Planned |
| 7   | `grif reset`                  | Medium   | Medium     | None         | Planned |
| 8   | `grif export` / `grif import` | Medium   | Medium     | None         | Planned |
| 9   | `grif merge`                  | Low      | High       | push/pull    | Planned |
| 10  | Node metadata annotations     | Low      | High       | None         | Planned |

---

## 1. `grif log` — Graph Commit History

Display the commit history for an infrastructure graph.

**Rationale**: The commit lineage already exists (each `grif commit` creates a chain of commits on the graph ref with `Source-Commit` trailers), but there is no way to inspect it. This is the most obvious gap and is foundational for all other history-related features.

**Scope**:

- Walk the commit chain starting from the graph ref's tip commit, following parent hashes
- Display commit hash, message, `Source-Commit` trailer, author, and timestamp
- Support `--oneline` flag for compact single-line-per-commit output
- Support `--max-count N` flag to limit the number of entries displayed
- Support `--json` flag for machine-readable output
- Accept an optional graph name argument (default to the only graph if one exists)

**Example output (human)**:

```text
commit a1b2c3d4e5f6a7b8
Date:   2026-02-14 10:30:00 -0700
Source:  f9e8d7c6a5b4c3d2

    Add network resources

commit c3d4e5f6a7b8c9d0
Date:   2026-02-14 10:00:00 -0700
Source:  d2c1b0a9e8f7d6c5

    Initialize graph "default"
```

**Example output (oneline)**:

```text
a1b2c3d4 Add network resources
c3d4e5f6 Initialize graph "default"
```

---

## 2. `grif show` — Read Historical Graph Snapshots

Read a node's content or a graph's state at a specific historical commit.

**Rationale**: Currently `grif get` only reads the latest staged/committed state. Users need to inspect what a node looked like at a previous point in time for auditing and debugging. This is the natural companion to `grif log` — you find a commit in the log, then use `show` to inspect it.

**Scope**:

- `grif show <graph>@<commit>/<path>` — read a node at a specific commit
- `grif show <graph>@<commit>` — list the root tree children at a specific commit
- For blob nodes, print content; for tree nodes, list children (same as `get` behavior)
- Support `--json` flag

**Example output**:

```text
# Show a blob at a historical commit
grif show default@a1b2c3d4/network/vpc
# Output: 10.0.0.0/16

# Show root tree at a historical commit
grif show default@a1b2c3d4
# Output:
# TYPE  NAME     ID
# tree  network  e5f6a7b8
```

---

## 3. `grif diff` — Diff Between Graph Commits

Show what changed between two graph commits or between the committed state and staged state.

**Rationale**: `grif status` only reports which paths changed (added/modified/deleted) between staged and committed state. It does not show the actual content differences. There is no way to see what changed in a specific historical commit or to compare two arbitrary points in the graph's history.

**Scope**:

- `grif diff <graph>` — diff staged changes vs. last commit, showing content-level diffs for blobs
- `grif diff <graph> <commit>` — diff a specific commit against its parent
- `grif diff <graph> <commit1> <commit2>` — diff two arbitrary commits
- Show added/modified/deleted nodes; for modified blobs, display a unified-diff-style content comparison
- Builds on the existing `diffTreesRecursive` infrastructure in the graph package
- Support `--json` flag

**Example output (human)**:

```text
grif diff default
# Output:
# added:   network/vpc
#   +10.0.0.0/16
# modified: network/subnet
#   -172.16.0.0/24
#   +172.16.0.0/20
# deleted: network/legacy
```

---

## 4. `grif push` / `grif pull` — Remote Sync

Push and pull graph refs to/from a remote Git repository.

**Rationale**: Graph refs (`refs/infra/*`) and staging refs (`refs/infra-stage/*`) live only locally. Syncing them to a remote enables collaboration, backup, and CI/CD integration. Since the data is stored as native Git objects, the transport layer is already solved — only ref advertisement and transfer need to be wired up.

**Scope**:

- `grif push [remote] [graph]` — push `refs/infra/<name>` to the specified remote (default remote: `origin`)
- `grif pull [remote] [graph]` — fetch `refs/infra/<name>` from the specified remote (default remote: `origin`)
- Use go-git remote operations (no dependency on the git binary)
- Do NOT push staging refs — only committed graph refs
- Support fast-forward updates; reject non-fast-forward pushes with a clear error directing users to `grif merge` or `--force`
- Support `--force` flag to allow non-fast-forward push (with a warning)
- Support `--json` flag
- When no graph name is specified and only one graph exists, use it by default (consistent with `commit` and `status`)

---

## 5. `grif tree` — Recursive Tree Listing

Display the full tree structure of a graph or subtree.

**Rationale**: `grif get` on a tree node shows only its immediate children. Understanding the full structure of a complex graph requires repeated `get` calls at each level. A recursive tree view provides a single-command overview of the entire graph hierarchy — essential for orientation and debugging.

**Scope**:

- `grif tree [<graph>[/<path>]]` — recursively list all descendants of the specified tree node
- When no argument is provided, display the tree for all graphs (if only one graph exists, show it; if multiple exist, show all)
- Display output as an indented tree with type indicators
- Support `--depth N` flag to limit recursion depth
- Support `--json` flag for machine-readable output

**Example output (human)**:

```text
grif tree
# Output (single graph):
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    ├── subnet  (blob, e5f6a7b8)
    └── vpc  (blob, a1b2c3d4)

grif tree
# Output (multiple graphs):
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    ├── subnet  (blob, e5f6a7b8)
    └── vpc  (blob, a1b2c3d4)
staging
└── compute
    └── instance  (blob, c3d4e5f6)

grif tree default
# Output:
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    ├── subnet    (blob, e5f6a7b8)
    └── vpc       (blob, a1b2c3d4)

grif tree default/network
# Output:
network
├── subnet  (blob, e5f6a7b8)
└── vpc     (blob, a1b2c3d4)
```

---

## 6. `grif mv` — Rename or Move Nodes

Move or rename a node within a graph.

**Rationale**: Currently, renaming or moving a node requires `grif rm` followed by `grif put`, which is error-prone (content must be manually re-supplied) and non-atomic with respect to staging. A dedicated `mv` command preserves content and performs both operations in a single staging update.

**Scope**:

- `grif mv <source-path> <dest-path>` — move or rename a node within the same graph
- Preserve blob content or entire subtree structure when moving
- Source and destination must be in the same graph (cross-graph moves are not supported)
- If the destination already exists, return an error (no implicit overwrite)
- Support `--force` flag to overwrite the destination
- Support `--json` flag
- Changes are staged, not committed, consistent with `put` and `rm`

---

## 7. `grif reset` — Rollback to a Previous Commit

Reset a graph ref to a previous commit in its lineage.

**Rationale**: If a commit introduces incorrect state, users need a way to roll back the graph to a known-good point. This is the graph-level equivalent of `git reset`.

**Naming note**: This command is named `reset` rather than `checkout` to align with Git semantics. In Git, `checkout` switches contextual state (branches, working tree files), while `reset` moves a ref to a different commit. Since this command updates the graph ref pointer, `reset` is the more precise analog.

**Scope**:

- `grif reset <graph> <commit>` — update `refs/infra/<name>` to point to the specified commit
- Verify the target commit is in the graph's lineage (prevent pointing to unrelated commits)
- Clear any staging ref for the graph
- Warn if there are uncommitted staged changes that would be lost
- Support `--force` to skip the staged-changes warning
- Support `--json` flag

---

## 8. `grif export` / `grif import` — Portable Serialization

Export a graph snapshot to a portable format, or import one.

**Rationale**: Graph data is stored as Git objects, which is optimal for versioning and deduplication but opaque to external tools. Exporting to a structured format (JSON) enables interoperability with IaC tools, backup workflows, migration between repositories, and inspection without Git tooling.

**Scope**:

- `grif export <graph> [--output <file>]` — serialize the current graph state (or a specific commit with `@<commit>`) to JSON, writing to stdout or a file
- `grif import <graph> [--input <file>]` — deserialize a JSON payload into a graph, reading from stdin or a file
- Export format includes the full tree structure and blob content (base64-encoded for binary safety)
- Import creates nodes via the existing `Put` path, so changes are staged and require `grif commit`
- Support `--json` flag on import for status output

**Example export format**:

```json
{
  "graph": "default",
  "commit": "a1b2c3d4e5f6a7b8",
  "nodes": [
    {"path": "network/vpc", "type": "blob", "content": "MTAuMC4wLjAvMTY="},
    {"path": "network/subnet", "type": "blob", "content": "MTcyLjE2LjAuMC8yMA=="},
    {"path": "compute", "type": "tree"}
  ]
}
```

---

## 9. `grif merge` — Merge Diverged Graph Lineages

Merge two diverged commit lineages for a graph ref.

**Rationale**: When `grif push` and `grif pull` are available, two collaborators can create diverged commit histories for the same graph. Without a merge operation, the only resolution is `--force` push, which discards history. A merge command enables safe reconciliation of concurrent changes. This feature completes the collaboration story started by push/pull.

**Scope**:

- `grif merge <graph>` — merge the remote-tracking lineage into the local graph ref
- Detect and auto-resolve non-conflicting changes (additions on different paths, deletions of different nodes)
- For conflicting changes (same path modified differently), report conflicts and abort
- Support `--json` flag
- Future consideration: interactive conflict resolution and `--strategy` flags

---

## 10. Node Metadata Annotations

Attach arbitrary key-value metadata to graph nodes.

**Rationale**: Currently nodes hold only raw `[]byte` blob content or act as tree containers. Infrastructure graphs benefit from structured metadata (e.g., `owner=platform-team`, `environment=production`, `cost-center=eng-123`) without imposing a schema at the graph layer.

**Scope**:

- Store annotations as a sibling blob (e.g., `.grif-meta/<node-name>`) or as a convention within the tree structure — design TBD
- `grif annotate <path> --set key=value` — set a metadata key
- `grif annotate <path> --get key` — read a metadata key
- `grif annotate <path> --list` — list all metadata for a node
- `grif annotate <path> --remove key` — remove a metadata key
- Annotations are staged like any other change and persisted via `grif commit`
- Include annotation changes in `grif diff` and `grif status` output
- Support `--json` flag

---

## Design Notes

- All features follow the project constitution: module-first architecture, CLI as sole interface, Git-native storage, test-first development, and graph-layer/app-layer separation.
- Each feature should get its own spec directory under `specs/` (e.g., `specs/003-graph-log/`) following the established pattern.
- Features 1–3 (log, show, diff) are recommended as the next implementation target since they build directly on the existing commit infrastructure with no new storage concepts required. Log and show are particularly low-risk since they are read-only operations.
- Features 5–6 (tree, mv) are low-complexity quality-of-life improvements that could be interleaved with higher-priority work.
- Feature 9 (merge) should be designed alongside push/pull even if implemented later, to ensure the ref update and conflict detection model is consistent.
