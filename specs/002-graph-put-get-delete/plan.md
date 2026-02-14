# Implementation Plan: Graph Node Put, Get, and Delete

**Branch**: `002-graph-put-get-delete` | **Date**: 2026-02-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-graph-put-get-delete/spec.md`

## Summary

Implement the path-addressed tree/blob node API with `Put`, `Get`, `Delete`, `Commit`, and `Status` operations for infrastructure graph nodes. `Put` creates or replaces blob or tree nodes at `/`-separated paths, auto-creating intermediate trees. `Get` reads blob content or lists tree children. `Delete` removes nodes (recursively for trees). Changes are staged in a per-graph staging ref (`refs/infra-stage/<name>`) and persisted via explicit `Commit`. `Status` reports uncommitted changes. All operations use stateless functions with `repoPath` and operate on the existing `go-git` backend from feature 001. The CLI exposes `grif put`, `grif get`, `grif rm`, `grif commit`, and `grif status` commands.

## Technical Context

**Language/Version**: Go 1.26.0 (per `go.mod`)
**Primary Dependencies**: `go-git` (`github.com/go-git/go-git/v5`) — pure-Go Git implementation for all object store operations (blobs, trees, commits, refs)
**Storage**: Git object database only (blobs, trees, commits, refs); per-graph staging ref (`refs/infra-stage/<name>`) for staging
**Testing**: `go test` with table-driven tests; integration tests against live Git repos in `testdata/` (git-ignored)
**Target Platform**: Cross-platform CLI (Linux, macOS, Windows)
**Project Type**: Single Go module
**Performance Goals**: All operations complete within 2 seconds for graphs containing up to 1,000 nodes on a local repository (SC-007)
**Constraints**: No external database; no modification of standard-namespace refs; per-graph index isolation; stateless function API (no session object)
**Scale/Scope**: Single-user CLI tool; paths up to 10 levels deep (SC-004); up to 1,000 nodes per graph

## Constitution Check (Pre-Research)

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
| --- | --------- | ------ | ----- |
| I | Module-First Architecture | PASS | Core module exposes `Put`, `Get`, `Delete`, `Commit`, `Status` in `graph` package; CLI is a thin adapter that parses args, calls module API, and formats output |
| II | CLI as Sole Interface | PASS | No web UI/REST/GUI; stdout for output, stderr for errors, `--json` flag for all commands; exit code 0/1 |
| III | Git-Native Storage | PASS | All data in Git objects (blobs, trees, commits, refs); per-graph staging ref for staging; `Source-Commit` trailer on commits |
| IV | Test-First Development | PASS | Integration tests with live Git repos in `testdata/`; unique names for parallel runs; `t.Cleanup` for teardown |
| V | Graph-Layer / App-Layer Separation | PASS | Module exposes untyped graph primitives only (blob content as `[]byte`, tree children as name/type/ID); no typed resources, schema validation, or reverse indexes |

**Gate result**: PASS — no violations. Proceed to Phase 0.

## Constitution Check (Post-Design)

*Re-evaluation after Phase 1 design is complete.*

| # | Principle | Status | Notes |
| --- | --------- | ------ | ----- |
| I | Module-First Architecture | PASS | `graph` package exposes `Put`, `Get`, `DeleteNode`, `Commit`, `Status`, plus types `NodeResult`, `NodeContent`, `NodeEntry`, `NodeType`, `StatusChange`, `StatusResult`, `CommitResult`. CLI in `cmd/grif` is a thin adapter that parses args/flags and formats output. No business logic in CLI. |
| II | CLI as Sole Interface | PASS | CLI commands `put`, `get`, `rm`, `commit`, `status` all support `--json` flag. Human and JSON output defined in contracts. Errors to stderr. Exit codes 0/1. README.md updated with new commands per constitution requirement. |
| III | Git-Native Storage | PASS | All data in Git objects (blobs, trees, commits). Staging via `refs/infra-stage/<name>` (Git refs, not external files). `Source-Commit` trailer on commits. Content-addressed integrity preserved. Structural sharing via tree hashing. |
| IV | Test-First Development | PASS | Integration tests with live Git repos in `testdata/` (git-ignored). Unique subdir per test via `setupTestRepo`. `t.Cleanup` removes on success. Tests run in parallel. Red-green-refactor cycle mandatory. |
| V | Graph-Layer / App-Layer Separation | PASS | All types are untyped graph primitives. `NodeContent` contains raw `[]byte` blob and untyped children listing. No type semantics, schema validation, or reverse indexes. Application layer responsibility is explicitly excluded. |

**Gate result**: PASS — no violations. Design is constitution-compliant.

## Project Structure

### Documentation (this feature)

```text
specs/002-graph-put-get-delete/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── go-api.md        # Go public API and CLI interface contracts
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
go.mod
go.sum
Makefile
README.md                    # Updated with new commands: put, get, rm, commit, status
src/
├── cmd/
│   └── grif/
│       └── main.go          # CLI entry point — adds put, get, rm, commit, status commands
└── graph/
    ├── graph.go             # Existing Init/List/Delete + new Put/Get/Delete/Commit/Status functions
    ├── graph_test.go        # Existing tests + new integration tests for Put/Get/Delete/Commit/Status
    ├── node.go              # New: NodeResult, NodeContent, NodeEntry types; path parsing/validation
    ├── node_test.go         # New: path validation unit tests
    ├── ref.go               # Existing ref helpers and ValidateGraphName
    ├── ref_test.go          # Existing ref validation tests
    ├── testhelper_test.go   # Existing test helper (setupTestRepo)
    └── internal/
        └── gitops/
            ├── objects.go       # Existing + new: blob creation, tree manipulation, index operations
            ├── objects_test.go  # Existing + new gitops tests

testdata/                    # Git-ignored scratch directory for temp repos
```

**Structure Decision**: Follows the established single Go module layout from feature 001. New node types and path utilities go in a new `node.go` file to keep `graph.go` focused. The `internal/gitops` package gains new functions for blob/tree composition and per-graph index management. No new packages needed — the existing structure scales well for this feature.

## Complexity Tracking

> No violations — this section is intentionally empty.
