# Git-Native Infrastructure Graph Model

The infrastructure graph is a versioned graph designed to work with Infrastructure as Code files stored in a Git repo. The graph is stored directly in Git native objects: node data in blobs, hierarchy (edges) in trees, snapshot versions in commits, and graph head tracking via a custom ref path `refs/infra`.

## Graph type

A **directed multigraph**:

- **Nodes** represent infrastructure entities.
- **Containment edges** represent hierarchy.
- **Reference edges** represent cross-node dependencies.
- **The graph is untyped.** All nodes and edges carry no type information at the graph data layer. Type semantics (for example, resource type, API version, or edge classification) live exclusively in blob content and are the responsibility of the application layer.

The graph is a rooted hierarchy. Cross-tree reference edges are also possible, and can be encoded in two ways: embedded in node blob content (resolved by application-layer parsing) or as specially named ref blob objects in the tree (visible at the Git object level).

## Graph constraints

- **Containment hierarchy** is traversed top-down only (parent to child).
- **Upward traversal** is not a first-class graph edge operation. The application layer can derive reverse indexes (child-to-parent lookups) if needed, but these are not stored as edges in the graph.
- **No self-loops**: a node cannot reference itself.
- **Cross-tree references** are allowed: a node may reference nodes outside its containment subtree.
- **Reference edges are directed (one-way)**; a reverse relationship must be represented by a separate reference edge.
- **Reference edges are not indexed.** The graph layer does not maintain any index over reference edges. If indexing is required, it is the responsibility of the application layer.

## Git object mapping

- **Blob objects** store node payloads (canonical node documents).
- **Tree objects** encode containment by path (parent tree to child tree/blob entries).
- **Commit objects** capture immutable graph snapshots (one commit = one graph version).
- **Custom private refs** (for example `refs/infra`) select the active snapshot lineage.

This model uses Git storage primitives directly and keeps infrastructure-graph pointers in a dedicated ref namespace, separate from normal branch/tag usage.

## Edge encoding

- **Containment edges** are implicit in tree traversal (`root -> ... -> node blob`).
- **Reference edges** can be encoded in two ways:
  1. **Embedded in node data** – References are stored within the node blob content. These require payload parsing and are only visible to the application layer; they are not first-class edges in Git object topology.
  2. **Ref objects in the tree** – References are represented as specially named objects in the tree using Git ref-style paths. A ref object encodes an edge from a blob in the current tree node to a blob elsewhere in the graph, using a path convention such as `refs/<blob name>/` → `refs/<tree node>/<tree node …n>/<blob>`. These edges are visible at the Git storage level without parsing blob content.

## Capabilities (graph terms)

- **Reachability** from a ref defines the live snapshot.
- **Versioned topology** is obtained by commit-to-commit graph evolution.
- **Structural sharing** occurs automatically through content-addressed Git objects.
- **Cross-namespace commit mapping** links each infrastructure graph commit to any commit in the standard Git refs namespace (not limited to a specific branch) by storing that source commit SHA in a commit trailer on the graph commit. This enables co-versioning the infrastructure graph with the rest of the repository — for example, a gitops repo storing infrastructure templates and property settings.

## Creation flow in Git terms

1. Write/update node blobs in the working tree (including any embedded reference values) and/or create ref objects for tree-level reference edges.
2. Reference a commit from the standard Git refs namespace in a commit trailer.
3. Stage with the Git index and commit the tree
4. Update the custom ref (for example `refs/infra`) to the new commit.
