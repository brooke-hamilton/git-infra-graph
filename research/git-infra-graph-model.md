# Git-Native Infrastructure Graph Model

Radius will create a versioned infrastructure graph stored directly in Git native objects: node data in blobs, hierarchy in trees, snapshot versions in commits, and graph head tracking via `refs/radius`.

## Graph type

A **typed directed multigraph**:

- **Nodes** represent infrastructure entities.
- **Containment edges** represent hierarchy.
- **Reference edges** represent cross-node dependencies.

The containment subgraph is a rooted hierarchy; cross-tree references are embedded in node blob content and are resolved by application-layer parsing.

## Graph constraints

- **Containment hierarchy** is traversed top-down only (parent to child).
- **Upward traversal** is not supported as a graph edge operation.
- **No self-loops**: a node cannot reference itself.
- **Cross-tree references** are allowed: a node may reference nodes outside its containment subtree.
- **Reference edges are directed (one-way)**; a reverse relationship must be represented by a separate `ref*` blob.

## Git object mapping

- **Blob objects** store node payloads (canonical node documents).
- **Tree objects** encode containment by path (parent tree to child tree/blob entries).
- **Commit objects** capture immutable graph snapshots (one commit = one graph version).
- **Custom private refs** (for example `refs/radius`) select the active snapshot lineage.

This model uses Git storage primitives directly and keeps infrastructure-graph pointers in a dedicated ref namespace, separate from normal branch/tag usage.

## Edge encoding

- **Containment edges** are implicit in tree traversal (`root -> ... -> node blob`).
- **Reference edges** are implemented inside node blob content and require payload parsing.
- These references are not first-class graph edges in Git object topology.
- References are not understandable at the raw graph-storage level and are only navigable at the application layer.

## Capabilities (graph terms)

- **Reachability** from a ref defines the live snapshot.
- **Versioned topology** is obtained by commit-to-commit graph evolution.
- **Structural sharing** occurs automatically through content-addressed Git objects.
- **Cross-namespace commit mapping** links each Radius graph commit to a commit in the standard Git refs namespace by storing that source commit SHA in a commit trailer on the graph commit.

## Creation flow in Git terms

1. Write/update node blobs in the working tree (including embedded reference values).
2. Reference a commit from the main git tree in a commit trailer.
3. Stage with the Git index and commit the tree
4. Update the custom ref (for example `refs/radius`) to the new commit.

## Open questions

- Should there be any type information about blobs?
- Are reference edges required or should they be implemented within the nodes according to whatever type system the nodes are in. This makes the graph less descriptive, but simpler. However, putting reference edges in the graph requires knowledge of the type system being overlaid upon the graph.
