# Feature Specification: Graph Init, List, and Delete

**Feature Branch**: `001-graph-init-list`
**Created**: 2026-02-13
**Status**: Draft
**Input**: User description: "Write the first operations for the infra graph: init, list, and delete. Init creates a named graph with a root node, commits it with a trailer referencing the current HEAD, and creates a custom ref. List shows all existing graphs. Delete removes the graph ref, allowing Git garbage collection to reclaim unreferenced objects."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Initialize a New Graph (Priority: P1)

A user wants to create a new infrastructure graph in their Git repository. They run the `init` command with a graph name. The system creates an empty root tree representing the graph's root node, commits it with a trailer that records the current HEAD commit SHA of the active branch, and creates a custom Git ref at `refs/infra/<graph name>` pointing to that commit. The user now has a named, versioned graph they can build upon.

**Why this priority**: This is the foundational operation. No other graph operation can exist without first initializing a graph. It establishes the core creation flow described in the design document.

**Independent Test**: Can be fully tested by running `init` with a graph name in a Git repository and verifying that the custom ref exists and points to a valid commit containing an empty root tree and the expected trailer.

**Acceptance Scenarios**:

1. **Given** a valid Git repository with at least one commit on the current branch, **When** the user runs `init` with graph name "my-infra", **Then** a new ref `refs/infra/my-infra` is created pointing to a commit that contains an empty root tree and a commit trailer referencing the current HEAD SHA.
2. **Given** a valid Git repository, **When** the user runs `init` with graph name "my-infra" and a graph with that name already exists (the ref `refs/infra/my-infra` is present), **Then** the command fails with a descriptive error message and a non-zero exit code, leaving the existing graph unchanged.
3. **Given** the current working directory is not inside a Git repository, **When** the user runs `init`, **Then** the command fails with a descriptive error message and a non-zero exit code.

---

### User Story 2 - List Existing Graphs (Priority: P2)

A user wants to see which infrastructure graphs have been created in their repository. They run the `list` command. The system enumerates all refs under `refs/infra/` and displays the graph names. If no graphs exist, the output is empty (no error).

**Why this priority**: Listing graphs is the natural complement to creating them. It provides discoverability and is essential for users managing multiple graphs, but it depends on at least one graph having been created first.

**Independent Test**: Can be fully tested by creating one or more graphs with `init`, then running `list` and verifying the output contains exactly the expected graph names.

**Acceptance Scenarios**:

1. **Given** a Git repository with two initialized graphs named "staging" and "production", **When** the user runs `list`, **Then** the output lists "production" before "staging" (alphabetical order), one per line in human-readable mode or as a sorted JSON array in JSON mode.
2. **Given** a Git repository with no initialized graphs, **When** the user runs `list`, **Then** the output is empty (human-readable mode produces no output lines; JSON mode produces an empty array) and the exit code is zero.
3. **Given** the current working directory is not inside a Git repository, **When** the user runs `list`, **Then** the command fails with a descriptive error message and a non-zero exit code.

---

### User Story 3 - Delete an Existing Graph (Priority: P3)

A user wants to remove an infrastructure graph from their repository. They run the `delete` command with the graph name. The system deletes the ref at `refs/infra/<graph name>`. Once the ref is removed, the commit and tree objects it pointed to become unreferenced and will be reclaimed by Git's garbage collection.

**Why this priority**: Deletion is the complement to creation. It enables cleanup of graphs that are no longer needed. It ranks below init and list because it requires a graph to exist first and is a less frequent operation.

**Independent Test**: Can be fully tested by creating a graph with `init`, verifying it appears in `list`, running `delete`, and confirming the graph no longer appears in `list` and the ref no longer exists.

**Acceptance Scenarios**:

1. **Given** a Git repository with an initialized graph named "my-infra", **When** the user runs `delete` with graph name "my-infra", **Then** the ref `refs/infra/my-infra` is removed, the command exits with code 0, and the graph no longer appears in `list` output.
2. **Given** a Git repository with no graph named "nonexistent", **When** the user runs `delete` with graph name "nonexistent", **Then** the command fails with a descriptive error message and a non-zero exit code.
3. **Given** the current working directory is not inside a Git repository, **When** the user runs `delete`, **Then** the command fails with a descriptive error message and a non-zero exit code.

---

### Edge Cases

- What happens when the graph name contains characters that are invalid in Git ref names (e.g., spaces, `..`, `~`, `^`, `:`, `\`, control characters)? The command MUST reject the name with a clear error before attempting to create any Git objects.
- What happens when the repository has no commits yet (empty repository with no HEAD)? The command MUST fail with an error explaining that at least one commit is required because the trailer must reference a valid HEAD SHA.
- What happens when multiple `init` commands are run concurrently with the same graph name? Only one should succeed; the other MUST fail with a duplicate-graph error.
- What happens when a ref path under `refs/infra/` exists but was not created by this tool (e.g., manually created)? The `list` command MUST still include it; the `init` command MUST still treat it as a duplicate; the `delete` command MUST still delete it.
- What happens when `delete` is run concurrently for the same graph name? Only one should succeed; the other MUST fail with a "graph not found" error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `init` command MUST accept a graph name as a required argument.
- **FR-002**: The `init` command MUST validate that the graph name is a legal Git ref name component (no spaces, `..`, `~`, `^`, `:`, `\`, control characters, or names ending in `.lock`).
- **FR-003**: The `init` command MUST verify that it is running inside a valid Git repository; otherwise it MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-004**: The `init` command MUST verify that `refs/infra/<graph name>` does not already exist; if it does, the command MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-005**: The `init` command MUST verify that HEAD resolves to a valid commit; if the repository is empty (no commits), the command MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-006**: The `init` command MUST create an empty root tree as the graph's initial root node.
- **FR-007**: The `init` command MUST create an orphan commit (no parent) whose tree is the empty root tree. The commit message MUST include a trailer that records the SHA of the current HEAD commit from the standard Git refs namespace. The relationship to the source commit is captured exclusively via the trailer, not via Git commit parentage.
- **FR-008**: The `init` command MUST create the ref `refs/infra/<graph name>` pointing to the newly created commit.
- **FR-009**: The `init` command MUST NOT modify any existing refs in the standard namespace (branches, tags, HEAD).
- **FR-010**: The `list` command MUST enumerate all refs under the `refs/infra/` namespace and output the graph name portion of each ref, sorted in lexicographic (alphabetical) order.
- **FR-011**: The `list` command MUST verify that it is running inside a valid Git repository; otherwise it MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-012**: The `list` command MUST return an empty result (not an error) when no graphs exist.
- **FR-013**: The `delete` command MUST accept a graph name as a required argument.
- **FR-013a**: The `delete` command MUST validate that the graph name is a legal Git ref name component (same rules as FR-002); if invalid, it MUST fail with a non-zero exit code and a descriptive error on stderr before attempting any ref lookup.
- **FR-014**: The `delete` command MUST verify that it is running inside a valid Git repository; otherwise it MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-015**: The `delete` command MUST verify that `refs/infra/<graph name>` exists; if it does not, the command MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-016**: The `delete` command MUST delete the ref `refs/infra/<graph name>`. It MUST NOT delete or modify any other refs.
- **FR-017**: The `delete` command MUST NOT directly delete Git objects (blobs, trees, commits). Object cleanup is delegated to Git's garbage collection of unreferenced objects.
- **FR-018**: All commands MUST support human-readable output (default) and JSON output when a JSON flag is provided.
- **FR-019**: All commands MUST write normal output to stdout and errors/diagnostics to stderr.
- **FR-020**: All commands MUST exit with code 0 on success and non-zero on failure.

### Key Entities

- **Graph**: A named infrastructure graph identified by a ref under `refs/infra/<name>`. Created by `init`, discoverable by `list`, removable by `delete`. Each graph has a root node (empty tree) and a lineage of commits.
- **Root Node**: The initial node of a graph, represented as an empty Git tree object. Serves as the top of the containment hierarchy.
- **Graph Commit**: An orphan Git commit object (no parent) that captures a graph snapshot. Its tree is the graph's root. Its commit message includes a trailer with the SHA of a commit from the standard Git refs namespace (enabling co-versioning). The graph commit lineage is fully independent from the repository's standard commit DAG; subsequent graph commits will form their own separate lineage under the graph ref.
- **Graph Ref**: A Git ref at `refs/infra/<graph name>` that points to the latest graph commit, tracking the head of the graph's snapshot lineage. Deleting this ref via the `delete` command makes the entire graph commit lineage unreferenced and eligible for Git garbage collection.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can initialize a new graph and verify its existence within a single terminal session (under 5 seconds wall-clock time for the `init` command on a local repository).
- **SC-002**: Running `list` after creating N graphs returns exactly N graph names with no duplicates or omissions.
- **SC-003**: Attempting to create a duplicate graph always fails with a clear error — no silent overwriting occurs under any circumstances.
- **SC-004**: All validation errors (not in a repo, duplicate name, invalid name, empty repo) produce a meaningful error message that identifies the specific problem, without requiring the user to interpret raw Git errors.
- **SC-005**: The graph commit's trailer correctly references the HEAD SHA that was current at the time of `init`, verifiable by inspecting the commit object.
- **SC-006**: After `init`, no standard-namespace refs (branches, tags, HEAD) are modified — verifiable by comparing ref state before and after.
- **SC-007**: After `delete`, the named graph no longer appears in `list` output and the ref `refs/infra/<name>` no longer exists.
- **SC-008**: After `delete`, no standard-namespace refs (branches, tags, HEAD) are modified — verifiable by comparing ref state before and after.

## Clarifications

### Session 2026-02-13

- Q: Should the graph commit created by `init` have a parent (linked to repo history) or be an orphan commit? → A: Orphan commit (no parent). Graph lineage is fully independent from standard branch history; the source commit relationship is captured solely via the trailer.
- Q: What sort order should the `list` command use for graph names? → A: Alphabetical (lexicographic by graph name).

## Assumptions

- The user has Git installed and available on their PATH.
- The repository has at least one commit before `init` is run (the trailer requires a valid HEAD SHA to reference).
- Graph names are simple identifiers (single ref-name component), not nested paths. For example, "my-infra" is valid but "team/my-infra" is not in scope for this feature.
- The trailer key name used in the commit message (e.g., `Source-Commit:`) will be defined during implementation; this spec does not prescribe the exact key.
