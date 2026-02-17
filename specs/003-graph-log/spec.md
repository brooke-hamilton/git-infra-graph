# Feature Specification: Graph Commit History (`grif log`)

<!-- markdownlint-disable MD013 -->

**Feature Branch**: `003-graph-log`
**Created**: 2026-02-17
**Status**: Draft
**Input**: User description: "grif log — Graph Commit History: Display the commit history for an infrastructure graph. Walk the commit chain starting from the graph ref tip commit, following parent hashes. Display commit hash, message, Source-Commit trailer, author, and timestamp. Support --oneline, --max-count N, and --json flags."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Full Commit History (Priority: P1)

A user has an infrastructure graph with multiple commits and wants to see the complete commit history. They run `grif log <graph>` and see the full commit chain displayed in reverse chronological order (newest first), with each entry showing the commit hash, date, Source-Commit trailer, and commit message. This gives the user a clear audit trail of every change made to their graph.

**Why this priority**: This is the core value of the feature. Without the ability to display the full commit log, no other log-related functionality is useful. It directly exposes the commit lineage that already exists but is invisible to users.

**Independent Test**: Can be fully tested by initializing a graph, making several commits with `grif commit`, running `grif log`, and verifying the output contains all commits in reverse chronological order with correct hashes, dates, source commits, and messages.

**Acceptance Scenarios**:

1. **Given** a graph "default" with 3 commits, **When** the user runs `grif log default`, **Then** all 3 commits are displayed in reverse chronological order (newest first), each showing commit hash, date, Source-Commit trailer value, and the indented commit message.
2. **Given** a graph "default" with a single commit (the init commit), **When** the user runs `grif log default`, **Then** exactly one commit entry is displayed showing the init commit's hash, date, Source-Commit trailer, and message (e.g., `Initialize graph "default"`).
3. **Given** no graph named "nonexistent" exists, **When** the user runs `grif log nonexistent`, **Then** the command fails with a descriptive error message and a non-zero exit code.

**Example — full log with multiple commits**:

```text
$ grif log default
commit a1b2c3d4e5f6a7b8
Date:   2026-02-14 10:30:00 -0700
Source: f9e8d7c6a5b4c3d2

    Add network resources

commit c3d4e5f6a7b8c9d0
Date:   2026-02-14 10:15:00 -0700
Source: d2c1b0a9e8f7d6c5

    Update graph "default"

commit e5f6a7b8c9d0e1f2
Date:   2026-02-14 10:00:00 -0700
Source: b0a9e8f7d6c5b4a3

    Initialize graph "default"
```

**Example — single init commit**:

```text
$ grif log default
commit e5f6a7b8c9d0e1f2
Date:   2026-02-14 10:00:00 -0700
Source: b0a9e8f7d6c5b4a3

    Initialize graph "default"
```

**Example — graph not found (stderr)**:

```text
$ grif log nonexistent
Error: graph 'nonexistent' not found
```

---

### User Story 2 - View Compact One-Line History (Priority: P2)

A user wants a quick overview of the commit history without the full detail. They run `grif log --oneline <graph>` and see each commit summarized on a single line showing the abbreviated commit hash (first 8 characters) and the first line of the commit message. This enables fast scanning of the graph's evolution.

**Why this priority**: One-line output is the most common alternative display format for commit logs. It delivers immediate value for quick scanning and is a natural complement to the full log.

**Independent Test**: Can be fully tested by creating a graph with multiple commits, running `grif log --oneline`, and verifying each line contains an 8-character abbreviated hash followed by the first line of the commit message.

**Acceptance Scenarios**:

1. **Given** a graph "default" with 3 commits, **When** the user runs `grif log --oneline default`, **Then** exactly 3 lines are printed, each in the format `<8-char-hash> <first line of commit message>`.
2. **Given** a graph "default" with a commit whose message has multiple lines, **When** the user runs `grif log --oneline default`, **Then** only the first line of the commit message is shown for that entry.

**Example — oneline output**:

```text
$ grif log --oneline default
a1b2c3d4 Add network resources
c3d4e5f6 Update graph "default"
e5f6a7b8 Initialize graph "default"
```

---

### User Story 3 - Limit Number of Displayed Commits (Priority: P2)

A user has a graph with a long history and only wants to see the most recent N commits. They run `grif log --max-count N <graph>` and see at most N entries. This is useful for large graphs with extensive histories where the user only cares about recent activity.

**Why this priority**: Limiting output is essential for usability on graphs with many commits. It shares P2 priority with oneline because both are display modifiers that enhance the core log but are independently valuable.

**Independent Test**: Can be fully tested by creating a graph with 5+ commits, running `grif log --max-count 2`, and verifying exactly 2 entries are displayed (the 2 most recent).

**Acceptance Scenarios**:

1. **Given** a graph "default" with 5 commits, **When** the user runs `grif log --max-count 2 default`, **Then** exactly 2 commit entries are displayed (the 2 most recent).
2. **Given** a graph "default" with 3 commits, **When** the user runs `grif log --max-count 10 default`, **Then** all 3 commits are displayed (max-count exceeds actual count).
3. **Given** a graph "default" with commits, **When** the user runs `grif log --max-count 0 default`, **Then** no commit entries are displayed.

**Example — limit to 2 most recent commits**:

```text
$ grif log --max-count 2 default
commit a1b2c3d4e5f6a7b8
Date:   2026-02-14 10:30:00 -0700
Source: f9e8d7c6a5b4c3d2

    Add network resources

commit c3d4e5f6a7b8c9d0
Date:   2026-02-14 10:15:00 -0700
Source: d2c1b0a9e8f7d6c5

    Update graph "default"
```

**Example — combined with oneline**:

```text
$ grif log --oneline --max-count 2 default
a1b2c3d4 Add network resources
c3d4e5f6 Update graph "default"
```

---

### User Story 4 - Machine-Readable JSON Output (Priority: P3)

A user or automation script needs machine-readable output of the commit history. They run `grif log --json <graph>` and receive a JSON array of commit objects, each containing the commit hash, date, source commit, author, and message. This enables integration with CI/CD pipelines, dashboards, and other tools.

**Why this priority**: JSON output is a standard pattern across all `grif` commands and enables programmatic consumption, but it is less commonly used interactively than human-readable formats.

**Independent Test**: Can be fully tested by creating a graph with commits, running `grif log --json`, and verifying the output is valid JSON containing an array of commit entries with all expected fields.

**Acceptance Scenarios**:

1. **Given** a graph "default" with 2 commits, **When** the user runs `grif log --json default`, **Then** valid JSON is written to stdout containing an array of 2 commit objects, each with `hash`, `date`, `sourceCommit`, `author`, and `message` fields.
2. **Given** a graph "default" with commits, **When** the user runs `grif log --json --max-count 1 default`, **Then** valid JSON is written containing an array of exactly 1 commit object.

**Example — JSON output**:

```text
$ grif log --json default
[
  {
    "hash": "a1b2c3d4e5f6a7b8",
    "date": "2026-02-14T10:30:00-07:00",
    "sourceCommit": "f9e8d7c6a5b4c3d2",
    "author": "git-infra-graph",
    "message": "Add network resources\n\nSource-Commit: f9e8d7c6a5b4c3d2\n"
  },
  {
    "hash": "e5f6a7b8c9d0e1f2",
    "date": "2026-02-14T10:00:00-07:00",
    "sourceCommit": "b0a9e8f7d6c5b4a3",
    "author": "git-infra-graph",
    "message": "Initialize graph \"default\"\n\nSource-Commit: b0a9e8f7d6c5b4a3\n"
  }
]
```

**Example — JSON with max-count**:

```text
$ grif log --json --max-count 1 default
[
  {
    "hash": "a1b2c3d4e5f6a7b8",
    "date": "2026-02-14T10:30:00-07:00",
    "sourceCommit": "f9e8d7c6a5b4c3d2",
    "author": "git-infra-graph",
    "message": "Add network resources\n\nSource-Commit: f9e8d7c6a5b4c3d2\n"
  }
]
```

---

### User Story 5 - Default Graph Selection (Priority: P3)

A user has only one graph in their repository and wants to view its log without explicitly naming it. They run `grif log` (no graph argument) and the command automatically selects the sole graph. If multiple graphs exist and no name is provided, the command fails with a clear message listing the available graphs.

**Why this priority**: This is a convenience behavior consistent with other `grif` commands (`commit`, `status`). It improves usability but is not required for the feature to function.

**Independent Test**: Can be fully tested by creating a single graph, running `grif log` without a graph argument, and verifying the correct graph's log is displayed.

**Acceptance Scenarios**:

1. **Given** a repository with exactly one graph named "default", **When** the user runs `grif log` (no graph argument), **Then** the log for "default" is displayed.
2. **Given** a repository with graphs "staging" and "production", **When** the user runs `grif log` (no graph argument), **Then** the command fails with an error listing the available graph names and prompting the user to specify one.
3. **Given** a repository with no graphs, **When** the user runs `grif log`, **Then** the command fails with a descriptive error indicating no graphs exist.

**Example — single graph auto-selected**:

```text
$ grif log
commit a1b2c3d4e5f6a7b8
Date:   2026-02-14 10:30:00 -0700
Source: f9e8d7c6a5b4c3d2

    Add network resources

commit e5f6a7b8c9d0e1f2
Date:   2026-02-14 10:00:00 -0700
Source: b0a9e8f7d6c5b4a3

    Initialize graph "default"
```

**Example — multiple graphs, no argument (stderr)**:

```text
$ grif log
Error: multiple graphs exist, specify one: production, staging
```

**Example — no graphs exist (stderr)**:

```text
$ grif log
Error: no graphs found
```

---

### Edge Cases

- What happens when a commit in the chain has a malformed or missing `Source-Commit` trailer? The `Source-Commit` field MUST be shown as empty, and the log MUST continue displaying remaining commits without error.
- What happens when `--max-count` is given a negative value? The command MUST return an error indicating the value must be a non-negative integer.
- What happens when `--oneline` and `--json` are used together? The `--json` flag takes precedence; JSON output is always the full structured format regardless of `--oneline`.
- What happens when the graph ref exists but the commit object it points to is corrupted or missing? The command MUST return a descriptive error rather than a panic or raw Git error.
- What happens when a commit in the middle of the chain is unreachable (broken parent link)? The log MUST display all reachable commits up to the break and then terminate with a warning indicating the chain is broken, rather than failing silently.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `log` command MUST accept an optional graph name argument identifying the target graph (ref `refs/infra/<name>`).
- **FR-002**: When no graph name is provided and exactly one graph exists, the command MUST default to that graph. When no graph name is provided and multiple graphs exist, the command MUST fail with a descriptive error listing available graphs. When no graphs exist, the command MUST fail with a descriptive error.
- **FR-003**: The `log` command MUST walk the commit chain starting from the graph ref's tip commit, following the `ParentHashes` of each commit object.
- **FR-004**: The `log` command MUST display commits in reverse chronological order (newest first), following the commit chain from tip to root.
- **FR-005**: For each commit in the default (human-readable) output, the command MUST display: the full 40-character commit hash, the date (formatted as `YYYY-MM-DD HH:MM:SS ±HHMM`), the `Source-Commit` trailer value, and the commit message (indented with 4 spaces).
- **FR-006**: The `log` command MUST support a `--oneline` flag that displays each commit on a single line in the format `<8-char-hash> <first line of commit message>`.
- **FR-007**: The `log` command MUST support a `--max-count N` flag that limits output to at most N commits. A value of 0 displays no commits. Negative values MUST produce an error.
- **FR-008**: The `log` command MUST support a `--json` flag that outputs a JSON array of commit objects, each containing: `hash` (full commit hash), `date` (ISO 8601 format), `sourceCommit` (Source-Commit trailer value), `author` (author name), and `message` (full commit message).
- **FR-009**: When `--json` and `--oneline` are both specified, `--json` MUST take precedence and produce the full structured JSON output.
- **FR-010**: The `--max-count` flag MUST work in combination with both human-readable and JSON output modes.
- **FR-011**: The `log` command MUST verify it is running inside a valid Git repository; otherwise it MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-012**: The `log` command MUST verify the specified graph exists (ref `refs/infra/<name>` is present); otherwise it MUST fail with a non-zero exit code and a descriptive error on stderr.
- **FR-013**: The `log` command MUST handle broken commit chains gracefully: if a parent commit object cannot be found, the command MUST display all reachable commits, emit a warning on stderr indicating the chain is broken, and exit with code 0 (partial success).
- **FR-014**: The `log` command MUST handle commits with missing or malformed `Source-Commit` trailers by displaying an empty value for the source commit field.
- **FR-015**: The `log` command MUST write normal output to stdout and errors/diagnostics to stderr.
- **FR-016**: The `log` command MUST exit with code 0 on success (including partial success when a broken chain is encountered per FR-013) and non-zero on failure.
- **FR-017**: This command is read-only; it MUST NOT modify any refs, objects, or repository state.

### Key Entities

- **Graph Commit**: A Git commit object in the graph's independent commit lineage. Contains a tree hash (the graph snapshot), parent hash(es) linking to the previous commit in the chain, author/committer signatures with timestamps, and a commit message with a `Source-Commit` trailer. The initial commit (from `grif init`) is an orphan (no parents); subsequent commits (from `grif commit`) have exactly one parent.
- **Commit Chain**: The ordered sequence of graph commits reachable by following parent hashes from the graph ref's tip commit back to the initial orphan commit. This forms a linear history (no branching or merging within a single graph's lineage at this point).
- **Graph Ref**: The Git ref at `refs/infra/<name>` that points to the latest (tip) commit in the graph's commit chain. This is the starting point for the log walk.
- **Source-Commit Trailer**: A line in the commit message in the format `Source-Commit: <hash>` that records the repository HEAD SHA at the time the graph commit was created. Provides traceability between graph history and standard Git history.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can view the complete commit history of any graph in under 2 seconds for graphs with up to 1,000 commits on a local repository.
- **SC-002**: The log output displays commits in correct reverse chronological order — verified by comparing displayed dates/hashes against the known commit chain.
- **SC-003**: Using `--max-count N` produces exactly N entries (or fewer if the graph has fewer than N commits) — verifiable by counting output lines or JSON array length.
- **SC-004**: JSON output is valid, parseable JSON that can be consumed by standard tools (e.g., `jq`) without errors.
- **SC-005**: All error conditions (missing graph, not a repo, invalid flags) produce clear, descriptive messages that identify the specific problem without exposing raw Git internals.
- **SC-006**: The command produces no side effects — running `grif log` does not modify any refs or objects in the repository.

## Assumptions

- A graph has already been initialized using `grif init` (from feature 001) and optionally has additional commits from `grif commit` (from feature 002) before `grif log` is used.
- The commit chain for a graph is linear (no merge commits) at this stage. Each commit has exactly one parent, except the initial orphan commit which has none.
- The `Source-Commit` trailer key in commit messages is `Source-Commit:`, consistent with the format established by `Init` and `Commit` in the existing codebase.
- The date displayed in human-readable output uses the committer timestamp from the Git commit object.
- The author displayed in JSON output uses the author name from the Git commit object's author signature.
- When the `log` command defaults to the sole graph (no argument provided), this is the same convention used by `grif commit` and `grif status`.

## Clarifications

### Session 2026-02-17

- Q: When a broken commit chain is detected (FR-013), should the exit code be 0 (partial success) or non-zero (failure)? → A: Exit 0 (partial success). Commits are displayed; warning goes to stderr. Scripts checking $? see success.
- Q: In the default (human-readable) log output, should the commit hash be the full 40-char hash or an 8-char abbreviation? → A: Full 40-char hash in default format. Matches `git log` convention. Oneline remains 8-char; JSON remains full hash.
