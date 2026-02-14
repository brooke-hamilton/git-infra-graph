package graph

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/brooke-hamilton/git-infra-graph/src/graph/internal/gitops"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/utils/merkletrie"
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
// repository at the given path by deleting refs/infra/<name> and any
// associated staging ref at refs/infra-stage/<name>.
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

	// Clean up staging ref if it exists (ignore errors — it may not exist)
	_ = gitops.DeleteStagingRef(repo, name)

	return gitops.DeleteRef(repo, refName)
}

// parentEntry holds the tree hash and child segment name at each level of the
// path walk, used to rebuild the tree chain bottom-up after a mutation.
type parentEntry struct {
	treeHash  plumbing.Hash
	childName string
}

// Put creates or replaces a node at the specified path within the graph.
// The first segment of path identifies the graph name (mapping to
// refs/infra/<name>); remaining segments address nodes within that graph's
// tree.
//
// When blob is non-nil, a blob node is created or replaced with the given
// content. When blob is nil, a tree node is created; if the tree already
// exists, this is a no-op that preserves existing children.
//
// Put automatically creates any missing intermediate tree nodes along the
// path. Changes are staged in a per-graph staging ref and are not committed
// until Commit is called.
func Put(repoPath string, path string, blob []byte) (*NodeResult, error) {
	graphName, segments, err := ParseNodePath(path)
	if err != nil {
		return nil, err
	}

	if len(segments) == 0 {
		return nil, errors.New("path must contain at least a graph name and node name")
	}

	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return nil, err
	}

	rootTreeHash, err := gitops.ResolveRootTree(repo, graphName)
	if err != nil {
		return nil, err
	}

	// Walk phase: collect parent tree stack (top-down)
	parents := make([]parentEntry, 0, len(segments))
	currentTree := rootTreeHash

	for i := 0; i < len(segments)-1; i++ {
		seg := segments[i]
		parents = append(parents, parentEntry{treeHash: currentTree, childName: seg})

		tree, err := gitops.GetTreeByHash(repo, currentTree)
		if err != nil {
			return nil, err
		}

		entry := findEntry(tree, seg)
		if entry == nil {
			// Auto-create missing intermediate tree
			emptyHash, err := gitops.CreateEmptyTree(repo)
			if err != nil {
				return nil, err
			}
			currentTree = emptyHash
		} else if entry.Mode == filemode.Dir {
			currentTree = entry.Hash
		} else {
			return nil, fmt.Errorf("blob at %q cannot have children", rebuildPath(graphName, segments[:i+1]))
		}
	}

	// Mutate phase: create/replace at target
	targetName := segments[len(segments)-1]
	targetTree, err := gitops.GetTreeByHash(repo, currentTree)
	if err != nil {
		return nil, err
	}

	existingEntry := findEntry(targetTree, targetName)

	var nodeHash plumbing.Hash
	var nodeType NodeType

	if blob != nil {
		// Creating/replacing a blob
		if existingEntry != nil && existingEntry.Mode == filemode.Dir {
			return nil, fmt.Errorf("cannot convert tree to blob at %q", rebuildPath(graphName, segments))
		}

		nodeHash, err = gitops.CreateBlob(repo, blob)
		if err != nil {
			return nil, err
		}
		nodeType = BlobNode

		currentTree, err = gitops.SetTreeEntry(repo, currentTree, object.TreeEntry{
			Name: targetName,
			Mode: filemode.Regular,
			Hash: nodeHash,
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Creating a tree (nil blob)
		if existingEntry != nil && existingEntry.Mode != filemode.Dir {
			return nil, fmt.Errorf("cannot convert blob to tree at %q", rebuildPath(graphName, segments))
		}

		if existingEntry != nil {
			// No-op: tree already exists
			nodeHash = existingEntry.Hash
		} else {
			nodeHash, err = gitops.CreateEmptyTree(repo)
			if err != nil {
				return nil, err
			}
			currentTree, err = gitops.SetTreeEntry(repo, currentTree, object.TreeEntry{
				Name: targetName,
				Mode: filemode.Dir,
				Hash: nodeHash,
			})
			if err != nil {
				return nil, err
			}
		}
		nodeType = TreeNode
	}

	// Rebuild phase: bottom-up
	childHash := currentTree
	for i := len(parents) - 1; i >= 0; i-- {
		p := parents[i]
		childHash, err = gitops.SetTreeEntry(repo, p.treeHash, object.TreeEntry{
			Name: p.childName,
			Mode: filemode.Dir,
			Hash: childHash,
		})
		if err != nil {
			return nil, err
		}
	}

	// Update staging ref
	if err := gitops.WriteStagingRef(repo, graphName, childHash); err != nil {
		return nil, err
	}

	normalizedPath := rebuildPath(graphName, segments)
	return &NodeResult{
		ID:   nodeHash.String(),
		Path: normalizedPath,
		Type: nodeType,
	}, nil
}

// findEntry looks up a tree entry by name, returning nil if not found
// or if tree is nil.
func findEntry(tree *object.Tree, name string) *object.TreeEntry {
	if tree == nil {
		return nil
	}
	for i := range tree.Entries {
		if tree.Entries[i].Name == name {
			return &tree.Entries[i]
		}
	}
	return nil
}

// rebuildPath reconstructs the full path from graph name and segments.
func rebuildPath(graphName string, segments []string) string {
	parts := make([]string, 0, 1+len(segments))
	parts = append(parts, graphName)
	parts = append(parts, segments...)
	return strings.Join(parts, "/")
}

// Get reads the node at the specified path within the graph. The first
// segment of path identifies the graph name; remaining segments address
// nodes within that graph's tree.
//
// Get reads from the current staged (uncommitted) tree if a staging ref
// exists, otherwise from the last committed tree.
//
// For blob nodes, returns NodeContent with Blob populated and Children nil.
// For tree nodes, returns NodeContent with Children populated and Blob nil.
func Get(repoPath string, path string) (*NodeContent, error) {
	graphName, segments, err := ParseNodePath(path)
	if err != nil {
		return nil, err
	}

	if len(segments) == 0 {
		return nil, errors.New("path must contain at least a graph name and node name")
	}

	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return nil, err
	}

	rootTreeHash, err := gitops.ResolveRootTree(repo, graphName)
	if err != nil {
		return nil, err
	}

	// Walk to target
	currentTree := rootTreeHash
	for i := 0; i < len(segments)-1; i++ {
		seg := segments[i]
		tree, err := gitops.GetTreeByHash(repo, currentTree)
		if err != nil {
			return nil, err
		}

		entry := findEntry(tree, seg)
		if entry == nil {
			return nil, fmt.Errorf("node not found at %q", rebuildPath(graphName, segments))
		}
		if entry.Mode != filemode.Dir {
			return nil, fmt.Errorf("blob at %q cannot be traversed", rebuildPath(graphName, segments[:i+1]))
		}
		currentTree = entry.Hash
	}

	// Look up the target
	targetName := segments[len(segments)-1]
	tree, err := gitops.GetTreeByHash(repo, currentTree)
	if err != nil {
		return nil, err
	}

	entry := findEntry(tree, targetName)
	if entry == nil {
		return nil, fmt.Errorf("node not found at %q", rebuildPath(graphName, segments))
	}

	normalizedPath := rebuildPath(graphName, segments)

	if entry.Mode == filemode.Dir {
		// Tree node: list children
		childTree, err := gitops.GetTreeByHash(repo, entry.Hash)
		if err != nil {
			return nil, err
		}

		children := make([]NodeEntry, len(childTree.Entries))
		for i, e := range childTree.Entries {
			childType := BlobNode
			if e.Mode == filemode.Dir {
				childType = TreeNode
			}
			children[i] = NodeEntry{
				Name: e.Name,
				Type: childType,
				ID:   e.Hash.String(),
			}
		}

		sort.Slice(children, func(i, j int) bool {
			return children[i].Name < children[j].Name
		})

		return &NodeContent{
			ID:       entry.Hash.String(),
			Path:     normalizedPath,
			Type:     TreeNode,
			Children: children,
		}, nil
	}

	// Blob node: read content
	blobData, err := gitops.ReadBlobContent(repo, entry.Hash)
	if err != nil {
		return nil, err
	}

	return &NodeContent{
		ID:   entry.Hash.String(),
		Path: normalizedPath,
		Type: BlobNode,
		Blob: blobData,
	}, nil
}

// DeleteNode removes the node at the specified path within the graph. The
// first segment of path identifies the graph name; remaining segments
// address nodes within that graph's tree.
//
// When the path resolves to a blob, the blob is removed. When the path
// resolves to a tree, the entire subtree is recursively removed.
//
// Changes are staged in the per-graph staging ref and are not committed
// until Commit is called. Parent trees are not automatically pruned when
// all their children are deleted.
func DeleteNode(repoPath string, path string) error {
	graphName, segments, err := ParseNodePath(path)
	if err != nil {
		return err
	}

	if len(segments) == 0 {
		return errors.New("path must contain at least a graph name and node name")
	}

	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return err
	}

	rootTreeHash, err := gitops.ResolveRootTree(repo, graphName)
	if err != nil {
		return err
	}

	// Walk phase: collect parent tree stack
	parents := make([]parentEntry, 0, len(segments))
	currentTree := rootTreeHash

	for i := 0; i < len(segments)-1; i++ {
		seg := segments[i]
		parents = append(parents, parentEntry{treeHash: currentTree, childName: seg})

		tree, err := gitops.GetTreeByHash(repo, currentTree)
		if err != nil {
			return err
		}

		entry := findEntry(tree, seg)
		if entry == nil {
			return fmt.Errorf("node not found at %q", rebuildPath(graphName, segments))
		}
		if entry.Mode != filemode.Dir {
			return fmt.Errorf("blob at %q cannot be traversed", rebuildPath(graphName, segments[:i+1]))
		}
		currentTree = entry.Hash
	}

	// Verify the target exists
	targetName := segments[len(segments)-1]
	tree, err := gitops.GetTreeByHash(repo, currentTree)
	if err != nil {
		return err
	}

	entry := findEntry(tree, targetName)
	if entry == nil {
		return fmt.Errorf("node not found at %q", rebuildPath(graphName, segments))
	}

	// Remove the entry from its parent tree
	currentTree, err = gitops.RemoveTreeEntry(repo, currentTree, targetName)
	if err != nil {
		return err
	}

	// Rebuild phase: bottom-up
	childHash := currentTree
	for i := len(parents) - 1; i >= 0; i-- {
		p := parents[i]
		childHash, err = gitops.SetTreeEntry(repo, p.treeHash, object.TreeEntry{
			Name: p.childName,
			Mode: filemode.Dir,
			Hash: childHash,
		})
		if err != nil {
			return err
		}
	}

	// Update staging ref
	return gitops.WriteStagingRef(repo, graphName, childHash)
}

// Commit persists all staged (uncommitted) changes for the specified graph
// as a new commit on the graph ref's lineage. The commit becomes the new
// tip of the graph ref.
//
// If message is empty, a default message is generated in the format:
//
//	Update graph "<name>"
//
//	Source-Commit: <HEAD>
//
// If message is provided, it replaces the default message but the
// Source-Commit trailer is still appended.
func Commit(repoPath string, graphName string, message string) (*CommitResult, error) {
	if err := ValidateGraphName(graphName); err != nil {
		return nil, err
	}

	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return nil, err
	}

	// Verify the graph exists
	graphRef := graphRefName(graphName)
	gRef, err := repo.Storer.Reference(plumbing.ReferenceName(graphRef))
	if err != nil {
		return nil, fmt.Errorf("graph '%s' not found", graphName)
	}

	// Read staging ref
	stageRef, err := repo.Storer.Reference(plumbing.ReferenceName(stageRefName(graphName)))
	if err != nil {
		return nil, fmt.Errorf("no staged changes for graph %q", graphName)
	}

	stagedTreeHash := stageRef.Hash()

	// Resolve HEAD for Source-Commit trailer
	headHash, err := gitops.ResolveHEAD(repo)
	if err != nil {
		return nil, err
	}

	// Build commit message
	if message == "" {
		message = fmt.Sprintf("Update graph %q\n\nSource-Commit: %s\n", graphName, headHash.String())
	} else {
		message = fmt.Sprintf("%s\n\nSource-Commit: %s\n", message, headHash.String())
	}

	// Create commit with previous graph commit as parent
	parentHash := gRef.Hash()
	sig := gitops.DefaultSignature()
	commitHash, err := gitops.CreateCommit(repo, stagedTreeHash, []plumbing.Hash{parentHash}, message, sig)
	if err != nil {
		return nil, err
	}

	// Update graph ref to new commit
	if err := gitops.UpdateRef(repo, graphRef, commitHash); err != nil {
		return nil, err
	}

	// Delete staging ref
	if err := gitops.DeleteStagingRef(repo, graphName); err != nil {
		return nil, err
	}

	return &CommitResult{
		Graph:  graphName,
		Commit: commitHash.String(),
		Ref:    graphRef,
	}, nil
}

// Status returns the uncommitted changes for the specified graph by
// comparing the last committed tree with the current staged tree.
//
// Returns a StatusResult with an empty Changes slice (not an error) if
// there are no uncommitted changes.
func Status(repoPath string, graphName string) (*StatusResult, error) {
	if err := ValidateGraphName(graphName); err != nil {
		return nil, err
	}

	repo, err := gitops.OpenRepo(repoPath)
	if err != nil {
		return nil, err
	}

	// Verify the graph exists and get committed tree
	graphRef := graphRefName(graphName)
	gRef, err := repo.Storer.Reference(plumbing.ReferenceName(graphRef))
	if err != nil {
		return nil, fmt.Errorf("graph '%s' not found", graphName)
	}

	commit, err := object.GetCommit(repo.Storer, gRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to read graph commit: %w", err)
	}

	committedTree, err := object.GetTree(repo.Storer, commit.TreeHash)
	if err != nil {
		return nil, fmt.Errorf("failed to read committed tree: %w", err)
	}

	// Check for staging ref
	stageRef, err := repo.Storer.Reference(plumbing.ReferenceName(stageRefName(graphName)))
	if err != nil {
		// No staging ref → no changes
		return &StatusResult{
			Graph:   graphName,
			Changes: []StatusChange{},
		}, nil
	}

	stagedTree, err := object.GetTree(repo.Storer, stageRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to read staged tree: %w", err)
	}

	// Recursively diff the two trees to produce leaf-level change paths
	statusChanges, err := diffTreesRecursive(repo.Storer, committedTree, stagedTree, "")
	if err != nil {
		return nil, fmt.Errorf("failed to diff trees: %w", err)
	}

	return &StatusResult{
		Graph:   graphName,
		Changes: statusChanges,
	}, nil
}

// diffTreesRecursive compares two trees recursively and produces leaf-level
// change entries (blobs only). prefix is the path prefix accumulated during
// recursion.
func diffTreesRecursive(store storer.EncodedObjectStorer, fromTree, toTree *object.Tree, prefix string) ([]StatusChange, error) {
	changes, err := object.DiffTree(fromTree, toTree)
	if err != nil {
		return nil, err
	}

	var result []StatusChange

	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			return nil, err
		}

		switch action {
		case merkletrie.Insert:
			entry := findEntry(toTree, change.To.Name)
			path := joinPath(prefix, change.To.Name)
			if entry != nil && entry.Mode == filemode.Dir {
				// Recurse into the new tree to report all leaf additions
				subTree, err := object.GetTree(store, entry.Hash)
				if err != nil {
					return nil, err
				}
				leafChanges, err := collectAllLeaves(store, subTree, path, "added")
				if err != nil {
					return nil, err
				}
				result = append(result, leafChanges...)
			} else {
				result = append(result, StatusChange{Path: path, Status: "added"})
			}

		case merkletrie.Delete:
			entry := findEntry(fromTree, change.From.Name)
			path := joinPath(prefix, change.From.Name)
			if entry != nil && entry.Mode == filemode.Dir {
				// Recurse into the removed tree to report all leaf deletions
				subTree, err := object.GetTree(store, entry.Hash)
				if err != nil {
					return nil, err
				}
				leafChanges, err := collectAllLeaves(store, subTree, path, "deleted")
				if err != nil {
					return nil, err
				}
				result = append(result, leafChanges...)
			} else {
				result = append(result, StatusChange{Path: path, Status: "deleted"})
			}

		case merkletrie.Modify:
			// Both sides exist. Check if they are trees (recurse) or blobs (leaf).
			fromEntry := findEntry(fromTree, change.From.Name)
			toEntry := findEntry(toTree, change.To.Name)
			path := joinPath(prefix, change.To.Name)

			fromIsTree := fromEntry != nil && fromEntry.Mode == filemode.Dir
			toIsTree := toEntry != nil && toEntry.Mode == filemode.Dir

			if fromIsTree && toIsTree {
				// Both trees → recurse
				subFrom, err := object.GetTree(store, fromEntry.Hash)
				if err != nil {
					return nil, err
				}
				subTo, err := object.GetTree(store, toEntry.Hash)
				if err != nil {
					return nil, err
				}
				subChanges, err := diffTreesRecursive(store, subFrom, subTo, path)
				if err != nil {
					return nil, err
				}
				result = append(result, subChanges...)
			} else if fromIsTree && !toIsTree {
				// Tree replaced by blob: delete all leaves, add new blob
				subFrom, err := object.GetTree(store, fromEntry.Hash)
				if err != nil {
					return nil, err
				}
				leafChanges, err := collectAllLeaves(store, subFrom, path, "deleted")
				if err != nil {
					return nil, err
				}
				result = append(result, leafChanges...)
				result = append(result, StatusChange{Path: path, Status: "added"})
			} else if !fromIsTree && toIsTree {
				// Blob replaced by tree: delete old blob, add all leaves
				result = append(result, StatusChange{Path: path, Status: "deleted"})
				subTo, err := object.GetTree(store, toEntry.Hash)
				if err != nil {
					return nil, err
				}
				leafChanges, err := collectAllLeaves(store, subTo, path, "added")
				if err != nil {
					return nil, err
				}
				result = append(result, leafChanges...)
			} else {
				// Both blobs → leaf modification
				result = append(result, StatusChange{Path: path, Status: "modified"})
			}
		}
	}

	return result, nil
}

// collectAllLeaves recursively collects all blob entries under a tree,
// returning them as StatusChange entries with the given status.
func collectAllLeaves(store storer.EncodedObjectStorer, tree *object.Tree, prefix, status string) ([]StatusChange, error) {
	var result []StatusChange
	for _, entry := range tree.Entries {
		path := joinPath(prefix, entry.Name)
		if entry.Mode == filemode.Dir {
			subTree, err := object.GetTree(store, entry.Hash)
			if err != nil {
				return nil, err
			}
			subChanges, err := collectAllLeaves(store, subTree, path, status)
			if err != nil {
				return nil, err
			}
			result = append(result, subChanges...)
		} else {
			result = append(result, StatusChange{Path: path, Status: status})
		}
	}
	return result, nil
}

// joinPath joins a prefix and name with "/", handling empty prefix.
func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "/" + name
}
