package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/brooke-hamilton/git-infra-graph/src/graph/internal/gitops"
	"github.com/go-git/go-git/v5/plumbing"
)

// Init creates a new named infrastructure graph in the Git repository at the
// given path. It creates an empty root tree, an orphan commit with a
// Source-Commit trailer referencing the current HEAD, and a ref at
// refs/infra/<name>.
func Init(repoPath string, name string) error {
	if err := ValidateGraphName(name); err != nil {
		return err
	}

	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return err
	}

	headHash, err := gitops.ResolveHEAD(repo)
	if err != nil {
		return err
	}

	refName := graphRefName(name)
	if gitops.RefExists(repo, refName) {
		return fmt.Errorf("graph '%s' already exists", name)
	}

	treeHash, err := gitops.CreateEmptyTree(repo)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("Initialize graph %q\n\nSource-Commit: %s\n", name, headHash.String())
	sig := gitops.DefaultSignature()

	commitHash, err := gitops.CreateOrphanCommit(repo, treeHash, message, sig)
	if err != nil {
		return err
	}

	return gitops.CreateRef(repo, refName, commitHash)
}

// InitInfo contains details about a newly created graph for JSON output.
type InitInfo struct {
	Name         string `json:"name"`
	Ref          string `json:"ref"`
	SourceCommit string `json:"sourceCommit"`
}

// GetInitInfo retrieves information about an existing graph for JSON output.
func GetInitInfo(repoPath string, name string) (*InitInfo, error) {
	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return nil, err
	}

	refName := graphRefName(name)
	ref, err := repo.Storer.Reference(plumbing.ReferenceName(refName))
	if err != nil {
		return nil, fmt.Errorf("graph '%s' not found", name)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to read commit: %w", err)
	}

	// Extract Source-Commit trailer from commit message
	sourceCommit := extractSourceCommit(commit.Message)

	return &InitInfo{
		Name:         name,
		Ref:          refName,
		SourceCommit: sourceCommit,
	}, nil
}

// extractSourceCommit parses the Source-Commit trailer from a commit message.
func extractSourceCommit(message string) string {
	const prefix = "Source-Commit: "
	for _, line := range strings.Split(message, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}

// List returns all infrastructure graphs in the Git repository at the given
// path, sorted alphabetically by name. Returns an empty slice (not an error)
// if no graphs exist.
func List(repoPath string) ([]GraphInfo, error) {
	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return nil, err
	}

	names, err := gitops.ListRefsByPrefix(repo, refPrefix)
	if err != nil {
		return nil, err
	}

	sort.Strings(names)

	graphs := make([]GraphInfo, len(names))
	for i, name := range names {
		graphs[i] = GraphInfo{Name: name}
	}
	return graphs, nil
}

// Delete removes the infrastructure graph with the given name from the Git
// repository at the given path by deleting refs/infra/<name>.
func Delete(repoPath string, name string) error {
	if err := ValidateGraphName(name); err != nil {
		return err
	}

	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return err
	}

	refName := graphRefName(name)
	if !gitops.RefExists(repo, refName) {
		return fmt.Errorf("graph '%s' not found", name)
	}

	return gitops.DeleteRef(repo, refName)
}
