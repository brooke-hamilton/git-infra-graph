package graph

import (
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("creates ref for new graph", func(t *testing.T) {
		t.Parallel()
		dir, headSHA := setupTestRepo(t)

		err := Init(dir, "my-infra")
		if err != nil {
			t.Fatalf("Init returned error: %v", err)
		}

		// Verify the ref exists
		repo, err := git.PlainOpen(dir)
		if err != nil {
			t.Fatalf("failed to open repo: %v", err)
		}
		ref, err := repo.Storer.Reference(plumbing.ReferenceName("refs/infra/my-infra"))
		if err != nil {
			t.Fatalf("ref not found after Init: %v", err)
		}

		// Verify commit is an orphan
		commit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			t.Fatalf("failed to read commit: %v", err)
		}
		if len(commit.ParentHashes) != 0 {
			t.Errorf("commit has %d parents, want 0 (orphan)", len(commit.ParentHashes))
		}

		// Verify tree is empty
		tree, err := repo.TreeObject(commit.TreeHash)
		if err != nil {
			t.Fatalf("failed to read tree: %v", err)
		}
		if len(tree.Entries) != 0 {
			t.Errorf("tree has %d entries, want 0", len(tree.Entries))
		}

		// Verify Source-Commit trailer matches HEAD
		if !strings.Contains(commit.Message, "Source-Commit: "+headSHA) {
			t.Errorf("commit message does not contain Source-Commit trailer with HEAD SHA\nmessage: %s\nexpected SHA: %s", commit.Message, headSHA)
		}
	})

	t.Run("fails for duplicate graph name", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "dup-graph"); err != nil {
			t.Fatalf("first Init failed: %v", err)
		}

		err := Init(dir, "dup-graph")
		if err == nil {
			t.Fatal("second Init should fail for duplicate graph")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("fails for invalid name", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		err := Init(dir, "a..b")
		if err == nil {
			t.Fatal("Init should fail for invalid name")
		}
	})

	t.Run("fails for not a repo", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		err := Init(dir, "test")
		if err == nil {
			t.Fatal("Init should fail for non-repo directory")
		}
		if !strings.Contains(err.Error(), "not a git repository") {
			t.Errorf("expected 'not a git repository' error, got: %v", err)
		}
	})

	t.Run("fails for empty repo (no commits)", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := git.PlainInit(dir, false)
		if err != nil {
			t.Fatalf("failed to create empty repo: %v", err)
		}

		err = Init(dir, "test")
		if err == nil {
			t.Fatal("Init should fail for empty repo")
		}
		if !strings.Contains(err.Error(), "repository has no commits") {
			t.Errorf("expected 'repository has no commits' error, got: %v", err)
		}
	})
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("lists multiple graphs in alphabetical order", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		// Create graphs in non-alphabetical order
		for _, name := range []string{"gamma", "alpha", "beta"} {
			if err := Init(dir, name); err != nil {
				t.Fatalf("Init(%s) failed: %v", name, err)
			}
		}

		graphs, err := List(dir)
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if len(graphs) != 3 {
			t.Fatalf("List returned %d graphs, want 3", len(graphs))
		}

		expected := []string{"alpha", "beta", "gamma"}
		for i, g := range graphs {
			if g.Name != expected[i] {
				t.Errorf("graphs[%d].Name = %q, want %q", i, g.Name, expected[i])
			}
		}
	})

	t.Run("returns empty slice when no graphs exist", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		graphs, err := List(dir)
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if len(graphs) != 0 {
			t.Errorf("List returned %d graphs, want 0", len(graphs))
		}
	})

	t.Run("fails for not a repo", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		_, err := List(dir)
		if err == nil {
			t.Fatal("List should fail for non-repo directory")
		}
		if !strings.Contains(err.Error(), "not a git repository") {
			t.Errorf("expected 'not a git repository' error, got: %v", err)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("removes ref and graph no longer in list", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "to-delete"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Verify it appears in list
		graphs, err := List(dir)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(graphs) != 1 || graphs[0].Name != "to-delete" {
			t.Fatalf("expected [to-delete] in list, got %v", graphs)
		}

		// Delete the graph
		if err := Delete(dir, "to-delete"); err != nil {
			t.Fatalf("Delete returned error: %v", err)
		}

		// Verify it no longer appears in list
		graphs, err = List(dir)
		if err != nil {
			t.Fatalf("List after delete failed: %v", err)
		}
		if len(graphs) != 0 {
			t.Errorf("expected empty list after delete, got %v", graphs)
		}

		// Verify ref no longer exists
		repo, err := git.PlainOpen(dir)
		if err != nil {
			t.Fatalf("failed to open repo: %v", err)
		}
		_, err = repo.Storer.Reference(plumbing.ReferenceName("refs/infra/to-delete"))
		if err == nil {
			t.Error("ref still exists after delete")
		}
	})

	t.Run("fails for non-existent graph", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		err := Delete(dir, "nonexistent")
		if err == nil {
			t.Fatal("Delete should fail for non-existent graph")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})

	t.Run("fails for invalid name", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		err := Delete(dir, "a..b")
		if err == nil {
			t.Fatal("Delete should fail for invalid name")
		}
	})

	t.Run("fails for not a repo", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		err := Delete(dir, "test")
		if err == nil {
			t.Fatal("Delete should fail for non-repo directory")
		}
		if !strings.Contains(err.Error(), "not a git repository") {
			t.Errorf("expected 'not a git repository' error, got: %v", err)
		}
	})

	t.Run("does not modify standard refs", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		// Get HEAD before
		repo, _ := git.PlainOpen(dir)
		headBefore, _ := repo.Head()

		if err := Init(dir, "safe-delete"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if err := Delete(dir, "safe-delete"); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Get HEAD after
		repo, _ = git.PlainOpen(dir)
		headAfter, _ := repo.Head()

		if headBefore.Hash() != headAfter.Hash() {
			t.Errorf("HEAD changed after delete: before=%s, after=%s", headBefore.Hash(), headAfter.Hash())
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("concurrent init with same name", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		// First init succeeds
		if err := Init(dir, "concurrent"); err != nil {
			t.Fatalf("first Init failed: %v", err)
		}
		// Second init with same name must fail
		err := Init(dir, "concurrent")
		if err == nil {
			t.Fatal("second Init should fail for duplicate name")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got: %v", err)
		}
	})

	t.Run("concurrent delete of same name", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "del-twice"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		// First delete succeeds
		if err := Delete(dir, "del-twice"); err != nil {
			t.Fatalf("first Delete failed: %v", err)
		}
		// Second delete must fail
		err := Delete(dir, "del-twice")
		if err == nil {
			t.Fatal("second Delete should fail for already-deleted graph")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})

	t.Run("ref created outside tool appears in list", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		// Manually create a ref under refs/infra/ using go-git
		repo, err := git.PlainOpen(dir)
		if err != nil {
			t.Fatalf("failed to open repo: %v", err)
		}
		head, _ := repo.Head()
		ref := plumbing.NewHashReference(plumbing.ReferenceName("refs/infra/external"), head.Hash())
		if err := repo.Storer.SetReference(ref); err != nil {
			t.Fatalf("failed to create external ref: %v", err)
		}

		graphs, err := List(dir)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		found := false
		for _, g := range graphs {
			if g.Name == "external" {
				found = true
				break
			}
		}
		if !found {
			t.Error("externally created ref not found in list")
		}
	})

	t.Run("standard-namespace refs unmodified after all operations", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		repo, _ := git.PlainOpen(dir)
		headBefore, _ := repo.Head()

		// Full lifecycle: init → list → delete
		if err := Init(dir, "lifecycle"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := List(dir); err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if err := Delete(dir, "lifecycle"); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		repo, _ = git.PlainOpen(dir)
		headAfter, _ := repo.Head()

		if headBefore.Hash() != headAfter.Hash() {
			t.Errorf("HEAD changed after lifecycle: before=%s, after=%s",
				headBefore.Hash(), headAfter.Hash())
		}
	})
}
