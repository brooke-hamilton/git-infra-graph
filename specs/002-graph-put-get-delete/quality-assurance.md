# Quality Assurance: Graph Node Put, Get, and Delete

**Feature**: 002-graph-put-get-delete
**Date**: 2026-02-13
**Scope**: Code review of implementation against plan, spec, and contracts

## Issues Found

### 1. BUG: `--data ""` doesn't create an empty blob — it creates a tree (Severity: High)

In `src/cmd/grif/main.go`, `flagValue("--data")` returns `""` for `--data ""`,
and the `switch` statement treats empty string as "no data provided":

```go
case dataVal != "":
    blob = []byte(dataVal)
```

Per FR-027 and the contract: *"Use `--data ""` for an empty blob."* The current
code silently creates a tree node instead. The fix requires distinguishing
between "flag not present" and "flag present with empty value". `flagValue`
should return a sentinel or the CLI should track whether `--data` was passed at
all.

### 2. BUG: `flagValue` has a technically correct but confusing bounds check (Severity: Low)

In `src/cmd/grif/main.go`:

```go
if arg == flag && i+1 < len(os.Args[2:])-0 {
```

The `-0` is a no-op and reads like an editing artifact. It works correctly but
is confusing. Should just be `i+1 < len(allArgs)`.

### 3. BUG: `runRm` calls `countDescendants` after the node is already deleted (Severity: Medium)

In `src/cmd/grif/main.go`, `countDescendants` recursively calls `graph.Get` to
count children. The `Get` call in `runRm` happens **before** `DeleteNode`,
which is correct. However, `countDescendants` calls `graph.Get` on child paths
**after** `DeleteNode` has already removed them. This means for nested trees,
the recursive `Get` calls will fail and descendants won't be counted properly.

The fix: `countDescendants` should only use the `content` that was already
fetched (and recursively pre-fetch children) **before** calling `DeleteNode`.
Alternatively, restructure to collect the full descendant count before deleting.

### 4. QUALITY: `Get` tree children not explicitly sorted (Severity: Low)

The contract states: *"Children are sorted alphabetically by Name (Git tree
entry order)."* The code in `src/graph/graph.go` relies on Git's internal tree
entry ordering, which happens to be alphabetical for entries of the same type
but uses a different sort key for trees vs blobs (trees get a trailing `/` in
Git's sort). The `StoreTree` function in `src/graph/internal/gitops/objects.go`
uses `object.TreeEntrySorter` which follows Git's canonical encoding order, not
strict alphabetical order. For most cases this works, but edge cases (e.g., a
blob named `ab` and a tree named `ab-x`) may not sort alphabetically. The
contract says "sorted alphabetically by Name" — consider adding an explicit
`sort.Slice` by `Name` on the `children` slice returned by `Get`.

### 5. QUALITY: `ParseNodePath` is exported but the contract says it "may be exported or unexported" (Severity: Info)

The contract note says: *"This may be exported or unexported depending on
whether external consumers need path parsing."* It is currently exported. This
is a conscious design choice — just flagging it for awareness.

### 6. QUALITY: `get` CLI human output uses tabs, not aligned columns (Severity: Low)

The contract shows tree output as:

```text
TYPE  NAME    ID
blob  vpc     a1b2c3d4
tree  subnet  e5f6a7b8
```

The implementation in `src/cmd/grif/main.go` uses `\t` (tab characters):

```go
fmt.Println("TYPE\tNAME\tID")
fmt.Printf("%s\t%s\t%s\n", child.Type, child.Name, shortID)
```

Tab-separated output will only align properly in a terminal with consistent tab
stops. The spec shows space-padded columns. Consider using `text/tabwriter` for
proper alignment.

### 7. QUALITY: `--data` and `--file` mutual exclusivity check is incomplete (Severity: Medium)

In `src/cmd/grif/main.go`, the code only checks if both `--data` and `--file`
are provided simultaneously. Per FR-027, stdin should also be mutually exclusive
with `--data` and `--file`. If a user pipes stdin and also passes `--data`, the
`--data` wins silently without error (because the `switch` checks
`dataVal != ""` first). This isn't technically wrong (the spec says "checked in
this order") but the contract says "mutually exclusive" — the CLI should error
if multiple sources are detected.

### 8. QUALITY: `Delete` (graph-level) doesn't clean up staging ref (Severity: Medium)

In `src/graph/graph.go`, the graph-level `Delete` function removes
`refs/infra/<name>` but does **not** remove `refs/infra-stage/<name>` if it
exists. If a user does `put` + `delete` (graph-level) without committing, a
dangling staging ref is left behind. This is from feature 001 but worth fixing
now.

### 9. QUALITY: No test for `Commit` default message format (Severity: Low)

The test "commit staged changes" commits with empty message (triggering default
format) but doesn't verify the commit message contains the expected
`Update graph "default"` prefix and `Source-Commit:` trailer. Only the custom
message test checks message content.

### 10. QUALITY: No test for `Get` returning empty children for empty tree (Severity: Low)

There's no test that creates an explicit empty tree via `Put(path, nil)` and
then verifies `Get` returns a `NodeContent` with `Type=TreeNode`, `Children` as
an empty slice (not nil), and `Blob` as nil.

### 11. QUALITY: `Status` only reports top-level changes (Severity: High)

The `Status` function in `src/graph/graph.go` uses `object.DiffTree` which only
compares entries in the **immediate** children of the root trees. It does
**not** recurse into subtrees. If a user does `Put("default/network/vpc",
data)`, the Status will report `network` as added/modified (the tree entry), not
`network/vpc` (the blob). This contradicts the contract which shows individual
leaf paths like `network/vpc` and `network/subnet` as separate status entries.

The fix requires recursive tree diffing — using
`object.DiffTreeWithOptions` or manually walking the diff to produce leaf-level
change paths.

## Summary

| # | Severity | Category | Description |
| --- | --- | --- | --- |
| 1 | High | Bug | `--data ""` creates tree instead of empty blob |
| 2 | Low | Quality | `flagValue` has `-0` no-op in bounds check |
| 3 | Medium | Bug | `countDescendants` calls `Get` after tree is deleted |
| 4 | Low | Quality | `Get` children rely on Git sort order, not explicit alpha sort |
| 5 | Info | Quality | `ParseNodePath` exported (deliberate choice, just noting) |
| 6 | Low | Quality | Tree listing uses raw tabs instead of `tabwriter` |
| 7 | Medium | Quality | Stdin not checked for mutual exclusivity with `--data`/`--file` |
| 8 | Medium | Quality | Graph-level `Delete` leaves orphaned staging ref |
| 9 | Low | Quality | No test for default commit message format |
| 10 | Low | Quality | No test for `Get` on empty tree returning empty `Children` slice |
| 11 | High | Bug | `Status` reports tree-level diffs, not leaf-level paths per contract |

## Recommended Fix Priority

1. **#11** — `Status` not reporting leaf-level paths (breaks contract)
2. **#1** — `--data ""` creating tree instead of empty blob (breaks contract)
3. **#3** — `countDescendants` failing for nested trees (incorrect output)
4. **#8** — Graph-level `Delete` leaving dangling staging ref
5. **#7** — Incomplete mutual exclusivity check for input sources
6. Remaining low-severity items
