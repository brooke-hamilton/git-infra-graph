# Implementation Plan: Recursive Tree Listing (`grif tree`)

**Branch**: `004-graph-tree` | **Date**: 2026-03-18 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-graph-tree/spec.md`

## Summary

Implement a `Tree` function in the `graph` package and a `grif tree` CLI command
that recursively displays the full hierarchy of a graph (or subtree) using
box-drawing characters matching Unix `tree` style. Supports optional positional
argument `[<graph>[/<path>]]`, `--depth N` flag for recursion limiting, `--json`
for machine-readable output, and a no-argument mode that displays all graphs.
The implementation reads existing Git tree/blob objects via go-git — no new
internal/gitops functions are needed since `GetTreeByHash` and `ResolveRootTree`
already provide the required tree traversal primitives.

## Technical Context

**Language/Version**: Go (per `go.mod`)
**Primary Dependencies**: go-git (`github.com/go-git/go-git/v5`) — pure-Go Git implementation
**Storage**: Git object database only — trees and blobs read via go-git
**Testing**: `go test` with table-driven tests; integration tests use live temp Git repos under `testdata/`
**Target Platform**: Linux, macOS, Windows (CLI binary)
**Project Type**: Single Go module with `cmd/` CLI and `graph/` library
**Performance Goals**: < 1 second for graphs with up to 1,000 nodes (SC-001)
**Constraints**: Read-only operation — must not modify refs or objects (FR-018)
**Scale/Scope**: Handles arbitrarily deep hierarchies without stack overflow (edge case requirement)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Module-First Architecture | PASS | `Tree` function implemented in `graph` package before CLI exposure |
| II. CLI as Sole Interface | PASS | `grif tree` command; stdout for output, stderr for errors; `--json` support |
| III. Git-Native Storage | PASS | Reads only from Git object database via go-git; no external storage |
| IV. Test-First Development | PASS | Tests written before implementation; integration tests with live repos |
| V. Graph/Application Separation | PASS | No type semantics; reads untyped blobs/trees; no indexes |
| README.md | PASS | README must be updated with `tree` command documentation |
| Conventional Commits | PASS | `feat:` prefix for new command |

No violations detected. Gate passes.

## Project Structure

### Documentation (this feature)

```text
specs/004-graph-tree/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── go-api.md
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
src/
├── cmd/
│   └── grif/
│       └── main.go          # Add "tree" case to command router + runTree handler
└── graph/
    ├── graph.go             # Add Tree(), TreeAll(), FormatTree(), FormatTreeAll() functions
    ├── graph_test.go        # Add Tree() integration tests
    ├── node.go              # Add TreeResult, TreeItem types (reuse NodeType, NodeEntry)
    └── internal/
        └── gitops/
            └── objects.go   # No changes needed — existing primitives sufficient
```

**Structure Decision**: Single project layout matching existing repository
structure. New code goes into existing files (`graph.go`, `node.go`, `main.go`)
with new types added to `node.go` for consistency with prior features.

## Complexity Tracking

No constitution violations — this section is not applicable.
