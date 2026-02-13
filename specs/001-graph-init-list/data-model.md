# Data Model: Graph Init, List, and Delete

**Feature**: 001-graph-init-list
**Date**: 2026-02-13

## Entities

### Graph

A named infrastructure graph identified by a Git ref under `refs/infra/<name>`.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Unique identifier; must be a valid single Git ref-name component |

**Constraints**:

- Name must pass `ValidateGraphName()` (see ref-name rules in
  [research.md](research.md) R2).
- Name must be unique within the `refs/infra/` namespace (enforced by checking
  ref existence before creation).

**State transitions**:

```text
(does not exist) --[init]--> active --[delete]--> (does not exist)
```

- `init`: Creates the ref `refs/infra/<name>` pointing to an orphan commit.
- `delete`: Removes the ref. The graph's Git objects become unreferenced and
  are eligible for garbage collection.

### Root Node

The initial (and currently only) node in a newly initialized graph.

| Field | Type | Description |
|-------|------|-------------|
| TreeHash | plumbing.Hash | SHA-1 of the empty Git tree object |

**Constraints**:

- Always an empty tree (`4b825dc642cb6eb9a060e54bf899d69f82cf7b04`).
- Represents the top of the containment hierarchy.

### Graph Commit

An orphan Git commit that captures a graph snapshot.

| Field | Type | Description |
|-------|------|-------------|
| Tree | plumbing.Hash | Points to the root node (empty tree for init) |
| ParentHashes | []plumbing.Hash | Empty slice (orphan commit — no parents) |
| Author | object.Signature | Author identity and timestamp |
| Committer | object.Signature | Committer identity and timestamp |
| Message | string | Commit message including the `Source-Commit` trailer |

**Constraints**:

- `ParentHashes` MUST be empty (orphan commit).
- `Message` MUST contain a `Source-Commit: <sha>` trailer referencing the
  current HEAD at time of creation.
- The commit lineage is independent from the repository's standard commit DAG.

**Trailer format**:

```text
Initialize graph "<name>"

Source-Commit: <40-char hex SHA of HEAD>
```

### Graph Ref

A Git reference at `refs/infra/<name>` that tracks the head of a graph's
commit lineage.

| Field | Type | Description |
|-------|------|-------------|
| Name | plumbing.ReferenceName | Full ref path: `refs/infra/<graph name>` |
| Hash | plumbing.Hash | Points to the latest graph commit |

**Constraints**:

- One ref per graph.
- Deleting the ref makes the entire commit lineage unreferenced.

## Relationships

```text
Graph Ref (refs/infra/<name>)
  └── points to → Graph Commit (orphan)
                     └── tree → Root Node (empty tree)
                     └── trailer → Source-Commit: <HEAD SHA>
```

- A **Graph** is identified by its **Graph Ref**.
- A **Graph Ref** points to the latest **Graph Commit**.
- A **Graph Commit** references a **Root Node** (empty tree) as its tree.
- A **Graph Commit** references a standard-namespace commit via
  its `Source-Commit` trailer.

## Validation Rules

### Graph Name Validation

A graph name is a single Git ref-name component. The following are rejected:

| Rule | Example Invalid Input | Error Message |
|------|----------------------|---------------|
| Empty string | `""` | "graph name must not be empty" |
| Single `@` | `"@"` | "graph name must not be '@'" |
| Leading dot | `".hidden"` | "graph name must not start with '.'" |
| Trailing dot | `"name."` | "graph name must not end with '.'" |
| `.lock` suffix | `"name.lock"` | "graph name must not end with '.lock'" |
| Double dot | `"a..b"` | "graph name must not contain '..'" |
| `@{` sequence | `"a@{b"` | "graph name must not contain '@{'" |
| Control chars | `"a\x00b"` | "graph name must not contain control character 0x00" |
| Forbidden chars | `"a b"`, `"a~b"`, `"a:b"` | "graph name must not contain ' '" |
| Slash | `"a/b"` | "graph name must not contain '/'" |

### Pre-Condition Validation

| Command | Pre-Condition | Error |
|---------|---------------|-------|
| `init` | Must be in a Git repository | "not a git repository" |
| `init` | HEAD must resolve to a commit | "repository has no commits" |
| `init` | `refs/infra/<name>` must not exist | "graph '<name>' already exists" |
| `list` | Must be in a Git repository | "not a git repository" |
| `delete` | Must be in a Git repository | "not a git repository" |
| `delete` | `refs/infra/<name>` must exist | "graph '<name>' not found" |
