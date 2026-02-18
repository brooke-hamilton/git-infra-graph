# Go Module API Contract: Graph Commit History (`grif log`)

**Feature**: 003-graph-log
**Date**: 2026-02-17
**Package**: `graph`

This project is a Go module with a CLI interface — not a REST API. The
"contracts" defined here are the public Go function signatures and types
exposed by the `graph` package, plus the CLI command interface.

## Go Public API

### Types

```go
package graph

import "time"

// LogEntry represents a single commit in the graph's commit history.
type LogEntry struct {
    Hash         string    `json:"hash"`         // Full 40-character commit hash
    Date         time.Time `json:"date"`         // Committer timestamp
    SourceCommit string    `json:"sourceCommit"` // Source-Commit trailer value (empty if missing)
    Author       string    `json:"author"`       // Author name
    Message      string    `json:"message"`      // Full commit message
}

// LogOptions controls the behavior of the Log function.
type LogOptions struct {
    MaxCount    int  // Maximum number of commits to return
    HasMaxCount bool // Whether MaxCount was explicitly set
}

// LogResult is the return value from a successful Log operation.
type LogResult struct {
    Entries []LogEntry `json:"entries"` // Commits in reverse chronological order
    Warning string     `json:"warning,omitempty"` // Non-empty if chain is broken
}
```

### Functions

#### Log

```go
// Log returns the commit history for the specified graph by walking the
// commit chain from the graph ref's tip commit back to the orphan root.
//
// Commits are returned in reverse chronological order (newest first),
// following the parent hashes of each commit object.
//
// Options:
//   - MaxCount: When HasMaxCount is true, limits the result to at most
//     MaxCount entries. A MaxCount of 0 with HasMaxCount true returns no
//     entries. Negative MaxCount values return an error.
//
// Broken chain handling: If a parent commit object cannot be found during
// the walk, all reachable commits collected so far are returned in
// LogResult.Entries, and LogResult.Warning is set to a descriptive message.
// This is a partial success, not an error.
//
// Returns an error if:
//   - repoPath is not a valid Git repository
//   - the graph name is invalid
//   - the graph does not exist (ref refs/infra/<name> not found)
//   - the tip commit object cannot be read
//   - MaxCount is negative (when HasMaxCount is true)
func Log(repoPath string, graphName string, opts LogOptions) (*LogResult, error)
```

**Preconditions**:

- `repoPath` is a path to a Git repository
- `graphName` has been initialized via `Init`
- If `opts.HasMaxCount` is true, `opts.MaxCount` must be non-negative

**Postconditions**:

- No state is modified (read-only, FR-017)
- Returns a `LogResult` with the commit chain entries and optional warning
- Entries are ordered newest-first (reverse chronological)

**Errors**:

| Condition | Error message pattern |
| --------- | --------------------- |
| Not a git repo | "not a git repository: ..." |
| Invalid graph name | (from `ValidateGraphName`) |
| Graph not found | "graph '...' not found" |
| Tip commit unreadable | "failed to read tip commit for graph '...': ..." |
| Negative max-count | "max-count must be a non-negative integer" |

**Non-error conditions** (partial success):

| Condition | Behavior |
| --------- | -------- |
| Broken chain (mid-walk) | `LogResult.Entries` contains reachable commits; `LogResult.Warning` describes the break |

## CLI Interface

### Command

#### `grif log [graph] [--oneline] [--max-count N] [--json]`

Display the commit history for an infrastructure graph.

| Argument/Flag | Required | Description |
| ------------- | -------- | ----------- |
| `[graph]` | No | Graph name; optional if exactly one graph exists |
| `--oneline` | No | Compact one-line format per commit |
| `--max-count N` | No | Limit output to at most N commits |
| `--json` | No | Output as JSON array |

**Default graph selection**:

- No graph argument + exactly one graph exists → auto-selects that graph
- No graph argument + multiple graphs exist → error listing available graphs
- No graph argument + no graphs exist → error

**Flag interactions**:

- `--json` + `--oneline` → `--json` takes precedence; full structured JSON output
- `--max-count` works with all output modes (default, `--oneline`, `--json`)

### Output Formats

#### Default (human-readable)

```text
commit <40-char-hash>
Date:   <YYYY-MM-DD HH:MM:SS ±HHMM>
Source: <source-commit-hash>

    <commit message line 1>
    <commit message line 2>
    ...

```

Each commit block is separated by a blank line. The commit message is
indented with 4 spaces. Only the first paragraph of the message (up to
the first blank line) is displayed, excluding the `Source-Commit:` trailer
line.

**Example**:

```text
commit a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
Date:   2026-02-14 10:30:00 -0700
Source: f9e8d7c6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0

    Add network resources

commit e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4
Date:   2026-02-14 10:00:00 -0700
Source: b0a9e8f7d6c5b4a3c2d1e0f9a8b7c6d5e4f3a2b1

    Initialize graph "default"
```

#### One-line (`--oneline`)

```text
<8-char-hash> <first line of commit message>
```

**Example**:

```text
a1b2c3d4 Add network resources
e5f6a7b8 Initialize graph "default"
```

#### JSON (`--json`)

JSON array of commit objects written to stdout:

```json
[
  {
    "hash": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
    "date": "2026-02-14T10:30:00-07:00",
    "sourceCommit": "f9e8d7c6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0",
    "author": "git-infra-graph",
    "message": "Add network resources\n\nSource-Commit: f9e8d7c6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d0\n"
  },
  {
    "hash": "e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4",
    "date": "2026-02-14T10:00:00-07:00",
    "sourceCommit": "b0a9e8f7d6c5b4a3c2d1e0f9a8b7c6d5e4f3a2b1",
    "author": "git-infra-graph",
    "message": "Initialize graph \"default\"\n\nSource-Commit: b0a9e8f7d6c5b4a3c2d1e0f9a8b7c6d5e4f3a2b1\n"
  }
]
```

### Error Output

All errors go to stderr. Exit code 1 on error.

**Warnings** (broken chain) go to stderr. Exit code 0 (partial success).

**Human-readable errors**:

```text
Error: graph 'nonexistent' not found
```

```text
Error: no graphs found
```

```text
Error: multiple graphs exist; specify a graph name
```

```text
Error: max-count must be a non-negative integer
```

**JSON errors**:

```json
{"error": "graph 'nonexistent' not found"}
```

**Warnings (stderr, both modes)**:

```text
Warning: broken commit chain at <hash>: <error detail>
```

### Exit Codes

| Condition | Exit Code |
| --------- | --------- |
| Success (full history displayed) | 0 |
| Partial success (broken chain, warning emitted) | 0 |
| Graph not found | 1 |
| Not a git repo | 1 |
| Invalid flags | 1 |
| No graphs found | 1 |
| Multiple graphs, no name specified | 1 |

### CLI-to-API Mapping

| CLI Command | Go API Call | Notes |
| ----------- | ----------- | ----- |
| `grif log default` | `Log(".", "default", LogOptions{})` | Full log, no limit |
| `grif log default --max-count 5` | `Log(".", "default", LogOptions{MaxCount: 5, HasMaxCount: true})` | Limited to 5 |
| `grif log default --max-count 0` | `Log(".", "default", LogOptions{MaxCount: 0, HasMaxCount: true})` | Empty result |
| `grif log default --json` | `Log(...)` then JSON marshal `result.Entries` | CLI formats output |
| `grif log default --oneline` | `Log(...)` then compact format | CLI formats output |
| `grif log default --json --oneline` | `Log(...)` then JSON marshal | `--json` takes precedence |
| `grif log` (auto-select) | `List(...)` then `Log(...)` | CLI resolves graph name |
