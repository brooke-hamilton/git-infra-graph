# Research Report: Git Operations in Go

**Feature Branch**: `001-graph-init-list`
**Date**: 2026-02-13
**Input**: Plan from [plan.md](plan.md)

## Question 1: go-git vs Shelling Out to the Git CLI

### Capability Matrix

| Capability | go-git (v5) | Shell out to `git` CLI |
|---|---|---|
| Create empty tree (well-known hash `4b825dc…`) | Yes — `object.TreeHash` for empty tree; or write a tree with `git hash-object` equivalent via plumbing API | Yes — `git hash-object -t tree /dev/null` or `git mktree < /dev/null` |
| Create orphan commits (no parent) | Yes — `object.Commit` with empty `ParentHashes` slice, write via `Storer.SetEncodedObject` | Yes — `git commit-tree <tree> -m <msg>` with no `-p` flag |
| Write custom refs (`refs/infra/<name>`) | Yes — `storer.SetReference(plumbing.NewHashReference(...))` | Yes — `git update-ref refs/infra/<name> <sha>` |
| Enumerate refs by prefix (`refs/infra/`) | Yes — `storer.IterReferences()` then filter by prefix, or use `ForEach` | Yes — `git for-each-ref refs/infra/ --format='%(refname:short)'` |
| Delete a single ref | Yes — `storer.RemoveReference(plumbing.ReferenceName(...))` | Yes — `git update-ref -d refs/infra/<name>` |
| Validate ref name components | No built-in validation function; must implement manually | Yes — `git check-ref-format --allow-onelevel <name>` |
| Commit trailers | No native trailer support; format manually in message body | No native API for trailers; `git interpret-trailers` exists but is a separate command; simpler to format manually |
| Cross-platform availability | Pure Go binary — no external dependencies | Requires `git` installed on the system and available on PATH |
| Performance for low-level ops | Fast — in-process, no fork/exec overhead, direct object store access | Slower — each operation spawns a subprocess; latency adds up for multiple operations in sequence |
| Testability | Excellent — `git.Init()` to create in-memory or on-disk repos in tests; no git binary needed; `memory.NewStorage()` for pure in-memory repos | Good — but requires `git` on the test machine; slightly more ceremony to set up and tear down repos |
| Maturity and maintenance | Actively maintained; widely used in Go ecosystem (Gitea, etc.); some edge cases around packfile handling; good issue tracker; `v5` is the current stable line | `git` CLI is the reference implementation; maximally correct and complete; versioned with Git releases |

### Deep-Dive on go-git for Required Operations

**Empty tree creation**: Git has a well-known empty tree hash (`4b825dc642cb6eb9a060e54bf899d69f82cf7b04`). In go-git, you can create an empty `object.Tree{}` and encode it to the object store via `storer.SetEncodedObject`. The hash is deterministic. Alternatively, you can reference the well-known constant directly since Git guarantees the hash of an empty tree is always the same. The go-git type `plumbing.Hash` can hold this constant.

**Orphan commit creation**: go-git's plumbing layer allows you to construct a `plumbing.MemoryObject` with type `plumbing.CommitObject`, encode a commit with zero parents, a tree reference, author/committer fields, and a message (including trailers). The `object.Commit.Encode()` method writes to an `EncodedObject`, which is then stored via `SetEncodedObject`. This returns the commit hash. No parent hashes means no parent — an orphan commit.

**Custom ref writing**: `storer.SetReference(plumbing.NewHashReference(plumbing.ReferenceName("refs/infra/name"), commitHash))` — straightforward one-liner.

**Ref enumeration**: `storer.IterReferences()` returns a `storer.ReferenceIter`. Call `.ForEach(func(ref *plumbing.Reference) error { ... })` and filter by `strings.HasPrefix(ref.Name().String(), "refs/infra/")`.

**Ref deletion**: `storer.RemoveReference(plumbing.ReferenceName("refs/infra/name"))` — single call.

### Deep-Dive on Shell-Out Approach

The git CLI approach requires composing and executing commands via `os/exec`:

```go
// Create empty tree
cmd := exec.Command("git", "mktree")
cmd.Stdin = strings.NewReader("")
cmd.Dir = repoPath
out, err := cmd.Output()
treeHash := strings.TrimSpace(string(out))

// Create orphan commit
cmd = exec.Command("git", "commit-tree", treeHash, "-m", message)
cmd.Dir = repoPath
out, err = cmd.Output()
commitHash := strings.TrimSpace(string(out))

// Write ref
exec.Command("git", "update-ref", "refs/infra/name", commitHash).Run()

// List refs
exec.Command("git", "for-each-ref", "refs/infra/", "--format=%(refname:short)").Output()

// Delete ref
exec.Command("git", "update-ref", "-d", "refs/infra/name").Run()
```

Each operation forks a subprocess. Error handling requires parsing stderr. Exit codes and output formats vary slightly across git versions.

### Decision

**Use go-git (github.com/go-git/go-git/v5)** as the primary Git backend.

### Rationale

1. **Zero external dependency**: The constitution states the CLI must work cross-platform (Linux, macOS, Windows). go-git is pure Go — no git binary required at runtime. This eliminates a class of user-environment issues (wrong git version, git not installed, PATH issues on Windows).

2. **Performance**: The `init`, `list`, and `delete` operations each require 2-4 low-level Git operations. With go-git, these are in-process calls (microseconds). Shelling out spawns 2-4 subprocesses per command (milliseconds each, with OS-dependent overhead). For the SC-001 target of < 5 seconds, both approaches easily suffice, but go-git has fundamentally lower overhead and scales better as the feature set grows.

3. **Testability**: go-git allows creating repositories entirely in Go test code without requiring git on the CI machine. Tests can use `git.PlainInit()` to create on-disk repos, or `memory.NewStorage()` for in-memory repos. The constitution mandates live Git repos for integration tests (not mocked), and go-git satisfies this: repos initialized with go-git are real Git repos that `git` can also read.

4. **API completeness for this project**: Every Git operation required by the spec (empty tree, orphan commit, custom refs, ref enumeration, ref deletion) is supported by go-git's plumbing layer. The only missing capability is ref-name validation, which must be implemented manually regardless of approach (git's `check-ref-format` is a porcelain convenience, not an API).

5. **Maintenance**: go-git v5 is actively maintained, used in production by Gitea, Fleet, and other projects. The project has regular releases and an active contributor base.

### Alternatives Considered

- **Shell out to `git` CLI**: Rejected. Adds a hard runtime dependency on git being installed. Subprocess overhead is unnecessary for the low-level operations this project needs. Error handling is more fragile (parsing stderr strings). However, this approach would provide access to `git check-ref-format` for ref validation, which is a minor advantage.

- **Hybrid approach (go-git + fallback to CLI)**: Rejected. Adds complexity without proportional benefit. If go-git handles all required operations, there is no need for a fallback path. Maintaining two code paths doubles the testing surface.

- **libgit2 via git2go**: Rejected. Requires CGo and a C compiler toolchain, which complicates cross-compilation and Windows builds. The CGo dependency is a significant operational burden for a project that targets cross-platform distribution.

## Question 2: Git Ref Name Validation in Go

### Git Ref Name Rules

Git ref names must follow the rules specified in `git-check-ref-format(1)`. For a **single component** (which is what this project needs — graph names like `my-infra` that become `refs/infra/my-infra`), the relevant rules are:

1. **No double dots**: The component must not contain `..` anywhere.
2. **No ASCII control characters**: No bytes with value < `\x20` (space) or `\x7f` (DEL).
3. **No space, tilde, caret, colon**: The characters ` `, `~`, `^`, `:` are forbidden.
4. **No question mark, asterisk, open bracket**: `?`, `*`, `[` are forbidden (glob characters).
5. **No backslash**: `\` is forbidden.
6. **No leading dot**: The component must not begin with `.`.
7. **No trailing dot**: The component must not end with `.`.
8. **No `.lock` suffix**: The component must not end with `.lock`.
9. **No `@{` sequence**: The sequence `@{` is forbidden anywhere in the name.
10. **Not the single character `@`**: The name must not be exactly `@`.
11. **No slash**: For a single component, `/` is not allowed (it would create a hierarchical ref path, which this project explicitly scopes out).
12. **Not empty**: The name must have at least one character.
13. **No trailing dot or leading/trailing whitespace**: Must not begin or end with a dot.

The complete set of forbidden patterns for a single ref-name component:

- Contains `..`
- Contains ASCII control characters (0x00-0x1F, 0x7F)
- Contains any of: ` ` `~` `^` `:` `?` `*` `[` `\` `/`
- Starts with `.`
- Ends with `.`
- Ends with `.lock`
- Contains `@{`
- Is exactly `@`
- Is empty

### Go Library Assessment

**No established Go library** implements Git ref-name validation as a reusable function. The go-git library does not export a ref-name validation function — it accepts `plumbing.ReferenceName` as a type alias for `string` with no validation.

### Decision

**Implement ref-name validation manually** in the `graph` package (e.g., in `ref.go`).

### Rationale

1. **Rules are simple and stable**: The ref-name rules have been unchanged in Git for many years. A ~30-line validation function covers all cases. There is no need for a dependency for this.

2. **Single component only**: The spec scopes graph names to single ref-name components (not hierarchical paths). This simplifies validation — no need to handle multi-component paths or hierarchical rules.

3. **No external dependency risk**: A manually implemented validator has zero dependency risk and is trivially testable with a table-driven test covering all edge cases.

4. **Error messages**: A custom validator can return domain-specific error messages ("graph name must not contain '..'") rather than generic ref-format errors.

### Implementation Sketch

```go
// ValidateGraphName checks that name is a legal single Git ref-name component
// suitable for use in refs/infra/<name>.
func ValidateGraphName(name string) error {
    if name == "" {
        return errors.New("graph name must not be empty")
    }
    if name == "@" {
        return errors.New("graph name must not be '@'")
    }
    if strings.HasPrefix(name, ".") {
        return errors.New("graph name must not start with '.'")
    }
    if strings.HasSuffix(name, ".") {
        return errors.New("graph name must not end with '.'")
    }
    if strings.HasSuffix(name, ".lock") {
        return errors.New("graph name must not end with '.lock'")
    }
    if strings.Contains(name, "..") {
        return errors.New("graph name must not contain '..'")
    }
    if strings.Contains(name, "@{") {
        return errors.New("graph name must not contain '@{'")
    }
    for _, c := range name {
        if c <= 0x1f || c == 0x7f {
            return fmt.Errorf("graph name must not contain control character 0x%02x", c)
        }
        if strings.ContainsRune(" ~^:?*[\\/", c) {
            return fmt.Errorf("graph name must not contain '%c'", c)
        }
    }
    return nil
}
```

### Alternatives Considered

- **Shell out to `git check-ref-format`**: Rejected. Would reintroduce a dependency on the git binary, contradicting the decision to use go-git. Also adds subprocess overhead for a pure validation operation.

- **Use a third-party Go validation library**: None found that specifically implements Git ref-name rules. General-purpose validators don't cover Git-specific rules.

## Question 3: Commit Trailers in Go

### Trailer Format

Git commit trailers follow the format defined by `git-interpret-trailers(1)`. They appear at the end of the commit message body, separated from the body text by a blank line. Each trailer is a key-value pair on a single line:

```text
<short summary line>

<optional body text>

Key: value
Another-Key: another value
```

Rules:

- Trailers are separated from the message body by **at least one blank line**.
- Each trailer is on its own line in the format `Key: value`.
- Keys use **Title-Case-With-Hyphens** by convention (e.g., `Signed-off-by`, `Co-authored-by`).
- The separator is `:` (colon followed by a space).
- Values extend to the end of the line.
- Multiple trailers can appear in the same block.
- Trailers must be the **last paragraph** of the commit message (no text after them).

For this project, the trailer will be:

```text
Source-Commit: <40-character hex SHA>
```

Example complete commit message:

```text
Initialize graph "my-infra"

Source-Commit: a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
```

### Decision

**Format trailers via simple string formatting** — no library needed.

### Rationale

1. **Trivial format**: A trailer is `Key: value` preceded by a blank line. For this project, there is exactly one trailer per commit (`Source-Commit: <sha>`). This is a one-liner in Go:

    ```go
    message := fmt.Sprintf("Initialize graph %q\n\nSource-Commit: %s\n", name, headSHA)
    ```

2. **No parsing needed**: The spec only requires *writing* trailers, not parsing them from existing commits. If future features need to read trailers, a simple line-based parser (split on last blank-line-delimited paragraph, parse `Key: value` lines) is straightforward.

3. **No Go trailer library exists**: There is no established Go library for Git trailer manipulation. The `go-git` library does not provide trailer support.

### Alternatives Considered

- **Shell out to `git interpret-trailers`**: Rejected. Reintroduces git binary dependency. Overkill for formatting a single key-value pair.

- **Write a general-purpose trailer parser/formatter**: Rejected for MVP. The spec requires only one trailer type. If future features need trailer parsing, a small utility function can be added later.

## Question 4: Integration Test Patterns for Git Repos in Go

### Test Repository Lifecycle

The constitution mandates:

- Integration tests use **live Git repositories** (no mocking the object database).
- Repos are created in `testdata/` (git-ignored).
- Each test creates a **uniquely named** subdirectory.
- Tests run in **parallel** without interference.
- Tests **clean up** after completion.

### Decision

**Use `testdata/` with unique subdirectories per test**, as prescribed by the constitution.

### Rationale

1. **Constitution compliance**: The constitution (IV, Integration Test Infrastructure) requires a dedicated scratch directory (e.g., `testdata/`) that is git-ignored. Using this directory directly respects the governing document.

2. **Post-mortem inspection**: When a test fails, the temporary Git repository is preserved on disk at a known, predictable location inside the project tree. This allows developers to inspect the repo state after failure — a critical debugging capability that `t.TempDir()` cannot provide because it auto-deletes on test completion, even on failure.

3. **Guaranteed uniqueness**: Each test creates a subdirectory named from the test name (sanitized) plus a random suffix, guaranteeing no collisions under `go test -parallel`.

4. **Cleanup on success only**: `t.Cleanup` removes the test's subdirectory when the test passes. On failure, the directory is intentionally left behind. The constitution's rule that "leftover directories MUST NOT cause subsequent test runs to fail" is satisfied because each run creates uniquely named subdirectories.

5. **Failure logging**: Every integration test MUST log the full path to its temporary repo via `t.Logf` so that on failure (visible with `go test -v` or in CI output) a developer can navigate directly to the repo for inspection.

### Test Setup Pattern

```go
const integrationDir = "testdata"

func setupTestRepo(t *testing.T) (*git.Repository, string) {
    t.Helper()

    // Create the scratch root if it doesn't exist
    if err := os.MkdirAll(integrationDir, 0o755); err != nil {
        t.Fatalf("failed to create integration dir: %v", err)
    }

    // Create a uniquely named subdirectory for this test
    dir, err := os.MkdirTemp(integrationDir, sanitizeTestName(t.Name())+"-")
    if err != nil {
        t.Fatalf("failed to create temp dir: %v", err)
    }

    // Log the full path so failures can be inspected
    absDir, _ := filepath.Abs(dir)
    t.Logf("integration test repo: %s", absDir)

    // Clean up on success only — leave on failure for inspection
    t.Cleanup(func() {
        if !t.Failed() {
            os.RemoveAll(dir)
        }
    })

    repo, err := git.PlainInit(dir, false)
    if err != nil {
        t.Fatalf("failed to init test repo: %v", err)
    }

    // Create an initial commit so HEAD is valid (required by spec)
    wt, _ := repo.Worktree()
    dummyPath := filepath.Join(dir, "README.md")
    os.WriteFile(dummyPath, []byte("# test\n"), 0o644)
    wt.Add("README.md")
    _, err = wt.Commit("Initial commit", &git.CommitOptions{
        Author: &object.Signature{
            Name:  "Test",
            Email: "test@test.com",
            When:  time.Now(),
        },
    })
    if err != nil {
        t.Fatalf("failed to create initial commit: %v", err)
    }

    return repo, dir
}

// sanitizeTestName replaces characters that are invalid in directory names.
func sanitizeTestName(name string) string {
    return strings.Map(func(r rune) rune {
        if r == '/' || r == '\\' || r == ' ' {
            return '_'
        }
        return r
    }, name)
}
```

### Test Structure

```go
func TestInitGraph(t *testing.T) {
    t.Parallel()

    t.Run("creates ref for new graph", func(t *testing.T) {
        t.Parallel()
        repo, dir := setupTestRepo(t)
        // ... test logic using repo and dir ...
    })

    t.Run("fails for duplicate graph name", func(t *testing.T) {
        t.Parallel()
        repo, dir := setupTestRepo(t)
        // ... test logic ...
    })
}
```

### Key Practices

- **Always call `t.Parallel()`** on both the parent test and subtests to enable parallel execution.
- **Use `t.Helper()`** in setup functions so test failure messages point to the calling test, not the helper.
- **Use `t.Fatalf()`** for setup failures (not `t.Errorf()`) — a failed setup means the test cannot proceed.
- **Use table-driven tests** for validation edge cases (ref name validation, error conditions).
- **Verify Git objects directly**: After `init`, use go-git's plumbing API to read the ref, load the commit, check parent count is zero, parse the tree hash, and verify the trailer in the message. This validates the spec requirements at the Git object level.
- **Test both success and error paths**: Each acceptance scenario in the spec has a corresponding test.
- **Do not share repos between tests**: Each test creates its own repo. This prevents test coupling and ordering dependencies.

### Alternatives Considered

- **`t.TempDir()`**: Go's built-in `t.TempDir()` provides automatic cleanup and unique names. However, it auto-deletes the directory even when the test fails, making post-mortem inspection impossible. Since inspecting a failed test's Git repo state is essential for debugging graph operations, this was rejected.

- **In-memory repos via `memory.NewStorage()`**: go-git supports fully in-memory repos. These are faster (no disk I/O) but the constitution requires "live Git repository instances" for integration tests. In-memory repos are real Git object stores but cannot be inspected with the `git` CLI for debugging. Reserve in-memory repos for unit tests of internal plumbing functions, not integration tests.

## Summary of Decisions

| # | Question | Decision | Key Rationale |
|---|----------|----------|---------------|
| 1 | go-git vs git CLI | **go-git** | Pure Go; no runtime dependency; better performance; excellent testability |
| 2 | Ref name validation | **Manual implementation** | Rules are simple and stable; no Go library exists; enables domain-specific errors |
| 3 | Commit trailers | **String formatting** | Single trailer type; trivial format; no library needed |
| 4 | Test repo pattern | **`testdata/` + `git.PlainInit()`** | Constitution-compliant; post-mortem inspection on failure; unique subdirs; cleanup on success only |
