# Copilot Instructions for git-infra-graph

## Build, Test, and Lint

```bash
make build        # Build the grif CLI binary to ./grif
make test         # Run all tests (go test ./...)
make lint         # Run golangci-lint (golangci-lint run ./...)
```

Run a single test:

```bash
go test ./src/graph/ -run TestFunctionName
```

Run a single test with verbose output:

```bash
go test ./src/graph/ -run TestFunctionName -v
```

## Architecture

`grif` is a versioned graph database that stores infrastructure data as native
Git objects. There is no external database — all state lives inside the Git
repository's object store.

### Git Object Mapping

| Concept | Git Object |
| ------- | ---------- |
| Node data (leaf content) | Blob |
| Hierarchy (containment) | Tree |
| Point-in-time snapshot | Commit |
| Graph identity / head pointer | Ref at `refs/infra/<name>` |
| Staging area (uncommitted changes) | Ref at `refs/infra-stage/<name>` |

Changes follow a **stage-then-commit** workflow: `put`/`rm` operations write to
a staging tree ref, and `commit` snapshots the staged tree as a new commit on
the graph ref. Every graph commit includes a `Source-Commit` trailer linking it
to the repository HEAD at the time of the commit.

### Module-First Layering

```text
src/cmd/grif/main.go          ← CLI: argument parsing + output formatting only
src/graph/                     ← Public API: Init, List, Delete, Put, Get, DeleteNode, Commit, Status, Log
src/graph/internal/gitops/     ← Internal: low-level Git object operations via go-git
```

All business logic lives in the `graph` package. The CLI (`main.go`) is a thin
adapter — no business logic is permitted there. The `gitops` package is internal
and handles raw Git object creation, tree manipulation, and ref management using
`go-git` (a pure-Go Git implementation — no runtime `git` binary required).

### Key Types

- `graph.NodeResult`, `graph.NodeContent`, `graph.NodeEntry` — Put/Get return types
- `graph.StatusResult`, `graph.StatusChange` — Status diff types
- `graph.CommitResult`, `graph.LogEntry`, `graph.LogResult` — Commit/Log types
- `graph.GraphInfo` — List return type
- `gitops.OpenRepo`, `gitops.ResolveRootTree`, `gitops.SetTreeEntry` — core internal operations

### Tree Mutation Pattern

`Put` and `DeleteNode` share a three-phase pattern:

1. **Walk** — Traverse the tree top-down, collecting parent entries in a stack
2. **Mutate** — Create/replace/remove the target entry at the leaf level
3. **Rebuild** — Walk the parent stack bottom-up, creating new tree objects at
   each level (Git trees are immutable; changing a child requires new hashes up
   the chain)

## Conventions

### Constitution

The project has a formal constitution at `.specify/memory/constitution.md` that
governs all development. Key rules:

- Module-first: implement in `graph` package before exposing via CLI
- CLI is the only user interface (no web UI, REST API, or GUI)
- Git-native storage only — no external databases or secondary indexes
- Test-first (red-green-refactor) development is mandatory
- Graph layer is untyped; type semantics belong to the application layer
- Commit messages use Conventional Commits (`feat:`, `fix:`, `docs:`, etc.)

### Testing

- Integration tests use **live Git repositories**, not mocks — mocking the Git
  object database is not permitted
- Tests create temp repos under `testdata/` (git-ignored) via `setupTestRepo()`
  in `testhelper_test.go`
- Each test gets a uniquely named subdirectory derived from `t.Name()` to allow
  parallel execution
- Failed test repos are preserved for inspection; passing tests clean up via
  `t.Cleanup`
- Use table-driven tests with `t.Run` subtests

### CLI Output

Every command supports `--json` for machine-readable output. Human-readable
output goes to stdout; errors go to stderr as `Error: <message>` (or
`{"error": "<message>"}` in JSON mode). Exit code 0 for success, 1 for failure.

### Spec-Driven Development

Feature specifications live in `specs/<number>-<feature>/` with structured
artifacts: `spec.md`, `plan.md`, `tasks.md`, `data-model.md`, and
`contracts/go-api.md`. The `.specify/` directory contains templates and the
project constitution.
