package gitops

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
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

func TestCreateBlob(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	t.Run("creates blob with content", func(t *testing.T) {
		t.Parallel()
		data := []byte("10.0.0.0/16")
		hash, err := CreateBlob(repo, data)
		if err != nil {
			t.Fatalf("CreateBlob returned error: %v", err)
		}
		if hash.IsZero() {
			t.Fatal("CreateBlob returned zero hash")
		}
	})

	t.Run("creates empty blob", func(t *testing.T) {
		t.Parallel()
		hash, err := CreateBlob(repo, []byte{})
		if err != nil {
			t.Fatalf("CreateBlob returned error: %v", err)
		}
		if hash.IsZero() {
			t.Fatal("CreateBlob returned zero hash for empty blob")
		}
	})
}

func TestReadBlobContent(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	data := []byte("hello world")
	hash, err := CreateBlob(repo, data)
	if err != nil {
		t.Fatalf("CreateBlob failed: %v", err)
	}

	content, err := ReadBlobContent(repo, hash)
	if err != nil {
		t.Fatalf("ReadBlobContent returned error: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("ReadBlobContent = %q, want %q", string(content), string(data))
	}
}

func TestSetTreeEntry(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	emptyTree, err := CreateEmptyTree(repo)
	if err != nil {
		t.Fatalf("CreateEmptyTree failed: %v", err)
	}

	blobHash, err := CreateBlob(repo, []byte("data"))
	if err != nil {
		t.Fatalf("CreateBlob failed: %v", err)
	}

	t.Run("adds entry to empty tree", func(t *testing.T) {
		t.Parallel()
		newHash, err := SetTreeEntry(repo, emptyTree, object.TreeEntry{
			Name: "file.txt",
			Mode: filemode.Regular,
			Hash: blobHash,
		})
		if err != nil {
			t.Fatalf("SetTreeEntry returned error: %v", err)
		}
		if newHash == emptyTree {
			t.Error("SetTreeEntry returned same hash as empty tree")
		}

		tree, err := GetTreeByHash(repo, newHash)
		if err != nil {
			t.Fatalf("GetTreeByHash failed: %v", err)
		}
		if len(tree.Entries) != 1 {
			t.Fatalf("tree has %d entries, want 1", len(tree.Entries))
		}
		if tree.Entries[0].Name != "file.txt" {
			t.Errorf("entry name = %q, want %q", tree.Entries[0].Name, "file.txt")
		}
	})

	t.Run("replaces existing entry", func(t *testing.T) {
		t.Parallel()
		firstHash, err := SetTreeEntry(repo, emptyTree, object.TreeEntry{
			Name: "file.txt",
			Mode: filemode.Regular,
			Hash: blobHash,
		})
		if err != nil {
			t.Fatalf("first SetTreeEntry failed: %v", err)
		}

		newBlob, _ := CreateBlob(repo, []byte("updated"))
		secondHash, err := SetTreeEntry(repo, firstHash, object.TreeEntry{
			Name: "file.txt",
			Mode: filemode.Regular,
			Hash: newBlob,
		})
		if err != nil {
			t.Fatalf("second SetTreeEntry failed: %v", err)
		}

		tree, _ := GetTreeByHash(repo, secondHash)
		if len(tree.Entries) != 1 {
			t.Fatalf("tree has %d entries, want 1", len(tree.Entries))
		}
		if tree.Entries[0].Hash != newBlob {
			t.Error("entry was not replaced")
		}
	})
}

func TestRemoveTreeEntry(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	emptyTree, _ := CreateEmptyTree(repo)
	blobHash, _ := CreateBlob(repo, []byte("data"))

	treeWithEntry, err := SetTreeEntry(repo, emptyTree, object.TreeEntry{
		Name: "file.txt",
		Mode: filemode.Regular,
		Hash: blobHash,
	})
	if err != nil {
		t.Fatalf("SetTreeEntry failed: %v", err)
	}

	t.Run("removes existing entry", func(t *testing.T) {
		t.Parallel()
		newHash, err := RemoveTreeEntry(repo, treeWithEntry, "file.txt")
		if err != nil {
			t.Fatalf("RemoveTreeEntry returned error: %v", err)
		}

		tree, _ := GetTreeByHash(repo, newHash)
		if len(tree.Entries) != 0 {
			t.Errorf("tree has %d entries, want 0", len(tree.Entries))
		}
	})

	t.Run("returns error for missing entry", func(t *testing.T) {
		t.Parallel()
		_, err := RemoveTreeEntry(repo, treeWithEntry, "nonexistent")
		if err == nil {
			t.Fatal("RemoveTreeEntry should fail for missing entry")
		}
	})
}

func TestStagingRef(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	emptyTree, _ := CreateEmptyTree(repo)

	t.Run("write and read staging ref", func(t *testing.T) {
		t.Parallel()
		err := WriteStagingRef(repo, "test-graph", emptyTree)
		if err != nil {
			t.Fatalf("WriteStagingRef returned error: %v", err)
		}

		if !RefExists(repo, "refs/infra-stage/test-graph") {
			t.Fatal("staging ref does not exist after write")
		}
	})

	t.Run("delete staging ref", func(t *testing.T) {
		t.Parallel()
		if err := WriteStagingRef(repo, "del-stage", emptyTree); err != nil {
			t.Fatalf("WriteStagingRef failed: %v", err)
		}

		if err := DeleteStagingRef(repo, "del-stage"); err != nil {
			t.Fatalf("DeleteStagingRef returned error: %v", err)
		}

		if RefExists(repo, "refs/infra-stage/del-stage") {
			t.Error("staging ref still exists after delete")
		}
	})
}

func TestResolveRootTree(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	// Create a graph ref with a commit containing an empty tree
	emptyTree, _ := CreateEmptyTree(repo)
	sig := DefaultSignature()
	commitHash, _ := CreateOrphanCommit(repo, emptyTree, "test\n", sig)
	if err := CreateRef(repo, "refs/infra/resolve-test", commitHash); err != nil {
		t.Fatalf("CreateRef failed: %v", err)
	}

	t.Run("resolves from committed tree when no staging", func(t *testing.T) {
		t.Parallel()
		hash, err := ResolveRootTree(repo, "resolve-test")
		if err != nil {
			t.Fatalf("ResolveRootTree returned error: %v", err)
		}
		if hash != emptyTree {
			t.Errorf("ResolveRootTree = %s, want %s (empty tree)", hash, emptyTree)
		}
	})

	t.Run("resolves from staging ref when present", func(t *testing.T) {
		t.Parallel()
		blobHash, _ := CreateBlob(repo, []byte("staged"))
		stagedTree, _ := StoreTree(repo, []object.TreeEntry{
			{Name: "staged.txt", Mode: filemode.Regular, Hash: blobHash},
		})

		if err := WriteStagingRef(repo, "resolve-test", stagedTree); err != nil {
			t.Fatalf("WriteStagingRef failed: %v", err)
		}

		hash, err := ResolveRootTree(repo, "resolve-test")
		if err != nil {
			t.Fatalf("ResolveRootTree returned error: %v", err)
		}
		if hash != stagedTree {
			t.Errorf("ResolveRootTree = %s, want %s (staged tree)", hash, stagedTree)
		}

		// Clean up staging ref for other subtests
		DeleteStagingRef(repo, "resolve-test")
	})

	t.Run("returns error for missing graph", func(t *testing.T) {
		t.Parallel()
		_, err := ResolveRootTree(repo, "nonexistent")
		if err == nil {
			t.Fatal("ResolveRootTree should fail for missing graph")
		}
	})
}

func TestCreateCommit(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	emptyTree, _ := CreateEmptyTree(repo)
	sig := DefaultSignature()

	t.Run("creates orphan commit (empty parents)", func(t *testing.T) {
		t.Parallel()
		hash, err := CreateCommit(repo, emptyTree, []plumbing.Hash{}, "orphan\n", sig)
		if err != nil {
			t.Fatalf("CreateCommit returned error: %v", err)
		}

		commit, _ := repo.CommitObject(hash)
		if len(commit.ParentHashes) != 0 {
			t.Errorf("commit has %d parents, want 0", len(commit.ParentHashes))
		}
	})

	t.Run("creates commit with parent", func(t *testing.T) {
		t.Parallel()
		parentHash, _ := CreateCommit(repo, emptyTree, []plumbing.Hash{}, "parent\n", sig)
		childHash, err := CreateCommit(repo, emptyTree, []plumbing.Hash{parentHash}, "child\n", sig)
		if err != nil {
			t.Fatalf("CreateCommit with parent returned error: %v", err)
		}

		commit, _ := repo.CommitObject(childHash)
		if len(commit.ParentHashes) != 1 {
			t.Fatalf("commit has %d parents, want 1", len(commit.ParentHashes))
		}
		if commit.ParentHashes[0] != parentHash {
			t.Errorf("parent hash = %s, want %s", commit.ParentHashes[0], parentHash)
		}
	})
}

func TestUpdateRef(t *testing.T) {
	t.Parallel()
	repo, _ := initTestRepo(t)

	headHash, _ := ResolveHEAD(repo)

	// Create initial ref
	if err := CreateRef(repo, "refs/infra/update-test", headHash); err != nil {
		t.Fatalf("CreateRef failed: %v", err)
	}

	// Create a new commit to update to
	emptyTree, _ := CreateEmptyTree(repo)
	sig := DefaultSignature()
	newHash, _ := CreateCommit(repo, emptyTree, []plumbing.Hash{}, "new\n", sig)

	// Update the ref
	if err := UpdateRef(repo, "refs/infra/update-test", newHash); err != nil {
		t.Fatalf("UpdateRef returned error: %v", err)
	}

	// Verify the ref was updated
	ref, _ := repo.Storer.Reference(plumbing.ReferenceName("refs/infra/update-test"))
	if ref.Hash() != newHash {
		t.Errorf("ref hash = %s, want %s", ref.Hash(), newHash)
	}
}
