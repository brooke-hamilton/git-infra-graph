# Tasks: Graph Commit History (`grif log`)

**Input**: Design documents from `/specs/003-graph-log/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/go-api.md, quickstart.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Per Constitution Principle IV, test tasks precede or accompany implementation tasks using the red-green-refactor cycle.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Define the new types and function signature required by all user stories

- [ ] T001 [P] Add `LogEntry`, `LogOptions`, and `LogResult` types to src/graph/node.go per contracts/go-api.md
- [ ] T002 [P] Add `log` command case to the CLI switch in src/cmd/grif/main.go (call placeholder `runLog` that prints "not implemented")

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement the core `Log` function that all user stories depend on

**CRITICAL**: No user story output formatting can work until the commit-chain walk is complete

- [ ] T003 Write integration tests for `graph.Log` in src/graph/graph_test.go — table-driven tests covering: single commit (init only), multiple commits in reverse chronological order, graph not found error, negative max-count error, max-count limits results, max-count 0 returns empty, max-count exceeding commit count returns all, missing Source-Commit trailer produces empty string, broken commit chain returns partial result with warning. Tests use live Git repos in testdata/ with unique subdirs and `t.Cleanup`. Tests run in parallel.
- [ ] T004 Implement `Log(repoPath, graphName, opts)` function in src/graph/graph.go — validate inputs (graph name, negative max-count), resolve graph ref, walk commit chain via `repo.CommitObject`, extract `LogEntry` fields (Hash, Date, SourceCommit, Author, Message), respect `MaxCount`/`HasMaxCount`, handle broken chains with `LogResult.Warning`, return `*LogResult`

**Checkpoint**: `graph.Log()` returns correct `*LogResult` for any graph — all integration tests pass. All user story CLI work can now begin.

---

## Phase 3: User Story 1 — View Full Commit History (Priority: P1) MVP

**Goal**: User runs `grif log <graph>` and sees full commit history in reverse chronological order with hash, date, source commit, and indented commit message

### Tests for User Story 1

- [ ] T005 [US1] Write integration tests for default human-readable log output in src/graph/graph_test.go — verify output format matches contract (40-char hash, Date line, Source line, indented first-paragraph message), multiple commits display newest-first, single init commit case, graph not found error message on stderr, exit code 1 on error

### Implementation for User Story 1

- [ ] T006 [US1] Implement `runLog` function in src/cmd/grif/main.go — parse positional graph name argument, call `graph.Log`, format default human-readable output per contract (commit hash, Date line, Source line, indented message), write warnings to stderr, handle errors with `printError` and exit code 1
- [ ] T007 [US1] Add `printLogUsage` help text function in src/cmd/grif/main.go
- [ ] T008 [US1] Update `printUsage` in src/cmd/grif/main.go to include the `log` command in the commands list

**Checkpoint**: `grif log <graph>` displays full commit history in human-readable format

---

## Phase 4: User Story 2 — View Compact One-Line History (Priority: P2)

**Goal**: User runs `grif log --oneline <graph>` and sees each commit on one line with 8-char abbreviated hash and first line of message

### Tests for User Story 2

- [ ] T009 [US2] Write tests for `--oneline` output format — verify each line is `<8-char-hash> <first line of commit message>`, multi-line commit messages show only first line

### Implementation for User Story 2

- [ ] T010 [US2] Add `--oneline` flag support to `runLog` in src/cmd/grif/main.go — when `--oneline` is set and `--json` is not, format each entry as `<hash[:8]> <first line of message>`

**Checkpoint**: `grif log --oneline <graph>` displays compact one-line output

---

## Phase 5: User Story 3 — Limit Number of Displayed Commits (Priority: P2)

**Goal**: User runs `grif log --max-count N <graph>` and sees at most N entries

### Tests for User Story 3

- [ ] T011 [US3] Write tests for `--max-count` flag — verify exact count of entries displayed, max-count 0 shows nothing, max-count exceeding commit count shows all, negative value produces error, max-count combined with `--oneline`

### Implementation for User Story 3

- [ ] T012 [US3] Add `--max-count` flag parsing to `runLog` in src/cmd/grif/main.go — parse integer value, set `LogOptions.MaxCount` and `LogOptions.HasMaxCount`, pass to `graph.Log`; validate parse errors in CLI before calling module
- [ ] T013 [US3] Update `positionalArgs` in src/cmd/grif/main.go to skip `--max-count` and its value argument, and `--oneline` as a no-value flag

**Checkpoint**: `grif log --max-count N` works with default and `--oneline` output modes

---

## Phase 6: User Story 4 — Machine-Readable JSON Output (Priority: P3)

**Goal**: User runs `grif log --json <graph>` and receives a JSON array of commit objects with hash, date (ISO 8601), sourceCommit, author, and message fields

### Tests for User Story 4

- [ ] T014 [US4] Write tests for `--json` output mode — verify output is valid JSON array, each object contains all expected fields (`hash`, `date`, `sourceCommit`, `author`, `message`), `--json` takes precedence over `--oneline` (FR-009), `--json --max-count N` limits the array, date uses ISO 8601 format

### Implementation for User Story 4

- [ ] T015 [US4] Add `--json` output mode to `runLog` in src/cmd/grif/main.go — when `--json` is set, JSON-marshal `result.Entries` as an array to stdout; `--json` takes precedence over `--oneline` (FR-009); JSON errors use `printError` format on stderr

**Checkpoint**: `grif log --json` produces valid JSON; `--json --max-count N` limits the array; `--json --oneline` outputs full JSON

---

## Phase 7: User Story 5 — Default Graph Selection (Priority: P3)

**Goal**: User runs `grif log` (no graph argument) and the sole graph is auto-selected; if multiple graphs exist, an error lists available graphs

### Tests for User Story 5

- [ ] T016 [US5] Write tests for default graph resolution — single graph auto-selected, multiple graphs produces error listing available graphs, no graphs produces error

### Implementation for User Story 5

- [ ] T017 [US5] Add default graph resolution to `runLog` in src/cmd/grif/main.go — when no positional argument provided, call `graph.List`, auto-select if exactly one, error if zero or multiple (matching existing `runCommit`/`runStatus` pattern)

**Checkpoint**: `grif log` auto-selects sole graph; errors correctly for zero/multiple graphs

---

## Phase 8: Polish and Cross-Cutting Concerns

**Purpose**: Documentation, validation, and performance verification

- [ ] T018 [P] Update README.md to document the `log` command, its flags (`--oneline`, `--max-count`, `--json`), and usage examples
- [ ] T019 [P] Update docs/git-infra-graph.md with `log` command documentation if applicable
- [ ] T020 Run quickstart.md scenarios manually to validate end-to-end behavior
- [ ] T021 Validate SC-001 performance: create a graph with 1,000 commits, run `grif log`, verify completion under 2 seconds (manual benchmark or scripted timing test)

---

## Dependencies and Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (types must exist before `Log` function)
- **User Stories (Phases 3–7)**: All depend on Phase 2 (`graph.Log` must be implemented)
  - US1 (Phase 3) should be completed first as it establishes `runLog` scaffolding
  - US2 (Phase 4), US3 (Phase 5), US4 (Phase 6), US5 (Phase 7) all extend `runLog` — implement sequentially after US1
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends on Phase 2 only — creates `runLog` function
- **US2 (P2)**: Depends on US1 — adds `--oneline` branch to existing `runLog`
- **US3 (P2)**: Depends on US1 — adds `--max-count` flag parsing to `runLog`
- **US4 (P3)**: Depends on US1 — adds `--json` output branch to `runLog`
- **US5 (P3)**: Depends on US1 — adds default graph selection to `runLog`

### Within Each User Story

- Test tasks come first (red phase), then implementation tasks (green phase)
- CLI formatting depends on `graph.Log` returning correct data
- Each story adds a flag or behavior to the shared `runLog` function

### Parallel Opportunities

- T001 and T002 (Phase 1) can run in parallel — different files
- T018 and T019 (Phase 8) can run in parallel — different files
- US2 through US5 modify the same function (`runLog`) so should be sequential

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T002)
2. Complete Phase 2: Tests then implementation (T003–T004)
3. Complete Phase 3: Tests then implementation (T005–T008)
4. **STOP and VALIDATE**: `grif log <graph>` shows full commit history, all tests pass
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → `graph.Log()` works, integration tests pass
2. Add US1 → Full human-readable log → **MVP!**
3. Add US2 → `--oneline` support (tests first)
4. Add US3 → `--max-count` support (tests first)
5. Add US4 → `--json` support (tests first)
6. Add US5 → Default graph auto-selection (tests first)
7. Polish → Documentation updates + quickstart validation + SC-001 benchmark

---

## Notes

- The `Log` function reuses the existing `extractSourceCommit` helper in src/graph/graph.go
- No new `internal/gitops` functions are needed — `repo.CommitObject()` is used directly via go-git
- The `runLog` CLI function follows the established pattern from `runCommit` and `runStatus`
- `positionalArgs()` must be updated to skip `--max-count` and `--oneline` flags
- Date formatting: human-readable uses `2006-01-02 15:04:05 -0700`; JSON uses `time.RFC3339`
- All user stories modify src/cmd/grif/main.go — sequential implementation recommended
- Per Constitution Principle IV, all test tasks use live Git repos in testdata/, unique subdirs, `t.Cleanup`, and parallel execution
