# Research Report: Graph Commit History (`grif log`)

<!-- markdownlint-disable MD024 -->

**Feature Branch**: `003-graph-log`
**Date**: 2026-02-17
**Input**: Plan from [plan.md](plan.md)

## R1: Commit Chain Walking with go-git

### Context

The `log` command must walk the commit chain starting from the graph ref's tip commit, following `ParentHashes` back to the orphan root. The codebase already uses `go-git` for all Git operations. The graph commit lineage is linear (each commit has at most one parent, the orphan root has zero parents).

### Approach

go-git provides `repo.CommitObject(hash)` which returns `*object.Commit` with:

- `Hash` — the commit's SHA-1 hash (`plumbing.Hash`)
- `Author` — `object.Signature` with `Name`, `Email`, `When` (`time.Time`)
- `Committer` — `object.Signature` with `Name`, `Email`, `When` (`time.Time`)
- `Message` — full commit message as `string`
- `ParentHashes` — `[]plumbing.Hash` (empty for orphan, one entry for linear)
- `TreeHash` — the tree hash (not needed for log)

The walk algorithm:

```go
ref, _ := repo.Storer.Reference(plumbing.ReferenceName("refs/infra/<name>"))
currentHash := ref.Hash()

for {
    commit, err := repo.CommitObject(currentHash)
    if err != nil {
        // Broken chain — emit warning and stop
        break
    }
    // Process commit...
    if len(commit.ParentHashes) == 0 {
        break // orphan root reached
    }
    currentHash = commit.ParentHashes[0]
}
```

### Decision

Use direct `CommitObject` iteration rather than go-git's `Log` iterator. Walk the chain manually with a simple loop following `ParentHashes[0]`.

### Rationale

1. **Direct control**: Manual walking gives full control over broken chain handling (FR-013), `--max-count` limiting, and error recovery — without needing to understand go-git's iterator semantics or options.
2. **Simplicity**: A `for` loop with `CommitObject` is straightforward and matches the established codebase pattern of direct object store access.
3. **Linear chain only**: Graph commit chains are guaranteed linear (no merges) per the spec assumptions. No need for graph traversal algorithms — simple parent-following suffices.

### Alternatives Considered

- **go-git `repo.Log(&git.LogOptions{From: hash})`**: Returns an iterator that walks commits. However, this iterator follows the standard Git commit graph and handles branching/merging, which adds unnecessary complexity. It also doesn't provide clean hooks for broken chain warnings or partial results. Manual iteration is simpler for the linear case.

## R2: Source-Commit Trailer Extraction

### Context

Each graph commit contains a `Source-Commit: <hash>` trailer in the commit message body. The `log` command must extract and display this value.

### Approach

The codebase already has `extractSourceCommit(message string) string` in [graph.go](../../src/graph/graph.go) that parses the `Source-Commit:` prefix from commit message lines. This function is unexported but can be reused directly since `Log` will be in the same `graph` package.

The function handles:

- Present trailer: returns the hash string
- Missing trailer: returns empty string `""`

This directly satisfies FR-014 (handle missing or malformed `Source-Commit` trailers by displaying an empty value).

### Decision

Reuse the existing `extractSourceCommit` function. No new code needed for trailer parsing.

### Rationale

The function is already tested implicitly through `Init` and `GetInitInfo` tests. It handles the exact format established by `Init` and `Commit` functions.

## R3: Date Formatting

### Context

The spec requires two date formats:

- **Human-readable (FR-005)**: `YYYY-MM-DD HH:MM:SS ±HHMM` (e.g., `2026-02-14 10:30:00 -0700`)
- **JSON (FR-008)**: ISO 8601 format (e.g., `2026-02-14T10:30:00-07:00`)

The source is `commit.Committer.When` which is a `time.Time`.

### Approach

Go's `time.Format` with layout strings:

```go
// Human-readable
commit.Committer.When.Format("2006-01-02 15:04:05 -0700")

// ISO 8601
commit.Committer.When.Format(time.RFC3339)
```

### Decision

Format dates at the point of use — the `Log` function returns `time.Time` in `LogEntry`, and the CLI layer handles formatting for human vs JSON output. Alternatively, the `Log` function can return pre-formatted strings to keep formatting consistent regardless of the caller.

After review, return `time.Time` in the struct and let the CLI format. This follows the module-first principle: the module returns rich data, the CLI formats for display.

### Rationale

Returning `time.Time` keeps the API clean and reusable. Callers (including Go library consumers) get the actual timestamp and can format as needed. The CLI applies the spec-required formats.

## R4: Broken Chain Handling (FR-013)

### Context

If a parent commit object cannot be found while walking the chain, the command must display all reachable commits, emit a warning on stderr, and exit with code 0 (partial success).

### Approach

During the commit walk loop, if `repo.CommitObject(parentHash)` returns an error, the walk terminates. The `Log` function returns:

1. All commits collected so far (partial result)
2. A warning message or flag indicating a broken chain

The CLI checks for the warning and writes it to stderr while still printing the partial results to stdout and exiting 0.

### Implementation Options

**Option A**: Return `([]LogEntry, error)` where error is nil even on broken chain; add a `Warning` field to a result struct.

**Option B**: Return `(*LogResult, error)` where `LogResult` has `Entries []LogEntry` and `Warning string`.

### Decision

Use **Option B** — return `*LogResult` with an optional `Warning` field. This cleanly separates partial success (broken chain with warning) from hard failure (graph not found, not a repo).

### Rationale

A `Warning` field on the result struct avoids overloading the error return for partial success cases. The caller can check `result.Warning != ""` to know if the chain was broken. This pattern doesn't exist elsewhere in the codebase yet, but it's the cleanest way to express "success with caveats."

## R5: Default Graph Selection (FR-002)

### Context

When no graph name is provided, the command should auto-select the sole graph (if exactly one exists) or fail with a list of available graphs.

### Approach

The codebase already implements this pattern in `runCommit` and `runStatus` in [main.go](../../src/cmd/grif/main.go):

```go
if len(args) == 0 {
    graphs, _ := graph.List(repoPath)
    switch len(graphs) {
    case 0:
        printError(jsonMode, "no graphs found")
        os.Exit(1)
    case 1:
        graphName = graphs[0].Name
    default:
        printError(jsonMode, "multiple graphs exist; specify a graph name")
        os.Exit(1)
    }
}
```

### Decision

Replicate this exact pattern in `runLog`. The default graph selection is a CLI concern (not a module concern), consistent with `commit` and `status`.

### Rationale

Keeps the module API simple (`Log` always takes an explicit graph name). The convenience of auto-selection is a CLI behavior, matching the established pattern.

### Additional Note (Spec Enhancement)

The spec says the error for multiple graphs should list the available names (e.g., `"multiple graphs exist, specify one: production, staging"`). The existing `commit`/`status` implementation does not list names. For consistency, `log` will follow the same pattern as existing commands for now. The spec's enhanced error message with graph listing could be added across all commands in a future enhancement.

## R6: Flag Interaction — `--oneline` and `--json` (FR-009)

### Context

When both `--oneline` and `--json` are specified, `--json` takes precedence. JSON output is always the full structured format.

### Approach

This is purely a CLI concern. The flag evaluation order in `runLog`:

1. Check `--json` first
2. If JSON mode: output full JSON array, ignore `--oneline`
3. If not JSON mode: check `--oneline` for compact format vs full format

The `Log` function in the module always returns the full data. Output formatting is the CLI's responsibility.

### Decision

Handle `--json` precedence in the CLI layer only. No module-level awareness of output formats needed.

### Rationale

Consistent with constitution principle I (module-first) and principle II (CLI as sole interface). The module returns complete data; the CLI decides how to render it.

## R7: `--max-count` Validation

### Context

`--max-count N` limits output to N commits. N=0 means no output. Negative N must produce an error (FR-007).

### Approach

**Option A**: Validate in the CLI, pass to `Log` as a field in `LogOptions`.

**Option B**: Validate in the module's `Log` function, return an error for negative values.

### Decision

Validate in the module's `Log` function. `LogOptions.MaxCount` is an `int`; if negative, return an error. If zero, return an empty result. If positive, limit the walk. A special sentinel value (e.g., -1 or 0 with a separate flag) means "no limit" — but since negative is invalid, use `MaxCount: 0` to mean "no limit" and introduce a `HasMaxCount bool` field, or use a `*int` pointer where nil means "no limit."

After consideration: use `MaxCount int` where 0 means no limit, positive means limit, and negative returns an error. This matches the spec: `--max-count 0` displays no commits (but 0 meaning "no limit" conflicts). Re-reading the spec: "A value of 0 displays no commits."

Use `MaxCount int` where:

- Negative: error
- 0: no limit (default, when flag is not provided)
- Positive: limit to N commits

But this conflicts with `--max-count 0` showing no commits. To resolve: add a `HasMaxCount bool` field.

Final design:

```go
type LogOptions struct {
    MaxCount    int  // Maximum number of commits to display
    HasMaxCount bool // Whether MaxCount was explicitly set
}
```

When `HasMaxCount` is true and `MaxCount` is 0: show no commits. When `HasMaxCount` is false: show all commits.

### Decision

Use `LogOptions` with `MaxCount int` and `HasMaxCount bool`. Validate negative `MaxCount` in the `Log` function.

### Rationale

Distinguishes between "not specified" (show all) and "explicitly set to 0" (show none) without overloading the zero value. The bool flag is explicit and avoids pointer semantics.
