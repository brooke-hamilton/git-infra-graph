package gitops

import (
	"fmt"
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
	if err != nil && err != storer.ErrStop {
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
