# Azure Resource Graph: How to describe the whole thing as a single graph

A good graph-theoretic description is:

Azure resources form a multi-relational directed graph (a directed multigraph) with at least two important edge types:

- Containment edges (scope / “contains”) that are nearly tree-structured
- Reference edges (configuration / attachment / association) that cross-cut the tree

So the overall structure is not a DAG (cycles can exist through references), and not a pure tree. It’s a hierarchical backbone plus cross edges—sometimes described as a tree with cross-links or a layered graph.

## Terms you can use precisely

- Containment hierarchy: “rooted tree / arborescence” (mostly) of scopes
- Cross-tree reference: “directed reference edge” or “non-hierarchical association edge”
- Whole model: “typed (multi-relational) directed graph with a containment backbone”