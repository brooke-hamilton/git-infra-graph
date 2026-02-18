# Data Model: Graph Commit History (`grif log`)

**Feature**: 003-graph-log
**Date**: 2026-02-17

## Entities

### LogEntry

A single commit in the log output, representing one snapshot in the graph's history.

| Field | Type | Description |
| ----- | ---- | ----------- |
| Hash | string | Full 40-character hex SHA-1 hash of the commit object |
| Date | time.Time | Committer timestamp from the Git commit object |
| SourceCommit | string | Value of the `Source-Commit` trailer from the commit message; empty string if missing or malformed |
| Author | string | Author name from the Git commit object's author signature |
| Message | string | Full commit message including trailers |

**Constraints**:

- `Hash` is always a valid 40-character hex SHA-1 hash
- `Date` preserves the original timezone from the commit's committer signature
- `SourceCommit` may be empty if the trailer is missing or malformed (FR-014)
- `Author` is the author name only (not email)
- `Message` is the complete commit message text as stored in the Git object

**JSON serialization**:

| Go Field | JSON Key | Format |
| -------- | -------- | ------ |
| Hash | `hash` | 40-character hex string |
| Date | `date` | ISO 8601 (`time.RFC3339`) |
| SourceCommit | `sourceCommit` | hex string or empty |
| Author | `author` | plain string |
| Message | `message` | full text with newlines |

### LogOptions

Options controlling the behavior of the `Log` function.

| Field | Type | Description |
| ----- | ---- | ----------- |
| MaxCount | int | Maximum number of commits to return; negative values produce an error |
| HasMaxCount | bool | Whether `MaxCount` was explicitly set; when false, all commits are returned |

**Constraints**:

- When `HasMaxCount` is false: `MaxCount` is ignored and all commits are returned
- When `HasMaxCount` is true and `MaxCount` is 0: no commits are returned
- When `HasMaxCount` is true and `MaxCount` is positive: at most `MaxCount` commits are returned
- When `HasMaxCount` is true and `MaxCount` is negative: the function returns an error

### LogResult

The return value from a successful `Log` operation.

| Field | Type | Description |
| ----- | ---- | ----------- |
| Entries | []LogEntry | Commit entries in reverse chronological order (newest first) |
| Warning | string | Non-empty if a broken commit chain was detected (FR-013); empty on full success |

**Constraints**:

- `Entries` is ordered from newest to oldest (tip commit first, orphan root last)
- `Entries` may be partial if a broken chain is detected; the warning explains the break
- `Warning` is empty on full success; non-empty with a descriptive message on broken chain
- When `HasMaxCount` is true and `MaxCount` is 0, `Entries` will be empty

## Git Object Mapping

### How Entities Map to Git Objects

| Entity | Git Object Type | Access Method |
| ------ | --------------- | ------------- |
| LogEntry | Git commit | `repo.CommitObject(hash)` |
| Graph Ref (start of walk) | Git ref | `repo.Storer.Reference("refs/infra/<name>")` |
| Commit chain link | ParentHashes field | `commit.ParentHashes[0]` |

### Commit Object Fields Used

| LogEntry Field | Git Commit Field | Notes |
| -------------- | ---------------- | ----- |
| Hash | `commit.Hash` | `plumbing.Hash.String()` for 40-char hex |
| Date | `commit.Committer.When` | Uses committer timestamp per spec assumptions |
| SourceCommit | `commit.Message` | Parsed via `extractSourceCommit()` |
| Author | `commit.Author.Name` | Author name only, not email |
| Message | `commit.Message` | Full text including trailers |

## Relationships

```text
Graph Ref (refs/infra/<name>)
  └── Tip Commit (LogEntry[0])
        ├── Message → Source-Commit trailer → SourceCommit field
        ├── Author.Name → Author field
        ├── Committer.When → Date field
        └── ParentHashes[0] → Previous Commit (LogEntry[1])
              └── ParentHashes[0] → Previous Commit (LogEntry[2])
                    └── ... → Orphan Root (LogEntry[N], no parents)
```

## State Transitions

This feature is read-only (FR-017). There are no state transitions — the `Log` function does not create, modify, or delete any Git objects, refs, or repository state.

### Walk Lifecycle

```text
(start)
  --[Resolve graph ref]--> Tip commit hash
  --[Read commit object]--> Extract LogEntry
  --[Check parents]-------> If no parents → done (orphan root)
                            If parent exists → follow to next commit
                            If parent missing → broken chain warning, done

(walk terminates when):
  - Orphan root is reached (normal completion)
  - MaxCount limit is reached (early stop)
  - Parent commit object cannot be found (broken chain, partial result)
```

## Validation Rules

### Input Validation

| Rule | Invalid Input | Error |
| ---- | ------------- | ----- |
| Graph name empty | `""` | (from `ValidateGraphName`) |
| Graph name invalid | `"a..b"` | (from `ValidateGraphName`) |
| Negative max-count | `-1` | "max-count must be a non-negative integer" |
| Not a git repo | non-repo path | "not a git repository: ..." |
| Graph not found | non-existent name | "graph '...' not found" |

### Runtime Conditions

| Condition | Behavior |
| --------- | -------- |
| Tip commit object corrupted/missing | Error: descriptive message, non-zero exit |
| Mid-chain parent missing | Partial result with warning string, exit 0 |
| Missing Source-Commit trailer | Empty string in SourceCommit field, continue |
| MaxCount exceeds actual commits | Return all available commits (no error) |
| MaxCount is 0 | Return empty Entries slice |
