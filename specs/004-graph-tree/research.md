# Research: Recursive Tree Listing (`grif tree`)

<!-- markdownlint-disable MD013 -->

## R1: Tree Traversal Strategy (go-git)

**Decision**: Recursive traversal using `gitops.GetTreeByHash` + `gitops.ResolveRootTree`,
matching the existing pattern in `Get`, `diffTreesRecursive`, and `collectAllLeaves`.

**Rationale**: The codebase already traverses Git trees recursively in multiple places.
`GetTreeByHash` returns `*object.Tree` with an `Entries` slice of `TreeEntry` structs
(each having `Name`, `Hash`, `Mode` fields). The `ResolveRootTree` function already
handles the staging-vs-committed resolution (check `refs/infra-stage/<name>` first,
fall back to `refs/infra/<name>` commit tree). No new gitops functions are needed.

**Alternatives considered**:

- **Iterative traversal with explicit stack**: Would avoid recursion depth limits but
  adds complexity. Go's default goroutine stack starts at 8 KB and grows dynamically
  up to 1 GB, so recursion depth is not a practical concern even for thousands of
  levels. The existing codebase uses recursion for tree traversal (`diffTreesRecursive`,
  `collectAllLeaves`), so recursive traversal is idiomatic here.
- **go-git's `Tree.Files()` iterator**: Only yields blobs, skipping intermediate tree
  nodes. We need to visit both trees and blobs to render the hierarchy, so this is
  insufficient.

## R2: Box-Drawing Character Rendering

**Decision**: Implement a recursive renderer that tracks prefix state for each level
using two constants: branch connector (`├── `) for non-last siblings and last connector
(`└── `) for the last sibling. Continuation lines use `│   ` (pipe + 3 spaces) for
non-last and `    ` (4 spaces) for last.

**Rationale**: This is the standard algorithm used by the Unix `tree` command. Each
recursive call receives a prefix string that accumulates the vertical continuation
characters from all ancestor levels. At each level, entries are sorted alphabetically
and the last entry uses `└──` while all others use `├──`.

**Box-drawing constants**:

```text
├── (non-last sibling connector)
└── (last sibling connector)
│   (non-last continuation)
    (last continuation, 4 spaces)
```

**Alternatives considered**:

- **External `tree` command**: Not portable; violates Git-native principle. The binary
  must work without external dependencies.
- **Third-party Go tree-rendering library**: Unnecessary dependency for a straightforward
  algorithm. The rendering logic is ~30 lines of code.

## R3: Depth Limiting

**Decision**: Add an integer `depth` parameter to the `Tree` function via a `TreeOptions`
struct. During recursive traversal, decrement depth at each level. When depth reaches 0,
stop recursion (show only the current node, no children). A negative value in the option
produces an error. Distinguish "not set" from "set to 0" using a `HasDepth bool` field
(same pattern as `LogOptions.HasMaxCount`).

**Rationale**: Depth limiting is a display concern but must be applied at the module
level because the module performs the tree walk. Applying it at the CLI level would
require fetching the entire tree and then truncating, which is wasteful for very large
graphs.

**Depth semantics**:

- Depth 0: Show only the root node (graph name or subtree root), no children
- Depth 1: Root + immediate children
- Depth N: Root + N levels of descendants
- No depth set: Unlimited recursion

**Alternatives considered**:

- **CLI-level truncation**: Fetch full tree then truncate output. Wastes resources for
  large graphs; inconsistent with the module-first principle.
- **Sentinel value (-1 = unlimited)**: Less explicit than a boolean flag; the
  `HasMaxCount` pattern is already established in `LogOptions`.

## R4: JSON Output Structure

**Decision**: Use a recursive JSON node structure with `name`, `type`, `id`, and
optional `children` fields. Single graph/subtree (`grif tree <graph>`) returns a
single node object. All-graphs mode (`grif tree --all`) returns a wrapper object
with a `graphs` array of node objects and an optional `warnings` array, matching
the `TreeAllResult` contract. Blob nodes omit `children`. Tree nodes include
`children` as an array (empty array for empty trees, omitted field for blobs).

**Rationale**: This directly matches the spec examples (FR-013) and the Go API/
CLI contract (`TreeAllResult` with `graphs` and optional `warnings`). The node
structure mirrors the Git tree hierarchy. Using `omitempty` on `children` means
blob nodes naturally exclude the field. The `id` field contains the first 8
characters of the SHA hash, consistent with the human-readable display.

**JSON schema** (node object shape, used for single graph and inside `graphs`):

```json
{
  "name": "string",
  "type": "tree|blob",
  "id": "string (8-char hash)",
  "children": [{ ...recursive... }]
}
```

**Alternatives considered**:

- **Flat list with path field**: Loses hierarchy information; harder for consumers
  to reconstruct the tree. The spec explicitly requires nested structure.
- **Full 40-char hash in JSON**: The spec examples show 8-char hashes. Keep
  consistent with human-readable output. The `id` field uses abbreviated hashes.

## R5: Argument Parsing (`[<graph>[/<path>]]`)

**Decision**: Reuse the existing `ParseNodePath` function for the case where an
argument is provided. When the argument contains only a graph name (single segment),
display the entire graph. When it contains a path (multiple segments), display the
subtree at that path. When no argument is given, list all graphs via
`gitops.ListRefsByPrefix`.

**Rationale**: `ParseNodePath` already splits `graph/path/segments` correctly and
validates graph names. The only difference from `Get` is that a single-segment
path (just graph name) is valid for `tree` but not for `get`. The `Tree` function
needs to handle the single-segment case by displaying the full graph.

**Parsing rules**:

- No argument → all graphs mode
- Single segment (e.g., `default`) → full graph tree
- Multiple segments (e.g., `default/network`) → subtree mode

**Alternatives considered**:

- **Separate `--graph` flag + positional path**: Overcomplicates the interface.
  The `graph/path` convention is already established by `put`, `get`, `rm`.
- **New parser for tree paths**: Unnecessary; `ParseNodePath` handles the
  splitting correctly.

## R6: Partial Success for All-Graphs Mode (FR-019)

**Decision**: When displaying all graphs and one graph fails to resolve (e.g.,
corrupted tree object), display all successfully resolved graphs, emit a warning
to stderr for each failed graph, and exit 0. Return a `TreeResult` struct with
a `Warnings []string` field (same pattern as `LogResult.Warning` but as a slice
since multiple graphs can fail).

**Rationale**: FR-019 explicitly requires partial success matching the `grif log`
broken chain pattern. Using a slice of warnings (rather than a single string)
handles the case where multiple graphs fail independently.

**Alternatives considered**:

- **Fail fast on first error**: Violates FR-019; prevents users from seeing
  healthy graphs when one is corrupted.
- **Single warning string with concatenated messages**: Harder to parse; a slice
  is cleaner for both human and JSON output.

## R7: Root Tree Resolution (Staged vs. Committed)

**Decision**: Use the existing `gitops.ResolveRootTree` function, which already
checks for a staging ref (`refs/infra-stage/<name>`) first and falls back to the
committed tree from the graph ref (`refs/infra/<name>`). This matches the behavior
of `Get`.

**Rationale**: The spec assumption states "the tree displays the current state:
the staged tree (from the staging ref) if uncommitted changes exist, otherwise
the last committed tree. This is consistent with how `grif get` resolves its
root tree." Using the same function guarantees consistency.

**Alternatives considered**:

- **Always show committed tree**: Would be inconsistent with `grif get` and miss
  uncommitted changes, which is confusing for users.
- **Flag to choose staged vs committed**: Over-engineering; not in the spec.

## R8: Entry Sorting

**Decision**: Sort entries alphabetically at each level using Go's `sort.Slice`
with string comparison on `Name`. This is case-sensitive, consistent with Git's
default tree entry ordering.

**Rationale**: The spec states "entries at each level MUST be sorted alphabetically
by name" (FR-009) and the assumptions section clarifies "case-sensitive alphabetical
order, consistent with Git's default tree entry ordering." Git trees are already
sorted, but we sort explicitly to guarantee correctness regardless of go-git's
internal ordering.

**Alternatives considered**:

- **Case-insensitive sort**: Not specified; would differ from Git's behavior.
- **Trust go-git ordering without re-sorting**: Risky if go-git changes internal
  behavior; explicit sort is cheap and safe.
