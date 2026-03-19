# Feature Specification: Recursive Tree Listing (`grif tree`)

<!-- markdownlint-disable MD013 -->

**Feature Branch**: `004-graph-tree`
**Created**: 2026-03-18
**Status**: Draft
**Input**: User description: "grif tree — Recursive Tree Listing: Display the full tree structure of a graph or subtree with optional graph/path argument, tree-style box-drawing output, depth limiting, and JSON support. When no argument is provided, display all graphs."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Full Tree of a Named Graph (Priority: P1)

A user has an infrastructure graph with a hierarchy of tree and blob nodes and wants to see the entire structure at a glance. They run `grif tree <graph>` and see a recursive listing of every node in the graph, rendered with box-drawing characters (`├──`, `└──`, `│`) matching the style of the Unix `tree` command. Tree nodes show their name; blob nodes show their name followed by `(blob, <8-char-hash>)`. This replaces the need to run repeated `grif get` calls at each level.

**Why this priority**: This is the core value of the feature — visualizing the complete hierarchy of a single graph. All other stories (no-arg default, subtree, depth limit, JSON) build on this foundation.

**Independent Test**: Can be fully tested by initializing a graph, putting several blob and tree nodes at various depths, running `grif tree <graph>`, and verifying the output matches the expected box-drawing tree format with correct node names, types, and hashes.

**Acceptance Scenarios**:

1. **Given** a graph "default" containing `network/vpc` (blob), `network/subnet` (blob), and `compute/instance` (blob), **When** the user runs `grif tree default`, **Then** the output displays the graph name as the root followed by a recursive tree listing using box-drawing characters, with entries sorted alphabetically at each level.
2. **Given** a graph "default" with no nodes (empty root tree), **When** the user runs `grif tree default`, **Then** the output displays only the graph name (`default`) with no children.
3. **Given** no graph named "nonexistent" exists, **When** the user runs `grif tree nonexistent`, **Then** the command fails with a descriptive error message and a non-zero exit code.

**Example — full tree**:

```text
$ grif tree default
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    ├── subnet  (blob, e5f6a7b8)
    └── vpc  (blob, a1b2c3d4)
```

**Example — empty graph**:

```text
$ grif tree default
default
```

**Example — graph not found (stderr)**:

```text
$ grif tree nonexistent
Error: graph 'nonexistent' not found
```

---

### User Story 2 - View Trees for All Graphs (Priority: P1)

A user wants to see the full tree structure of every graph in the repository without specifying each one individually. They run `grif tree` with no arguments and see the tree output for all graphs, listed alphabetically. When only one graph exists, just that graph's tree is shown. When no graphs exist, the command fails with a descriptive error.

**Why this priority**: This story shares P1 because the no-argument behavior is the primary convenience enhancement requested for this feature. It makes `grif tree` immediately useful for orientation in any repository.

**Independent Test**: Can be fully tested by creating two graphs with nodes, running `grif tree` with no arguments, and verifying both graphs appear in alphabetical order with correct tree output.

**Acceptance Scenarios**:

1. **Given** a repository with exactly one graph "default" containing `network/vpc` (blob), **When** the user runs `grif tree` (no argument), **Then** the tree for "default" is displayed.
2. **Given** a repository with graphs "default" (containing `network/vpc`) and "staging" (containing `compute/instance`), **When** the user runs `grif tree` (no argument), **Then** the trees for both graphs are displayed in alphabetical order ("default" first, then "staging"), separated by a blank line.
3. **Given** a repository with no graphs, **When** the user runs `grif tree`, **Then** the command fails with a descriptive error indicating no graphs exist.

**Example — multiple graphs**:

```text
$ grif tree
default
├── compute
│   └── instance  (blob, c3d4e5f6)
└── network
    ├── subnet  (blob, e5f6a7b8)
    └── vpc  (blob, a1b2c3d4)

staging
└── compute
    └── instance  (blob, c3d4e5f6)
```

**Example — no graphs (stderr)**:

```text
$ grif tree
Error: no graphs found
```

---

### User Story 3 - View Subtree at a Specific Path (Priority: P2)

A user wants to inspect only a portion of a graph's hierarchy. They run `grif tree <graph>/<path>` and see a tree listing rooted at the specified subtree node. The root node displayed is the last segment of the path, not the full path.

**Why this priority**: Subtree viewing is the natural refinement once full-graph viewing works. It enables targeted inspection of deeply nested structures without visual clutter from unrelated branches.

**Independent Test**: Can be fully tested by creating a graph with a deep hierarchy, running `grif tree <graph>/<subtree-path>`, and verifying only the specified subtree is displayed with correct box-drawing formatting.

**Acceptance Scenarios**:

1. **Given** a graph "default" containing `network/vpc` (blob) and `network/subnet` (blob), **When** the user runs `grif tree default/network`, **Then** the output shows `network` as the root with `subnet` and `vpc` as children.
2. **Given** a graph "default" where `network/vpc` is a blob, **When** the user runs `grif tree default/network/vpc`, **Then** the command displays `vpc  (blob, <8-char-hash>)` as a single leaf entry.
3. **Given** a graph "default" with no node at `nonexistent/path`, **When** the user runs `grif tree default/nonexistent/path`, **Then** the command fails with a descriptive error.

**Example — subtree**:

```text
$ grif tree default/network
network
├── subnet  (blob, e5f6a7b8)
└── vpc  (blob, a1b2c3d4)
```

**Example — leaf blob**:

```text
$ grif tree default/network/vpc
vpc  (blob, a1b2c3d4)
```

---

### User Story 4 - Limit Recursion Depth (Priority: P2)

A user has a deeply nested graph and wants to see only the top levels of the hierarchy. They run `grif tree <graph> --depth N` and see the tree truncated at depth N. Depth 0 shows only the root; depth 1 shows the root and its immediate children, and so on.

**Why this priority**: Depth limiting is a display modifier that enhances usability for large graphs. It is independently valuable but depends on the core tree display being implemented first.

**Independent Test**: Can be fully tested by creating a graph with 3+ levels of nesting, running `grif tree --depth 1`, and verifying only the root and immediate children are shown (no deeper descendants).

**Acceptance Scenarios**:

1. **Given** a graph "default" with `compute/instance` (blob) and `network/subnet` (blob), **When** the user runs `grif tree default --depth 1`, **Then** only `compute` and `network` are shown as children of `default` (no deeper nodes).
2. **Given** a graph "default" with nodes, **When** the user runs `grif tree default --depth 0`, **Then** only the graph name `default` is displayed with no children.
3. **Given** the `--depth` flag with a negative value, **When** the user runs `grif tree default --depth -1`, **Then** the command fails with a descriptive error.

**Example — depth 1**:

```text
$ grif tree default --depth 1
default
├── compute
└── network
```

**Example — depth 0**:

```text
$ grif tree default --depth 0
default
```

---

### User Story 5 - Machine-Readable JSON Output (Priority: P3)

A user or automation script needs machine-readable output of the graph tree structure. They run `grif tree --json [<graph>[/<path>]]` and receive a JSON representation of the tree hierarchy. This enables integration with tooling that consumes graph structure programmatically.

**Why this priority**: JSON output is a standard pattern across all `grif` commands and enables programmatic consumption, but it is less commonly used interactively than the human-readable tree format.

**Independent Test**: Can be fully tested by creating a graph with nodes, running `grif tree --json`, and verifying the output is valid JSON containing a nested tree of node objects.

**Acceptance Scenarios**:

1. **Given** a graph "default" with `network/vpc` (blob), **When** the user runs `grif tree --json default`, **Then** valid JSON is written to stdout containing a tree structure with name, type, id, and children fields.
2. **Given** two graphs, **When** the user runs `grif tree --json` (no argument), **Then** valid JSON is written containing an object with a `graphs` array of tree objects (one per graph) and an optional `warnings` array.
3. **Given** the `--depth` flag is also specified, **When** the user runs `grif tree --json --depth 1 default`, **Then** the JSON output respects the depth limit.

**Example — JSON output for a single graph**:

```text
$ grif tree --json default
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
    },
    {
      "name": "network",
      "type": "tree",
      "id": "a7b8c9d0",
      "children": [
        {
          "name": "subnet",
          "type": "blob",
          "id": "e5f6a7b8"
        },
        {
          "name": "vpc",
          "type": "blob",
          "id": "a1b2c3d4"
        }
      ]
    }
  ]
}
```

**Example — JSON output for all graphs**:

```text
$ grif tree --json
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

---

### Edge Cases

- What happens when the graph has an empty root tree (no nodes)? The output MUST display only the graph name with no child lines.
- What happens when `--depth` is given a negative value? The command MUST return an error indicating the value must be a non-negative integer.
- What happens when the target path resolves to a blob? The command MUST display a single line showing the blob name with its type and abbreviated hash.
- What happens when the target path does not exist within the graph? The command MUST fail with a descriptive error identifying the missing path.
- What happens when the graph ref exists but the tree object it points to is corrupted or missing? The command MUST return a descriptive error rather than a panic or raw Git error.
- What happens when `grif tree` is run with no arguments and no graphs exist? The command MUST fail with a descriptive error indicating no graphs were found.
- What happens with very deep trees and no depth limit? The command MUST handle arbitrarily deep hierarchies without stack overflow (iterative or bounded recursion).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `tree` command MUST accept an optional positional argument in the form `[<graph>[/<path>]]` identifying the target graph and optional subtree path.
- **FR-002**: When no argument is provided, the command MUST display the tree for all graphs in the repository, listed in alphabetical order by graph name. When multiple graphs are shown, they MUST be separated by a blank line.
- **FR-003**: When no argument is provided and no graphs exist, the command MUST fail with a descriptive error and a non-zero exit code.
- **FR-004**: When a graph name is provided, the command MUST display the recursive tree structure of that graph's current state (staged tree if a staging ref exists, otherwise the last committed tree).
- **FR-005**: When a graph name and path are provided, the command MUST display the recursive tree starting from the subtree rooted at that path. The displayed root is the last path segment, not the full path.
- **FR-006**: When the specified graph does not exist, the command MUST fail with a descriptive error and a non-zero exit code.
- **FR-007**: When the specified path does not exist within the graph, the command MUST fail with a descriptive error and a non-zero exit code.
- **FR-008**: The human-readable output MUST use box-drawing characters matching the Unix `tree` command style: `├──` for non-last siblings, `└──` for last siblings, and `│` for vertical continuation lines.
- **FR-009**: Entries at each level MUST be sorted in case-sensitive alphabetical order (Go `sort.Strings` on entry names), consistent with Git's default tree entry ordering for non-directory entries.
- **FR-010**: Tree nodes MUST be displayed with only their name. Blob nodes MUST be displayed with their name followed by two spaces and `(blob, <8-char-hash>)` where the hash is the first 8 characters of the object's SHA. No column alignment is applied — spacing is fixed regardless of sibling name lengths.
- **FR-011**: When the target resolves to a blob node, the command MUST display a single line with the blob name, type, and abbreviated hash.
- **FR-012**: The `tree` command MUST support a `--depth N` flag that limits recursion to N levels below the root. Depth 0 shows only the root node. Depth 1 shows the root and its immediate children. Negative values MUST produce an error.
- **FR-013**: The `tree` command MUST support a `--json` flag that outputs the tree as a JSON structure. For a single graph or subtree, the output is a single JSON object with `name`, `type`, `id`, and optional `children` fields. For all graphs (no argument), the output is always a JSON object with a `graphs` array and an optional `warnings` array.
- **FR-014**: The `--depth` flag MUST work in combination with both human-readable and JSON output modes. When used with the no-argument (all-graphs) form, the depth limit MUST apply uniformly to every graph displayed.
- **FR-015**: The `tree` command MUST verify it is running inside a valid Git repository; otherwise it MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-016**: The `tree` command MUST write normal output to stdout and errors to stderr.
- **FR-017**: The `tree` command MUST exit with code 0 on success and non-zero on failure.
- **FR-018**: This command is read-only; it MUST NOT modify any refs, objects, or repository state.
- **FR-019**: When displaying all graphs (no argument) and one or more graphs fail to resolve (e.g., corrupted tree object), the command MUST display all successfully resolved graphs, emit a warning to stderr for each failed graph, and exit with code 0 (partial success). In JSON mode, warnings MUST be included in the `warnings` array of the wrapper object. This is consistent with the partial-success pattern in `grif log` (broken chain handling).

### Key Entities

- **Tree Node**: A Git tree object within the graph's hierarchy. Acts as a container with named children. Displayed with only its name in human-readable output; includes `type: "tree"` and `id` in JSON output.
- **Blob Node**: A Git blob object at the leaf level of the graph's hierarchy. Contains raw byte content. Displayed with name, type indicator, and abbreviated hash in human-readable output; includes `type: "blob"` and `id` in JSON output.
- **Graph Root**: The top-level tree pointed to by the graph's staging ref (if present) or the committed tree from the graph ref. This is the starting point for tree traversal when only a graph name is provided.
- **Graph Ref**: The Git ref at `refs/infra/<name>` that identifies a graph. Used to resolve the root tree for display.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can view the complete tree structure of any graph in under 1 second for graphs with up to 1,000 nodes on a local repository.
- **SC-002**: Running `grif tree` with no arguments displays all graphs in correct alphabetical order with complete recursive hierarchies.
- **SC-003**: The output uses correct box-drawing characters that visually match the Unix `tree` command — verifiable by visual comparison.
- **SC-004**: Using `--depth N` produces output limited to exactly N levels below the root — verifiable by counting nesting levels.
- **SC-005**: JSON output is valid, parseable JSON that can be consumed by standard tools (e.g., `jq`) without errors.
- **SC-006**: All error conditions (missing graph, invalid path, invalid depth, not a repo) produce clear messages that identify the specific problem without exposing raw Git internals.
- **SC-007**: The command produces no side effects — running `grif tree` does not modify any refs or objects in the repository.

## Assumptions

- A graph has already been initialized using `grif init` and optionally has nodes created via `grif put` before `grif tree` is used.
- The tree displays the current state: the staged tree (from the staging ref) if uncommitted changes exist, otherwise the last committed tree. This is consistent with how `grif get` resolves its root tree.
- Entry sorting uses Go `sort.Strings` (case-sensitive lexicographic order), which is consistent with Git's default tree entry ordering for non-directory entries.
- The graph name itself serves as the root label in human-readable output, not the ref path.
- When displaying all graphs (no argument), each graph's tree is independently resolved and rendered. A failure resolving one graph does not prevent displaying others; the command emits a warning to stderr for the failed graph and exits 0 (partial success per FR-019).

## Clarifications

### Session 2026-03-18

- Q: Should sibling blob nodes have column-aligned annotations (padded to longest name) or fixed spacing? → A: Fixed spacing — two spaces between name and `(blob, hash)`, no column alignment.
- Q: When `grif tree` (no args) encounters a graph whose tree cannot be resolved, should it fail entirely or show partial results? → A: Partial success — display all successfully resolved graphs, emit a warning to stderr for each failed graph, exit 0.
- Q: When `--depth` is used with the no-argument (all-graphs) form, does the depth limit apply uniformly to all graphs? → A: Yes, depth applies uniformly to all graphs.
