package gitops

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// OpenRepo opens an existing Git repository at the given path.
// Returns a descriptive error if the path is not a valid Git repository.
func OpenRepo(repoPath string) (*git.Repository, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}
	return repo, nil
}

// ResolveHEAD resolves HEAD to a commit hash. Returns an error if
// the repository has no commits.
func ResolveHEAD(repo *git.Repository) (plumbing.Hash, error) {
	ref, err := repo.Head()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("repository has no commits")
	}
	return ref.Hash(), nil
}

// CreateEmptyTree writes an empty tree object to the repository's object
// store and returns its hash. The empty tree hash is always
// 4b825dc642cb6eb9a060e54bf899d69f82cf7b04.
func CreateEmptyTree(repo *git.Repository) (plumbing.Hash, error) {
	tree := &object.Tree{}
	store := repo.Storer
	obj := store.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode empty tree: %w", err)
	}
	hash, err := store.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store empty tree: %w", err)
	}
	return hash, nil
}

// CreateOrphanCommit creates an orphan commit (no parents) with the given
// tree, message, and author/committer signature. Returns the commit hash.
func CreateOrphanCommit(repo *git.Repository, treeHash plumbing.Hash, message string, sig object.Signature) (plumbing.Hash, error) {
	commit := &object.Commit{
		TreeHash:     treeHash,
		ParentHashes: []plumbing.Hash{}, // orphan — no parents
		Author:       sig,
		Committer:    sig,
		Message:      message,
	}

	store := repo.Storer
	obj := store.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode commit: %w", err)
	}
	hash, err := store.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store commit: %w", err)
	}
	return hash, nil
}

// CreateRef creates a new ref pointing to the given hash. Returns an error
// if the ref already exists.
func CreateRef(repo *git.Repository, refName string, hash plumbing.Hash) error {
	// Check if ref already exists
	_, err := repo.Storer.Reference(plumbing.ReferenceName(refName))
	if err == nil {
		return fmt.Errorf("ref already exists: %s", refName)
	}

	ref := plumbing.NewHashReference(plumbing.ReferenceName(refName), hash)
	return repo.Storer.SetReference(ref)
}

// DeleteRef removes a ref from the repository.
func DeleteRef(repo *git.Repository, refName string) error {
	return repo.Storer.RemoveReference(plumbing.ReferenceName(refName))
}

// RefExists checks whether a ref exists in the repository.
func RefExists(repo *git.Repository, refName string) bool {
	_, err := repo.Storer.Reference(plumbing.ReferenceName(refName))
	return err == nil
}

// ListRefsByPrefix returns all ref names that start with the given prefix.
func ListRefsByPrefix(repo *git.Repository, prefix string) ([]string, error) {
	var names []string
	refs, err := repo.Storer.IterReferences()
	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().String()
		if strings.HasPrefix(name, prefix) {
			// Extract the component after the prefix
			component := strings.TrimPrefix(name, prefix)
			if component != "" {
				names = append(names, component)
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, storer.ErrStop) {
		return nil, fmt.Errorf("failed to list refs: %w", err)
	}
	return names, nil
}

// DefaultSignature returns a default author/committer signature for graph
// operations.
func DefaultSignature() object.Signature {
	return object.Signature{
		Name:  "git-infra-graph",
		Email: "git-infra-graph@localhost",
		When:  time.Now(),
	}
}

// CreateBlob writes a blob object to the repository's object store and
// returns its hash.
func CreateBlob(repo *git.Repository, data []byte) (plumbing.Hash, error) {
	store := repo.Storer
	obj := store.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(data)))

	w, err := obj.Writer()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to create blob writer: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		w.Close()
		return plumbing.ZeroHash, fmt.Errorf("failed to write blob data: %w", err)
	}
	if err := w.Close(); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to close blob writer: %w", err)
	}

	hash, err := store.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store blob: %w", err)
	}
	return hash, nil
}

// ReadBlobContent reads the content of a blob object by its hash.
func ReadBlobContent(repo *git.Repository, hash plumbing.Hash) ([]byte, error) {
	blob, err := object.GetBlob(repo.Storer, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob: %w", err)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to create blob reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob content: %w", err)
	}
	return data, nil
}

// GetTreeByHash retrieves a tree object by its hash.
func GetTreeByHash(repo *git.Repository, hash plumbing.Hash) (*object.Tree, error) {
	tree, err := object.GetTree(repo.Storer, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}
	return tree, nil
}

// StoreTree encodes and stores a tree object, returning its hash.
func StoreTree(repo *git.Repository, entries []object.TreeEntry) (plumbing.Hash, error) {
	sort.Sort(object.TreeEntrySorter(entries))
	tree := &object.Tree{Entries: entries}

	store := repo.Storer
	obj := store.NewEncodedObject()
	if err := tree.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode tree: %w", err)
	}

	hash, err := store.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store tree: %w", err)
	}
	return hash, nil
}

// SetTreeEntry adds or replaces an entry in a tree and returns the new tree hash.
func SetTreeEntry(repo *git.Repository, treeHash plumbing.Hash, entry object.TreeEntry) (plumbing.Hash, error) {
	tree, err := GetTreeByHash(repo, treeHash)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	entries := make([]object.TreeEntry, 0, len(tree.Entries)+1)
	replaced := false
	for _, e := range tree.Entries {
		if e.Name == entry.Name {
			entries = append(entries, entry)
			replaced = true
		} else {
			entries = append(entries, e)
		}
	}
	if !replaced {
		entries = append(entries, entry)
	}

	return StoreTree(repo, entries)
}

// RemoveTreeEntry removes an entry from a tree and returns the new tree hash.
func RemoveTreeEntry(repo *git.Repository, treeHash plumbing.Hash, name string) (plumbing.Hash, error) {
	tree, err := GetTreeByHash(repo, treeHash)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	entries := make([]object.TreeEntry, 0, len(tree.Entries))
	found := false
	for _, e := range tree.Entries {
		if e.Name == name {
			found = true
		} else {
			entries = append(entries, e)
		}
	}

	if !found {
		return plumbing.ZeroHash, fmt.Errorf("entry %q not found in tree", name)
	}

	return StoreTree(repo, entries)
}

// WriteStagingRef creates or updates the staging ref for a graph to point to a tree hash.
func WriteStagingRef(repo *git.Repository, graphName string, treeHash plumbing.Hash) error {
	refName := "refs/infra-stage/" + graphName
	ref := plumbing.NewHashReference(plumbing.ReferenceName(refName), treeHash)
	return repo.Storer.SetReference(ref)
}

// DeleteStagingRef removes the staging ref for a graph.
func DeleteStagingRef(repo *git.Repository, graphName string) error {
	refName := "refs/infra-stage/" + graphName
	return repo.Storer.RemoveReference(plumbing.ReferenceName(refName))
}

// ResolveRootTree reads the root tree hash from the staging ref if it exists,
// otherwise from the graph ref's committed tree.
func ResolveRootTree(repo *git.Repository, graphName string) (plumbing.Hash, error) {
	// Try staging ref first
	stageRef := "refs/infra-stage/" + graphName
	ref, err := repo.Storer.Reference(plumbing.ReferenceName(stageRef))
	if err == nil {
		return ref.Hash(), nil
	}

	// Fall back to committed tree from graph ref
	graphRef := "refs/infra/" + graphName
	gRef, err := repo.Storer.Reference(plumbing.ReferenceName(graphRef))
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("graph '%s' not found", graphName)
	}

	commit, err := object.GetCommit(repo.Storer, gRef.Hash())
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to read graph commit: %w", err)
	}

	return commit.TreeHash, nil
}

// CreateCommit creates a commit with the given tree, parent hashes, message,
// and signature. parentHashes can be empty for an orphan commit.
func CreateCommit(repo *git.Repository, treeHash plumbing.Hash, parentHashes []plumbing.Hash, message string, sig object.Signature) (plumbing.Hash, error) {
	commit := &object.Commit{
		TreeHash:     treeHash,
		ParentHashes: parentHashes,
		Author:       sig,
		Committer:    sig,
		Message:      message,
	}

	store := repo.Storer
	obj := store.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to encode commit: %w", err)
	}

	hash, err := store.SetEncodedObject(obj)
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("failed to store commit: %w", err)
	}
	return hash, nil
}

// UpdateRef unconditionally creates or updates a ref to point to the given hash.
func UpdateRef(repo *git.Repository, refName string, hash plumbing.Hash) error {
	ref := plumbing.NewHashReference(plumbing.ReferenceName(refName), hash)
	return repo.Storer.SetReference(ref)
}
