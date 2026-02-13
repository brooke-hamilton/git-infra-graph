# Implementation Plan: Graph Init, List, and Delete

**Branch**: `001-graph-init-list` | **Date**: 2026-02-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-graph-init-list/spec.md`

## Summary

Implement the three foundational graph lifecycle commands — `init`, `list`, and `delete` — as a core Go module backed by Git-native storage, exposed through a thin CLI. `init` creates a named graph by writing an empty root tree, an orphan commit with a source-commit trailer, and a custom ref under `refs/infra/<name>`. `list` enumerates all refs under that namespace. `delete` removes a graph's ref, leaving object cleanup to Git GC. All commands support human-readable and JSON output.

## Technical Context

**Language/Version**: Go (version TBD in `go.mod`; target Go 1.22+)
**Primary Dependencies**: `go-git` (`github.com/go-git/go-git/v5`) — pure-Go Git implementation; no runtime `git` binary required (see [research.md](research.md) R1)
**Storage**: Git object database only (blobs, trees, commits, refs)
**Testing**: `go test` with table-driven tests; integration tests against live Git repos in `testdata/integration/` (git-ignored)
**Target Platform**: Cross-platform CLI (Linux, macOS, Windows)
**Project Type**: Single Go module
**Performance Goals**: All commands complete in < 5 seconds on a local repository (per SC-001)
**Constraints**: No external database; no modification of standard-namespace refs; orphan commits only
**Scale/Scope**: Single-user CLI tool operating on local Git repositories; hundreds of graphs per repo is a reasonable upper bound

## Constitution Check (Pre-Research)

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | Module-First Architecture | PASS | Core module exposes `Init`, `List`, `Delete` functions; CLI is a thin adapter |
| II | CLI as Sole Interface | PASS | No web UI/REST/GUI; stdout for output, stderr for errors, JSON flag supported |
| III | Git-Native Storage | PASS | All data in Git objects (blobs, trees, commits, refs); no external store; trailer for source commit |
| IV | Test-First Development | PASS | Integration tests required; live Git repos in `testdata/integration/`; unique names for parallel runs; `t.Cleanup` for teardown |
| V | Graph-Layer / App-Layer Separation | PASS | Module exposes untyped graph primitives only; no typed resources, schema validation, or reverse indexes |

**Gate result**: PASS — no violations. Proceed to Phase 0.

## Constitution Check (Post-Design)

*Re-evaluation after Phase 1 design is complete.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | Module-First Architecture | PASS | `graph` package exposes `Init`, `List`, `Delete`, `ValidateGraphName`; `GraphInfo` type; CLI in `cmd/grif` is a thin adapter |
| II | CLI as Sole Interface | PASS | CLI supports `--json` flag; human and JSON output defined in contracts; errors to stderr; exit codes 0/1; README.md with install instructions, command list, and usage examples |
| III | Git-Native Storage | PASS | go-git for all Git operations; empty tree, orphan commit, custom refs, ref enumeration/deletion; trailer for co-versioning |
| IV | Test-First Development | PASS | Live repos in `testdata/integration/` (git-ignored); unique subdir per test; `t.Cleanup` removes on success; path logged on failure for inspection |
| V | Graph-Layer / App-Layer Separation | PASS | `GraphInfo` contains only `Name`; no type semantics, schema, or reverse indexes; data model is untyped |

**Gate result**: PASS — no violations. Design is constitution-compliant.

## Project Structure

### Documentation (this feature)

```text
specs/001-graph-init-list/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
go.mod
go.sum
README.md                # Installation, command reference, usage examples
cmd/
└── grif/
    └── main.go          # CLI entry point

graph/                   # Core module: public API
├── graph.go             # Init, List, Delete functions
├── graph_test.go        # Unit tests (API surface)
├── ref.go               # Ref-name validation, ref helpers
├── ref_test.go          # Ref validation unit tests
└── internal/            # Unexported implementation details
    └── gitops/
        ├── objects.go   # Low-level Git object creation (tree, commit, ref)
        └── objects_test.go

testdata/
└── integration/         # Git-ignored scratch directory for temp repos
    └── (created at test runtime)
```

**Structure Decision**: Single Go module following the constitution's repository layout (`cmd/` for CLI, library packages at module root). Go convention uses `cmd/<binary>/` for the entry point and top-level packages for the library. `internal/` hides implementation details. Integration test repos live under `testdata/integration/`, which is added to `.gitignore`. A `README.md` at the repo root provides installation instructions (`go install`), a command reference, and usage examples per the constitution's README requirements.

## Complexity Tracking

> No violations — this section is intentionally empty.
