package graph

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

func TestLog(t *testing.T) {
	t.Parallel()

	t.Run("single commit (init only)", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Log(dir, "default", LogOptions{})
		if err != nil {
			t.Fatalf("Log returned error: %v", err)
		}

		if len(result.Entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result.Entries))
		}
		if result.Warning != "" {
			t.Errorf("expected no warning, got %q", result.Warning)
		}

		entry := result.Entries[0]
		if len(entry.Hash) != 40 {
			t.Errorf("expected 40-char hash, got %d chars: %q", len(entry.Hash), entry.Hash)
		}
		if entry.Date.IsZero() {
			t.Error("entry.Date is zero")
		}
		if entry.Author == "" {
			t.Error("entry.Author is empty")
		}
		if !strings.Contains(entry.Message, "Initialize graph") {
			t.Errorf("expected init message, got %q", entry.Message)
		}
		if entry.SourceCommit == "" {
			t.Error("entry.SourceCommit is empty for init commit")
		}
	})

	t.Run("multiple commits in reverse chronological order", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// First commit
		if _, err := Put(dir, "default/a", []byte("first")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "first commit"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Second commit
		if _, err := Put(dir, "default/b", []byte("second")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "second commit"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		result, err := Log(dir, "default", LogOptions{})
		if err != nil {
			t.Fatalf("Log returned error: %v", err)
		}

		if len(result.Entries) != 3 {
			t.Fatalf("expected 3 entries (2 commits + init), got %d", len(result.Entries))
		}

		// Newest first
		if !strings.Contains(result.Entries[0].Message, "second commit") {
			t.Errorf("entries[0] should be newest, got %q", result.Entries[0].Message)
		}
		if !strings.Contains(result.Entries[1].Message, "first commit") {
			t.Errorf("entries[1] should be middle, got %q", result.Entries[1].Message)
		}
		if !strings.Contains(result.Entries[2].Message, "Initialize graph") {
			t.Errorf("entries[2] should be init, got %q", result.Entries[2].Message)
		}
	})

	t.Run("graph not found error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		_, err := Log(dir, "nonexistent", LogOptions{})
		if err == nil {
			t.Fatal("Log should fail for non-existent graph")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})

	t.Run("negative max-count error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		_, err := Log(dir, "default", LogOptions{MaxCount: -1, HasMaxCount: true})
		if err == nil {
			t.Fatal("Log should fail for negative max-count")
		}
		if !strings.Contains(err.Error(), "max-count must be a non-negative integer") {
			t.Errorf("expected 'max-count must be a non-negative integer' error, got: %v", err)
		}
	})

	t.Run("max-count limits results", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/a", []byte("first")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "first commit"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
		if _, err := Put(dir, "default/b", []byte("second")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "second commit"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		result, err := Log(dir, "default", LogOptions{MaxCount: 2, HasMaxCount: true})
		if err != nil {
			t.Fatalf("Log returned error: %v", err)
		}

		if len(result.Entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(result.Entries))
		}
		if !strings.Contains(result.Entries[0].Message, "second commit") {
			t.Errorf("entries[0] should be newest, got %q", result.Entries[0].Message)
		}
	})

	t.Run("max-count 0 returns empty", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Log(dir, "default", LogOptions{MaxCount: 0, HasMaxCount: true})
		if err != nil {
			t.Fatalf("Log returned error: %v", err)
		}

		if len(result.Entries) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(result.Entries))
		}
	})

	t.Run("max-count exceeding commit count returns all", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Log(dir, "default", LogOptions{MaxCount: 100, HasMaxCount: true})
		if err != nil {
			t.Fatalf("Log returned error: %v", err)
		}

		if len(result.Entries) != 1 {
			t.Fatalf("expected 1 entry (init only), got %d", len(result.Entries))
		}
	})

	t.Run("missing Source-Commit trailer produces empty string", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Manually create a commit without a Source-Commit trailer
		repo, err := git.PlainOpen(dir)
		if err != nil {
			t.Fatalf("failed to open repo: %v", err)
		}

		ref, err := repo.Storer.Reference(plumbing.ReferenceName("refs/infra/default"))
		if err != nil {
			t.Fatalf("failed to get ref: %v", err)
		}

		tipCommit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			t.Fatalf("failed to read tip commit: %v", err)
		}

		// Create a new commit with no Source-Commit trailer
		newCommit := &object.Commit{
			Author:       tipCommit.Author,
			Committer:    tipCommit.Committer,
			Message:      "No trailer here\n",
			TreeHash:     tipCommit.TreeHash,
			ParentHashes: []plumbing.Hash{ref.Hash()},
		}

		obj := repo.Storer.NewEncodedObject()
		if err := newCommit.Encode(obj); err != nil {
			t.Fatalf("failed to encode commit: %v", err)
		}
		newHash, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			t.Fatalf("failed to store commit: %v", err)
		}

		// Update the ref
		newRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/infra/default"), newHash)
		if err := repo.Storer.SetReference(newRef); err != nil {
			t.Fatalf("failed to update ref: %v", err)
		}

		result, logErr := Log(dir, "default", LogOptions{})
		if logErr != nil {
			t.Fatalf("Log returned error: %v", logErr)
		}

		if len(result.Entries) < 1 {
			t.Fatal("expected at least 1 entry")
		}
		if result.Entries[0].SourceCommit != "" {
			t.Errorf("expected empty SourceCommit, got %q", result.Entries[0].SourceCommit)
		}
	})

	t.Run("broken commit chain returns partial result with warning", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// Create a commit whose parent hash points to a non-existent object
		repo, err := git.PlainOpen(dir)
		if err != nil {
			t.Fatalf("failed to open repo: %v", err)
		}

		ref, err := repo.Storer.Reference(plumbing.ReferenceName("refs/infra/default"))
		if err != nil {
			t.Fatalf("failed to get ref: %v", err)
		}

		tipCommit, err := repo.CommitObject(ref.Hash())
		if err != nil {
			t.Fatalf("failed to read tip commit: %v", err)
		}

		// Create a commit with a broken parent hash
		fakeParent := plumbing.NewHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		newCommit := &object.Commit{
			Author:       tipCommit.Author,
			Committer:    tipCommit.Committer,
			Message:      "Commit with broken parent\n\nSource-Commit: abc123\n",
			TreeHash:     tipCommit.TreeHash,
			ParentHashes: []plumbing.Hash{fakeParent},
		}

		obj := repo.Storer.NewEncodedObject()
		if err := newCommit.Encode(obj); err != nil {
			t.Fatalf("failed to encode commit: %v", err)
		}
		newHash, err := repo.Storer.SetEncodedObject(obj)
		if err != nil {
			t.Fatalf("failed to store commit: %v", err)
		}

		// Update the ref to the new commit
		newRef := plumbing.NewHashReference(plumbing.ReferenceName("refs/infra/default"), newHash)
		if err := repo.Storer.SetReference(newRef); err != nil {
			t.Fatalf("failed to update ref: %v", err)
		}

		result, logErr := Log(dir, "default", LogOptions{})
		if logErr != nil {
			t.Fatalf("Log should not return error for broken chain, got: %v", logErr)
		}

		if len(result.Entries) != 1 {
			t.Fatalf("expected 1 entry (reachable commit only), got %d", len(result.Entries))
		}
		if result.Warning == "" {
			t.Error("expected warning for broken chain, got empty")
		}
	})

	t.Run("not a git repository", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		_, err := Log(dir, "default", LogOptions{})
		if err == nil {
			t.Fatal("Log should fail for non-repo directory")
		}
		if !strings.Contains(err.Error(), "not a git repository") {
			t.Errorf("expected 'not a git repository' error, got: %v", err)
		}
	})

	t.Run("invalid graph name", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		_, err := Log(dir, "a..b", LogOptions{})
		if err == nil {
			t.Fatal("Log should fail for invalid graph name")
		}
	})
}

func TestTree(t *testing.T) {
	t.Parallel()

	t.Run("full graph tree with multiple levels", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/network/vpc", []byte("vpc-data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Put(dir, "default/network/subnet", []byte("subnet-data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Put(dir, "default/compute/instance", []byte("instance-data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", ""); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		result, err := Tree(dir, "default", TreeOptions{})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}

		if result.Name != "default" {
			t.Errorf("Name = %q, want %q", result.Name, "default")
		}
		if result.Type != TreeNode {
			t.Errorf("Type = %q, want %q", result.Type, TreeNode)
		}
		if len(result.Children) != 2 {
			t.Fatalf("expected 2 children (compute, network), got %d", len(result.Children))
		}
		// Children should be sorted alphabetically
		if result.Children[0].Name != "compute" {
			t.Errorf("first child = %q, want %q", result.Children[0].Name, "compute")
		}
		if result.Children[1].Name != "network" {
			t.Errorf("second child = %q, want %q", result.Children[1].Name, "network")
		}
		// Verify nested children
		if len(result.Children[0].Children) != 1 {
			t.Fatalf("compute should have 1 child, got %d", len(result.Children[0].Children))
		}
		if result.Children[0].Children[0].Name != "instance" {
			t.Errorf("compute child = %q, want %q", result.Children[0].Children[0].Name, "instance")
		}
		if result.Children[0].Children[0].Type != BlobNode {
			t.Errorf("instance type = %q, want %q", result.Children[0].Children[0].Type, BlobNode)
		}
		if len(result.Children[0].Children[0].ID) != 8 {
			t.Errorf("ID length = %d, want 8", len(result.Children[0].Children[0].ID))
		}
	})

	t.Run("empty graph", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "empty"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := Tree(dir, "empty", TreeOptions{})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		if result.Name != "empty" {
			t.Errorf("Name = %q, want %q", result.Name, "empty")
		}
		if result.Type != TreeNode {
			t.Errorf("Type = %q, want %q", result.Type, TreeNode)
		}
		if len(result.Children) != 0 {
			t.Errorf("expected 0 children, got %d", len(result.Children))
		}
	})

	t.Run("graph not found", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		_, err := Tree(dir, "nonexistent", TreeOptions{})
		if err == nil {
			t.Fatal("expected error for nonexistent graph")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' in error, got: %v", err)
		}
	})

	t.Run("subtree path resolving to tree node", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/network/vpc", []byte("vpc-data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Put(dir, "default/network/subnet", []byte("subnet-data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Tree(dir, "default/network", TreeOptions{})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		if result.Name != "network" {
			t.Errorf("Name = %q, want %q", result.Name, "network")
		}
		if result.Type != TreeNode {
			t.Errorf("Type = %q, want %q", result.Type, TreeNode)
		}
		if len(result.Children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(result.Children))
		}
		if result.Children[0].Name != "subnet" {
			t.Errorf("first child = %q, want %q", result.Children[0].Name, "subnet")
		}
		if result.Children[1].Name != "vpc" {
			t.Errorf("second child = %q, want %q", result.Children[1].Name, "vpc")
		}
	})

	t.Run("subtree path resolving to blob node", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/network/vpc", []byte("vpc-data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Tree(dir, "default/network/vpc", TreeOptions{})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		if result.Name != "vpc" {
			t.Errorf("Name = %q, want %q", result.Name, "vpc")
		}
		if result.Type != BlobNode {
			t.Errorf("Type = %q, want %q", result.Type, BlobNode)
		}
		if result.Children != nil {
			t.Errorf("expected nil children for blob, got %v", result.Children)
		}
		if len(result.ID) != 8 {
			t.Errorf("ID length = %d, want 8", len(result.ID))
		}
	})

	t.Run("path not found", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		_, err := Tree(dir, "default/nonexistent", TreeOptions{})
		if err == nil {
			t.Fatal("expected error for nonexistent path")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' in error, got: %v", err)
		}
	})

	t.Run("negative depth error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		_, err := Tree(dir, "default", TreeOptions{Depth: -1, HasDepth: true})
		if err == nil {
			t.Fatal("expected error for negative depth")
		}
		if !strings.Contains(err.Error(), "depth must be a non-negative integer") {
			t.Errorf("expected depth error, got: %v", err)
		}
	})

	t.Run("depth 0 returns root only", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/compute/instance", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Tree(dir, "default", TreeOptions{Depth: 0, HasDepth: true})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		if result.Name != "default" {
			t.Errorf("Name = %q, want %q", result.Name, "default")
		}
		if result.Children != nil {
			t.Errorf("depth 0 should have nil children, got %v", result.Children)
		}
	})

	t.Run("depth 1 returns root and immediate children", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/compute/instance", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Put(dir, "default/network/vpc", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Tree(dir, "default", TreeOptions{Depth: 1, HasDepth: true})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		if len(result.Children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(result.Children))
		}
		// Children should be trees but have nil children (depth exhausted)
		for _, child := range result.Children {
			if child.Type != TreeNode {
				t.Errorf("child %q type = %q, want %q", child.Name, child.Type, TreeNode)
			}
			if child.Children != nil {
				t.Errorf("child %q should have nil children at depth 1, got %d", child.Name, len(child.Children))
			}
		}
	})

	t.Run("depth exceeding actual depth returns full tree", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/compute/instance", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Tree(dir, "default", TreeOptions{Depth: 100, HasDepth: true})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		if len(result.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(result.Children))
		}
		if len(result.Children[0].Children) != 1 {
			t.Fatalf("expected 1 grandchild, got %d", len(result.Children[0].Children))
		}
	})

	t.Run("single segment path returns full tree", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/compute/instance", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Tree(dir, "default", TreeOptions{})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		if result.Name != "default" {
			t.Errorf("Name = %q, want %q", result.Name, "default")
		}
		if len(result.Children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(result.Children))
		}
	})

	t.Run("staged tree preferred over committed tree", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if _, err := Put(dir, "default/compute/instance", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", "first commit"); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Stage a new node without committing
		if _, err := Put(dir, "default/network/vpc", []byte("vpc-data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		result, err := Tree(dir, "default", TreeOptions{})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		// Staged tree should have both compute and network
		if len(result.Children) != 2 {
			t.Fatalf("expected 2 children (staged tree), got %d", len(result.Children))
		}
	})

	t.Run("not a Git repo", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		_, err := Tree(dir, "default", TreeOptions{})
		if err == nil {
			t.Fatal("expected error for non-repo directory")
		}
		if !strings.Contains(err.Error(), "not a git repository") {
			t.Errorf("expected 'not a git repository' error, got: %v", err)
		}
	})

	t.Run("empty path returns error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		_, err := Tree(dir, "", TreeOptions{})
		if err == nil {
			t.Fatal("expected error for empty path")
		}
		if !strings.Contains(err.Error(), "path must not be empty") {
			t.Errorf("expected 'path must not be empty' error, got: %v", err)
		}
	})

	t.Run("children sorted alphabetically at each level", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		// Insert in non-alphabetical order
		for _, name := range []string{"zebra", "alpha", "middle"} {
			if _, err := Put(dir, "default/"+name, []byte("data")); err != nil {
				t.Fatalf("Put failed: %v", err)
			}
		}

		result, err := Tree(dir, "default", TreeOptions{})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}
		if len(result.Children) != 3 {
			t.Fatalf("expected 3 children, got %d", len(result.Children))
		}
		expected := []string{"alpha", "middle", "zebra"}
		for i, child := range result.Children {
			if child.Name != expected[i] {
				t.Errorf("child[%d] = %q, want %q", i, child.Name, expected[i])
			}
		}
	})
}

func TestTreeAll(t *testing.T) {
	t.Parallel()

	t.Run("multiple graphs in alphabetical order", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		for _, name := range []string{"staging", "default", "production"} {
			if err := Init(dir, name); err != nil {
				t.Fatalf("Init(%s) failed: %v", name, err)
			}
		}
		if _, err := Put(dir, "default/compute/instance", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "default", ""); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
		if _, err := Put(dir, "staging/network/vpc", []byte("data")); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		if _, err := Commit(dir, "staging", ""); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		result, err := TreeAll(dir, TreeOptions{})
		if err != nil {
			t.Fatalf("TreeAll returned error: %v", err)
		}

		if len(result.Graphs) != 3 {
			t.Fatalf("expected 3 graphs, got %d", len(result.Graphs))
		}
		expected := []string{"default", "production", "staging"}
		for i, g := range result.Graphs {
			if g.Name != expected[i] {
				t.Errorf("graphs[%d].Name = %q, want %q", i, g.Name, expected[i])
			}
		}
	})

	t.Run("single graph", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "only"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		result, err := TreeAll(dir, TreeOptions{})
		if err != nil {
			t.Fatalf("TreeAll returned error: %v", err)
		}
		if len(result.Graphs) != 1 {
			t.Fatalf("expected 1 graph, got %d", len(result.Graphs))
		}
		if result.Graphs[0].Name != "only" {
			t.Errorf("Name = %q, want %q", result.Graphs[0].Name, "only")
		}
	})

	t.Run("no graphs found", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		_, err := TreeAll(dir, TreeOptions{})
		if err == nil {
			t.Fatal("expected error when no graphs found")
		}
		if !strings.Contains(err.Error(), "no graphs found") {
			t.Errorf("expected 'no graphs found' error, got: %v", err)
		}
	})

	t.Run("negative depth error", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		_, err := TreeAll(dir, TreeOptions{Depth: -1, HasDepth: true})
		if err == nil {
			t.Fatal("expected error for negative depth")
		}
		if !strings.Contains(err.Error(), "depth must be a non-negative integer") {
			t.Errorf("expected depth error, got: %v", err)
		}
	})

	t.Run("depth limiting applied uniformly", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		for _, name := range []string{"alpha", "beta"} {
			if err := Init(dir, name); err != nil {
				t.Fatalf("Init failed: %v", err)
			}
			if _, err := Put(dir, name+"/compute/instance", []byte("data")); err != nil {
				t.Fatalf("Put failed: %v", err)
			}
			if _, err := Commit(dir, name, ""); err != nil {
				t.Fatalf("Commit failed: %v", err)
			}
		}

		result, err := TreeAll(dir, TreeOptions{Depth: 1, HasDepth: true})
		if err != nil {
			t.Fatalf("TreeAll returned error: %v", err)
		}
		for _, g := range result.Graphs {
			if len(g.Children) != 1 {
				t.Errorf("graph %q: expected 1 child at depth 1, got %d", g.Name, len(g.Children))
			}
			for _, child := range g.Children {
				if child.Children != nil {
					t.Errorf("graph %q child %q: expected nil children at depth 1", g.Name, child.Name)
				}
			}
		}
	})
}

func TestFormatTree(t *testing.T) {
	t.Parallel()

	t.Run("box-drawing connectors", func(t *testing.T) {
		t.Parallel()

		tree := &TreeResult{
			Name: "default",
			Type: TreeNode,
			ID:   "d4e5f6a7",
			Children: []TreeItem{
				{
					Name: "compute",
					Type: TreeNode,
					ID:   "b8c9d0e1",
					Children: []TreeItem{
						{Name: "instance", Type: BlobNode, ID: "c3d4e5f6"},
					},
				},
				{
					Name: "network",
					Type: TreeNode,
					ID:   "a7b8c9d0",
					Children: []TreeItem{
						{Name: "subnet", Type: BlobNode, ID: "e5f6a7b8"},
						{Name: "vpc", Type: BlobNode, ID: "a1b2c3d4"},
					},
				},
			},
		}

		got := FormatTree(tree)
		expected := "default\n" +
			"├── compute\n" +
			"│   └── instance  (blob, c3d4e5f6)\n" +
			"└── network\n" +
			"    ├── subnet  (blob, e5f6a7b8)\n" +
			"    └── vpc  (blob, a1b2c3d4)\n"

		if got != expected {
			t.Errorf("FormatTree mismatch:\ngot:\n%s\nexpected:\n%s", got, expected)
		}
	})

	t.Run("blob annotation format", func(t *testing.T) {
		t.Parallel()

		tree := &TreeResult{
			Name: "default",
			Type: TreeNode,
			ID:   "abcdef12",
			Children: []TreeItem{
				{Name: "myblob", Type: BlobNode, ID: "12345678"},
			},
		}

		got := FormatTree(tree)
		if !strings.Contains(got, "myblob  (blob, 12345678)") {
			t.Errorf("expected blob annotation format, got:\n%s", got)
		}
	})

	t.Run("tree nodes show name only", func(t *testing.T) {
		t.Parallel()

		tree := &TreeResult{
			Name: "default",
			Type: TreeNode,
			ID:   "abcdef12",
			Children: []TreeItem{
				{
					Name:     "subtree",
					Type:     TreeNode,
					ID:       "11223344",
					Children: []TreeItem{},
				},
			},
		}

		got := FormatTree(tree)
		// Tree nodes should not have annotation
		if strings.Contains(got, "(tree,") {
			t.Errorf("tree nodes should not have type annotation, got:\n%s", got)
		}
		if !strings.Contains(got, "└── subtree\n") {
			t.Errorf("expected tree name only, got:\n%s", got)
		}
	})

	t.Run("empty graph shows only graph name", func(t *testing.T) {
		t.Parallel()

		tree := &TreeResult{
			Name:     "empty",
			Type:     TreeNode,
			ID:       "aaaabbbb",
			Children: []TreeItem{},
		}

		got := FormatTree(tree)
		if got != "empty\n" {
			t.Errorf("expected only graph name, got:\n%s", got)
		}
	})

	t.Run("blob at root displays correctly", func(t *testing.T) {
		t.Parallel()

		tree := &TreeResult{
			Name: "vpc",
			Type: BlobNode,
			ID:   "a1b2c3d4",
		}

		got := FormatTree(tree)
		if got != "vpc  (blob, a1b2c3d4)\n" {
			t.Errorf("expected blob format, got:\n%s", got)
		}
	})

	t.Run("entries sorted alphabetically at each level", func(t *testing.T) {
		t.Parallel()
		dir, _ := setupTestRepo(t)

		if err := Init(dir, "default"); err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		for _, name := range []string{"zebra", "alpha", "middle"} {
			if _, err := Put(dir, "default/"+name, []byte("data")); err != nil {
				t.Fatalf("Put failed: %v", err)
			}
		}

		result, err := Tree(dir, "default", TreeOptions{})
		if err != nil {
			t.Fatalf("Tree returned error: %v", err)
		}

		got := FormatTree(result)
		lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
		// Lines should be: default, alpha, middle, zebra
		if len(lines) != 4 {
			t.Fatalf("expected 4 lines, got %d: %v", len(lines), lines)
		}
		if !strings.Contains(lines[1], "alpha") {
			t.Errorf("expected alpha first, got: %s", lines[1])
		}
		if !strings.Contains(lines[2], "middle") {
			t.Errorf("expected middle second, got: %s", lines[2])
		}
		if !strings.Contains(lines[3], "zebra") {
			t.Errorf("expected zebra third, got: %s", lines[3])
		}
	})
}

func TestFormatTreeAll(t *testing.T) {
	t.Parallel()

	t.Run("multiple graphs separated by blank line", func(t *testing.T) {
		t.Parallel()

		result := &TreeAllResult{
			Graphs: []TreeResult{
				{
					Name: "alpha",
					Type: TreeNode,
					ID:   "aaaa1111",
					Children: []TreeItem{
						{Name: "node1", Type: BlobNode, ID: "11111111"},
					},
				},
				{
					Name: "beta",
					Type: TreeNode,
					ID:   "bbbb2222",
					Children: []TreeItem{
						{Name: "node2", Type: BlobNode, ID: "22222222"},
					},
				},
			},
		}

		got := FormatTreeAll(result)
		expected := "alpha\n" +
			"└── node1  (blob, 11111111)\n" +
			"\n" +
			"beta\n" +
			"└── node2  (blob, 22222222)\n"

		if got != expected {
			t.Errorf("FormatTreeAll mismatch:\ngot:\n%s\nexpected:\n%s", got, expected)
		}
	})

	t.Run("single graph no trailing blank line", func(t *testing.T) {
		t.Parallel()

		result := &TreeAllResult{
			Graphs: []TreeResult{
				{
					Name: "only",
					Type: TreeNode,
					ID:   "cccc3333",
					Children: []TreeItem{
						{Name: "node", Type: BlobNode, ID: "33333333"},
					},
				},
			},
		}

		got := FormatTreeAll(result)
		expected := "only\n" +
			"└── node  (blob, 33333333)\n"

		if got != expected {
			t.Errorf("FormatTreeAll mismatch:\ngot:\n%s\nexpected:\n%s", got, expected)
		}
	})
}

func BenchmarkTree1000Nodes(b *testing.B) {
	// Setup: create a graph with 1000 blobs across multiple levels
	dir, err := os.MkdirTemp(integrationDir, "BenchmarkTree1000Nodes-")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}
	b.Cleanup(func() { os.RemoveAll(dir) })

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		b.Fatalf("failed to init repo: %v", err)
	}

	wt, _ := repo.Worktree()
	dummyPath := dir + "/README.md"
	if err := os.WriteFile(dummyPath, []byte("# bench\n"), 0o644); err != nil {
		b.Fatalf("failed to write dummy: %v", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		b.Fatalf("failed to add: %v", err)
	}
	if _, err := wt.Commit("Initial", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	}); err != nil {
		b.Fatalf("failed to commit: %v", err)
	}

	if err := Init(dir, "bench"); err != nil {
		b.Fatalf("Init failed: %v", err)
	}

	// Create 1000 blobs in 10 groups of 100
	for group := 0; group < 10; group++ {
		for node := 0; node < 100; node++ {
			path := fmt.Sprintf("bench/group%02d/node%03d", group, node)
			if _, err := Put(dir, path, []byte(fmt.Sprintf("data-%d-%d", group, node))); err != nil {
				b.Fatalf("Put failed: %v", err)
			}
		}
	}

	if _, err := Commit(dir, "bench", "Benchmark data"); err != nil {
		b.Fatalf("Commit failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := Tree(dir, "bench", TreeOptions{})
		if err != nil {
			b.Fatalf("Tree failed: %v", err)
		}
		if result.Name != "bench" {
			b.Fatalf("unexpected name: %s", result.Name)
		}
	}
}
