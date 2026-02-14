# Tasks: Graph Node Put, Get, and Delete

**Input**: Design documents from `/specs/002-graph-put-get-delete/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/go-api.md, quickstart.md

**Tests**: Included — the constitution mandates test-first development with integration tests against live Git repos in `testdata/`.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Create new source files with type definitions required by all subsequent phases

- [X] T001 Create src/graph/node.go with NodeType constants (BlobNode, TreeNode), NodeResult, NodeContent, NodeEntry, StatusChange, StatusResult, CommitResult types, and stageRefPrefix constant per contracts/go-api.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Path parsing, validation, and core gitops helpers that all user story operations depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 Implement ParseNodePath function in src/graph/node.go (trim leading/trailing slashes, split on `/`, validate non-empty, validate no empty segments, validate graph name via ValidateGraphName, return graphName and segments)
- [X] T003 [P] Create src/graph/node_test.go with table-driven tests for ParseNodePath (empty path, slashes-only, empty segments like `a//b`, leading/trailing slashes, single segment, multi-segment, invalid graph name)
- [X] T004 [P] Add CreateBlob and ReadBlobContent helpers in src/graph/internal/gitops/objects.go (CreateBlob writes a blob object and returns its hash; ReadBlobContent reads blob bytes by hash)
- [X] T005 Add tree manipulation helpers in src/graph/internal/gitops/objects.go (GetTreeByHash retrieves a tree by hash; SetTreeEntry adds or replaces an entry in a tree and returns the new tree hash; RemoveTreeEntry removes an entry from a tree and returns the new tree hash)
- [X] T006 Add staging ref helpers in src/graph/internal/gitops/objects.go (WriteStagingRef creates or updates `refs/infra-stage/<name>` to point to a tree hash; DeleteStagingRef removes a staging ref; ResolveRootTree reads from staging ref if it exists, else from the graph ref committed tree hash)
- [X] T007 Add CreateCommit (generalized with optional parent hashes) and UpdateRef (unconditional upsert) helpers in src/graph/internal/gitops/objects.go, refactoring CreateOrphanCommit to delegate to CreateCommit with empty parents
- [X] T008 [P] Add unit tests for new gitops helpers (CreateBlob, ReadBlobContent, tree manipulation, staging ref operations) in src/graph/internal/gitops/objects_test.go

**Checkpoint**: Foundation ready — all gitops primitives and path utilities are in place for user story implementation

---

## Phase 3: User Story 1 — Put a Blob Node (Priority: P1) MVP

**Goal**: Store blob data at a `/`-separated path within an initialized graph, auto-creating intermediate tree nodes

**Independent Test**: Initialize a graph, call `Put` with a path and blob content, verify the returned `NodeResult` contains correct path, type (`BlobNode`), and a valid hash ID

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T009 [US1] Write integration tests for Put blob operations in src/graph/graph_test.go: put a blob at a single-segment node path, put a blob at a multi-segment path (verify auto-created intermediate trees), replace an existing blob (verify new hash), verify NodeResult fields (ID, Path, Type), verify staging ref is created, put with empty blob content `[]byte{}`

### Implementation for User Story 1

- [X] T010 [US1] Implement walkAndRebuild helper in src/graph/graph.go (top-down walk collecting parent tree stack, mutation at target, bottom-up tree rebuild) and the Put function handling blob creation/replacement with auto-parent-tree creation and staging ref write
- [X] T011 [US1] Add put command to CLI in src/cmd/grif/main.go (--data flag for inline content, --file flag for file content, piped stdin detection, mutually exclusive input validation, --json flag, human-readable output format `Put blob at "<path>" (id: <short-hash>)`)

**Checkpoint**: User Story 1 is fully functional — blobs can be stored at any valid path and verified via return value

---

## Phase 4: User Story 4 — Put a Tree Node Explicitly (Priority: P2)

**Goal**: Create an empty tree node at a path by calling `Put` with nil blob, enabling organizational structure before populating with blobs

**Independent Test**: Call `Put` with nil blob, then call `Get` on that path and verify a `TreeNode` with an empty children list is returned (Get available after Phase 5; alternatively verify NodeResult type is `TreeNode`)

### Tests for User Story 4

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T012 [US4] Write integration tests for Put tree (nil blob) in src/graph/graph_test.go: create a tree with nil blob, no-op on existing tree (verify children preserved), tree-to-blob error (put non-nil blob on existing tree), blob-to-tree error (put nil blob on existing blob), put blob under explicitly created tree

### Implementation for User Story 4

- [X] T013 [US4] Extend Put function in src/graph/graph.go to handle nil blob: create tree node when blob is nil, preserve existing trees and children on no-op, return error for tree-to-blob conversion (FR-006), return error for blob-to-tree conversion (FR-007)

**Checkpoint**: Put handles both blob and tree creation — the full write API is complete

---

## Phase 5: User Story 2 — Get a Node at a Path (Priority: P2)

**Goal**: Read blob content or list tree children at a specified path, reading from staged (uncommitted) state

**Independent Test**: Put one or more nodes, call `Get` on a blob path (verify content), call `Get` on a tree path (verify children listing)

### Tests for User Story 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T014 [US2] Write integration tests for Get in src/graph/graph_test.go: get blob content, get tree children listing (verify sorted by name), node not found error, blob traversal error, verify Get reads staged (uncommitted) changes, verify NodeContent fields (ID, Path, Type, Blob, Children)

### Implementation for User Story 2

- [X] T015 [US2] Implement Get function in src/graph/graph.go (parse path, resolve root tree from staging ref or committed tree, walk to target, return NodeContent with blob bytes or children listing)
- [X] T016 [US2] Add get command to CLI in src/cmd/grif/main.go (raw blob content to stdout in human mode, tabular tree children listing with TYPE/NAME/ID columns, --json flag with `content` field for blobs and `children` array for trees)

**Checkpoint**: Users can store data and read it back — the read path is complete

---

## Phase 6: User Story 3 — Delete a Node at a Path (Priority: P3)

**Goal**: Remove a blob or recursively remove a tree and all its descendants from the graph

**Independent Test**: Put nodes into a graph, call `DeleteNode`, then verify with `Get` that the nodes no longer exist

### Tests for User Story 3

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T017 [US3] Write integration tests for DeleteNode in src/graph/graph_test.go: delete a blob, recursive tree delete (verify all descendants removed), node not found error, blob traversal error, parent tree preserved after child deletion, empty parent not auto-pruned, verify delete blob then put tree at same path (and vice versa) for type change workflow

### Implementation for User Story 3

- [X] T018 [US3] Implement DeleteNode function in src/graph/graph.go (parse path, walk to target using walkAndRebuild with remove mutation, update staging ref, return error for not-found and blob traversal)
- [X] T019 [US3] Add rm command to CLI in src/cmd/grif/main.go (--json flag, human output `Removed "<path>"` with descendant count for trees, JSON output with `path`, `removed`, `type`, `descendants` fields). **Note**: The CLI must call Get before DeleteNode to determine node type and count descendants, since DeleteNode returns only an error.

**Checkpoint**: Full CRUD lifecycle is operational — nodes can be created, read, and deleted

---

## Phase 7: Commit and Status

**Purpose**: Persist staged changes and report uncommitted state to complete the workflow

### Tests

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T020 Write integration tests for Commit in src/graph/graph_test.go: commit staged changes (verify new commit on graph ref, staging ref deleted, CommitResult fields), commit with custom message (verify message body), commit with default message (verify Source-Commit trailer), no staged changes error, commit with parent (verify linear history)
- [X] T021 [P] Write integration tests for Status in src/graph/graph_test.go: status with additions, status with modifications, status with deletions, status with mixed changes, empty status (no changes returns empty slice not error), status with no staging ref

### Implementation

- [X] T022 Implement Commit function in src/graph/graph.go (read staging ref tree, resolve HEAD for Source-Commit trailer, create commit with previous graph commit as parent, update graph ref, delete staging ref, return CommitResult)
- [X] T023 Implement Status function in src/graph/graph.go (resolve committed tree and staged tree, use object.DiffTree to compare, map merkletrie.Insert/Delete/Modify to StatusChange entries, return empty Changes slice when no staging ref exists)
- [X] T024 Add commit command to CLI in src/cmd/grif/main.go (--message flag, --json flag, human output `Committed graph "<name>" (commit: <short-hash>)`)
- [X] T025 Add status command to CLI in src/cmd/grif/main.go (--json flag, human output listing changes with `added:/modified:/deleted:` prefixes, `No uncommitted changes` message when empty, exit code 0 for both cases)

**Checkpoint**: Complete workflow operational — put, get, rm, commit, and status all functional

---

## Phase 8: Polish and Cross-Cutting Concerns

**Purpose**: Documentation, validation, and refinements that span multiple user stories

- [X] T026 Update README.md with put, get, rm, commit, and status command documentation including usage examples and flag descriptions
- [X] T027 [P] Update Makefile example and example-report targets to demonstrate the put/get/commit workflow
- [X] T028 [P] Run quickstart.md end-to-end validation (build grif, execute all quickstart scenarios, verify outputs match expected)
- [X] T029 [P] Validate SC-007 performance criterion (operations complete within 2 seconds for graphs containing up to 1,000 nodes) by manual inspection or optional benchmark test

---

## Dependencies and Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup (T001) — BLOCKS all user stories
- **US1 Put Blob (Phase 3)**: Depends on Foundational (Phase 2) completion
- **US4 Put Tree (Phase 4)**: Depends on US1 (Phase 3) — extends the same Put function
- **US2 Get (Phase 5)**: Depends on Foundational (Phase 2) — uses Put for test setup but only needs it functional, not its tests
- **US3 Delete (Phase 6)**: Depends on Foundational (Phase 2) — uses Put and Get for test setup
- **Commit and Status (Phase 7)**: Depends on US1 (Phase 3) — needs Put to stage changes for testing
- **Polish (Phase 8)**: Depends on all prior phases being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational (Phase 2) — no dependencies on other stories
- **US4 (P2)**: Depends on US1 — extends the same Put implementation
- **US2 (P2)**: Can start after Foundational (Phase 2) — needs Put functional for test data setup
- **US3 (P3)**: Can start after Foundational (Phase 2) — needs Put and Get functional for test setup and verification
- **Commit/Status**: Can start after US1 — needs Put functional to create staged changes

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Core function implementation before CLI command
- Story complete before moving to next priority

### Parallel Opportunities

- T003 (path tests) and T004 (blob helpers) can run in parallel (different files)
- T008 (gitops tests) can run in parallel with T003 and with T004-T007 group
- US2 (Phase 5) and Commit/Status (Phase 7) could run in parallel after US1 completes
- T027 and T028 can run in parallel in Polish phase

---

## Parallel Example: User Story 1

```bash
# After Phase 2 (Foundational) is complete:

# Write tests first (T009):
Task: "Write integration tests for Put blob operations in src/graph/graph_test.go"

# Implement Put (T010):
Task: "Implement walkAndRebuild helper and Put function in src/graph/graph.go"

# Add CLI command (T011):
Task: "Add put command to CLI in src/cmd/grif/main.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (types)
2. Complete Phase 2: Foundational (path parsing, gitops helpers)
3. Complete Phase 3: User Story 1 — Put Blob
4. **STOP AND VALIDATE**: Test Put independently via return values
5. Continue to Phase 4 (Put Tree) and Phase 5 (Get) to enable read-back verification

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. US1 (Put Blob) → Can store blobs (MVP)
3. US4 (Put Tree) → Full write API
4. US2 (Get) → Can read back stored data
5. US3 (Delete) → Full CRUD lifecycle
6. Commit + Status → Persistence and change tracking
7. Polish → Documentation and validation

### Key Files Modified

| File | Phases | Changes |
| ---- | ------ | ------- |
| src/graph/node.go | 1, 2 | New: types, ParseNodePath |
| src/graph/node_test.go | 2 | New: path validation tests |
| src/graph/graph.go | 3, 4, 5, 6, 7 | New: Put, Get, DeleteNode, Commit, Status, walkAndRebuild |
| src/graph/graph_test.go | 3, 4, 5, 6, 7 | New: integration tests for all operations |
| src/graph/internal/gitops/objects.go | 2 | New: blob, tree, staging ref, commit helpers |
| src/graph/internal/gitops/objects_test.go | 2 | New: unit tests for gitops helpers |
| src/cmd/grif/main.go | 3, 5, 6, 7 | New: put, get, rm, commit, status commands |
| README.md | 8 | Updated: new command documentation |
| Makefile | 8 | Updated: example targets |

---

## Notes

- [P] tasks = different files, no dependencies on concurrent tasks
- [Story] label maps task to specific user story for traceability
- Each user story phase is independently completable and testable
- Verify tests fail before implementing (red-green-refactor per constitution)
- Commit after each task or logical group
- The walkAndRebuild helper (T010) is shared by Put and DeleteNode — implemented once, reused
- Staging uses `refs/infra-stage/<name>` refs (not index files) per research.md R1 decision
- Status uses `object.DiffTree` per research.md R3 decision
- CreateCommit generalizes CreateOrphanCommit per research.md R4 decision
