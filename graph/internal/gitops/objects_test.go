package gitops

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func initTestRepo(t *testing.T) (*git.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	// Create initial commit so HEAD is valid
	wt, _ := repo.Worktree()
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
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

func TestOpenRepo(t *testing.T) {
	t.Parallel()

	t.Run("opens valid repo", func(t *testing.T) {
		t.Parallel()
		_, dir := initTestRepo(t)
		repo, err := OpenRepo(dir)
		if err != nil {
			t.Fatalf("OpenRepo(%q) returned error: %v", dir, err)
		}
		if repo == nil {
			t.Fatal("OpenRepo returned nil repo")
		}
	})

	t.Run("returns error for non-repo", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := OpenRepo(dir)
		if err == nil {
			t.Fatal("OpenRepo should fail for non-repo directory")
		}
	})
}

func TestResolveHEAD(t *testing.T) {
	t.Parallel()

	t.Run("resolves HEAD in repo with commits", func(t *testing.T) {
		t.Parallel()
		repo, _ := initTestRepo(t)
		hash, err := ResolveHEAD(repo)
		if err != nil {
			t.Fatalf("ResolveHEAD returned error: %v", err)
		}
		if hash.IsZero() {
			t.Fatal("ResolveHEAD returned zero hash")
		}
	})

	t.Run("returns error for empty repo", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		repo, err := git.PlainInit(dir, false)
		if err != nil {
			t.Fatalf("failed to init repo: %v", err)
		}
		_, err = ResolveHEAD(repo)
		if err == nil {
			t.Fatal("ResolveHEAD should fail for empty repo")
		}
	})
}

func TestCreateEmptyTree(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)
	hash, err := CreateEmptyTree(repo)
	if err != nil {
		t.Fatalf("CreateEmptyTree returned error: %v", err)
	}
	if hash.IsZero() {
		t.Fatal("CreateEmptyTree returned zero hash")
	}

	// Verify the tree object exists and has no entries
	tree, err := repo.TreeObject(hash)
	if err != nil {
		t.Fatalf("failed to read tree object: %v", err)
	}
	if len(tree.Entries) != 0 {
		t.Errorf("empty tree has %d entries, want 0", len(tree.Entries))
	}
}

func TestCreateOrphanCommit(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	treeHash, err := CreateEmptyTree(repo)
	if err != nil {
		t.Fatalf("CreateEmptyTree failed: %v", err)
	}

	sig := object.Signature{
		Name:  "Test",
		Email: "test@test.com",
		When:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	commitHash, err := CreateOrphanCommit(repo, treeHash, "test message\n\nSource-Commit: abc123\n", sig)
	if err != nil {
		t.Fatalf("CreateOrphanCommit returned error: %v", err)
	}
	if commitHash.IsZero() {
		t.Fatal("CreateOrphanCommit returned zero hash")
	}

	// Verify the commit object
	commit, err := repo.CommitObject(commitHash)
	if err != nil {
		t.Fatalf("failed to read commit: %v", err)
	}
	if len(commit.ParentHashes) != 0 {
		t.Errorf("commit has %d parents, want 0 (orphan)", len(commit.ParentHashes))
	}
	if commit.TreeHash != treeHash {
		t.Errorf("commit tree = %s, want %s", commit.TreeHash, treeHash)
	}
}

func TestCreateRef(t *testing.T) {
	t.Parallel()

	t.Run("creates new ref", func(t *testing.T) {
		t.Parallel()
		repo, _ := initTestRepo(t)
		headHash, _ := ResolveHEAD(repo)

		err := CreateRef(repo, "refs/infra/test", headHash)
		if err != nil {
			t.Fatalf("CreateRef returned error: %v", err)
		}

		// Verify ref exists
		ref, err := repo.Storer.Reference(plumbing.ReferenceName("refs/infra/test"))
		if err != nil {
			t.Fatalf("ref not found after creation: %v", err)
		}
		if ref.Hash() != headHash {
			t.Errorf("ref hash = %s, want %s", ref.Hash(), headHash)
		}
	})

	t.Run("fails for duplicate ref", func(t *testing.T) {
		t.Parallel()
		repo, _ := initTestRepo(t)
		headHash, _ := ResolveHEAD(repo)

		if err := CreateRef(repo, "refs/infra/test", headHash); err != nil {
			t.Fatalf("first CreateRef failed: %v", err)
		}
		err := CreateRef(repo, "refs/infra/test", headHash)
		if err == nil {
			t.Fatal("second CreateRef should fail for duplicate ref")
		}
	})
}

func TestDeleteRef(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)
	headHash, _ := ResolveHEAD(repo)

	if err := CreateRef(repo, "refs/infra/to-delete", headHash); err != nil {
		t.Fatalf("CreateRef failed: %v", err)
	}

	if err := DeleteRef(repo, "refs/infra/to-delete"); err != nil {
		t.Fatalf("DeleteRef returned error: %v", err)
	}

	if RefExists(repo, "refs/infra/to-delete") {
		t.Error("ref still exists after deletion")
	}
}

func TestRefExists(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)
	headHash, _ := ResolveHEAD(repo)

	if RefExists(repo, "refs/infra/nonexistent") {
		t.Error("RefExists returned true for nonexistent ref")
	}

	if err := CreateRef(repo, "refs/infra/exists", headHash); err != nil {
		t.Fatalf("CreateRef failed: %v", err)
	}
	if !RefExists(repo, "refs/infra/exists") {
		t.Error("RefExists returned false for existing ref")
	}
}

func TestListRefsByPrefix(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)
	headHash, _ := ResolveHEAD(repo)

	// No refs initially
	names, err := ListRefsByPrefix(repo, "refs/infra/")
	if err != nil {
		t.Fatalf("ListRefsByPrefix returned error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 refs, got %d", len(names))
	}

	// Create some refs
	for _, name := range []string{"beta", "alpha", "gamma"} {
		if err := CreateRef(repo, "refs/infra/"+name, headHash); err != nil {
			t.Fatalf("CreateRef(%s) failed: %v", name, err)
		}
	}

	names, err = ListRefsByPrefix(repo, "refs/infra/")
	if err != nil {
		t.Fatalf("ListRefsByPrefix returned error: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 refs, got %d", len(names))
	}
}
