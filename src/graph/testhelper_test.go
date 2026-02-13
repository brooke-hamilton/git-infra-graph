package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const integrationDir = "../../testdata"

// setupTestRepo creates a new Git repository in a unique subdirectory of
// testdata/ (repo root) for integration testing. The repo is initialized
// with a single commit so HEAD is valid.
//
// On test success the directory is removed. On failure it is preserved
// for post-mortem inspection and its path is logged.
func setupTestRepo(t *testing.T) (string, string) {
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
	if err := os.WriteFile(dummyPath, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	commitHash, err := wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return dir, commitHash.String()
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
