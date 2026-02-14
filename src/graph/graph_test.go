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

func TestPutBlob(t *testing.T) {
	t.Parallel()

	t.Run("put blob at single-segment node path", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Put(dir, "default/vpc", []byte("10.0.0.0/16"))
		if err != nil {
			t.Fatalf("Put returned error: %v", err)
		}

		if result.Path != "default/vpc" {
			t.Errorf("result.Path = %q, want %q", result.Path, "default/vpc")
		}
		if result.Type != BlobNode {
			t.Errorf("result.Type = %q, want %q", result.Type, BlobNode)
		}
		if result.ID == "" {
			t.Error("result.ID is empty")
		}
	})

	t.Run("put blob at multi-segment path auto-creates trees", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Put(dir, "default/network/vpc", []byte("10.0.0.0/16"))
		if err != nil {
			t.Fatalf("Put returned error: %v", err)
		}

		if result.Path != "default/network/vpc" {
			t.Errorf("result.Path = %q, want %q", result.Path, "default/network/vpc")
		}
		if result.Type != BlobNode {
			t.Errorf("result.Type = %q, want %q", result.Type, BlobNode)
		}
		if result.ID == "" {
			t.Error("result.ID is empty")
		}
	})

	t.Run("replace existing blob returns new hash", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result1, err := Put(dir, "default/network/vpc", []byte("10.0.0.0/16"))
		if err != nil {
			t.Fatalf("first Put failed: %v", err)
		}

		result2, err := Put(dir, "default/network/vpc", []byte("172.16.0.0/12"))
		if err != nil {
			t.Fatalf("second Put failed: %v", err)
		}

		if result1.ID == result2.ID {
			t.Error("replacing blob should produce different hash")
		}
		if result2.Type != BlobNode {
			t.Errorf("result.Type = %q, want %q", result2.Type, BlobNode)
		}
	})

	t.Run("put creates staging ref", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/node", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		repo, _ := git.PlainOpen(dir)
		_, err := repo.Storer.Reference(plumbing.ReferenceName("refs/infra-stage/default"))
		if err != nil {
			t.Error("staging ref does not exist after Put")
		}
	})

	t.Run("put with empty blob content", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Put(dir, "default/empty", []byte{})
		if err != nil {
			t.Fatalf("Put with empty blob failed: %v", err)
		}

		if result.Type != BlobNode {
			t.Errorf("result.Type = %q, want %q", result.Type, BlobNode)
		}
	})

	t.Run("put fails for invalid path", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		_, err := Put(dir, "", []byte("data"))
		if err == nil {
			t.Fatal("Put should fail for empty path")
		}
	})

	t.Run("put fails for path with only graph name", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		_, err := Put(dir, "default", []byte("data"))
		if err == nil {
			t.Fatal("Put should fail for path with only graph name")
		}
		if !strings.Contains(err.Error(), "at least a graph name and node name") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("put fails for non-existent graph", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		_, err := Put(dir, "nonexistent/node", []byte("data"))
		if err == nil {
			t.Fatal("Put should fail for non-existent graph")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})

	t.Run("put fails when intermediate is blob", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/network", []byte("data")); err != nil {
			t.Fatalf("first Put failed: %v", err)
		}

		_, err := Put(dir, "default/network/vpc", []byte("more"))
		if err == nil {
			t.Fatal("Put should fail when intermediate is a blob")
		}
		if !strings.Contains(err.Error(), "cannot have children") {
			t.Errorf("expected 'cannot have children' error, got: %v", err)
		}
	})
}

func TestPutTree(t *testing.T) {
	t.Parallel()

	t.Run("create tree with nil blob", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Put(dir, "default/network", nil)
		if err != nil {
			t.Fatalf("Put tree failed: %v", err)
		}

		if result.Type != TreeNode {
			t.Errorf("result.Type = %q, want %q", result.Type, TreeNode)
		}
		if result.Path != "default/network" {
			t.Errorf("result.Path = %q, want %q", result.Path, "default/network")
		}
	})

	t.Run("no-op on existing tree preserves children", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Create tree with a child blob
		if _, err := Put(dir, "default/network/vpc", []byte("data")); err != nil {
			t.Fatalf("Put blob failed: %v", err)
		}

		// Put nil blob on existing tree should be no-op
		result, err := Put(dir, "default/network", nil)
		if err != nil {
			t.Fatalf("Put tree no-op failed: %v", err)
		}

		if result.Type != TreeNode {
			t.Errorf("result.Type = %q, want %q", result.Type, TreeNode)
		}
	})

	t.Run("tree-to-blob error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Create a tree
		if _, err := Put(dir, "default/network", nil); err != nil {
			t.Fatalf("Put tree failed: %v", err)
		}

		// Try to put non-nil blob on existing tree
		_, err := Put(dir, "default/network", []byte("data"))
		if err == nil {
			t.Fatal("Put should fail for tree-to-blob conversion")
		}
		if !strings.Contains(err.Error(), "cannot convert tree to blob") {
			t.Errorf("expected 'cannot convert tree to blob' error, got: %v", err)
		}
	})

	t.Run("blob-to-tree error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Create a blob
		if _, err := Put(dir, "default/node", []byte("data")); err != nil {
			t.Fatalf("Put blob failed: %v", err)
		}

		// Try to put nil blob on existing blob
		_, err := Put(dir, "default/node", nil)
		if err == nil {
			t.Fatal("Put should fail for blob-to-tree conversion")
		}
		if !strings.Contains(err.Error(), "cannot convert blob to tree") {
			t.Errorf("expected 'cannot convert blob to tree' error, got: %v", err)
		}
	})

	t.Run("put blob under explicitly created tree", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Create tree first
		if _, err := Put(dir, "default/network", nil); err != nil {
			t.Fatalf("Put tree failed: %v", err)
		}

		// Then put a blob under it
		result, err := Put(dir, "default/network/vpc", []byte("10.0.0.0/16"))
		if err != nil {
			t.Fatalf("Put blob under tree failed: %v", err)
		}

		if result.Type != BlobNode {
			t.Errorf("result.Type = %q, want %q", result.Type, BlobNode)
		}
	})
}

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("get blob content", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		data := []byte("10.0.0.0/16")
		putResult, err := Put(dir, "default/network/vpc", data)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		content, err := Get(dir, "default/network/vpc")
		if err != nil {
			t.Fatalf("Get returned error: %v", err)
		}

		if content.Type != BlobNode {
			t.Errorf("content.Type = %q, want %q", content.Type, BlobNode)
		}
		if string(content.Blob) != string(data) {
			t.Errorf("content.Blob = %q, want %q", string(content.Blob), string(data))
		}
		if content.Children != nil {
			t.Error("content.Children should be nil for blob")
		}
		if content.ID != putResult.ID {
			t.Errorf("content.ID = %q, want %q", content.ID, putResult.ID)
		}
		if content.Path != "default/network/vpc" {
			t.Errorf("content.Path = %q, want %q", content.Path, "default/network/vpc")
		}
	})

	t.Run("get tree children listing sorted by name", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Create children in non-alphabetical order
		if _, err := Put(dir, "default/network/vpc", []byte("vpc-data")); err != nil {
			t.Fatalf("Put vpc failed: %v", err)
		}
		if _, err := Put(dir, "default/network/acl", []byte("acl-data")); err != nil {
			t.Fatalf("Put acl failed: %v", err)
		}
		if _, err := Put(dir, "default/network/subnet", nil); err != nil {
			t.Fatalf("Put subnet tree failed: %v", err)
		}

		content, err := Get(dir, "default/network")
		if err != nil {
			t.Fatalf("Get returned error: %v", err)
		}

		if content.Type != TreeNode {
			t.Errorf("content.Type = %q, want %q", content.Type, TreeNode)
		}
		if content.Blob != nil {
			t.Error("content.Blob should be nil for tree")
		}
		if len(content.Children) != 3 {
			t.Fatalf("expected 3 children, got %d", len(content.Children))
		}

		// Should be sorted: acl, subnet, vpc
		expected := []struct {
			name     string
			nodeType NodeType
		}{
			{"acl", BlobNode},
			{"subnet", TreeNode},
			{"vpc", BlobNode},
		}
		for i, exp := range expected {
			if content.Children[i].Name != exp.name {
				t.Errorf("children[%d].Name = %q, want %q", i, content.Children[i].Name, exp.name)
			}
			if content.Children[i].Type != exp.nodeType {
				t.Errorf("children[%d].Type = %q, want %q", i, content.Children[i].Type, exp.nodeType)
			}
			if content.Children[i].ID == "" {
				t.Errorf("children[%d].ID is empty", i)
			}
		}
	})

	t.Run("node not found error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		_, err := Get(dir, "default/network/missing")
		if err == nil {
			t.Fatal("Get should fail for missing node")
		}
		if !strings.Contains(err.Error(), "node not found") {
			t.Errorf("expected 'node not found' error, got: %v", err)
		}
	})

	t.Run("blob traversal error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/network/vpc", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		_, err := Get(dir, "default/network/vpc/child")
		if err == nil {
			t.Fatal("Get should fail for blob traversal")
		}
		if !strings.Contains(err.Error(), "cannot be traversed") {
			t.Errorf("expected 'cannot be traversed' error, got: %v", err)
		}
	})

	t.Run("get reads staged uncommitted changes", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Put without commit
		data := []byte("staged-data")
		if _, err := Put(dir, "default/node", data); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Get should read staged data
		content, err := Get(dir, "default/node")
		if err != nil {
			t.Fatalf("Get returned error: %v", err)
		}

		if string(content.Blob) != string(data) {
			t.Errorf("content.Blob = %q, want %q", string(content.Blob), string(data))
		}
	})

	t.Run("get empty tree returns empty children slice", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Create an explicit empty tree
		if _, err := Put(dir, "default/empty-tree", nil); err != nil {
			t.Fatalf("Put tree failed: %v", err)
		}

		content, err := Get(dir, "default/empty-tree")
		if err != nil {
			t.Fatalf("Get returned error: %v", err)
		}

		if content.Type != TreeNode {
			t.Errorf("content.Type = %q, want %q", content.Type, TreeNode)
		}
		if content.Children == nil {
			t.Error("content.Children should be empty slice, not nil")
		}
		if len(content.Children) != 0 {
			t.Errorf("expected 0 children, got %d", len(content.Children))
		}
		if content.Blob != nil {
			t.Error("content.Blob should be nil for tree node")
		}
	})
}

func TestDeleteNode(t *testing.T) {
	t.Parallel()

	t.Run("delete a blob", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/network/vpc", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		if err := DeleteNode(dir, "default/network/vpc"); err != nil {
			t.Fatalf("DeleteNode returned error: %v", err)
		}

		// Verify the blob is gone
		_, err := Get(dir, "default/network/vpc")
		if err == nil {
			t.Fatal("Get should fail after delete")
		}
		if !strings.Contains(err.Error(), "node not found") {
			t.Errorf("expected 'node not found', got: %v", err)
		}
	})

	t.Run("recursive tree delete removes all descendants", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/network/vpc", []byte("vpc")); err != nil {
			t.Fatalf("Put vpc failed: %v", err)
		}
		if _, err := Put(dir, "default/network/subnet", []byte("subnet")); err != nil {
			t.Fatalf("Put subnet failed: %v", err)
		}

		if err := DeleteNode(dir, "default/network"); err != nil {
			t.Fatalf("DeleteNode returned error: %v", err)
		}

		// All should be gone
		for _, path := range []string{"default/network", "default/network/vpc", "default/network/subnet"} {
			_, err := Get(dir, path)
			if err == nil {
				t.Errorf("Get(%q) should fail after tree delete", path)
			}
		}
	})

	t.Run("node not found error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		err := DeleteNode(dir, "default/nonexistent")
		if err == nil {
			t.Fatal("DeleteNode should fail for missing node")
		}
		if !strings.Contains(err.Error(), "node not found") {
			t.Errorf("expected 'node not found', got: %v", err)
		}
	})

	t.Run("blob traversal error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/network/vpc", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		err := DeleteNode(dir, "default/network/vpc/child")
		if err == nil {
			t.Fatal("DeleteNode should fail for blob traversal")
		}
		if !strings.Contains(err.Error(), "cannot be traversed") {
			t.Errorf("expected 'cannot be traversed', got: %v", err)
		}
	})

	t.Run("parent tree preserved after child deletion", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/network/vpc", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		if err := DeleteNode(dir, "default/network/vpc"); err != nil {
			t.Fatalf("DeleteNode failed: %v", err)
		}

		// Parent tree should still exist
		content, err := Get(dir, "default/network")
		if err != nil {
			t.Fatalf("Get parent tree failed: %v", err)
		}
		if content.Type != TreeNode {
			t.Errorf("parent type = %q, want %q", content.Type, TreeNode)
		}
		if len(content.Children) != 0 {
			t.Errorf("parent has %d children, want 0", len(content.Children))
		}
	})

	t.Run("delete blob then put tree at same path", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)
		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/node", []byte("data")); err != nil {
			t.Fatalf("Put blob failed: %v", err)
		}

		if err := DeleteNode(dir, "default/node"); err != nil {
			t.Fatalf("DeleteNode failed: %v", err)
		}

		result, err := Put(dir, "default/node", nil)
		if err != nil {
			t.Fatalf("Put tree at deleted blob path failed: %v", err)
		}
		if result.Type != TreeNode {
			t.Errorf("result.Type = %q, want %q", result.Type, TreeNode)
		}
	})
}

func TestCommit(t *testing.T) {
	t.Parallel()

	t.Run("commit staged changes", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		if _, err := Put(dir, "default/node", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Commit(dir, "default", "")
		if err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		if result.Graph != "default" {
			t.Errorf("result.Graph = %q, want %q", result.Graph, "default")
		}
		if result.Ref != "refs/infra/default" {
			t.Errorf("result.Ref = %q, want %q", result.Ref, "refs/infra/default")
		}
		if result.Commit == "" {
			t.Error("result.Commit is empty")
		}

		// Staging ref should be deleted
		repo, _ := git.PlainOpen(dir)
		_, err = repo.Storer.Reference(plumbing.ReferenceName("refs/infra-stage/default"))
		if err == nil {
			t.Error("staging ref still exists after commit")
		}

		// Committed data should be readable via Get (no staging ref)
		content, err := Get(dir, "default/node")
		if err != nil {
			t.Fatalf("Get after commit failed: %v", err)
		}
		if string(content.Blob) != "data" {
			t.Errorf("content.Blob = %q, want %q", string(content.Blob), "data")
		}
	})

	t.Run("commit with custom message", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/node", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Commit(dir, "default", "Custom message")
		if err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Verify the commit message contains the custom message
		repo, _ := git.PlainOpen(dir)
		ref, _ := repo.Storer.Reference(plumbing.ReferenceName("refs/infra/default"))
		commit, _ := repo.CommitObject(ref.Hash())
		if !strings.HasPrefix(commit.Message, "Custom message") {
			t.Errorf("commit message = %q, want prefix %q", commit.Message, "Custom message")
		}
		if !strings.Contains(commit.Message, "Source-Commit:") {
			t.Errorf("commit message should contain Source-Commit trailer")
		}
		_ = result
	})

	t.Run("commit with default message format", func(t *testing.T) {
		t.Parallel()
		dir, headSHA := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/node", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		_, err := Commit(dir, "default", "")
		if err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		repo, _ := git.PlainOpen(dir)
		ref, _ := repo.Storer.Reference(plumbing.ReferenceName("refs/infra/default"))
		commit, _ := repo.CommitObject(ref.Hash())

		expectedPrefix := `Update graph "default"`
		if !strings.HasPrefix(commit.Message, expectedPrefix) {
			t.Errorf("commit message = %q, want prefix %q", commit.Message, expectedPrefix)
		}
		expectedTrailer := "Source-Commit: " + headSHA
		if !strings.Contains(commit.Message, expectedTrailer) {
			t.Errorf("commit message should contain %q, got %q", expectedTrailer, commit.Message)
		}
	})

	t.Run("no staged changes error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		_, err := Commit(dir, "default", "")
		if err == nil {
			t.Fatal("expected error for no staged changes")
		}
		if !strings.Contains(err.Error(), "no staged changes") {
			t.Errorf("error = %q, want to contain %q", err.Error(), "no staged changes")
		}
	})

	t.Run("commit with parent creates linear history", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// First commit
		if _, err := Put(dir, "default/a", []byte("first")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		result1, err := Commit(dir, "default", "first commit")
		if err != nil {
			t.Fatalf("first Commit failed: %v", err)
		}

		// Second commit
		if _, err := Put(dir, "default/b", []byte("second")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		result2, err := Commit(dir, "default", "second commit")
		if err != nil {
			t.Fatalf("second Commit failed: %v", err)
		}

		// Verify the second commit has the first as parent
		repo, _ := git.PlainOpen(dir)
		commit2Hash := plumbing.NewHash(result2.Commit)
		commit2, _ := repo.CommitObject(commit2Hash)
		if len(commit2.ParentHashes) != 1 {
			t.Fatalf("parent count = %d, want 1", len(commit2.ParentHashes))
		}
		if commit2.ParentHashes[0].String() != result1.Commit {
			t.Errorf("parent = %s, want %s", commit2.ParentHashes[0].String(), result1.Commit)
		}
	})
}

func TestStatus(t *testing.T) {
	t.Parallel()

	t.Run("status with additions", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/network/vpc", []byte("10.0.0.0/16")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Status(dir, "default")
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if result.Graph != "default" {
			t.Errorf("result.Graph = %q, want %q", result.Graph, "default")
		}
		if len(result.Changes) == 0 {
			t.Fatal("expected changes, got none")
		}

		// Should report the vpc blob as added
		found := false
		for _, c := range result.Changes {
			if c.Path == "network/vpc" && c.Status == "added" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected added change at network/vpc, got %v", result.Changes)
		}
	})

	t.Run("status with modifications", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/node", []byte("original")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "initial"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Modify the node
		if _, err := Put(dir, "default/node", []byte("updated")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Status(dir, "default")
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}

		found := false
		for _, c := range result.Changes {
			if c.Path == "node" && c.Status == "modified" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected modified change at node, got %v", result.Changes)
		}
	})

	t.Run("status with deletions", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/node", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "initial"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Delete the node
		if err := DeleteNode(dir, "default/node"); err != nil {
			t.Fatalf("DeleteNode failed: %v", err)
		}

		result, err := Status(dir, "default")
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}

		found := false
		for _, c := range result.Changes {
			if c.Path == "node" && c.Status == "deleted" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected deleted change at node, got %v", result.Changes)
		}
	})

	t.Run("status with mixed changes", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/keep", []byte("keep")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Put(dir, "default/modify", []byte("old")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Put(dir, "default/remove", []byte("gone")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "initial"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Mixed changes
		if _, err := Put(dir, "default/modify", []byte("new")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if err := DeleteNode(dir, "default/remove"); err != nil {
			t.Fatalf("DeleteNode failed: %v", err)
		}
		if _, err := Put(dir, "default/added", []byte("new node")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Status(dir, "default")
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}

		statuses := make(map[string]string)
		for _, c := range result.Changes {
			statuses[c.Path] = c.Status
		}

		if statuses["modify"] != "modified" {
			t.Errorf("expected modified for modify, got %q", statuses["modify"])
		}
		if statuses["remove"] != "deleted" {
			t.Errorf("expected deleted for remove, got %q", statuses["remove"])
		}
		if statuses["added"] != "added" {
			t.Errorf("expected added for added, got %q", statuses["added"])
		}
		// keep should not appear
		if _, ok := statuses["keep"]; ok {
			t.Errorf("keep should not appear in changes")
		}
	})

	t.Run("empty status returns empty changes not error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/node", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "initial"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		result, err := Status(dir, "default")
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if result.Changes == nil {
			t.Error("Changes should be empty slice, not nil")
		}
		if len(result.Changes) != 0 {
			t.Errorf("expected 0 changes, got %d", len(result.Changes))
		}
	})

	t.Run("status with no staging ref returns empty changes", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Status(dir, "default")
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if len(result.Changes) != 0 {
			t.Errorf("expected 0 changes, got %d", len(result.Changes))
		}
	})
}
