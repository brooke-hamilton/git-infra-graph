# Go Module API Contract: Graph Init, List, and Delete

**Feature**: 001-graph-init-list
**Date**: 2026-02-13
**Package**: `graph`

This project is a Go module with a CLI interface — not a REST API. The
"contracts" defined here are the public Go function signatures and types
exposed by the `graph` package, plus the CLI command interface.

## Go Public API

### Types

```go
package graph

// GraphInfo represents a named graph returned by List.
type GraphInfo struct {
    Name string // Graph name (the component after refs/infra/)
}
```

### Functions

#### Init

```go
// Init creates a new named infrastructure graph in the Git repository at the
// given path. It creates an empty root tree, an orphan commit with a
// Source-Commit trailer referencing the current HEAD, and a ref at
// refs/infra/<name>.
//
// Returns an error if:
//   - repoPath is not a valid Git repository
//   - the repository has no commits (HEAD does not resolve)
//   - name fails ref-name validation
//   - a graph with the given name already exists
func Init(repoPath string, name string) error
```

**Preconditions**:

- `repoPath` is a path to a Git repository (with `.git` directory)
- `name` passes `ValidateGraphName()`
- `refs/infra/<name>` does not exist
- HEAD resolves to a valid commit

**Postconditions**:

- An empty tree object exists in the object database
- An orphan commit exists with the empty tree as its tree, and a
  `Source-Commit: <HEAD SHA>` trailer
- `refs/infra/<name>` points to the new commit
- No standard-namespace refs are modified

**Errors**:

| Condition | Error message pattern |
|-----------|----------------------|
| Not a git repo | "not a git repository: ..." |
| No commits | "repository has no commits" |
| Invalid name | (from `ValidateGraphName`) |
| Duplicate graph | "graph '<name>' already exists" |

#### List

```go
// List returns all infrastructure graphs in the Git repository at the given
// path, sorted alphabetically by name. Returns an empty slice (not an error)
// if no graphs exist.
//
// Returns an error if repoPath is not a valid Git repository.
func List(repoPath string) ([]GraphInfo, error)
```

**Preconditions**:

- `repoPath` is a path to a Git repository

**Postconditions**:

- Returns a slice of `GraphInfo` sorted alphabetically by `Name`
- Returns an empty slice if no refs exist under `refs/infra/`
- No state is modified

**Errors**:

| Condition | Error message pattern |
|-----------|----------------------|
| Not a git repo | "not a git repository: ..." |

#### Delete

```go
// Delete removes the infrastructure graph with the given name from the Git
// repository at the given path by deleting refs/infra/<name>. Git objects
// that become unreferenced are left for Git garbage collection.
//
// Returns an error if:
//   - repoPath is not a valid Git repository
//   - name fails ref-name validation
//   - no graph with the given name exists
func Delete(repoPath string, name string) error
```

**Preconditions**:

- `repoPath` is a path to a Git repository
- `name` passes `ValidateGraphName()`
- `refs/infra/<name>` exists

**Postconditions**:

- `refs/infra/<name>` no longer exists
- No other refs are modified
- Git objects are not directly deleted

**Errors**:

| Condition | Error message pattern |
|-----------|----------------------|
| Not a git repo | "not a git repository: ..." |
| Invalid name | (from `ValidateGraphName`) |
| Graph not found | "graph '<name>' not found" |

#### ValidateGraphName

```go
// ValidateGraphName checks that name is a legal single Git ref-name component
// suitable for use in refs/infra/<name>. Returns nil if valid, or an error
// describing the specific validation failure.
func ValidateGraphName(name string) error
```

## CLI Interface

### Command Structure

```text
grif <command> [flags] [args]
```

### Commands

#### `grif init <name> [--json]`

Create a new infrastructure graph.

| Argument/Flag | Required | Description |
|---------------|----------|-------------|
| `<name>` | Yes | Graph name (positional argument) |
| `--json` | No | Output in JSON format |

**Success output (human)**:

```text
Initialized graph "my-infra"
```

**Success output (JSON)**:

```json
{"name": "my-infra", "ref": "refs/infra/my-infra", "sourceCommit": "a1b2c3..."}
```

**Exit codes**: 0 on success, 1 on error.

#### `grif list [--json]`

List all infrastructure graphs.

| Argument/Flag | Required | Description |
|---------------|----------|-------------|
| `--json` | No | Output in JSON format |

**Success output (human)** (one graph per line, alphabetical):

```text
production
staging
```

**Success output (JSON)**:

```json
["production", "staging"]
```

**Empty result (human)**: No output, exit code 0.
**Empty result (JSON)**: `[]`, exit code 0.

**Exit codes**: 0 on success, 1 on error.

#### `grif delete <name> [--json]`

Delete an infrastructure graph.

| Argument/Flag | Required | Description |
|---------------|----------|-------------|
| `<name>` | Yes | Graph name (positional argument) |
| `--json` | No | Output in JSON format |

**Success output (human)**:

```text
Deleted graph "my-infra"
```

**Success output (JSON)**:

```json
{"name": "my-infra", "deleted": true}
```

**Exit codes**: 0 on success, 1 on error.

### Error Output

All errors go to stderr. In JSON mode, errors are formatted as:

```json
{"error": "graph 'my-infra' already exists"}
```

In human-readable mode, errors are plain text:

```text
Error: graph 'my-infra' already exists
```
