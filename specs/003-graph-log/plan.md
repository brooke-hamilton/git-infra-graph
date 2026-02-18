# Implementation Plan: Graph Commit History (`grif log`)

**Branch**: `003-graph-log` | **Date**: 2026-02-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-graph-log/spec.md`

## Summary

Implement a read-only `Log` function in the `graph` package that walks the commit chain for an infrastructure graph, starting from the graph ref's tip commit and following parent hashes back to the orphan root. The CLI exposes `grif log` with `--oneline`, `--max-count N`, and `--json` flags. The implementation uses `go-git`'s `CommitObject` to traverse the parent chain, extracting commit hash, committer timestamp, `Source-Commit` trailer, author, and message from each commit object.

## Technical Context

**Language/Version**: Go 1.26.0 (per `go.mod`)
**Primary Dependencies**: `go-git` (`github.com/go-git/go-git/v5`) — pure-Go Git implementation for reading commit objects and refs
**Storage**: Git object database only (commits, refs); read-only access — no writes
**Testing**: `go test` with table-driven tests; integration tests against live Git repos in `testdata/` (git-ignored)
**Target Platform**: Cross-platform CLI (Linux, macOS, Windows)
**Project Type**: Single Go module
**Performance Goals**: Complete commit history walkable in under 2 seconds for graphs with up to 1,000 commits on a local repository (SC-001)
**Constraints**: No external database; read-only operation (FR-017); no modification of any refs or objects
**Scale/Scope**: Single-user CLI tool; linear commit chains (no merges); up to 1,000 commits per graph

## Constitution Check (Pre-Research)

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
| --- | --------- | ------ | ----- |
| I | Module-First Architecture | PASS | Core module exposes `Log` function in `graph` package; CLI is a thin adapter that parses args/flags, calls `Log`, and formats output |
| II | CLI as Sole Interface | PASS | No web UI/REST/GUI; stdout for output, stderr for errors; `--json` flag for machine-readable output; exit code 0/1 |
| III | Git-Native Storage | PASS | Reads commit objects and refs from Git object store only; no external storage; `Source-Commit` trailer extracted from commit messages |
| IV | Test-First Development | PASS | Integration tests with live Git repos in `testdata/`; unique names for parallel runs; `t.Cleanup` for teardown |
| V | Graph-Layer / App-Layer Separation | PASS | Module returns untyped commit metadata (hash, date, message, author, source commit); no typed resources or domain-specific interpretation |

**Gate result**: PASS — no violations. Proceed to Phase 0.

## Constitution Check (Post-Design)

*Re-evaluation after Phase 1 design is complete.*

| # | Principle | Status | Notes |
| --- | --------- | ------ | ----- |
| I | Module-First Architecture | PASS | `graph` package exposes `Log` function plus `LogOptions` and `LogEntry` types. CLI in `cmd/grif` is a thin adapter that parses `--oneline`, `--max-count`, `--json` flags and formats output. No business logic in CLI. |
| II | CLI as Sole Interface | PASS | CLI command `log` supports `--json`, `--oneline`, `--max-count` flags. Human and JSON output defined in contracts. Errors to stderr. Exit codes 0/1. README.md to be updated with new command per constitution requirement. |
| III | Git-Native Storage | PASS | Reads commit objects via `repo.CommitObject(hash)` and refs via `repo.Storer.Reference()`. No writes. `Source-Commit` trailer parsed from commit message text. No external storage. |
| IV | Test-First Development | PASS | Integration tests with live Git repos in `testdata/` (git-ignored). Unique subdir per test via `setupTestRepo`. `t.Cleanup` removes on success. Tests run in parallel. Red-green-refactor cycle mandatory. |
| V | Graph-Layer / App-Layer Separation | PASS | `LogEntry` contains untyped commit metadata only (hash, date string, source commit string, author string, message string). No typed resources, schema validation, or reverse indexes. |

**Gate result**: PASS — no violations. Design is constitution-compliant.

## Project Structure

### Documentation (this feature)

```text
specs/003-graph-log/
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
README.md                    # Updated with new command: log
src/
├── cmd/
│   └── grif/
│       └── main.go          # CLI entry point — adds log command with --oneline, --max-count, --json
└── graph/
    ├── graph.go             # Existing + new Log function
    ├── graph_test.go        # Existing tests + new integration tests for Log
    ├── node.go              # Existing types (unchanged)
    ├── node_test.go         # Existing tests (unchanged)
    ├── ref.go               # Existing ref helpers (unchanged)
    ├── ref_test.go          # Existing tests (unchanged)
    ├── testhelper_test.go   # Existing test helper (unchanged)
    └── internal/
        └── gitops/
            ├── objects.go       # Existing (no new functions needed — commit reading already available via go-git)
            └── objects_test.go  # Existing (unchanged)

testdata/                    # Git-ignored scratch directory for temp repos
```

**Structure Decision**: Follows the established single Go module layout from features 001 and 002. The `LogEntry`, `LogOptions`, and `LogResult` types are added to `node.go` (consistent with existing types like `NodeResult`, `CommitResult`, `StatusResult`). The `Log` function is added to `graph.go`. No new `internal/gitops` functions are needed — commit object reading is handled directly via `go-git`'s `repo.CommitObject()` which is already available. No new packages required.

## Complexity Tracking

> No violations — this section is intentionally empty.
