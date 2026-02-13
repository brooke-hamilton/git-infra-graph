# Tasks: Graph Init, List, and Delete

**Input**: Design documents from `/specs/001-graph-init-list/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/go-api.md, quickstart.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize the Go module, install dependencies, and create the project directory structure.

- [ ] T001 Initialize Go module (go.mod) and add go-git dependency (`github.com/go-git/go-git/v5`) in go.mod
- [ ] T002 Create project directory structure per plan.md: `cmd/grif/`, `graph/`, `graph/internal/gitops/`, `testdata/integration/`
- [ ] T003 [P] Add .gitignore entries for `testdata/integration/` and the `grif` binary

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core building blocks that MUST be complete before ANY user story can be implemented. Includes ref-name validation, low-level Git object helpers, and test infrastructure.

**CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T004 [P] Implement `GraphInfo` type and `ValidateGraphName` function in graph/ref.go per data-model.md validation rules and research.md R2 implementation sketch
- [ ] T005 [P] Implement low-level Git object helpers (open repo, create empty tree, create orphan commit with trailer, create ref, list refs by prefix, delete ref, resolve HEAD) in graph/internal/gitops/objects.go using go-git plumbing API
- [ ] T006 [P] Write table-driven unit tests for `ValidateGraphName` covering all validation rules (empty, `@`, leading/trailing dot, `.lock`, `..`, `@{`, control chars, forbidden chars, slash, valid names) in graph/ref_test.go
- [ ] T007 Write unit tests for gitops helpers (empty tree creation, orphan commit, ref CRUD) in graph/internal/gitops/objects_test.go
- [ ] T008 Implement shared integration test helper (`setupTestRepo` with unique subdirs, cleanup-on-success, failure path logging) in graph/testhelper_test.go per research.md Q4 pattern

**Checkpoint**: Foundation ready — user story implementation can now begin.

---

## Phase 3: User Story 1 — Initialize a New Graph (Priority: P1) MVP

**Goal**: A user can run `init` with a graph name to create an empty root tree, an orphan commit with a `Source-Commit` trailer referencing the current HEAD, and a custom ref at `refs/infra/<name>`.

**Independent Test**: Run `init` with a graph name in a Git repository and verify the custom ref exists, points to a valid orphan commit containing an empty root tree and the expected `Source-Commit` trailer.

### Tests for User Story 1

> Write these tests FIRST, ensure they FAIL before implementation.

- [ ] T009 [US1] Write Init integration tests in graph/graph_test.go: successful init (ref created, commit is orphan, tree is empty, trailer matches HEAD), duplicate name error, invalid name error, not-a-repo error, empty-repo (no commits) error

### Implementation for User Story 1

- [ ] T010 [US1] Implement `Init(repoPath string, name string) error` in graph/graph.go: validate name, open repo, resolve HEAD, check ref does not exist, create empty tree, create orphan commit with `Source-Commit` trailer, create ref at `refs/infra/<name>`
- [ ] T011 [US1] Implement CLI scaffolding (subcommand routing, `--json` flag) and `init` subcommand in cmd/grif/main.go: parse args, call `graph.Init`, print human/JSON success output to stdout, print errors to stderr, exit code 0/1

**Checkpoint**: User Story 1 is fully functional and independently testable. `go test ./...` passes. `./grif init my-infra` works end-to-end.

---

## Phase 4: User Story 2 — List Existing Graphs (Priority: P2)

**Goal**: A user can run `list` to see all infrastructure graphs in the repository, sorted alphabetically, in human-readable or JSON format.

**Independent Test**: Create one or more graphs with `init`, run `list`, and verify the output contains exactly the expected graph names in alphabetical order.

### Tests for User Story 2

> Write these tests FIRST, ensure they FAIL before implementation.

- [ ] T012 [US2] Write List integration tests in graph/graph_test.go: list with multiple graphs (alphabetical order), list with no graphs (empty result, no error), not-a-repo error

### Implementation for User Story 2

- [ ] T013 [US2] Implement `List(repoPath string) ([]GraphInfo, error)` in graph/graph.go: open repo, iterate refs with `refs/infra/` prefix, extract name component, sort alphabetically, return `[]GraphInfo`
- [ ] T014 [US2] Add `list` subcommand to CLI in cmd/grif/main.go: call `graph.List`, print one name per line (human) or JSON array (JSON mode), empty output for no graphs

**Checkpoint**: User Stories 1 AND 2 both work independently. `./grif init` then `./grif list` works end-to-end.

---

## Phase 5: User Story 3 — Delete an Existing Graph (Priority: P3)

**Goal**: A user can run `delete` with a graph name to remove the ref at `refs/infra/<name>`, leaving unreferenced objects for Git garbage collection.

**Independent Test**: Create a graph with `init`, verify it appears in `list`, run `delete`, confirm the graph no longer appears in `list` and the ref no longer exists.

### Tests for User Story 3

> Write these tests FIRST, ensure they FAIL before implementation.

- [ ] T015 [US3] Write Delete integration tests in graph/graph_test.go: successful delete (ref removed, no longer in list), delete non-existent graph error, invalid name error, not-a-repo error, no standard refs modified after delete

### Implementation for User Story 3

- [ ] T016 [US3] Implement `Delete(repoPath string, name string) error` in graph/graph.go: validate name, open repo, verify ref exists, remove ref at `refs/infra/<name>`, do not delete Git objects
- [ ] T017 [US3] Add `delete` subcommand to CLI in cmd/grif/main.go: parse args, call `graph.Delete`, print human/JSON success output to stdout, print errors to stderr, exit code 0/1

**Checkpoint**: All three user stories are independently functional. Full lifecycle (init → list → delete → list) works end-to-end.

---

## Phase 6: Polish and Cross-Cutting Concerns

**Purpose**: Documentation, edge case coverage, and end-to-end validation.

- [ ] T018 [P] Create README.md at repository root with installation instructions (`go install`), command reference for init/list/delete, `--json` flag usage, and usage examples per quickstart.md
- [ ] T019 [P] Add edge case integration tests in graph/graph_test.go: concurrent init with same name, concurrent delete of same name, ref created outside tool still appears in list, standard-namespace refs unmodified after all operations
- [ ] T020 Run quickstart.md end-to-end validation: build binary, execute all quickstart scenarios (init, list, delete with human and JSON output, error cases), verify outputs match documented examples

---

## Dependencies and Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Stories (Phases 3–5)**: All depend on Foundational phase completion
  - User stories can then proceed in priority order (P1 → P2 → P3)
  - US2 and US3 are independent of each other but both depend on US1 being complete (US2 and US3 tests use `Init` for setup)
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) — no dependencies on other stories
- **User Story 2 (P2)**: Tests call `Init` for setup, so US1 implementation (T010) must be complete
- **User Story 3 (P3)**: Tests call `Init` and `List` for setup/verification, so US1 (T010) and US2 (T013) must be complete

### Within Each User Story

- Integration tests MUST be written and FAIL before implementation
- Implementation in graph/graph.go before CLI in cmd/grif/main.go
- Story is complete when `go test ./...` passes and CLI subcommand works

### Parallel Opportunities

- T004, T005, T006 can all run in parallel (different packages/files)
- T018 and T019 can run in parallel (README.md vs test file)
- Within Phase 2, ref.go and gitops/objects.go are independent packages

---

## Parallel Example: Foundational Phase

```text
# These three tasks can run simultaneously (different files/packages):
Task T004: "Implement GraphInfo type and ValidateGraphName in graph/ref.go"
Task T005: "Implement low-level Git object helpers in graph/internal/gitops/objects.go"
Task T006: "Write table-driven unit tests for ValidateGraphName in graph/ref_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1 (Init)
4. **STOP and VALIDATE**: `go test ./...` passes, `./grif init my-infra` works
5. Deploy/demo if ready — a user can create graphs

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 (Init) → Test independently → MVP
3. Add User Story 2 (List) → Test independently → Users can discover graphs
4. Add User Story 3 (Delete) → Test independently → Full lifecycle complete
5. Polish → README, edge cases, quickstart validation
6. Each story adds value without breaking previous stories
