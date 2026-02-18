package graph

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// stageRefPrefix is the namespace under which all staging refs are stored.
const stageRefPrefix = "refs/infra-stage/"

// NodeType represents the kind of node in the graph.
type NodeType string

const (
	// BlobNode is a leaf node containing binary content.
	BlobNode NodeType = "blob"
	// TreeNode is a container node with named children.
	TreeNode NodeType = "tree"
)

// NodeResult is the return value from a successful Put operation.
type NodeResult struct {
	ID   string   `json:"id"`   // Content-addressable hash (SHA-1)
	Path string   `json:"path"` // Full path including graph name
	Type NodeType `json:"type"` // BlobNode or TreeNode
}

// NodeContent is the return value from a successful Get operation.
type NodeContent struct {
	ID       string      `json:"id"`                 // Content-addressable hash
	Path     string      `json:"path"`               // Full path including graph name
	Type     NodeType    `json:"type"`               // BlobNode or TreeNode
	Blob     []byte      `json:"content,omitempty"`  // Raw content (blobs only)
	Children []NodeEntry `json:"children,omitempty"` // Immediate children (trees only)
}

// NodeEntry is a child descriptor within a NodeContent children listing.
type NodeEntry struct {
	Name string   `json:"name"` // Single path segment name
	Type NodeType `json:"type"` // BlobNode or TreeNode
	ID   string   `json:"id"`   // Content-addressable hash
}

// StatusChange represents a single change in the Status response.
type StatusChange struct {
	Path   string `json:"path"`   // Node path relative to graph root
	Status string `json:"status"` // "added", "modified", or "deleted"
}

// StatusResult is the return value from a successful Status operation.
type StatusResult struct {
	Graph   string         `json:"graph"`   // Graph name
	Changes []StatusChange `json:"changes"` // List of changes since last commit
}

// CommitResult is the return value from a successful Commit operation.
type CommitResult struct {
	Graph  string `json:"graph"`  // Graph name
	Commit string `json:"commit"` // Hash of the new commit
	Ref    string `json:"ref"`    // Full ref path
}

// LogEntry represents a single commit in the graph's commit history.
type LogEntry struct {
	Hash         string    `json:"hash"`         // Full 40-character commit hash
	Date         time.Time `json:"date"`         // Committer timestamp
	SourceCommit string    `json:"sourceCommit"` // Source-Commit trailer value (empty if missing)
	Author       string    `json:"author"`       // Author name
	Message      string    `json:"message"`      // Full commit message
}

// LogOptions controls the behavior of the Log function.
type LogOptions struct {
	MaxCount    int  // Maximum number of commits to return
	HasMaxCount bool // Whether MaxCount was explicitly set
}

// LogResult is the return value from a successful Log operation.
type LogResult struct {
	Entries []LogEntry `json:"entries"`           // Commits in reverse chronological order
	Warning string     `json:"warning,omitempty"` // Non-empty if chain is broken
}

// ParseNodePath splits a path into a graph name and node path segments.
// It trims leading/trailing slashes, validates the path is non-empty and
// has no empty segments, and validates the graph name.
//
// Returns the graph name, remaining path segments, and any validation error.
func ParseNodePath(path string) (graphName string, segments []string, err error) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", nil, errors.New("invalid path: empty")
	}

	parts := strings.Split(trimmed, "/")
	for _, p := range parts {
		if p == "" {
			return "", nil, fmt.Errorf("invalid path: empty segment in %q", path)
		}
	}

	if err := ValidateGraphName(parts[0]); err != nil {
		return "", nil, err
	}

	return parts[0], parts[1:], nil
}

// stageRefName returns the full staging ref path for a graph name.
func stageRefName(name string) string {
	return stageRefPrefix + name
}
