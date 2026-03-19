# Tasks: Recursive Tree Listing (`grif tree`)

<!-- markdownlint-disable MD013 -->

**Input**: Design documents from `/specs/004-graph-tree/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/go-api.md, quickstart.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Per Constitution Principle IV, test tasks precede or accompany implementation tasks using the red-green-refactor cycle.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Define the new types and function signatures required by all user stories

- [x] T001 [P] Add `TreeItem`, `TreeOptions`, `TreeResult` (type alias for `TreeItem`), and `TreeAllResult` types to src/graph/node.go per contracts/go-api.md — `TreeItem` has `Name`, `Type`, `ID`, `Children` fields; `TreeOptions` has `Depth` and `HasDepth` fields; `TreeAllResult` has `Graphs` and `Warnings` fields
- [x] T002 [P] Add `tree` command case to the CLI switch in src/cmd/grif/main.go (call placeholder `runTree` that prints "not implemented") and update `printUsage` to include the `tree` command in the commands list

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement the core `Tree` and `TreeAll` functions that all user stories depend on

**CRITICAL**: No user story output formatting can work until the tree traversal functions are complete

- [x] T003 Write integration tests for `graph.Tree` in src/graph/graph_test.go — table-driven tests covering: full graph tree with multiple levels, empty graph (no nodes), graph not found error, subtree path resolving to tree node, subtree path resolving to blob node, path not found error, negative depth error, depth 0 returns root only, depth 1 returns root and immediate children, depth exceeding actual depth returns full tree, single-segment path (graph name only) returns full tree, blob at root of path displays correctly, staged tree preferred over committed tree (FR-004), deeply nested tree (500+ levels) completes without panic, not-a-Git-repo returns descriptive error (FR-015). Tests use live Git repos in testdata/ with unique subdirs derived from `t.Name()` and `t.Cleanup`. Tests run in parallel.
- [x] T004 Write integration tests for `graph.TreeAll` in src/graph/graph_test.go — table-driven tests covering: multiple graphs returned in alphabetical order, single graph, no graphs found error, negative depth error, depth limiting applied uniformly. Tests use live Git repos in testdata/ with unique subdirs and `t.Cleanup`. Tests run in parallel.
- [x] T005 Implement `Tree(repoPath, path, opts)` function in src/graph/graph.go — validate depth (negative → error), parse path (graph name + optional segments), validate graph name via `ValidateGraphName`, resolve root tree via `gitops.ResolveRootTree`, walk to target subtree/blob if path has segments, build recursive `TreeItem` children respecting depth limit, sort children alphabetically at each level, return `*TreeResult`
- [x] T006 Implement `TreeAll(repoPath, opts)` function in src/graph/graph.go — validate depth (negative → error), list all graphs via `gitops.ListRefsByPrefix`, error if no graphs found, sort alphabetically, call `Tree` for each graph, collect warnings for failed graphs (partial success per FR-019), return `*TreeAllResult`

**Checkpoint**: `graph.Tree()` and `graph.TreeAll()` return correct results for any graph — all integration tests pass. CLI work can now begin.

---

## Phase 3: User Story 1 — View Full Tree of a Named Graph (Priority: P1) MVP

**Goal**: User runs `grif tree <graph>` and sees the complete recursive hierarchy using box-drawing characters

**Independent Test**: Initialize a graph, put several blob nodes at various depths, run `grif tree <graph>`, verify box-drawing output matches expected format

### Tests for User Story 1

- [x] T007 [US1] Write tests verifying box-drawing output format of `graph.FormatTree` in src/graph/graph_test.go — verify `├──` for non-last siblings, `└──` for last siblings, `│` for continuation lines, blob annotation format `(blob, <8-char-hash>)` with two-space separator, tree nodes show name only, entries sorted alphabetically at each level, empty graph shows only graph name, graph not found produces error

### Implementation for User Story 1

- [x] T008 [US1] Implement `FormatTree` function in src/graph/graph.go and `runTree` handler in src/cmd/grif/main.go — `FormatTree(*TreeResult) string` recursively formats the `TreeItem` hierarchy with correct prefix tracking (`├── `/`└── ` connectors, `│   `/`    ` continuations) and returns the formatted string; `runTree` parses positional argument, calls `graph.Tree`, calls `graph.FormatTree`, writes output to stdout, handles errors with `printError` and exit code 1
- [x] T009 [US1] Add `printTreeUsage` help text function in src/cmd/grif/main.go

**Checkpoint**: `grif tree <graph>` displays full recursive hierarchy with correct box-drawing characters

---

## Phase 4: User Story 2 — View Trees for All Graphs (Priority: P1)

**Goal**: User runs `grif tree` with no arguments and sees tree output for all graphs in alphabetical order, separated by blank lines

**Independent Test**: Create two graphs with nodes, run `grif tree` with no arguments, verify both graphs appear in alphabetical order with correct tree output separated by a blank line

### Tests for User Story 2

- [x] T010 [US2] Write tests for no-argument mode in src/graph/graph_test.go — verify multiple graphs displayed in alphabetical order separated by blank line, single graph displays without trailing blank line, no graphs produces error, partial success emits warnings to stderr and exits 0

### Implementation for User Story 2

- [x] T011 [US2] Add no-argument mode to `runTree` in src/cmd/grif/main.go — when no positional argument provided, call `graph.TreeAll`, call `graph.FormatTreeAll` to render each graph's tree separated by blank lines, emit warnings to stderr for failed graphs per FR-019, exit 0 on partial success

**Checkpoint**: `grif tree` (no argument) displays all graphs alphabetically with correct formatting and partial-success handling

---

## Phase 5: User Story 3 — View Subtree at a Specific Path (Priority: P2)

**Goal**: User runs `grif tree <graph>/<path>` and sees tree listing rooted at the specified subtree, with root label being the last path segment

**Independent Test**: Create a graph with deep hierarchy, run `grif tree <graph>/<subtree-path>`, verify only the specified subtree is displayed with the last segment as root label

### Tests for User Story 3

- [x] T012 [US3] Write tests for subtree mode in src/graph/graph_test.go — verify subtree rooted at intermediate tree node, subtree resolving to blob displays single line with `(blob, hash)` format, path not found produces error, deeply nested subtree renders correctly

### Verification for User Story 3

- [x] T013 [US3] Verify subtree handling in `runTree` in src/cmd/grif/main.go — `graph.Tree` already handles subtree paths via path walking; verify the CLI renderer correctly uses the last path segment as root label (not the full path); add test coverage for blob-at-path single-line display

**Checkpoint**: `grif tree <graph>/<path>` displays correct subtree with last segment as root label

---

## Phase 6: User Story 4 — Limit Recursion Depth (Priority: P2)

**Goal**: User runs `grif tree <graph> --depth N` and sees tree truncated at N levels below root

**Independent Test**: Create a graph with 3+ levels of nesting, run `grif tree --depth 1`, verify only root and immediate children shown

### Tests for User Story 4

- [x] T014 [US4] Write tests for `--depth` flag in src/graph/graph_test.go — verify depth 0 shows root only, depth 1 shows root + immediate children (trees without their children), depth exceeds actual depth shows full tree, negative depth produces error, depth works with no-argument (all-graphs) mode applying uniformly

### Implementation for User Story 4

- [x] T015 [US4] Add `--depth` flag parsing to `runTree` in src/cmd/grif/main.go — parse integer value from `flagValue("--depth")`, set `TreeOptions.Depth` and `TreeOptions.HasDepth`, pass to `graph.Tree` or `graph.TreeAll`; validate parse errors in CLI before calling module
- [x] T016 [US4] Update `positionalArgs` in src/cmd/grif/main.go to skip `--depth` and its value argument so they are not treated as positional arguments

**Checkpoint**: `grif tree --depth N` works with single graph, subtree, and all-graphs modes

---

## Phase 7: User Story 5 — Machine-Readable JSON Output (Priority: P3)

**Goal**: User runs `grif tree --json [<graph>[/<path>]]` and receives JSON representation of the tree hierarchy

**Independent Test**: Create a graph with nodes, run `grif tree --json`, verify output is valid JSON with correct nested structure

### Tests for User Story 5

- [x] T017 [US5] Write tests for `--json` output mode in src/graph/graph_test.go — verify single graph produces JSON object with `name`, `type`, `id`, `children` fields, all-graphs mode produces JSON wrapper object with `graphs` array, blob node omits `children` field, empty tree has empty `children` array, `--json --depth N` limits JSON depth, all-graphs with warnings includes `warnings` array in wrapper object, output is valid parseable JSON

### Implementation for User Story 5

- [x] T018 [US5] Add `--json` output mode to `runTree` in src/cmd/grif/main.go — when `--json` is set: single graph/subtree marshals `TreeResult` directly; all-graphs marshals full `TreeAllResult` object (always wrapper with `graphs` array); JSON errors use `printError` format on stderr

**Checkpoint**: `grif tree --json` produces valid JSON; `--json --depth N` limits the tree; all-graphs JSON includes warnings when present

---

## Phase 8: Polish and Cross-Cutting Concerns

**Purpose**: Documentation, validation, and final verification

- [x] T019 [P] Update README.md to document the `tree` command, its flags (`--depth`, `--json`), and usage examples per Constitution README.md requirements
- [x] T020 Write a benchmark test `BenchmarkTree1000Nodes` in src/graph/graph_test.go — creates a graph with 1,000 blob nodes distributed across multiple tree levels and benchmarks `graph.Tree`; validates SC-001 performance goal (< 1 second on local repository)
- [x] T021 Run quickstart.md scenarios manually to validate end-to-end behavior
- [x] T022 Run `make lint` and `make test` to verify all checks pass

---

## Dependencies and Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (types must exist before `Tree`/`TreeAll` functions)
- **User Stories (Phases 3–7)**: All depend on Phase 2 (`graph.Tree` and `graph.TreeAll` must be implemented)
  - US1 (Phase 3) must be completed first — establishes `runTree` and box-drawing renderer
  - US2 (Phase 4) extends `runTree` with no-argument mode
  - US3 (Phase 5) verifies subtree handling works end-to-end
  - US4 (Phase 6) adds `--depth` flag parsing
  - US5 (Phase 7) adds `--json` output branch
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends on Phase 2 only — creates `runTree` function and box-drawing renderer
- **US2 (P1)**: Depends on US1 — adds no-argument branch to existing `runTree`
- **US3 (P2)**: Depends on US1 — verifies subtree path handling in CLI
- **US4 (P2)**: Depends on US1 — adds `--depth` flag parsing to `runTree`
- **US5 (P3)**: Depends on US1 — adds `--json` output branch to `runTree`

### Within Each User Story

- Test tasks come first (red phase), then implementation tasks (green phase)
- CLI formatting depends on `graph.Tree`/`graph.TreeAll` returning correct data
- Each story adds a flag or behavior to the shared `runTree` function

### Parallel Opportunities

- T001 and T002 (Phase 1) can run in parallel — different files
- T003 and T004 (Phase 2) can run in parallel — both are test tasks for different functions
- T019 (Phase 8) can run in parallel with T020/T021
- US3 through US5 modify the same function (`runTree`) so should be sequential after US2

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (T001–T002)
2. Complete Phase 2: Tests then implementation (T003–T006)
3. Complete Phase 3: US1 tests then implementation (T007–T009)
4. Complete Phase 4: US2 tests then implementation (T010–T011)
5. **STOP and VALIDATE**: `grif tree` and `grif tree <graph>` work with box-drawing output, all tests pass
6. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → `graph.Tree()` and `graph.TreeAll()` work, integration tests pass
2. Add US1 → Box-drawing tree for named graph → **Core MVP!**
3. Add US2 → No-argument all-graphs mode → **Full MVP!**
4. Add US3 → Subtree path support (tests first)
5. Add US4 → `--depth` support (tests first)
6. Add US5 → `--json` support (tests first)
7. Polish → README update + quickstart validation + lint/test

---

## Notes

- The `Tree` function reuses existing `gitops.ResolveRootTree` and `gitops.GetTreeByHash` — no new internal/gitops functions needed
- The box-drawing renderer (`FormatTree`, `FormatTreeAll`) lives in the `graph` package (`graph.go`) as a public API — it is output formatting that is testable independently of the CLI
- `TreeResult = TreeItem` is a type alias, so JSON output naturally produces the correct nested structure
- All-graphs JSON: always a wrapper object `{"graphs":[...],"warnings":[...]}` for consistent schema; `warnings` omitted via `omitempty` when empty
- `positionalArgs()` must be updated to skip `--depth` and its value argument (same pattern as `--max-count` in `runLog`)
- Per Constitution Principle IV, all test tasks use live Git repos in testdata/, unique subdirs derived from `t.Name()`, `t.Cleanup`, and parallel execution
- Entry sorting is case-sensitive alphabetical, consistent with Git's default tree entry ordering
