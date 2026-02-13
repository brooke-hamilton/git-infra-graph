<!--
Sync Impact Report
===================
Version change: 1.1.0 → 1.2.0
Modified sections:
  - Development Workflow → added README.md requirements
Added sections: None (added subsection within Development Workflow)
Removed sections: None
Templates requiring updates:
  - .specify/templates/plan-template.md ✅ no changes needed
  - .specify/templates/spec-template.md ✅ no changes needed
  - .specify/templates/tasks-template.md ✅ no changes needed
Follow-up TODOs: None
-->

# grif Constitution

## Core Principles

### I. Module-First Architecture

- Every feature MUST be implemented in the core Go module before
  being exposed through the CLI.
- The core module MUST be independently importable, compilable,
  and testable without the CLI package.
- The module MUST have a clear public API surface; internal
  implementation details MUST use Go `internal` packages or
  unexported identifiers.
- No business logic is permitted in the CLI layer; the CLI is a
  thin adapter that parses arguments, calls the module API, and
  formats output.

### II. CLI as Sole Interface

- The CLI is the only user-facing interface for this project.
  No web UI, REST API server, or GUI is permitted.
- Input MUST come from command-line arguments and stdin.
- Normal output MUST go to stdout; errors and diagnostics MUST
  go to stderr.
- The CLI MUST support both human-readable and JSON output
  formats for every command that produces output.
- Exit codes MUST follow POSIX conventions: 0 for success,
  non-zero for failure.

### III. Git-Native Storage

- The infrastructure graph MUST be stored exclusively using Git
  object primitives: blobs for node payloads, trees for
  containment hierarchy, commits for graph snapshots, and custom
  refs (e.g., `refs/infra`) for head tracking.
- No external database, file-based store, or secondary index
  outside of Git objects is permitted at the graph layer.
- All graph operations MUST preserve Git's content-addressed
  integrity and structural sharing properties.
- Graph commits MUST reference a source commit from the standard
  Git refs namespace via a commit trailer, enabling co-versioning.

### IV. Test-First Development

- Tests MUST be written before implementation code. The
  red-green-refactor cycle is mandatory.
- Unit tests MUST cover the core module's public API surface.
- Integration tests MUST verify correct creation and reading of
  Git objects (blobs, trees, commits, refs).
- Integration tests are REQUIRED for every major feature.
- End-to-end tests MUST exercise CLI commands against a real
  (temporary) Git repository.
- All tests MUST pass before a pull request can be merged.

#### Integration Test Infrastructure

- Integration tests MUST use live Git repository instances —
  mocking the Git object database is not permitted for
  integration-level coverage.
- The repository MUST contain a dedicated integration-test
  scratch directory (e.g., `testdata/`) that is
  listed in the project's `.gitignore` so that temporary
  repositories are never committed.
- Each test MUST create a uniquely named Git repository inside
  a subdirectory of the scratch directory. Unique names MUST
  be derived deterministically (e.g., from the test name) or
  via a random/UUID suffix to guarantee no collisions.
- This naming convention MUST allow tests to run in parallel
  (`go test -parallel`) without interference between
  concurrent repository instances.
- Tests MUST clean up their temporary repositories after
  completion (use `t.Cleanup` or equivalent). Leftover
  directories in the scratch folder MUST NOT cause subsequent
  test runs to fail.

### V. Graph-Layer and Application-Layer Separation

- "Application layer" means any external program that imports
  the grif Go module and uses it to store graph data in
  Git. The CLI included in this project is NOT an application
  layer; it is a thin interface to the graph primitives exposed
  by the module.
- The graph data layer MUST remain untyped. No type semantics
  (resource type, API version, edge classification) are stored
  at the graph level.
- Type information, schema validation, and semantic
  interpretation are the responsibility of the application layer
  (i.e., the external consumer), not of this module.
- Reverse indexes (child-to-parent lookups) and reference-edge
  indexes are the responsibility of the application layer; the
  graph layer MUST NOT maintain them.
- This module MUST NOT include application-layer concerns such
  as typed resource models, schema registries, or domain-specific
  validation. Its public API surface MUST expose only untyped
  graph primitives.

## Technology Stack

- **Language**: Go (minimum version determined by `go.mod`).
- **Storage**: Git object database only — no external
  dependencies for persistence.
- **Repository layout**: One Go module containing the core
  library packages; one `cmd/` directory containing the CLI
  entry point.
- **Build**: Standard `go build` / `go install` toolchain.
- **Testing**: `go test` with the standard `testing` package.
  Table-driven tests are preferred. Integration tests use live
  Git repositories created under `testdata/`
  (git-ignored).
- **Linting**: `golangci-lint` with project-level configuration.
- **License**: MIT.

## Development Workflow

- All changes MUST be submitted via pull request against `main`.
- Every pull request MUST include or update tests that cover the
  changed behavior.
- CI MUST run `go vet`, `golangci-lint`, and `go test ./...`
  and all checks MUST pass before merge.
- Commit messages MUST follow Conventional Commits format
  (e.g., `feat:`, `fix:`, `docs:`, `test:`, `refactor:`).
- Each pull request MUST be reviewed for constitution compliance
  as part of the review checklist.

### README.md

- A comprehensive `README.md` MUST be maintained at the
  repository root.
- The README MUST include installation instructions using
  `go install` syntax (e.g.,
  `go install github.com/brooke-hamilton/rad-graph/cmd/grif@latest`)
  so that users can compile and install the tool in one step.
- The README MUST list every available CLI command with a brief
  description of what each command does.
- The README MUST include runnable usage examples (command +
  expected output) for every command.
- When a command is added, removed, or its interface changes,
  the README MUST be updated in the same pull request.

## Governance

- This constitution supersedes all other project practices and
  conventions. In case of conflict, the constitution governs.
- Amendments MUST be proposed via pull request, include a
  rationale, and update the version number below according to
  semantic versioning:
  - **MAJOR**: Principle removal, redefinition, or backward-
    incompatible governance change.
  - **MINOR**: New principle or section added, or material
    expansion of existing guidance.
  - **PATCH**: Clarifications, wording fixes, non-semantic
    refinements.
- All pull requests and code reviews MUST verify compliance with
  constitutional principles. Deviations MUST be justified in the
  PR description and approved explicitly.

**Version**: 1.2.0 | **Ratified**: 2026-02-13 | **Last Amended**: 2026-02-13
